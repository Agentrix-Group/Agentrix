// Package config reads process configuration from the environment and
// validates it. Invalid configuration is a startup error, never a silent
// fallback: outside dev/test there are no default secrets.
package config

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	ModeDev        = "dev"
	ModeTest       = "test"
	ModeDemo       = "demo"
	ModeProduction = "production"
)

type Database struct {
	URL      string
	Host     string
	Port     string
	Name     string
	Username string
	Password string
	SSLMode  string
	MaxConns int
}

type Auth struct {
	AccessSecret     []byte
	AccessTTL        time.Duration
	RefreshIdleTTL   time.Duration
	SessionMaxTTL    time.Duration
	MaxSessions      int
	CookieSecure     bool
	CookieSameSite   string
	CookiePath       string
	EphemeralSecret  bool
	RegistrationOpen bool
}

type HTTP struct {
	Port           string
	AllowedOrigins []string
	TrustedProxies []*net.IPNet
	RateLimits     RateLimits
}

// RateLimits are requests per minute per client IP.
type RateLimits struct {
	Login    int
	Register int
	Refresh  int
}

func DefaultRateLimits() RateLimits { return RateLimits{Login: 10, Register: 5, Refresh: 60} }

type Worker struct {
	ID             string
	Concurrency    int
	LeaseTTL       time.Duration
	PollInterval   time.Duration
	MaxAttempts    int
	EngineBinary   string
	EngineCommit   string
	Sandbox        string
	PodmanImage    string
	ReconcileEvery time.Duration
}

type Logging struct {
	Level  string
	Format string
	Color  string
}

type Config struct {
	Mode         string
	Database     Database
	Auth         Auth
	HTTP         HTTP
	Worker       Worker
	ArtifactsDir string
	GamesDir     string
	Logging      Logging
}

func (c *Config) Development() bool { return c.Mode == ModeDev || c.Mode == ModeTest }

