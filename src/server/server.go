// Package server is the HTTP layer: routing, authentication, the route
// policy table, request validation and DTO presentation. Business rules and
// horizontal authorization live in the service package.
package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/config"
	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/Agentrix-Group/Agentrix/src/service"
	"github.com/Agentrix-Group/Agentrix/src/tracer"
	"github.com/google/uuid"
)

// Policy is the authorization rule of a route. Every route declares one;
// there is no default.
type Policy struct {
	// Public routes are reachable without evaluating capabilities (health
	// probes and the session endpoints that establish identity).
	Public bool
	// Capabilities: the principal needs at least one of them. Anonymous
	// visitors hold the capabilities of the spectator role. Horizontal
	// ownership is checked by the service ("own" vs "any").
	Capabilities []model.Capability
	// Authenticated requires a signed-in user in addition to the capability.
	Authenticated bool
}

func PublicPolicy() Policy { return Policy{Public: true} }

func Requires(c ...model.Capability) Policy { return Policy{Capabilities: c} }

func SignedIn(c ...model.Capability) Policy { return Policy{Capabilities: c, Authenticated: true} }

func (p Policy) allows(principal *model.Principal) bool {
	for _, c := range p.Capabilities {
		if principal.Can(c) {
			return true
		}
	}
	return false
}

type Route struct {
	Method  string
	Pattern string
	Policy  Policy
	Handler http.HandlerFunc
	// CookieAuth marks endpoints authenticated by the refresh cookie; they
	// require the X-Requested-With header as CSRF defense.
	CookieAuth bool
}

type Server struct {
	svc      *service.Service
	cfg      *config.Config
	limiters map[string]*RateLimiter
	routes   []Route
	handler  http.Handler
}

const (
	RefreshCookieName = "agentrix_refresh"
	CSRFHeader        = "X-Requested-With"
	CSRFHeaderValue   = "agentrix"
	maxJSONBody       = 64 << 10
)

func New(svc *service.Service, cfg *config.Config) *Server {
	limits := cfg.HTTP.RateLimits
	if limits.Login == 0 {
		limits = config.DefaultRateLimits()
	}
	s := &Server{svc: svc, cfg: cfg, limiters: map[string]*RateLimiter{
		"login":    NewRateLimiter(limits.Login, time.Minute, 10000),
		"register": NewRateLimiter(limits.Register, time.Minute, 10000),
		"refresh":  NewRateLimiter(limits.Refresh, time.Minute, 10000),
	}}
	s.routes = s.buildRoutes()
	mux := http.NewServeMux()
	for _, route := range s.routes {
		route := route
		mux.Handle(route.Method+" "+route.Pattern, s.authorize(route))
	}
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, r, model.NotFound("route_not_found", "route not found"))
	})
	s.handler = s.recoverer(s.requestContext(s.cors(mux)))
	return s
}

func (s *Server) Handler() http.Handler { return s.handler }

// Routes exposes the route table (policy coverage and OpenAPI contract tests).
func (s *Server) Routes() []Route { return append([]Route(nil), s.routes...) }

// Close stops background goroutines of the rate limiters.
func (s *Server) Close() {
	for _, l := range s.limiters {
		l.Stop()
	}
}

