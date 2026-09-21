package server

import (
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"

	"github.com/Agentrix-Group/Agentrix/src/auth"
	"github.com/Agentrix-Group/Agentrix/src/service"
)

type Server struct {
	Handler        http.Handler
	Service        service.Service
	SessionStore   *sessions.CookieStore
	Auth           *auth.Auth
	SessionManager *auth.SessionManager
	RateLimiter    *RateLimiter
	AllowedOrigins []string
}

func NewServer(svc service.Service) *Server {
	sessionSecret := os.Getenv("SESSION_SECRET")
	if sessionSecret == "" {
		sessionSecret = "agentrix-session-secret-key-change-in-prod"
	}

	accessSecret := os.Getenv("ACCESS_SECRET")
	if accessSecret == "" {
		accessSecret = "agentrix-access-secret-key-change-in-prod"
	}

	refreshSecret := os.Getenv("REFRESH_SECRET")
	if refreshSecret == "" {
		refreshSecret = "agentrix-refresh-secret-key-change-in-prod"
	}

	authMod := auth.NewAuth(accessSecret, refreshSecret)

	var sessionMgr *auth.SessionManager
	if svc != nil {
		func() {
			defer func() { _ = recover() }()
			if repo := svc.GetRepository(); repo != nil {
				sessionMgr = auth.NewSessionManager(repo, authMod)
			}
		}()
	}

	allowedOriginsStr := os.Getenv("CORS_ALLOWED_ORIGINS")
	var allowedOrigins []string
	if allowedOriginsStr != "" {
		for _, o := range strings.Split(allowedOriginsStr, ",") {
			o = strings.TrimSpace(o)
			if o != "" {
				allowedOrigins = append(allowedOrigins, o)
			}
		}
	}
	if len(allowedOrigins) == 0 {
		allowedOrigins = []string{
			"http://localhost:3000",
			"http://localhost:5173",
			"http://127.0.0.1:3000",
			"http://127.0.0.1:5173",
		}
	}

	s := &Server{
		Service:        svc,
		SessionStore:   sessions.NewCookieStore([]byte(sessionSecret)),
		Auth:           authMod,
		SessionManager: sessionMgr,
		RateLimiter:    NewRateLimiter(60, time.Minute),
		AllowedOrigins: allowedOrigins,
	}
	s.Handler = s.buildHandler()
	return s
}