// Load reads the environment (and ./.env when present, without overriding
// variables already set) and validates the result.
func Load() (*Config, error) {
	loadDotEnv(".env")
	cfg := &Config{
		Mode: env("MODE", ModeDev),
		Database: Database{
			URL:      os.Getenv("DATABASE_URL"),
			Host:     env("DB_HOST", "localhost"),
			Port:     env("DB_PORT", "5432"),
			Name:     env("DB_NAME", "agentrix"),
			Username: env("DB_USER", "postgres"),
			Password: os.Getenv("DB_PASSWORD"),
			SSLMode:  env("DB_SSLMODE", "disable"),
		},
		ArtifactsDir: env("ARTIFACTS_DIR", "./artifacts"),
		GamesDir:     env("GAMES_DIR", "./games"),
		Logging: Logging{
			Level:  env("LOG_LEVEL", "info"),
			Format: env("LOG_FORMAT", "console"),
			Color:  env("LOG_COLOR", "auto"),
		},
	}
	var errs []error
	intVar := func(name string, def int, dst *int) {
		v, err := strconv.Atoi(env(name, strconv.Itoa(def)))
		if err != nil || v <= 0 {
			errs = append(errs, fmt.Errorf("%s must be a positive integer", name))
			return
		}
		*dst = v
	}
	durVar := func(name string, def time.Duration, dst *time.Duration) {
		v, err := time.ParseDuration(env(name, def.String()))
		if err != nil || v <= 0 {
			errs = append(errs, fmt.Errorf("%s must be a positive duration", name))
			return
		}
		*dst = v
	}
	boolVar := func(name string, def bool, dst *bool) {
		v, err := strconv.ParseBool(env(name, strconv.FormatBool(def)))
		if err != nil {
			errs = append(errs, fmt.Errorf("%s must be a boolean", name))
			return
		}
		*dst = v
	}

	switch cfg.Mode {
	case ModeDev, ModeTest, ModeDemo, ModeProduction:
	default:
		errs = append(errs, fmt.Errorf("MODE must be dev, test, demo or production (got %q)", cfg.Mode))
	}
	intVar("DB_MAX_CONNS", 20, &cfg.Database.MaxConns)

	// --- Auth -------------------------------------------------------------
	secret := os.Getenv("ACCESS_SECRET")
	switch {
	case secret == "" && cfg.Development():
		random := make([]byte, 48)
		if _, err := rand.Read(random); err != nil {
			errs = append(errs, fmt.Errorf("generate ephemeral secret: %w", err))
		}
		cfg.Auth.AccessSecret = random
		cfg.Auth.EphemeralSecret = true
	case secret == "":
		errs = append(errs, errors.New("ACCESS_SECRET is required outside dev/test"))
	case len(secret) < 32:
		errs = append(errs, errors.New("ACCESS_SECRET must contain at least 32 bytes"))
	case strings.Contains(strings.ToLower(secret), "change") || strings.Contains(strings.ToLower(secret), "secret-key"):
		errs = append(errs, errors.New("ACCESS_SECRET looks like a placeholder"))
	default:
		cfg.Auth.AccessSecret = []byte(secret)
	}
	durVar("ACCESS_TOKEN_TTL", 10*time.Minute, &cfg.Auth.AccessTTL)
	durVar("REFRESH_IDLE_TTL", 7*24*time.Hour, &cfg.Auth.RefreshIdleTTL)
	durVar("SESSION_MAX_TTL", 30*24*time.Hour, &cfg.Auth.SessionMaxTTL)
	intVar("MAX_SESSIONS_PER_USER", 10, &cfg.Auth.MaxSessions)
	boolVar("COOKIE_SECURE", cfg.Mode == ModeProduction, &cfg.Auth.CookieSecure)
	boolVar("REGISTRATION_OPEN", true, &cfg.Auth.RegistrationOpen)
	cfg.Auth.CookieSameSite = env("COOKIE_SAMESITE", "strict")
	cfg.Auth.CookiePath = "/api/v1/auth"
	if cfg.Auth.AccessTTL > 30*time.Minute {
		errs = append(errs, errors.New("ACCESS_TOKEN_TTL must not exceed 30m"))
	}
	if cfg.Auth.RefreshIdleTTL > cfg.Auth.SessionMaxTTL {
		errs = append(errs, errors.New("REFRESH_IDLE_TTL must not exceed SESSION_MAX_TTL"))
	}
	if cfg.Mode == ModeProduction && !cfg.Auth.CookieSecure {
		errs = append(errs, errors.New("COOKIE_SECURE must be true in production"))
	}
	switch cfg.Auth.CookieSameSite {
	case "strict", "lax":
	case "none":
		if !cfg.Auth.CookieSecure {
			errs = append(errs, errors.New("COOKIE_SAMESITE=none requires COOKIE_SECURE=true"))
		}
	default:
		errs = append(errs, errors.New("COOKIE_SAMESITE must be strict, lax or none"))
	}

	// --- HTTP -------------------------------------------------------------
	cfg.HTTP.Port = env("PORT", "8080")
	defaults := DefaultRateLimits()
	intVar("RATE_LIMIT_LOGIN", defaults.Login, &cfg.HTTP.RateLimits.Login)
	intVar("RATE_LIMIT_REGISTER", defaults.Register, &cfg.HTTP.RateLimits.Register)
	intVar("RATE_LIMIT_REFRESH", defaults.Refresh, &cfg.HTTP.RateLimits.Refresh)
	for _, origin := range splitCSV(os.Getenv("CORS_ALLOWED_ORIGINS")) {
		if origin == "*" {
			errs = append(errs, errors.New("CORS_ALLOWED_ORIGINS cannot contain '*': the API uses credentials"))
			continue
		}
		u, err := url.Parse(origin)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.Path != "" {
			errs = append(errs, fmt.Errorf("invalid CORS origin %q", origin))
			continue
		}
		cfg.HTTP.AllowedOrigins = append(cfg.HTTP.AllowedOrigins, origin)
	}
	for _, cidr := range splitCSV(os.Getenv("TRUSTED_PROXIES")) {
		if !strings.Contains(cidr, "/") {
			if strings.Contains(cidr, ":") {
				cidr += "/128"
			} else {
				cidr += "/32"
			}
		}
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			errs = append(errs, fmt.Errorf("invalid TRUSTED_PROXIES entry %q", cidr))
			continue
		}
		cfg.HTTP.TrustedProxies = append(cfg.HTTP.TrustedProxies, network)
	}

	// --- Worker -----------------------------------------------------------
	host, _ := os.Hostname()
	cfg.Worker.ID = env("WORKER_ID", fmt.Sprintf("%s-%d", host, os.Getpid()))
	intVar("WORKER_CONCURRENCY", 2, &cfg.Worker.Concurrency)
	intVar("MAX_RUN_ATTEMPTS", 3, &cfg.Worker.MaxAttempts)
	durVar("LEASE_TTL", 60*time.Second, &cfg.Worker.LeaseTTL)
	durVar("POLL_INTERVAL", 500*time.Millisecond, &cfg.Worker.PollInterval)
	durVar("RECONCILE_INTERVAL", 5*time.Second, &cfg.Worker.ReconcileEvery)
	cfg.Worker.EngineBinary = os.Getenv("AGENTRIX_ENGINE_BIN")
	cfg.Worker.EngineCommit = os.Getenv("AGENTRIX_ENGINE_COMMIT")
	cfg.Worker.Sandbox = env("AGENTRIX_SANDBOX", "bubblewrap")
	cfg.Worker.PodmanImage = env("AGENTRIX_PODMAN_IMAGE", "docker.io/library/python:3.12-slim")
	switch cfg.Worker.Sandbox {
	case "bubblewrap", "podman":
	case "direct":
		if !cfg.Development() {
			errs = append(errs, errors.New("AGENTRIX_SANDBOX=direct runs untrusted code without isolation and is only allowed in dev/test"))
		}
	default:
		errs = append(errs, fmt.Errorf("AGENTRIX_SANDBOX must be bubblewrap, podman or direct (got %q)", cfg.Worker.Sandbox))
	}
	if cfg.Worker.LeaseTTL < 10*time.Second {
		errs = append(errs, errors.New("LEASE_TTL must be at least 10s"))
	}

	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	return cfg, nil
}