func (s *Server) buildRoutes() []Route {
	return []Route{
		{Method: "GET", Pattern: "/health/live", Policy: PublicPolicy(), Handler: s.live},
		{Method: "GET", Pattern: "/health/ready", Policy: PublicPolicy(), Handler: s.ready},

		{Method: "POST", Pattern: "/api/v1/auth/register", Policy: PublicPolicy(), Handler: s.limit("register", s.register)},
		{Method: "POST", Pattern: "/api/v1/auth/login", Policy: PublicPolicy(), Handler: s.limit("login", s.login)},
		{Method: "POST", Pattern: "/api/v1/auth/refresh", Policy: PublicPolicy(), Handler: s.limit("refresh", s.refresh), CookieAuth: true},
		{Method: "POST", Pattern: "/api/v1/auth/logout", Policy: PublicPolicy(), Handler: s.logout, CookieAuth: true},
		{Method: "POST", Pattern: "/api/v1/auth/logout-all", Policy: SignedIn(model.CapUsersReadOwn), Handler: s.logoutAll},

		{Method: "GET", Pattern: "/api/v1/me", Policy: SignedIn(model.CapUsersReadOwn), Handler: s.me},
		{Method: "GET", Pattern: "/api/v1/users", Policy: SignedIn(model.CapUsersReadAny), Handler: s.listUsers},
		{Method: "POST", Pattern: "/api/v1/users", Policy: SignedIn(model.CapUsersRolesManage), Handler: s.createUser},
		{Method: "GET", Pattern: "/api/v1/users/{id}", Policy: SignedIn(model.CapUsersReadOwn, model.CapUsersReadAny), Handler: s.getUser},
		{Method: "PATCH", Pattern: "/api/v1/users/{id}", Policy: SignedIn(model.CapUsersUpdateOwn, model.CapUsersUpdateAny), Handler: s.updateUser},
		{Method: "PUT", Pattern: "/api/v1/users/{id}/roles", Policy: SignedIn(model.CapUsersRolesManage), Handler: s.replaceRoles},
		{Method: "PUT", Pattern: "/api/v1/users/{id}/status", Policy: SignedIn(model.CapUsersUpdateAny), Handler: s.setUserStatus},

		{Method: "GET", Pattern: "/api/v1/games", Policy: Requires(model.CapContestsView), Handler: s.listGames},
		{Method: "GET", Pattern: "/api/v1/games/{id}", Policy: Requires(model.CapContestsView), Handler: s.getGame},

		{Method: "GET", Pattern: "/api/v1/agents", Policy: SignedIn(model.CapAgentsReadOwn, model.CapAgentsReadAny), Handler: s.listAgents},
		{Method: "POST", Pattern: "/api/v1/agents", Policy: SignedIn(model.CapAgentsCreate), Handler: s.createAgent},
		{Method: "GET", Pattern: "/api/v1/agents/{id}", Policy: SignedIn(model.CapAgentsReadOwn, model.CapAgentsReadAny), Handler: s.getAgent},
		{Method: "PATCH", Pattern: "/api/v1/agents/{id}", Policy: SignedIn(model.CapAgentsUpdateOwn, model.CapAgentsUpdateAny), Handler: s.updateAgent},
		{Method: "PUT", Pattern: "/api/v1/agents/{id}/status", Policy: SignedIn(model.CapAgentsUpdateOwn, model.CapAgentsUpdateAny), Handler: s.setAgentStatus},
		{Method: "GET", Pattern: "/api/v1/agents/{id}/submissions", Policy: SignedIn(model.CapSubmissionsReadOwn, model.CapSubmissionsReadAny), Handler: s.listSubmissions},
		{Method: "POST", Pattern: "/api/v1/agents/{id}/submissions", Policy: SignedIn(model.CapSubmissionsCreate), Handler: s.uploadSubmission},
		{Method: "GET", Pattern: "/api/v1/submissions/{id}", Policy: SignedIn(model.CapSubmissionsReadOwn, model.CapSubmissionsReadAny), Handler: s.getSubmission},
		{Method: "POST", Pattern: "/api/v1/submissions/{id}/disable", Policy: SignedIn(model.CapAgentsUpdateOwn, model.CapAgentsUpdateAny), Handler: s.disableSubmission},

		{Method: "GET", Pattern: "/api/v1/contests", Policy: Requires(model.CapContestsView), Handler: s.listContests},
		{Method: "POST", Pattern: "/api/v1/contests", Policy: SignedIn(model.CapContestsManage), Handler: s.createContest},
		{Method: "GET", Pattern: "/api/v1/contests/{id}", Policy: Requires(model.CapContestsView), Handler: s.getContest},
		{Method: "PATCH", Pattern: "/api/v1/contests/{id}", Policy: SignedIn(model.CapContestsManage), Handler: s.updateContest},
		{Method: "POST", Pattern: "/api/v1/contests/{id}/transitions", Policy: SignedIn(model.CapContestsManage), Handler: s.transitionContest},
		{Method: "GET", Pattern: "/api/v1/contests/{id}/entries", Policy: Requires(model.CapContestsView), Handler: s.listEntries},
		{Method: "POST", Pattern: "/api/v1/contests/{id}/entries", Policy: SignedIn(model.CapEntriesCreateOwn, model.CapEntriesManageAny), Handler: s.enroll},
		{Method: "PUT", Pattern: "/api/v1/contests/{id}/entries/{entryId}/submission", Policy: SignedIn(model.CapEntriesCreateOwn, model.CapEntriesManageAny), Handler: s.resubmitEntry},
		{Method: "POST", Pattern: "/api/v1/contests/{id}/entries/{entryId}/withdraw", Policy: SignedIn(model.CapEntriesCreateOwn, model.CapEntriesManageAny), Handler: s.withdrawEntry},
		{Method: "POST", Pattern: "/api/v1/contests/{id}/entries/{entryId}/disqualify", Policy: SignedIn(model.CapEntriesManageAny), Handler: s.disqualifyEntry},
		{Method: "GET", Pattern: "/api/v1/contests/{id}/rankings", Policy: Requires(model.CapRankingsView), Handler: s.getRankings},
		{Method: "POST", Pattern: "/api/v1/contests/{id}/rankings/recalculate", Policy: SignedIn(model.CapRankingsPublish), Handler: s.recalculateRankings},
		{Method: "GET", Pattern: "/api/v1/contests/{id}/rankings/snapshots", Policy: Requires(model.CapRankingsView), Handler: s.listSnapshots},
		{Method: "POST", Pattern: "/api/v1/contests/{id}/rankings/snapshots", Policy: SignedIn(model.CapRankingsPublish), Handler: s.publishSnapshot},
		{Method: "GET", Pattern: "/api/v1/contests/{id}/rankings/snapshots/{version}", Policy: Requires(model.CapRankingsView), Handler: s.getSnapshot},

		{Method: "GET", Pattern: "/api/v1/matches", Policy: Requires(model.CapMatchesView), Handler: s.listMatches},
		{Method: "POST", Pattern: "/api/v1/matches", Policy: SignedIn(model.CapMatchesCreate), Handler: s.createMatch},
		{Method: "GET", Pattern: "/api/v1/matches/{id}", Policy: Requires(model.CapMatchesView), Handler: s.getMatch},
		{Method: "POST", Pattern: "/api/v1/matches/{id}/runs", Policy: SignedIn(model.CapMatchesRun), Handler: s.scheduleRun},
		{Method: "POST", Pattern: "/api/v1/matches/{id}/cancel", Policy: SignedIn(model.CapMatchesCancel), Handler: s.cancelMatch},

		{Method: "GET", Pattern: "/api/v1/replays/{id}", Policy: Requires(model.CapReplaysView), Handler: s.getReplay},
		{Method: "GET", Pattern: "/api/v1/replays/{id}/stream", Policy: Requires(model.CapReplaysView), Handler: s.streamReplay},

		{Method: "GET", Pattern: "/api/v1/admin/readiness", Policy: SignedIn(model.CapAdminAccess), Handler: s.adminReadiness},
	}
}

