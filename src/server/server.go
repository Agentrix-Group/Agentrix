package server

import (
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"

	"github.com/F4nk1/Agentrix/src/auth"
	"github.com/F4nk1/Agentrix/src/service"
)

type Server struct {
	Handler      http.Handler
	Service      service.Service
	SessionStore *sessions.CookieStore
	Auth         *auth.Auth
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

	s := &Server{
		Service:      svc,
		SessionStore: sessions.NewCookieStore([]byte(sessionSecret)),
		Auth:         auth.NewAuth(accessSecret, refreshSecret),
	}
	s.Handler = s.buildHandler()
	return s
}

func (s *Server) buildHandler() http.Handler {
	router := mux.NewRouter()
	router.Use(s.correlationMiddleware)
	router.Use(s.corsMiddleware)
	router.Use(s.requestLoggingMiddleware)

	api := router.PathPrefix("/api/v1").Subrouter()

	// Public endpoints
	api.HandleFunc("/login", s.login).Methods(http.MethodPost, http.MethodOptions)
	api.HandleFunc("/register", s.register).Methods(http.MethodPost, http.MethodOptions)
	api.HandleFunc("/refresh", s.refreshToken).Methods(http.MethodPost, http.MethodOptions)
	api.HandleFunc("/auth/login", s.login).Methods(http.MethodPost, http.MethodOptions)
	api.HandleFunc("/auth/register", s.register).Methods(http.MethodPost, http.MethodOptions)
	api.HandleFunc("/auth/refresh", s.refreshToken).Methods(http.MethodPost, http.MethodOptions)
	api.HandleFunc("/contests", s.listPublicContests).Methods(http.MethodGet, http.MethodOptions)
	api.HandleFunc("/contests/{id}", s.getPublicContest).Methods(http.MethodGet, http.MethodOptions)
	api.HandleFunc("/contests/{id}/agents", s.listContestAgents).Methods(http.MethodGet, http.MethodOptions)

	// Public spectator routes (Blueprint Section 4: matches, rankings, replays)
	api.HandleFunc("/matches", s.listMatches).Methods(http.MethodGet, http.MethodOptions)
	api.HandleFunc("/matches/{id}", s.getMatch).Methods(http.MethodGet, http.MethodOptions)
	api.HandleFunc("/rankings", s.listRankings).Methods(http.MethodGet, http.MethodOptions)
	api.HandleFunc("/replays/{id}", s.getReplay).Methods(http.MethodGet, http.MethodOptions)

	// Protected routes
	protected := api.NewRoute().Subrouter()
	protected.Use(s.authMiddleware)
	protected.Use(s.permissionMiddleware)

	// User session
	protected.HandleFunc("/me", s.checkSession).Methods(http.MethodGet, http.MethodOptions)

	// Participants
	protected.HandleFunc("/participants", s.listParticipants).Methods(http.MethodGet)
	protected.HandleFunc("/participants/{id}", s.getParticipant).Methods(http.MethodGet)
	protected.HandleFunc("/participants/{id}", s.updateParticipant).Methods(http.MethodPut)
	protected.HandleFunc("/participants/{id}", s.activateParticipant).Methods(http.MethodPatch)

	// Contests
	protected.HandleFunc("/contests", s.createContest).Methods(http.MethodPost)
	protected.HandleFunc("/contests/{id}", s.updateContest).Methods(http.MethodPut)
	protected.HandleFunc("/contests/{id}", s.activateContest).Methods(http.MethodPatch)
	protected.HandleFunc("/contests/{id}/agents", s.enrollAgent).Methods(http.MethodPost)

	// Categories
	protected.HandleFunc("/categories", s.listCategories).Methods(http.MethodGet)
	protected.HandleFunc("/categories", s.createCategory).Methods(http.MethodPost)
	protected.HandleFunc("/categories/{id}", s.getCategory).Methods(http.MethodGet)
	protected.HandleFunc("/categories/{id}", s.updateCategory).Methods(http.MethodPut)
	protected.HandleFunc("/categories/{id}", s.activateCategory).Methods(http.MethodPatch)

	// Games
	protected.HandleFunc("/games", s.listGames).Methods(http.MethodGet)
	protected.HandleFunc("/games", s.createGame).Methods(http.MethodPost)
	protected.HandleFunc("/games/{id}", s.getGame).Methods(http.MethodGet)
	protected.HandleFunc("/games/{id}", s.updateGame).Methods(http.MethodPut)
	protected.HandleFunc("/games/{id}", s.activateGame).Methods(http.MethodPatch)

	// Agents
	protected.HandleFunc("/agents", s.listAgents).Methods(http.MethodGet)
	protected.HandleFunc("/agents", s.createAgent).Methods(http.MethodPost)
	protected.HandleFunc("/agents/{id}", s.getAgent).Methods(http.MethodGet)
	protected.HandleFunc("/agents/{id}", s.updateAgent).Methods(http.MethodPut)
	protected.HandleFunc("/agents/{id}", s.activateAgent).Methods(http.MethodPatch)

	// Submissions
	protected.HandleFunc("/submissions", s.listSubmissions).Methods(http.MethodGet)
	protected.HandleFunc("/submissions", s.createSubmission).Methods(http.MethodPost)
	protected.HandleFunc("/submissions/{id}", s.getSubmission).Methods(http.MethodGet)
	protected.HandleFunc("/submissions/{id}", s.updateSubmission).Methods(http.MethodPut)
	protected.HandleFunc("/submissions/{id}", s.activateSubmission).Methods(http.MethodPatch)

	// Matches (Scheduling & Execution)
	protected.HandleFunc("/matches", s.createMatch).Methods(http.MethodPost)
	protected.HandleFunc("/matches/{id}", s.updateMatch).Methods(http.MethodPut)
	protected.HandleFunc("/matches/{id}", s.activateMatch).Methods(http.MethodPatch)
	protected.HandleFunc("/matches/{id}/run", s.runMatch).Methods(http.MethodPost)

	// Results
	protected.HandleFunc("/results", s.listResults).Methods(http.MethodGet)
	protected.HandleFunc("/results/{id}", s.getResult).Methods(http.MethodGet)

	// Replay stream
	protected.HandleFunc("/replays/{id}/stream", s.streamReplay).Methods(http.MethodGet)

	return router
}
