package main

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/config"
	"github.com/Agentrix-Group/Agentrix/src/connection"
	"github.com/Agentrix-Group/Agentrix/src/database"
	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/Agentrix-Group/Agentrix/src/replay"
)

func main() {
	var (
		skipSeeds bool
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
		AgentID     string
		SubmissionID string
		BotFile     string
		FallbackSrc string
		ManifestName string
	}{
		{
			AgentID:      "agent-star-hunter",
			SubmissionID: "sub-star-hunter-001",
			BotFile:      "./games/starfighter/examples/bot_hunter.py",
			FallbackSrc:  `import sys, json` + "\n" + `def act(state): return {"thrust": 1.0, "steer": 0.2, "fire": True}` + "\n",
			ManifestName: "StarHunter",
		},
		{
			AgentID:      "agent-star-evasive",
			SubmissionID: "sub-star-evasive-001",
			BotFile:      "./games/starfighter/examples/bot_evasive.py",
			FallbackSrc:  `import sys, json` + "\n" + `def act(state): return {"thrust": 0.8, "steer": -0.5, "fire": False}` + "\n",
			ManifestName: "StarEvasive",
		},
	}

	for _, b := range bots {
		botCode, err := os.ReadFile(b.BotFile)
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

	// 5. Generate and seed authoritative demo match with results and replay
	fmt.Println("🏆 Step 5: Generating Starfighter demo match and replay...")
	demoMatchID := "match-star-demo-001"
	demoReplayID := "replay-demo-001"
	contestID := "starfighter-cup-2026"
	gameID := "starfighter"

	// Create replay NDJSON data
	now := time.Now().UTC()
	replayMeta := model.ReplayMetadata{
		Type:            "metadata",
		ReplayID:        demoReplayID,
		MatchID:         demoMatchID,
		GameID:          gameID,
		Seed:            1337,
		Participants:    []string{"agent-star-hunter", "agent-star-evasive"},
		FixedTimestepMs: 16,
		CreatedAt:       now,
	}

	var replayBuf bytes.Buffer
	writer, err := replay.NewStreamWriter(&nopCloser{&replayBuf}, replayMeta)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error creating replay stream writer: %v\n", err)
		os.Exit(1)
	}

	// Write 5 sample snapshots simulating Starfighter dogfight
	for tick := 0; tick < 5; tick++ {
		snapshotJSON := fmt.Sprintf(`{"tick":%d,"hunter":{"x":%d,"y":%d,"shield":100},"evasive":{"x":%d,"y":%d,"shield":%d}}`,
			tick, 100+tick*5, 200, 300-tick*4, 200, 100-tick*15)
		_ = writer.WriteSnapshot(model.ReplaySnapshot{
			Type:           "snapshot",
			Tick:           tick,
			PublicSnapshot: json.RawMessage(snapshotJSON),
			StateHash:      fmt.Sprintf("hash-tick-%d", tick),
		})
	}
	_ = writer.Complete(model.ReplayResult{
		Type:           "result",
		FinalTick:      5,
		Winner:         "agent-star-hunter",
		Scores:         map[string]int{"agent-star-hunter": 10, "agent-star-evasive": 2},
		Reason:         "shield_depleted",
		FinalStateHash: "hash-tick-5-final",
		FinishedAt:     now,
	})

	replayBytes := replayBuf.Bytes()
	replayHashBytes := sha256.Sum256(replayBytes)
	replaySHA := hex.EncodeToString(replayHashBytes[:])

	replaySubpath := fmt.Sprintf("replays/%s.ndjson", demoReplayID)
	replaySavedPath, err := artifacts.Save(ctx, replaySubpath, replayBytes)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed saving replay to artifacts: %v\n", err)
		os.Exit(1)
	}

	// Upsert Match
	_, err = conn.Db.ExecContext(ctx, `
		INSERT INTO matches (id, contest_id, game_id, status, seed, replay_id, active, created_at, finished_at)
		VALUES ($1, $2, $3, 'finished', 1337, $4, TRUE, NOW() - INTERVAL '10 minutes', NOW())
		ON CONFLICT (id) DO UPDATE SET status = EXCLUDED.status, replay_id = EXCLUDED.replay_id, finished_at = EXCLUDED.finished_at;
	`, demoMatchID, contestID, gameID, demoReplayID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed creating demo match: %v\n", err)
		os.Exit(1)
	}

	// Upsert Replay
	_, err = conn.Db.ExecContext(ctx, `
		INSERT INTO replays (id, match_id, file_path, duration_ticks, summary, sha256, size_bytes, active, created_at)
		VALUES ($1, $2, $3, 5, 'Starfighter Demo Dogfight - Hunter vs Evasive', $4, $5, TRUE, NOW())
		ON CONFLICT (id) DO UPDATE SET file_path = EXCLUDED.file_path, sha256 = EXCLUDED.sha256, size_bytes = EXCLUDED.size_bytes;
	`, demoReplayID, demoMatchID, replaySavedPath, replaySHA, int64(len(replayBytes)))
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed creating demo replay: %v\n", err)
		os.Exit(1)
	}

	// Upsert Results
	_, err = conn.Db.ExecContext(ctx, `
		INSERT INTO results (id, match_id, submission_id, score, "rank", status, details, created_at) VALUES
		('res-hunter-001', $1, 'sub-star-hunter-001', 10, 1, 'finished', 'Target destroyed', NOW()),
		('res-evasive-001', $1, 'sub-star-evasive-001', 2, 2, 'finished', 'Shields depleted', NOW())
		ON CONFLICT (id) DO UPDATE SET score = EXCLUDED.score, "rank" = EXCLUDED."rank", status = EXCLUDED.status;
	`, demoMatchID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed creating demo results: %v\n", err)
		os.Exit(1)
	}

	// Update rankings with match results
	_, err = conn.Db.ExecContext(ctx, `
		UPDATE rankings SET score = 10, matches_played = 1, wins = 1, losses = 0, draws = 0, "rank" = 1, updated_at = NOW()
		WHERE contest_id = $1 AND agent_id = 'agent-star-hunter';
		UPDATE rankings SET score = 2, matches_played = 1, wins = 0, losses = 1, draws = 0, "rank" = 2, updated_at = NOW()
		WHERE contest_id = $1 AND agent_id = 'agent-star-evasive';
	`, contestID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed updating rankings: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("   Demo match %s created with replay %s (%d bytes)\n", demoMatchID, demoReplayID, len(replayBytes))

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

		payload, _ := json.Marshal(map[string]any{
			"match_id": liveMatchID,
			"game_id":  gameID,
			"agents": []string{
				"agent-star-hunter",
				"agent-star-evasive",
			},
		})

		_, err = conn.Db.ExecContext(ctx, `
			INSERT INTO match_jobs (id, match_id, contest_id, game_id, payload, status, created_at, scheduled_at)
			VALUES ($1, $2, $3, $4, $5, 'queued', NOW(), NOW());
		`, "job-"+liveMatchID, liveMatchID, contestID, gameID, string(payload))
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
	fmt.Println("  📼 Replay:    /api/v1/replays/replay-demo-001")
	fmt.Println("==================================================")
}

type nopCloser struct {
	*bytes.Buffer
}

func (n *nopCloser) Close() error {
	return nil
}
