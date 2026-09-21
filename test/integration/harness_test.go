// Package integration exercises Agentrix against a real PostgreSQL, the real
// HTTP stack, the real sandbox and an engine subprocess. Set
// AGENTRIX_REQUIRE_DB=1 (CI does) to turn a missing database into a failure
// instead of a skip.
package integration

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/auth"
	"github.com/Agentrix-Group/Agentrix/src/config"
	"github.com/Agentrix-Group/Agentrix/src/connection"
	"github.com/Agentrix-Group/Agentrix/src/database"
	"github.com/Agentrix-Group/Agentrix/src/executor"
	"github.com/Agentrix-Group/Agentrix/src/game"
	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/Agentrix-Group/Agentrix/src/repository"
	"github.com/Agentrix-Group/Agentrix/src/server"
	"github.com/Agentrix-Group/Agentrix/src/service"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
)

var (
	fakeEngineBin string
	repoRoot      string
	dbCounter     struct {
		sync.Mutex
		n int
	}
)

func TestMain(m *testing.M) {
	root, err := filepath.Abs("../..")
	if err != nil {
		panic(err)
	}
	repoRoot = root
	auth.SetActiveArgon2Params(auth.FastArgon2Params)
	dir, err := os.MkdirTemp("", "agentrix-it-*")
	if err != nil {
		panic(err)
	}
	fakeEngineBin = filepath.Join(dir, "fake-engine")
	build := exec.Command("go", "build", "-o", fakeEngineBin, "./cmd/fake-engine")
	build.Dir = repoRoot
	if out, err := build.CombinedOutput(); err != nil {
		panic(fmt.Sprintf("build fake engine: %v\n%s", err, out))
	}
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

func adminDSN(name string) string {
	host := envOr("DB_HOST", "localhost")
	port := envOr("DB_PORT", "5432")
	user := envOr("DB_USER", "postgres")
	u := url.URL{Scheme: "postgres", Host: host + ":" + port, Path: "/" + name, RawQuery: "sslmode=disable"}
	if pass := os.Getenv("DB_PASSWORD"); pass != "" {
		u.User = url.UserPassword(user, pass)
	} else {
		u.User = url.User(user)
	}
	return u.String()
}

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

// freshDB creates an empty database, applies the canonical migration and
// returns a pool. The database is dropped when the test ends.
func freshDB(t *testing.T) *sql.DB {
	t.Helper()
	admin, err := sql.Open("pgx", adminDSN("postgres"))
	require.NoError(t, err)
	if err := admin.Ping(); err != nil {
		_ = admin.Close()
		if os.Getenv("AGENTRIX_REQUIRE_DB") == "1" {
			t.Fatalf("PostgreSQL is required (AGENTRIX_REQUIRE_DB=1) but unreachable: %v", err)
		}
		t.Skipf("PostgreSQL unreachable: %v", err)
	}
	dbCounter.Lock()
	dbCounter.n++
	name := fmt.Sprintf("agentrix_it_%d_%d", os.Getpid(), dbCounter.n)
	dbCounter.Unlock()
	_, err = admin.Exec(`CREATE DATABASE ` + name)
	require.NoError(t, err)
	db, err := sql.Open("pgx", adminDSN(name))
	require.NoError(t, err)
	db.SetMaxOpenConns(30)
	require.NoError(t, database.Migrate(db))
	t.Cleanup(func() {
		_ = db.Close()
		_, _ = admin.Exec(`DROP DATABASE IF EXISTS ` + name + ` WITH (FORCE)`)
		_ = admin.Close()
	})
	return db
}

type env struct {
	t         *testing.T
	db        *sql.DB
	store     *repository.Store
	svc       *service.Service
	cfg       *config.Config
	server    *server.Server
	http      *httptest.Server
	games     *game.Registry
	artifacts *connection.ArtifactStore
	clock     *testClock
	engineBin string
	engineSHA string
	runtime   executor.BotRuntime
	admin     string
}

// adminID returns the bootstrap administrator, creating it on first use.
func (e *env) adminID() string {
	if e.admin == "" {
		u, err := e.svc.BootstrapAdmin(context.Background(), "root", "root@example.com", "password-root")
		require.NoError(e.t, err)
		e.admin = u.ID
	}
	return e.admin
}

type testClock struct {
	mu     sync.Mutex
	offset time.Duration
}

func (c *testClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return time.Now().UTC().Add(c.offset).Truncate(time.Microsecond)
}

func (c *testClock) Advance(d time.Duration) {
	c.mu.Lock()
	c.offset += d
	c.mu.Unlock()
}

type envOptions struct {
	engineBin string // defaults to the fake engine
	sandbox   string // defaults to AGENTRIX_TEST_SANDBOX or bubblewrap
	opts      *service.Options
}

func newEnv(t *testing.T, o envOptions) *env {
	t.Helper()
	db := freshDB(t)
	games, err := game.LoadRegistry(filepath.Join(repoRoot, "games"))
	require.NoError(t, err)
	artifacts, err := connection.NewArtifactStore(t.TempDir())
	require.NoError(t, err)
	tokens, err := auth.NewTokens([]byte("integration-test-secret-0123456789abcdef"), 10*time.Minute)
	require.NoError(t, err)
	opts := service.DefaultOptions()
	if o.opts != nil {
		opts = *o.opts
	}
	clock := &testClock{}
	store := repository.NewStore(db)
	svc := service.New(store, games, artifacts, tokens, opts).WithClock(clock.Now)
	require.NoError(t, svc.SyncGames(context.Background()))
	cfg := &config.Config{Mode: config.ModeTest, Auth: config.Auth{CookieSameSite: "strict", CookiePath: "/api/v1/auth"},
		HTTP: config.HTTP{AllowedOrigins: []string{"http://localhost:5173"},
			RateLimits: config.RateLimits{Login: 1000, Register: 1000, Refresh: 1000}}}
	srv := server.New(svc, cfg)
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(func() {
		ts.Close()
		srv.Close()
	})
	engineBin := o.engineBin
	if engineBin == "" {
		engineBin = fakeEngineBin
	}
	sha, _, err := connection.SHA256File(engineBin)
	require.NoError(t, err)
	sandbox := o.sandbox
	if sandbox == "" {
		sandbox = envOr("AGENTRIX_TEST_SANDBOX", "bubblewrap")
	}
	runtime, err := executor.NewRuntime(sandbox, "")
	require.NoError(t, err)
	require.NoError(t, runtime.Check(), "sandbox %s must be available for integration tests", sandbox)
	return &env{t: t, db: db, store: store, svc: svc, cfg: cfg, server: srv, http: ts, games: games, artifacts: artifacts,
		clock: clock, engineBin: engineBin, engineSHA: sha, runtime: runtime}
}

// registerWorker announces an engine artifact as a live worker would.
func (e *env) registerWorker(id string) {
	e.t.Helper()
	module, _ := e.games.Get("starfighter")
	version := "0.1.0-fake"
	if e.engineBin != fakeEngineBin {
		version = engineVersion(e.t, e.engineBin)
	}
	require.NoError(e.t, e.svc.RegisterWorker(context.Background(), model.WorkerInfo{ID: id, SandboxRuntime: e.runtime.Name()},
		model.EngineArtifact{SHA256: e.engineSHA, GameID: module.ID, GameVersion: module.Version, EngineVersion: version,
			ProtocolVersion: module.EngineProtocol}))
}

func engineVersion(t *testing.T, bin string) string {
	t.Helper()
	client := enginePkgClient()
	require.NoError(t, client.Start(context.Background(), engineStart(bin)))
	v := client.EngineVersion()
	require.NoError(t, client.Close(context.Background()))
	return v
}

func (e *env) worker(id string) *executor.Worker {
	exec := executor.NewExecutor(e.runtime, e.artifacts, executor.SubprocessEngineFactory(e.engineBin))
	return executor.NewWorker(e.svc, exec, executor.NewAdmission(e.runtime), executor.WorkerOptions{
		ID: id, EngineSHA256: e.engineSHA, Sandbox: e.runtime.Name(), Concurrency: 1, PollInterval: 50 * time.Millisecond,
		LeaseTTL: 60 * time.Second, ReconcileEvery: 100 * time.Millisecond})
}

// ---------------------------------------------------------------------------
// HTTP client
// ---------------------------------------------------------------------------

type client struct {
	e     *env
	http  *http.Client
	token string
	user  map[string]any
}

func (e *env) client() *client {
	jar, _ := cookiejar.New(nil)
	return &client{e: e, http: &http.Client{Jar: jar, Timeout: 30 * time.Second}}
}

type response struct {
	Status int
	Header http.Header
	Body   []byte
}

func (r response) JSON(t *testing.T) map[string]any {
	t.Helper()
	var out map[string]any
	require.NoError(t, json.Unmarshal(r.Body, &out), "body: %s", r.Body)
	return out
}

func (r response) ErrorCode(t *testing.T) string {
	t.Helper()
	body := r.JSON(t)
	errObj, ok := body["error"].(map[string]any)
	require.True(t, ok, "expected error envelope, got %s", r.Body)
	return errObj["code"].(string)
}

func (c *client) do(method, path string, body any, headers ...string) response {
	c.e.t.Helper()
	var reader io.Reader
	contentType := ""
	switch b := body.(type) {
	case nil:
	case *multipartBody:
		reader, contentType = bytes.NewReader(b.data), b.contentType
	default:
		raw, err := json.Marshal(b)
		require.NoError(c.e.t, err)
		reader, contentType = bytes.NewReader(raw), "application/json"
	}
	req, err := http.NewRequest(method, c.e.http.URL+path, reader)
	require.NoError(c.e.t, err)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	res, err := c.http.Do(req)
	require.NoError(c.e.t, err)
	defer res.Body.Close()
	raw, err := io.ReadAll(res.Body)
	require.NoError(c.e.t, err)
	return response{Status: res.StatusCode, Header: res.Header, Body: raw}
}

func (c *client) expect(status int, method, path string, body any, headers ...string) map[string]any {
	c.e.t.Helper()
	res := c.do(method, path, body, headers...)
	require.Equal(c.e.t, status, res.Status, "%s %s -> %s", method, path, res.Body)
	if len(res.Body) == 0 {
		return nil
	}
	return res.JSON(c.e.t)
}

type multipartBody struct {
	data        []byte
	contentType string
}

func bundleUpload(t *testing.T, name string, code []byte) *multipartBody {
	t.Helper()
	var archive bytes.Buffer
	zw := zip.NewWriter(&archive)
	manifest, _ := json.Marshal(map[string]string{"name": name, "entrypoint": "bot.py", "protocol_version": "1.0"})
	for fname, content := range map[string][]byte{"agentrix.json": manifest, "bot.py": code} {
		w, err := zw.Create(fname)
		require.NoError(t, err)
		_, err = w.Write(content)
		require.NoError(t, err)
	}
	require.NoError(t, zw.Close())
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, err := mw.CreateFormFile("bundle", "bot.zip")
	require.NoError(t, err)
	_, err = fw.Write(archive.Bytes())
	require.NoError(t, err)
	require.NoError(t, mw.Close())
	return &multipartBody{data: body.Bytes(), contentType: mw.FormDataContentType()}
}

func referenceBot(t *testing.T, name string) []byte {
	t.Helper()
	code, err := os.ReadFile(filepath.Join(repoRoot, "games", "starfighter", "examples", name))
	require.NoError(t, err)
	return code
}

// ---------------------------------------------------------------------------
// Scenario builders
// ---------------------------------------------------------------------------

// user creates an account with roles directly through the service and
// returns a logged-in client.
func (e *env) user(username string, roles ...string) *client {
	e.t.Helper()
	ctx := context.Background()
	admin := model.Principal{UserID: e.adminID(), Capabilities: model.Capabilities}
	_, err := e.svc.CreateUser(ctx, admin, username, username+"@example.com", "password-"+username, roles)
	require.NoError(e.t, err)
	c := e.client()
	c.login(username, "password-"+username)
	return c
}

func (c *client) login(username, password string) {
	c.e.t.Helper()
	body := c.expect(http.StatusOK, "POST", "/api/v1/auth/login", map[string]string{"username": username, "password": password})
	c.token = body["access_token"].(string)
	c.user = body["user"].(map[string]any)
}

func (c *client) id() string { return c.user["id"].(string) }

// agentWithReadySubmission creates an agent, uploads a bot and runs the
// sandboxed admission with the given worker.
func (c *client) agentWithReadySubmission(w *executor.Worker, name, bot string) (agentID, submissionID string) {
	c.e.t.Helper()
	agent := c.expect(http.StatusCreated, "POST", "/api/v1/agents", map[string]string{"game_id": "starfighter", "name": name})
	agentID = agent["id"].(string)
	sub := c.expect(http.StatusAccepted, "POST", "/api/v1/agents/"+agentID+"/submissions", bundleUpload(c.e.t, name, referenceBot(c.e.t, bot)))
	require.Equal(c.e.t, "validating", sub["status"])
	submissionID = sub["id"].(string)
	admitAll(c.e.t, c.e.svc, w)
	got := c.expect(http.StatusOK, "GET", "/api/v1/submissions/"+submissionID, nil)
	require.Equal(c.e.t, "ready", got["status"], "admission: %v", got["admission_error"])
	return agentID, submissionID
}

func admitAll(t *testing.T, svc *service.Service, w *executor.Worker) {
	t.Helper()
	w.ReconcileOnce(context.Background(), false)
}

// competitiveMatch sets up a running contest with two enrolled agents and a
// scheduled competitive match. Returns contest, match and entry ids.
type competition struct {
	contestID string
	matchID   string
	entries   []string
	subs      []string
	organizer *client
	players   []*client
}

func (e *env) competition(w *executor.Worker, bots ...string) *competition {
	e.t.Helper()
	if len(bots) == 0 {
		bots = []string{"bot_hunter.py", "bot_evasive.py"}
	}
	org := e.user("organizer"+shortID(), model.RoleOrganizer)
	contest := org.expect(http.StatusCreated, "POST", "/api/v1/contests", map[string]any{"game_id": "starfighter", "name": "Cup " + shortID()})
	cid := contest["id"].(string)
	org.expect(http.StatusOK, "POST", "/api/v1/contests/"+cid+"/transitions", map[string]string{"state": "registration_open"})
	comp := &competition{contestID: cid, organizer: org}
	for i, bot := range bots {
		p := e.user(fmt.Sprintf("player%d%s", i, shortID()), model.RolePlayer)
		agentID, subID := p.agentWithReadySubmission(w, fmt.Sprintf("agent-%d", i), bot)
		entry := p.expect(http.StatusCreated, "POST", "/api/v1/contests/"+cid+"/entries",
			map[string]string{"agent_id": agentID, "submission_id": subID})
		comp.entries = append(comp.entries, entry["id"].(string))
		comp.subs = append(comp.subs, subID)
		comp.players = append(comp.players, p)
	}
	org.expect(http.StatusOK, "POST", "/api/v1/contests/"+cid+"/transitions", map[string]string{"state": "registration_closed"})
	org.expect(http.StatusOK, "POST", "/api/v1/contests/"+cid+"/transitions", map[string]string{"state": "running"})
	match := org.expect(http.StatusCreated, "POST", "/api/v1/matches", map[string]any{"contest_id": cid, "mode": "competitive",
		"entry_ids": comp.entries[:2], "seed": 42})
	comp.matchID = match["id"].(string)
	return comp
}

func shortID() string { return strings.ReplaceAll(uuid.NewString()[:8], "-", "") }

func (e *env) count(query string, args ...any) int {
	e.t.Helper()
	var n int
	require.NoError(e.t, e.db.QueryRow(query, args...).Scan(&n))
	return n
}

// waitMatch polls the match until it reaches one of the states.
func (c *client) waitMatch(matchID string, timeout time.Duration, states ...string) map[string]any {
	c.e.t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		m := c.expect(http.StatusOK, "GET", "/api/v1/matches/"+matchID, nil)
		for _, s := range states {
			if m["state"] == s {
				return m
			}
		}
		if time.Now().After(deadline) {
			c.e.t.Fatalf("match %s stuck in %v (wanted %v)", matchID, m["state"], states)
		}
		time.Sleep(100 * time.Millisecond)
	}
}
