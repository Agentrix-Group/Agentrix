package main

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/config"
	"github.com/Agentrix-Group/Agentrix/src/connection"
	"github.com/Agentrix-Group/Agentrix/src/database"
	"github.com/Agentrix-Group/Agentrix/src/executor"
	"github.com/Agentrix-Group/Agentrix/src/game"
	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/Agentrix-Group/Agentrix/src/repository"
	"github.com/Agentrix-Group/Agentrix/src/service"
	"github.com/Agentrix-Group/Agentrix/src/tracer"
)

func main() {
	var (
		skipSeeds  bool
		queueMatch bool
	)
	flag.BoolVar(&skipSeeds, "skip-seeds", false, "Skip executing SQL seeds")
	flag.BoolVar(&queueMatch, "queue-match", false, "Queue a live match in match_jobs for the worker pool")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fmt.Println("==================================================")
	fmt.Println("🚀 Agentrix - Idempotent Starfighter Demo Bootstrap")
	fmt.Println("==================================================")

	cfg := config.NewConfiguration()
	conn, err := connection.NewConnection(ctx, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error connecting to database (%s:%s/%s): %v\n",
			cfg.Database.Host, cfg.Database.Port, cfg.Database.Name, err)
		os.Exit(1)
	}
	defer conn.Close()

	// 1. Run migrations to ensure target schema
	fmt.Println("📦 Step 1: Applying migrations...")
	if err := database.Migrate(conn.Db); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Migration failed: %v\n", err)
		os.Exit(1)
	}
	ver, _ := database.GetCurrentVersion(conn.Db)
	fmt.Printf("   Schema version verified: %d (Target: %d)\n", ver, database.TargetSchemaVersion)

	// 2. Execute SQL Seeds (idempotent ON CONFLICT)
	if !skipSeeds {
		fmt.Println("🌱 Step 2: Applying SQL seeds...")
		seedPath := "./script/data/00_seeds_postgresql.sql"
		if _, err := os.Stat(seedPath); err != nil {
			seedPath = "../../script/data/00_seeds_postgresql.sql"
		}
		seedSQL, err := os.ReadFile(seedPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Could not read seed file %s: %v\n", seedPath, err)
			os.Exit(1)
		}
		if _, err := conn.Db.ExecContext(ctx, string(seedSQL)); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to execute seed SQL: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("   SQL seeds applied successfully.")
	}

	// 3. Prepare Artifact Store
	fmt.Println("💾 Step 3: Preparing artifact store...")
	artifacts, err := connection.NewArtifactStore(ctx, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to initialize artifact store: %v\n", err)
		os.Exit(1)
	}

	// 4. Create and admit bot submission bundles
	fmt.Println("🤖 Step 4: Creating Starfighter demo bot bundles...")
	bots := []struct {
		AgentID      string
		SubmissionID string
		BotFile      string
		FallbackSrc  string
		ManifestName string
	}{
		{
			AgentID:      "agent-star-hunter",
			SubmissionID: "sub-star-hunter-1",
			BotFile:      "./games/starfighter/examples/bot_hunter.py",
			FallbackSrc:  `import sys, json` + "\n" + `def act(state): return {"thrust": 1.0, "steer": 0.2, "fire": True}` + "\n",
			ManifestName: "StarHunter",
		},
		{
			AgentID:      "agent-star-evasive",
			SubmissionID: "sub-star-evasive-1",
			BotFile:      "./games/starfighter/examples/bot_evasive.py",
			FallbackSrc:  `import sys, json` + "\n" + `def act(state): return {"thrust": 0.8, "steer": -0.5, "fire": False}` + "\n",
			ManifestName: "StarEvasive",
		},
	}

	for _, b := range bots {
		botCode, err := os.ReadFile(b.BotFile)
		if err != nil {
			// try parent path
			botCode, err = os.ReadFile(filepath.Join("../..", b.BotFile))
		}
		if err != nil {
			fmt.Printf("   ⚠️  Could not read %s, using fallback bot code\n", b.BotFile)
			botCode = []byte(b.FallbackSrc)
		}

		manifestData, err := json.MarshalIndent(model.AgentPackageManifest{
			Name:            b.ManifestName,
			Entrypoint:      "bot.py",
			ProtocolVersion: "1.0",
		}, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error marshaling manifest for %s: %v\n", b.AgentID, err)
			os.Exit(1)
		}

		// Create in-memory zip bundle
		zipBuf := new(bytes.Buffer)
		zw := zip.NewWriter(zipBuf)

		// 1) bot.py
		fBot, err := zw.Create("bot.py")
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error creating zip entry bot.py: %v\n", err)
			os.Exit(1)
		}
		if _, err := fBot.Write(botCode); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error writing bot.py to zip: %v\n", err)
			os.Exit(1)
		}

		// 2) agentrix.json
		fMan, err := zw.Create("agentrix.json")
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error creating zip entry agentrix.json: %v\n", err)
			os.Exit(1)
		}
		if _, err := fMan.Write(manifestData); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error writing agentrix.json to zip: %v\n", err)
			os.Exit(1)
		}
		_ = zw.Close()

		bundleBytes := zipBuf.Bytes()

		// Save files into artifact store
		subpathPrefix := fmt.Sprintf("submissions/%s/v1", b.AgentID)
		codePath, err := artifacts.Save(ctx, subpathPrefix+"/bot.py", botCode)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed saving bot.py to artifacts: %v\n", err)
			os.Exit(1)
		}
		if _, err := artifacts.Save(ctx, subpathPrefix+"/agentrix.json", manifestData); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed saving agentrix.json to artifacts: %v\n", err)
			os.Exit(1)
		}
		if _, err := artifacts.Save(ctx, subpathPrefix+"/bundle.zip", bundleBytes); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed saving bundle.zip to artifacts: %v\n", err)
			os.Exit(1)
		}

		// Upsert submission record
		_, err = conn.Db.ExecContext(ctx, `
			INSERT INTO submissions (id, agent_id, version, code_path, language, status, active, created_at)
			VALUES ($1, $2, 1, $3, 'python', 'ready', TRUE, NOW())
			ON CONFLICT (id) DO UPDATE SET code_path = EXCLUDED.code_path, status = EXCLUDED.status, active = EXCLUDED.active;
		`, b.SubmissionID, b.AgentID, codePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed upserting submission for %s: %v\n", b.AgentID, err)
			os.Exit(1)
		}
		fmt.Printf("   Admitted %s (id: %s, bundle: %d bytes)\n", b.AgentID, b.SubmissionID, len(bundleBytes))
	}

	// 5. Execute authoritative demo match through real simulation engine
	fmt.Println("🏆 Step 5: Executing authoritative Starfighter match through pinned engine...")
	demoMatchID := "match-star-demo-001"
	demoRunID := "run-star-demo-001"
	contestID := "starfighter-cup-2026"
	gameID := "starfighter"

	// Check if already executed
	var matchStatus string
	var matchRunID sql.NullString
	err = conn.Db.QueryRowContext(ctx, "SELECT status, committed_run_id FROM matches WHERE id = $1", demoMatchID).Scan(&matchStatus, &matchRunID)
	if err == nil && matchStatus == "finished" && matchRunID.Valid {
		fmt.Printf("   Demo match %s is already finished and committed with run %s. Idempotent skip.\n", demoMatchID, matchRunID.String)
	} else {
		// Load game registry
		gamesDir := "./games"
		if _, err := os.Stat(gamesDir); err != nil {
			gamesDir = "../../games"
		}
		_ = game.GetRegistry().LoadGamesFromDir(gamesDir)
		manifest := game.GetRegistry().GetManifest("starfighter")
		if manifest != nil && manifest.BinaryPath == "" {
			manifest.BinaryPath = "bin/starfighter-engine"
		}

		repo := repository.NewRepository(conn)
		queue := connection.NewJobQueue(10)
		sandbox := executor.NewSandbox(0)
		svc := service.NewService(repo, artifacts, queue, sandbox)
		exec := executor.NewMatchExecutor(svc, sandbox)

		// Ensure contest entries are mapped to submissions
		_, _ = conn.Db.ExecContext(ctx, `
			INSERT INTO contest_entries (id, contest_id, agent_id, user_id, submission_id, status)
			VALUES
			('entry-hunter-001', $1, 'agent-star-hunter', 'usr-pilot-001', 'sub-star-hunter-1', 'active'),
			('entry-evasive-001', $1, 'agent-star-evasive', 'usr-pilot-002', 'sub-star-evasive-1', 'active')
			ON CONFLICT (id) DO UPDATE SET submission_id = EXCLUDED.submission_id, status = 'active';
		`, contestID)

		// Ensure match slots exist
		_, _ = conn.Db.ExecContext(ctx, `
			INSERT INTO matches (id, contest_id, game_id, status, seed, active, created_at)
			VALUES ($1, $2, $3, 'scheduled', 1337, TRUE, NOW())
			ON CONFLICT (id) DO UPDATE SET status = 'scheduled', active = TRUE;
		`, demoMatchID, contestID, gameID)

		_, _ = conn.Db.ExecContext(ctx, `
			INSERT INTO match_slots (id, match_id, slot_index, contest_entry_id, submission_id, agent_name, username, created_at)
			VALUES
			('slot-demo-1', $1, 0, 'entry-hunter-001', 'sub-star-hunter-1', 'StarHunter', 'pilot_alpha', NOW()),
			('slot-demo-2', $1, 1, 'entry-evasive-001', 'sub-star-evasive-1', 'StarEvasive', 'pilot_beta', NOW())
			ON CONFLICT (id) DO NOTHING;
		`, demoMatchID)

		_, _ = conn.Db.ExecContext(ctx, `
			INSERT INTO match_jobs (id, match_id, contest_id, game_id, status, fencing_token, available_at)
			VALUES ('job-demo-001', $1, $2, $3, 'reserved', 1, NOW())
			ON CONFLICT (id) DO UPDATE SET status = 'reserved', fencing_token = 1;
		`, demoMatchID, contestID, gameID)

		job := &connection.MatchJob{
			JobId:         "job-demo-001",
			RunId:         demoRunID,
			MatchId:       demoMatchID,
			ContestId:     contestID,
			GameId:        gameID,
			SubmissionIds: []string{"sub-star-hunter-1", "sub-star-evasive-1"},
			Seed:          1337,
			FencingToken:  1,
		}

		tracer.InfoEvent(ctx, tracer.ScopeMatch, "bootstrap.match.start", "Ejecutando partida demo con motor y bots reales")
		if err := exec.Execute(ctx, job); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Real match execution failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("   Match %s executed successfully with engine!\n", demoMatchID)

		// Recalculate contest rankings and publish snapshot
		rankings, err := svc.RecalculateContestRankings(ctx, contestID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "⚠️ Could not recalculate rankings: %v\n", err)
		} else {
			fmt.Printf("   Leaderboard recalculated: %d ranked participants\n", len(rankings))
			snapshot, snapErr := svc.PublishRankingSnapshot(ctx, contestID, "usr-admin-001")
			if snapErr == nil {
				fmt.Printf("   Published ranking snapshot v%d\n", snapshot.Version)
			}
		}
	}

	// 6. Optionally queue a live match in PostgreSQL queue for worker
	if queueMatch {
		fmt.Println("⚡ Step 6: Queuing live match in authoritative PostgreSQL queue...")
		liveMatchID := fmt.Sprintf("match-live-%d", time.Now().Unix()%100000)
		_, err = conn.Db.ExecContext(ctx, `
			INSERT INTO matches (id, contest_id, game_id, status, seed, active, created_at)
			VALUES ($1, $2, $3, 'pending', 42, TRUE, NOW());
		`, liveMatchID, contestID, gameID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed creating live match: %v\n", err)
			os.Exit(1)
		}

		submissions, _ := json.Marshal([]string{"sub-star-hunter-1", "sub-star-evasive-1"})
		_, err = conn.Db.ExecContext(ctx, `
			INSERT INTO match_jobs (id, match_id, contest_id, game_id, submission_ids, seed, status, fencing_token, available_at, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5::jsonb, 42, 'pending', 0, NOW(), NOW(), NOW())
			ON CONFLICT (id) DO UPDATE SET status = 'pending';
		`, "job-"+liveMatchID, liveMatchID, contestID, gameID, string(submissions))
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed queuing match job: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("   Match %s queued in match_jobs successfully!\n", liveMatchID)
	}

	fmt.Println("\n==================================================")
	fmt.Println("✅ Starfighter Platform Bootstrap Completed!")
	fmt.Println("==================================================")
	fmt.Println("Credentials & Demo Resources:")
	fmt.Println("  👑 Admin:     admin / admin123")
	fmt.Println("  🚀 Pilot 1:   pilot_alpha / pilot123 (Bot: StarHunter)")
	fmt.Println("  🚀 Pilot 2:   pilot_beta / pilot123  (Bot: StarEvasive)")
	fmt.Println("  🏆 Contest:   starfighter-cup-2026")
	fmt.Println("  🎮 Game:      starfighter")
	fmt.Println("==================================================")
}