// ---------------------------------------------------------------------------
// Middleware
// ---------------------------------------------------------------------------

type principalKey struct{}

func principalFrom(r *http.Request) model.Principal {
	p, _ := r.Context().Value(principalKey{}).(*model.Principal)
	if p == nil {
		return model.Principal{}
	}
	return *p
}

// authorize resolves the principal and enforces the route policy before the
// handler runs. A request with an invalid token is rejected, never
// downgraded to anonymous.
func (s *Server) authorize(route Route) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if route.CookieAuth && r.Header.Get(CSRFHeader) != CSRFHeaderValue {
			writeError(w, r, model.Forbidden("csrf_header_missing", "the %s header is required", CSRFHeader))
			return
		}
		bearer := ""
		if header := r.Header.Get("Authorization"); header != "" {
			token, ok := strings.CutPrefix(header, "Bearer ")
			if !ok || token == "" {
				writeError(w, r, model.Unauthorized("invalid_token", "Authorization header must be 'Bearer <token>'"))
				return
			}
			bearer = token
		}
		principal, err := s.svc.Principal(r.Context(), bearer)
		if err != nil {
			if !route.Policy.Public {
				writeError(w, r, err)
				return
			}
			// Public routes (login, refresh, logout, health) never depend on
			// a stale bearer: they proceed as anonymous.
			principal = &model.Principal{}
		}
		if !route.Policy.Public {
			if route.Policy.Authenticated && principal.Anonymous() {
				writeError(w, r, model.Unauthorized("authentication_required", "authentication is required"))
				return
			}
			if !route.Policy.allows(principal) {
				if principal.Anonymous() {
					writeError(w, r, model.Unauthorized("authentication_required", "authentication is required"))
				} else {
					writeError(w, r, model.Forbidden("missing_capability", "missing capability %s", route.Policy.Capabilities[0]))
				}
				return
			}
		}
		ctx := context.WithValue(r.Context(), principalKey{}, principal)
		if principal.UserID != "" {
			ctx = tracer.WithActorID(ctx, principal.UserID)
		}
		route.Handler(w, r.WithContext(ctx))
	})
}