// DSN builds the connection string with proper escaping.
func (c *Config) DSN() string {
	if c.Database.URL != "" {
		return c.Database.URL
	}
	u := url.URL{
		Scheme:   "postgres",
		Host:     net.JoinHostPort(c.Database.Host, c.Database.Port),
		Path:     "/" + c.Database.Name,
		RawQuery: url.Values{"sslmode": {c.Database.SSLMode}}.Encode(),
	}
	if c.Database.Password != "" {
		u.User = url.UserPassword(c.Database.Username, c.Database.Password)
	} else {
		u.User = url.User(c.Database.Username)
	}
	return u.String()
}

// Redacted describes the configuration for logs without secrets.
func (c *Config) Redacted() map[string]string {
	return map[string]string{
		"mode": c.Mode, "db_host": c.Database.Host, "db_name": c.Database.Name,
		"artifacts_dir": c.ArtifactsDir, "games_dir": c.GamesDir, "sandbox": c.Worker.Sandbox,
		"ephemeral_secret": strconv.FormatBool(c.Auth.EphemeralSecret),
	}
}

// RandomToken returns n random bytes encoded for URLs.
func RandomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func env(name, def string) string {
	if v, ok := os.LookupEnv(name); ok && v != "" {
		return v
	}
	return def
}

func splitCSV(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

// loadDotEnv reads KEY=VALUE lines without overriding existing variables.
func loadDotEnv(filename string) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key, val = strings.TrimSpace(key), strings.TrimSpace(val)
		if len(val) >= 2 && (val[0] == '"' || val[0] == '\'') && val[len(val)-1] == val[0] {
			val = val[1 : len(val)-1]
		}
		if _, exists := os.LookupEnv(key); !exists {
			_ = os.Setenv(key, val)
		}
	}
}
