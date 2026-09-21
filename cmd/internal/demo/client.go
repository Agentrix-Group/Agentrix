// Package demo is a minimal HTTP client of the Agentrix API used by the demo
// bootstrap and the smoke test. Both drive the real API; neither writes SQL.
package demo

import (
	"archive/zip"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"os"
	"strings"
	"time"
)

type Client struct {
	Base  string
	HTTP  *http.Client
	Token string
	User  map[string]any
}

func New(base string) *Client {
	jar, _ := cookiejar.New(nil)
	return &Client{Base: strings.TrimRight(base, "/"), HTTP: &http.Client{Jar: jar, Timeout: 60 * time.Second}}
}

type Response struct {
	Status int
	Header http.Header
	Body   []byte
}

func (r Response) JSON() map[string]any {
	var out map[string]any
	_ = json.Unmarshal(r.Body, &out)
	return out
}

func (r Response) ErrorCode() string {
	if e, ok := r.JSON()["error"].(map[string]any); ok {
		code, _ := e["code"].(string)
		return code
	}
	return ""
}

type Form struct {
	Data        []byte
	ContentType string
}

func (c *Client) Do(method, path string, body any, headers ...string) (Response, error) {
	var reader io.Reader
	contentType := ""
	switch b := body.(type) {
	case nil:
	case *Form:
		reader, contentType = bytes.NewReader(b.Data), b.ContentType
	default:
		raw, err := json.Marshal(b)
		if err != nil {
			return Response{}, err
		}
		reader, contentType = bytes.NewReader(raw), "application/json"
	}
	req, err := http.NewRequest(method, c.Base+path, reader)
	if err != nil {
		return Response{}, err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	res, err := c.HTTP.Do(req)
	if err != nil {
		return Response{}, err
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return Response{}, err
	}
	return Response{Status: res.StatusCode, Header: res.Header, Body: raw}, nil
}

// Expect performs a request and fails unless the status matches.
func (c *Client) Expect(status int, method, path string, body any, headers ...string) (map[string]any, error) {
	res, err := c.Do(method, path, body, headers...)
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w", method, path, err)
	}
	if res.Status != status {
		return nil, fmt.Errorf("%s %s: expected %d, got %d: %s", method, path, status, res.Status, truncate(string(res.Body), 400))
	}
	return res.JSON(), nil
}

func (c *Client) Login(username, password string) error {
	body, err := c.Expect(http.StatusOK, "POST", "/api/v1/auth/login", map[string]string{"username": username, "password": password})
	if err != nil {
		return err
	}
	c.Token, _ = body["access_token"].(string)
	c.User, _ = body["user"].(map[string]any)
	return nil
}

func (c *Client) UserID() string { id, _ := c.User["id"].(string); return id }