func (s *Server) buildHandler() http.Handler {
	router := mux.NewRouter()
	router.Use(s.correlationMiddleware)
	router.Use(s.corsMiddleware)
	router.Use(s.requestLoggingMiddleware)

	// Health & Readiness Probes (Phase 1 / ATD-020)
	router.HandleFunc("/health/live", s.livenessProbe).Methods(http.MethodGet, http.MethodOptions)
	router.HandleFunc("/health/ready", s.readinessProbe).Methods(http.MethodGet, http.MethodOptions)
	router.HandleFunc("/health", s.livenessProbe).Methods(http.MethodGet, http.MethodOptions)

	api := router.PathPrefix("/api/v1").Subrouter()

	api.HandleFunc("/health/live", s.livenessProbe).Methods(http.MethodGet, http.MethodOptions)
	api.HandleFunc("/health/ready", s.readinessProbe).Methods(http.MethodGet, http.MethodOptions)
	api.HandleFunc("/health", s.livenessProbe).Methods(http.MethodGet, http.MethodOptions)

	// Public auth endpoints protected with rate limiting
	authLimiter := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if s.RateLimiter != nil {
				s.RateLimiter.Middleware(next).ServeHTTP(w, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
	api.Handle("/login", authLimiter(http.HandlerFunc(s.login))).Methods(http.MethodPost, http.MethodOptions)
	api.Handle("/register", authLimiter(http.HandlerFunc(s.register))).Methods(http.MethodPost, http.MethodOptions)
	api.Handle("/refresh", authLimiter(http.HandlerFunc(s.refreshToken))).Methods(http.MethodPost, http.MethodOptions)
	api.Handle("/auth/login", authLimiter(http.HandlerFunc(s.login))).Methods(http.MethodPost, http.MethodOptions)
	api.Handle("/auth/register", authLimiter(http.HandlerFunc(s.register))).Methods(http.MethodPost, http.MethodOptions)
	api.Handle("/auth/refresh", authLimiter(http.HandlerFunc(s.refreshToken))).Methods(http.MethodPost, http.MethodOptions)
	api.HandleFunc("/auth/logout", s.logout).Methods(http.MethodPost, http.MethodOptions)
	api.HandleFunc("/auth/logout-all", s.logoutAll).Methods(http.MethodPost, http.MethodOptions)
	api.HandleFunc("/logout", s.logout).Methods(http.MethodPost, http.MethodOptions)

	api.HandleFunc("/contests", s.listPublicContests).Methods(http.MethodGet, http.MethodOptions)
	api.HandleFunc("/contests/{id}", s.getPublicContest).Methods(http.MethodGet, http.MethodOptions)
	api.HandleFunc("/contests/{id}/agents", s.listContestAgents).Methods(http.MethodGet, http.MethodOptions)
	api.HandleFunc("/contests/{id}/entries", s.listContestEntries).Methods(http.MethodGet, http.MethodOptions)
	api.HandleFunc("/contests/{id}/rankings", s.listContestRankings).Methods(http.MethodGet, http.MethodOptions)
	api.HandleFunc("/contests/{id}/rankings/snapshots", s.listRankingSnapshots).Methods(http.MethodGet, http.MethodOptions)
	api.HandleFunc("/contests/{id}/rankings/snapshots/{version}", s.getRankingSnapshot).Methods(http.MethodGet, http.MethodOptions)

	// Public spectator routes (Blueprint Section 4: matches, rankings, replays)
	api.HandleFunc("/matches", s.listMatches).Methods(http.MethodGet, http.MethodOptions)
	api.HandleFunc("/matches/{id}", s.getMatch).Methods(http.MethodGet, http.MethodOptions)
	api.HandleFunc("/rankings", s.listRankings).Methods(http.MethodGet, http.MethodOptions)
	api.HandleFunc("/rankings/{id}", s.getRanking).Methods(http.MethodGet, http.MethodOptions)
	api.HandleFunc("/replays/{id}", s.getReplay).Methods(http.MethodGet, http.MethodOptions)
	api.HandleFunc("/replays/{id}/stream", s.streamReplay).Methods(http.MethodGet, http.MethodOptions)

	// Protected routes
	protected := api.NewRoute().Subrouter()
	protected.Use(s.authMiddleware)
	protected.Use(s.permissionMiddleware)

	// User session
	protected.HandleFunc("/me", s.checkSession).Methods(http.MethodGet, http.MethodOptions)
	protected.HandleFunc("/me/agents", s.getMyAgents).Methods(http.MethodGet, http.MethodOptions)

	// Admin panel
	protected.PathPrefix("/admin").HandlerFunc(s.adminOverview).Methods(http.MethodGet, http.MethodPost, http.MethodPut, http.MethodOptions)

	// Users (Canonical Identity)
	protected.HandleFunc("/users", s.listUsers).Methods(http.MethodGet)
	protected.HandleFunc("/users/{id}", s.getUser).Methods(http.MethodGet)
	protected.HandleFunc("/users/{id}", s.updateUser).Methods(http.MethodPut)
	protected.HandleFunc("/users/{id}", s.activateUser).Methods(http.MethodPatch)

	// Contests
	protected.HandleFunc("/contests", s.createContest).Methods(http.MethodPost)
	protected.HandleFunc("/contests/{id}", s.updateContest).Methods(http.MethodPut)
	protected.HandleFunc("/contests/{id}", s.activateContest).Methods(http.MethodPatch)
	protected.HandleFunc("/contests/{id}/agents", s.enrollAgent).Methods(http.MethodPost)
	protected.HandleFunc("/contests/{id}/entries", s.enrollAgent).Methods(http.MethodPost)
	protected.HandleFunc("/contests/{id}/rankings/recalculate", s.recalculateRankings).Methods(http.MethodPost)
	protected.HandleFunc("/contests/{id}/rankings/publish", s.publishRankingSnapshot).Methods(http.MethodPost)

	// Categories
	protected.HandleFunc("/categories", s.listCategories).Methods(http.MethodGet)
	protected.HandleFunc("/categories", s.createCategory).Methods(http.MethodPost)
	protected.HandleFunc("/categories/{id}", s.getCategory).Methods(http.MethodGet)
	protected.HandleFunc("/categories/{id}", s.updateCategory).Methods(http.MethodPut)
	protected.HandleFunc("/categories/{id}", s.activateCategory).Methods(http.MethodPatch)

	// Games
	protected.HandleFunc("/games", s.listGames).Methods(http.MethodGet)
	protected.HandleFunc("/games/{id}", s.getGame).Methods(http.MethodGet)

	// Agents
	protected.HandleFunc("/agents", s.listAgents).Methods(http.MethodGet)
	protected.HandleFunc("/agents", s.createAgent).Methods(http.MethodPost)
	protected.HandleFunc("/agents/{id}", s.getAgent).Methods(http.MethodGet)
	protected.HandleFunc("/agents/{id}", s.updateAgent).Methods(http.MethodPut)
	protected.HandleFunc("/agents/{id}", s.activateAgent).Methods(http.MethodPatch)

	// Submissions
	protected.HandleFunc("/submissions", s.listSubmissions).Methods(http.MethodGet)
	protected.HandleFunc("/submissions/upload", s.uploadSubmissionBundle).Methods(http.MethodPost)
	protected.HandleFunc("/submissions/{id}", s.getSubmission).Methods(http.MethodGet)

	// Matches (Scheduling & Execution)
	protected.HandleFunc("/matches", s.createMatch).Methods(http.MethodPost)
	protected.HandleFunc("/matches/{id}", s.updateMatch).Methods(http.MethodPut)
	protected.HandleFunc("/matches/{id}", s.activateMatch).Methods(http.MethodPatch)
	protected.HandleFunc("/matches/{id}/run", s.runMatch).Methods(http.MethodPost)

	// Results
	protected.HandleFunc("/results", s.listResults).Methods(http.MethodGet)
	protected.HandleFunc("/results/{id}", s.getResult).Methods(http.MethodGet)

	return router
}
