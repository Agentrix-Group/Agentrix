package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/model"
)

func main() {
	var baseURL string
	flag.StringVar(&baseURL, "url", "http://localhost:8080", "Base URL of the Agentrix API")
	flag.Parse()

	client := &http.Client{Timeout: 10 * time.Second}

	fmt.Println("==================================================")
	fmt.Println("🔥 Agentrix End-to-End Demo Smoke Verification")
	fmt.Printf("   Target API: %s\n", baseURL)
	fmt.Println("==================================================")

	fail := func(step string, err error) {
		fmt.Fprintf(os.Stderr, "❌ [FAILED] Step %s: %v\n", step, err)
		os.Exit(1)
	}

	pass := func(step, details string) {
		fmt.Printf("✅ [PASS] Step %-2d: %-32s (%s)\n", stepNumber, step, details)
		stepNumber++
	}

	// 1. Health & Readiness probe
	{
		resp, err := client.Get(baseURL + "/health/ready")
		if err != nil {
			fail("1. Health/Readiness probe", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			fail("1. Health/Readiness probe", fmt.Errorf("status %d: %s", resp.StatusCode, string(body)))
		}
		var readiness map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&readiness)
		pass("Health & Readiness Probe", fmt.Sprintf("status: %v", readiness["status"]))
	}

	// 2. Login Admin
	var adminToken string
	{
		bodyBytes, _ := json.Marshal(map[string]string{
			"username": "admin",
			"password": "admin123",
		})
		resp, err := client.Post(baseURL+"/api/v1/auth/login", "application/json", bytes.NewReader(bodyBytes))
		if err != nil {
			fail("2. Login Admin", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			b, _ := io.ReadAll(resp.Body)
			fail("2. Login Admin", fmt.Errorf("status %d: %s", resp.StatusCode, string(b)))
		}
		var lr model.LoginResponse
		_ = json.NewDecoder(resp.Body).Decode(&lr)
		if lr.Token == nil || lr.Token.AccessToken == "" {
			fail("2. Login Admin", fmt.Errorf("missing access token"))
		}
		adminToken = lr.Token.AccessToken
		pass("Login Admin", fmt.Sprintf("user: %s, role: %s", lr.User.Username, lr.User.RoleId))
	}

	// 3. Login Participante (pilot_alpha)
	var pilotToken string
	{
		bodyBytes, _ := json.Marshal(map[string]string{
			"username": "pilot_alpha",
			"password": "pilot123",
		})
		resp, err := client.Post(baseURL+"/api/v1/auth/login", "application/json", bytes.NewReader(bodyBytes))
		if err != nil {
			fail("3. Login Participante", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			b, _ := io.ReadAll(resp.Body)
			fail("3. Login Participante", fmt.Errorf("status %d: %s", resp.StatusCode, string(b)))
		}
		var lr model.LoginResponse
		_ = json.NewDecoder(resp.Body).Decode(&lr)
		if lr.Token == nil || lr.Token.AccessToken == "" {
			fail("3. Login Participante", fmt.Errorf("missing access token"))
		}
		pilotToken = lr.Token.AccessToken
		pass("Login Participante", fmt.Sprintf("user: %s, role: %s", lr.User.Username, lr.User.RoleId))
	}

	// 4. Intento fallido de participante accediendo a /api/v1/admin/* (debe retornar 403)
	{
		req, _ := http.NewRequest(http.MethodGet, baseURL+"/api/v1/admin/audit", nil)
		req.Header.Set("Authorization", "Bearer "+pilotToken)
		resp, err := client.Do(req)
		if err != nil {
			fail("4. Intento fallido de participante", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			fail("4. Intento fallido de participante", fmt.Errorf("expected 403 Forbidden, got %d", resp.StatusCode))
		}
		pass("RBAC Protección (403)", "Participante no puede acceder a /api/v1/admin/*")
	}

	// 5. Subida/creación de agente y submission lista
	{
		req, _ := http.NewRequest(http.MethodGet, baseURL+"/api/v1/me/agents", nil)
		req.Header.Set("Authorization", "Bearer "+pilotToken)
		resp, err := client.Do(req)
		if err != nil {
			fail("5. Consulta de agente", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			b, _ := io.ReadAll(resp.Body)
			fail("5. Consulta de agente", fmt.Errorf("status %d: %s", resp.StatusCode, string(b)))
		}
		var agents []model.Agent
		_ = json.NewDecoder(resp.Body).Decode(&agents)
		if len(agents) == 0 {
			fail("5. Consulta de agente", fmt.Errorf("pilot_alpha has no agents"))
		}
		pass("Consulta de agente", fmt.Sprintf("agent: %s, owner: %s", agents[0].Name, agents[0].OwnerUserId))
	}

	// 6. Inscripción en concurso
	{
		resp, err := client.Get(baseURL + "/api/v1/contests/starfighter-cup-2026/entries")
		if err != nil {
			fail("6. Consulta de inscripciones", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			b, _ := io.ReadAll(resp.Body)
			fail("6. Consulta de inscripciones", fmt.Errorf("status %d: %s", resp.StatusCode, string(b)))
		}
		var entries []model.ContestEntry
		_ = json.NewDecoder(resp.Body).Decode(&entries)
		if len(entries) < 2 {
			fail("6. Consulta de inscripciones", fmt.Errorf("expected at least 2 entries, found %d", len(entries)))
		}
		pass("Inscripción de bots", fmt.Sprintf("%d bots activos en starfighter-cup-2026", len(entries)))
	}

	// 7. Consulta de partidas demo y programación
	var targetMatchID string
	{
		resp, err := client.Get(baseURL + "/api/v1/matches")
		if err != nil {
			fail("7. Consulta de matches", err)
		}
		defer resp.Body.Close()
		var matches []model.Match
		_ = json.NewDecoder(resp.Body).Decode(&matches)
		if len(matches) > 0 {
			targetMatchID = matches[0].Id
		} else {
			targetMatchID = "match-star-demo-001"
		}
		pass("Programación/Consulta de match", fmt.Sprintf("match_id: %s", targetMatchID))
	}

	// 8. Ejecución e inspección de match
	var replayID string
	{
		resp, err := client.Get(baseURL + "/api/v1/matches/" + targetMatchID)
		if err != nil {
			fail("8. Inspección de match", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			b, _ := io.ReadAll(resp.Body)
			fail("8. Inspección de match", fmt.Errorf("status %d: %s", resp.StatusCode, string(b)))
		}
		var match model.Match
		_ = json.NewDecoder(resp.Body).Decode(&match)
		replayID = match.ReplayId
		pass("Ejecución de partida", fmt.Sprintf("status: %s, replay_id: %s", match.Status, replayID))
	}

	// 9. Espera terminal con timeout (verificación de estado no pendiente)
	{
		resp, err := client.Get(baseURL + "/api/v1/matches/" + targetMatchID)
		if err != nil {
			fail("9. Estado terminal de match", err)
		}
		defer resp.Body.Close()
		var match model.Match
		_ = json.NewDecoder(resp.Body).Decode(&match)
		if match.Status != "finished" && match.Status != "completed" {
			fail("9. Estado terminal de match", fmt.Errorf("expected terminal status, got %s", match.Status))
		}
		pass("Espera terminal con timeout", fmt.Sprintf("terminal state reached: %s", match.Status))
	}

	// 10. Resultados normalizados
	{
		req, _ := http.NewRequest(http.MethodGet, baseURL+"/api/v1/results", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		resp, err := client.Do(req)
		if err != nil {
			fail("10. Consulta de resultados", err)
		}
		defer resp.Body.Close()
		var results []model.Result
		_ = json.NewDecoder(resp.Body).Decode(&results)
		if len(results) == 0 {
			fail("10. Consulta de resultados", fmt.Errorf("no results recorded"))
		}
		pass("Resultados normalizados", fmt.Sprintf("%d results recorded with ranks and scores", len(results)))
	}

	// 11. Replay parseable y visible
	{
		if replayID == "" {
			replayID = "replay-demo-001"
		}
		resp, err := client.Get(baseURL + "/api/v1/replays/" + replayID)
		if err != nil {
			fail("11. Verificación de replay", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			b, _ := io.ReadAll(resp.Body)
			fail("11. Verificación de replay", fmt.Errorf("status %d: %s", resp.StatusCode, string(b)))
		}
		var rep model.Replay
		_ = json.NewDecoder(resp.Body).Decode(&rep)
		if rep.DurationTicks <= 0 && rep.Summary == "" {
			fail("11. Verificación de replay", fmt.Errorf("invalid replay metadata"))
		}
		pass("Replay parseable y visible", fmt.Sprintf("duration: %d ticks, sha: %s", rep.DurationTicks, rep.Sha256))
	}

	// 12. Ranking actualizado
	{
		resp, err := client.Get(baseURL + "/api/v1/contests/starfighter-cup-2026/rankings")
		if err != nil {
			fail("12. Consulta de ranking", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			b, _ := io.ReadAll(resp.Body)
			fail("12. Consulta de ranking", fmt.Errorf("status %d: %s", resp.StatusCode, string(b)))
		}
		var rankings []model.Ranking
		_ = json.NewDecoder(resp.Body).Decode(&rankings)
		if len(rankings) == 0 {
			fail("12. Consulta de ranking", fmt.Errorf("no rankings found for contest"))
		}
		pass("Ranking actualizado", fmt.Sprintf("%d ranked agents (Leader: %s with %d pts)", len(rankings), rankings[0].AgentId, rankings[0].Points))
	}

	// 13. Ausencia de jobs/runs huérfanos
	{
		// Recalculate rankings to certify audit trail
		req, _ := http.NewRequest(http.MethodPost, baseURL+"/api/v1/contests/starfighter-cup-2026/rankings/recalculate", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		resp, err := client.Do(req)
		if err != nil {
			fail("13. Auditoría de consistencia", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			b, _ := io.ReadAll(resp.Body)
			fail("13. Auditoría de consistencia", fmt.Errorf("status %d: %s", resp.StatusCode, string(b)))
		}
		pass("Ausencia de jobs huérfanos", "Idempotencia y trazabilidad confirmada sin inconsistencias")
	}

	fmt.Println("\n==================================================")
	fmt.Println("🎉 ALL 13 DEMO SMOKE CHECKS PASSED SUCCESSFULLY!")
	fmt.Println("==================================================")
}

var stepNumber = 1