// Bundle builds the multipart upload of a bot bundle from a Python file.
func Bundle(name, botPath string) (*Form, error) {
	code, err := os.ReadFile(botPath)
	if err != nil {
		return nil, err
	}
	var archive bytes.Buffer
	zw := zip.NewWriter(&archive)
	manifest, _ := json.Marshal(map[string]string{"name": name, "entrypoint": "bot.py", "protocol_version": "1.0"})
	for _, f := range []struct {
		name string
		data []byte
	}{{"agentrix.json", manifest}, {"bot.py", code}} {
		w, err := zw.Create(f.name)
		if err != nil {
			return nil, err
		}
		if _, err := w.Write(f.data); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, err := mw.CreateFormFile("bundle", name+".zip")
	if err != nil {
		return nil, err
	}
	if _, err := fw.Write(archive.Bytes()); err != nil {
		return nil, err
	}
	if err := mw.Close(); err != nil {
		return nil, err
	}
	return &Form{Data: body.Bytes(), ContentType: mw.FormDataContentType()}, nil
}

// Poll calls fn until it returns true, an error, or the timeout expires.
func Poll(timeout time.Duration, what string, fn func() (bool, error)) error {
	deadline := time.Now().Add(timeout)
	for {
		done, err := fn()
		if err != nil {
			return err
		}
		if done {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout after %s waiting for %s", timeout, what)
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func RandomKey() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// Scenario plays the canonical competitive flow with the given clients:
// agents, uploads, admission, enrollment with exact submissions, contest
// transitions, match creation, idempotent scheduling and polling.
type Scenario struct {
	Organizer   *Client
	Player      *Client
	BotsDir     string
	ContestName string
	Timeout     time.Duration
	Log         func(format string, args ...any)

	ContestID string
	MatchID   string
	RunID     string
	EntryIDs  []string
}

func (s *Scenario) Play() error {
	agents := []struct{ name, bot string }{{"Hunter", "bot_hunter.py"}, {"Evasive", "bot_evasive.py"}}
	var agentIDs, subIDs []string
	suffix := RandomKey()[:6]
	for _, a := range agents {
		agent, err := s.Player.Expect(http.StatusCreated, "POST", "/api/v1/agents",
			map[string]string{"game_id": "starfighter", "name": a.name + " " + suffix, "description": "Reference bot " + a.bot})
		if err != nil {
			return err
		}
		form, err := Bundle(strings.ToLower(a.name), s.BotsDir+"/"+a.bot)
		if err != nil {
			return err
		}
		sub, err := s.Player.Expect(http.StatusAccepted, "POST", "/api/v1/agents/"+agent["id"].(string)+"/submissions", form)
		if err != nil {
			return err
		}
		agentIDs = append(agentIDs, agent["id"].(string))
		subIDs = append(subIDs, sub["id"].(string))
		s.Log("uploaded %s v%v (%s), admission pending", a.name, sub["version"], sub["artifact_sha256"])
	}
	for _, id := range subIDs {
		id := id
		if err := Poll(s.Timeout, "submission admission", func() (bool, error) {
			sub, err := s.Player.Expect(http.StatusOK, "GET", "/api/v1/submissions/"+id, nil)
			if err != nil {
				return false, err
			}
			switch sub["status"] {
			case "ready":
				return true, nil
			case "rejected":
				return false, fmt.Errorf("submission %s rejected: %v", id, sub["admission_error"])
			}
			return false, nil
		}); err != nil {
			return err
		}
	}
	s.Log("submissions admitted by the worker sandbox")

	contest, err := s.Organizer.Expect(http.StatusCreated, "POST", "/api/v1/contests",
		map[string]any{"game_id": "starfighter", "name": s.ContestName, "description": "Starfighter 1v1 demo contest"})
	if err != nil {
		return err
	}
	s.ContestID = contest["id"].(string)
	if _, err := s.Organizer.Expect(http.StatusOK, "POST", "/api/v1/contests/"+s.ContestID+"/transitions",
		map[string]string{"state": "registration_open"}); err != nil {
		return err
	}
	for i := range agentIDs {
		entry, err := s.Player.Expect(http.StatusCreated, "POST", "/api/v1/contests/"+s.ContestID+"/entries",
			map[string]string{"agent_id": agentIDs[i], "submission_id": subIDs[i]})
		if err != nil {
			return err
		}
		if entry["submission_id"] != subIDs[i] {
			return fmt.Errorf("entry locked submission %v, expected %s", entry["submission_id"], subIDs[i])
		}
		s.EntryIDs = append(s.EntryIDs, entry["id"].(string))
	}
	for _, state := range []string{"registration_closed", "running"} {
		if _, err := s.Organizer.Expect(http.StatusOK, "POST", "/api/v1/contests/"+s.ContestID+"/transitions",
			map[string]string{"state": state}); err != nil {
			return err
		}
	}
	s.Log("contest %s running with %d locked entries", s.ContestID, len(s.EntryIDs))

	match, err := s.Organizer.Expect(http.StatusCreated, "POST", "/api/v1/matches",
		map[string]any{"contest_id": s.ContestID, "mode": "competitive", "entry_ids": s.EntryIDs})
	if err != nil {
		return err
	}
	s.MatchID = match["id"].(string)
	key := "demo-" + RandomKey()
	ticket, err := s.Organizer.Expect(http.StatusAccepted, "POST", "/api/v1/matches/"+s.MatchID+"/runs", nil, "Idempotency-Key", key)
	if err != nil {
		return err
	}
	s.RunID, _ = ticket["run_id"].(string)
	if s.RunID == "" || ticket["job_id"] == "" {
		return fmt.Errorf("schedule returned an empty run/job: %v", ticket)
	}
	replayed, err := s.Organizer.Do("POST", "/api/v1/matches/"+s.MatchID+"/runs", nil, "Idempotency-Key", key)
	if err != nil {
		return err
	}
	if replayed.Status != http.StatusAccepted || replayed.JSON()["run_id"] != s.RunID || replayed.Header.Get("Idempotency-Replayed") != "true" {
		return fmt.Errorf("idempotent replay mismatch: %d %s", replayed.Status, replayed.Body)
	}
	s.Log("run %s queued (idempotent replay verified)", s.RunID)

	return Poll(s.Timeout, "match completion", func() (bool, error) {
		m, err := s.Organizer.Expect(http.StatusOK, "GET", "/api/v1/matches/"+s.MatchID, nil)
		if err != nil {
			return false, err
		}
		switch m["state"] {
		case "finished":
			return true, nil
		case "failed", "cancelled":
			return false, fmt.Errorf("match ended %v: runs %v", m["state"], m["runs"])
		}
		return false, nil
	})
}