func (s *Server) requestContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-Id")
		if !safeRequestID(id) {
			id = uuid.NewString()
		}
		ctx := tracer.BeginRequest(r.Context(), id)
		w.Header().Set("X-Request-Id", id)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-store")
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		start := time.Now()
		next.ServeHTTP(recorder, r.WithContext(ctx))
		tracer.CompleteRequest(ctx, r.Method, r.Pattern, recorder.status, time.Since(start))
	})
}

func (s *Server) recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				if v == http.ErrAbortHandler {
					panic(v)
				}
				tracer.ErrorEvent(r.Context(), tracer.ScopeHTTP, "http.panic", "Handler panic",
					tracer.Origin(tracer.OriginPlatform), tracer.String("panic", fmt.Sprint(v)))
				writeError(w, r, errors.New("internal panic"))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (s *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			if !slices.Contains(s.cfg.HTTP.AllowedOrigins, origin) {
				if r.Method == http.MethodOptions {
					w.WriteHeader(http.StatusForbidden)
					return
				}
			} else {
				h := w.Header()
				h.Set("Access-Control-Allow-Origin", origin)
				h.Set("Access-Control-Allow-Credentials", "true")
				h.Add("Vary", "Origin")
				h.Set("Access-Control-Expose-Headers", "X-Request-Id, Idempotency-Replayed")
				if r.Method == http.MethodOptions {
					h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, OPTIONS")
					h.Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Idempotency-Key, X-Request-Id, "+CSRFHeader)
					h.Set("Access-Control-Max-Age", "600")
					w.WriteHeader(http.StatusNoContent)
					return
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (r *statusRecorder) WriteHeader(code int) {
	if !r.wroteHeader {
		r.status, r.wroteHeader = code, true
		r.ResponseWriter.WriteHeader(code)
	}
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	return r.ResponseWriter.Write(b)
}

func (r *statusRecorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func safeRequestID(v string) bool {
	if v == "" || len(v) > 64 {
		return false
	}
	for _, c := range v {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || strings.ContainsRune("-_.:", c)) {
			return false
		}
	}
	return true
}

// clientIP returns the client address. X-Forwarded-For is honored only when
// the direct peer is a configured trusted proxy; the right-most untrusted
// hop is used.
func (s *Server) clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	if !s.trusted(host) {
		return host
	}
	hops := strings.Split(r.Header.Get("X-Forwarded-For"), ",")
	for i := len(hops) - 1; i >= 0; i-- {
		hop := strings.TrimSpace(hops[i])
		if hop == "" || net.ParseIP(hop) == nil {
			continue
		}
		if !s.trusted(hop) {
			return hop
		}
	}
	return host
}

func (s *Server) trusted(addr string) bool {
	ip := net.ParseIP(addr)
	if ip == nil {
		return false
	}
	for _, network := range s.cfg.HTTP.TrustedProxies {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

func (s *Server) limit(name string, next http.HandlerFunc) http.HandlerFunc {
	limiter := s.limiters[name]
	return func(w http.ResponseWriter, r *http.Request) {
		if ok, retry := limiter.Allow(s.clientIP(r)); !ok {
			w.Header().Set("Retry-After", fmt.Sprint(int(retry.Seconds())+1))
			writeJSON(w, http.StatusTooManyRequests, errorBody(r, "rate_limited", "too many requests, retry later", nil))
			return
		}
		next(w, r)
	}
}

// ---------------------------------------------------------------------------
// JSON helpers and error mapping
// ---------------------------------------------------------------------------

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if body != nil {
		_ = json.NewEncoder(w).Encode(body)
	}
}

type errorEnvelope struct {
	Error errorPayload `json:"error"`
}

type errorPayload struct {
	Code      string         `json:"code"`
	Message   string         `json:"message"`
	Details   map[string]any `json:"details,omitempty"`
	RequestID string         `json:"request_id"`
}

func errorBody(r *http.Request, code, message string, details map[string]any) errorEnvelope {
	return errorEnvelope{Error: errorPayload{Code: code, Message: message, Details: details, RequestID: tracer.RequestID(r.Context())}}
}

var kindStatus = map[model.Kind]int{
	model.KindValidation:   http.StatusUnprocessableEntity,
	model.KindBadRequest:   http.StatusBadRequest,
	model.KindNotFound:     http.StatusNotFound,
	model.KindConflict:     http.StatusConflict,
	model.KindTransition:   http.StatusConflict,
	model.KindForbidden:    http.StatusForbidden,
	model.KindUnauthorized: http.StatusUnauthorized,
	model.KindUnavailable:  http.StatusServiceUnavailable,
}

// writeError maps domain errors to their status; anything else is an
// unexpected infrastructure failure: logged with detail, answered with a
// generic 500 that leaks no SQL, paths or internals.
func writeError(w http.ResponseWriter, r *http.Request, err error) {
	if domain, ok := model.AsError(err); ok {
		status := kindStatus[domain.Kind]
		if status == 0 {
			status = http.StatusInternalServerError
		}
		writeJSON(w, status, errorBody(r, domain.Code, domain.Message, domain.Details))
		return
	}
	if errors.Is(err, context.Canceled) {
		return
	}
	tracer.ErrorEvent(r.Context(), tracer.ScopeHTTP, "http.internal_error", "Unexpected error",
		tracer.Origin(tracer.OriginInfrastructure), tracer.Err(err))
	writeJSON(w, http.StatusInternalServerError, errorBody(r, "internal_error", "unexpected server error", nil))
}

// decodeJSON reads a bounded JSON body strictly (unknown fields rejected).
func decodeJSON(r *http.Request, dst any) error {
	if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		return model.BadRequest("unsupported_media_type", "Content-Type must be application/json")
	}
	decoder := json.NewDecoder(io.LimitReader(r.Body, maxJSONBody+1))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return model.BadRequest("invalid_json", "request body is not valid: %v", sanitizeJSONError(err))
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return model.BadRequest("invalid_json", "request body must contain a single JSON object")
	}
	return nil
}

func sanitizeJSONError(err error) string {
	msg := err.Error()
	if len(msg) > 200 {
		msg = msg[:200]
	}
	return msg
}
