package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/engine"
)

func main() {
	mode := flag.String("mode", "normal", "Failure or operational mode for testing")
	flag.Parse()

	_, _ = fmt.Fprintf(os.Stderr, "[fake-engine] Started (mode=%s, pid=%d)\n", *mode, os.Getpid())

	// Handshake failure modes
	if *mode == "crash_on_start" {
		_, _ = fmt.Fprintf(os.Stderr, "[fake-engine] Crashing intentionally on startup\n")
		os.Exit(2)
	}

	if *mode == "timeout_on_start" {
		_, _ = fmt.Fprintf(os.Stderr, "[fake-engine] Sleeping to trigger handshake timeout\n")
		time.Sleep(10 * time.Second)
		return
	}

	if *mode == "invalid_json_on_start" {
		_, _ = fmt.Println("{invalid-json-payload-corrupted")
		return
	}

	protocolVersion := engine.ProtocolVersion
	if *mode == "incompatible_version" {
		protocolVersion = "agentrix-engine/99.0"
	}

	// 1. Emit engine_ready
	var sendSeq uint64 = 1
	readyEnv := engine.Envelope{
		ProtocolVersion: protocolVersion,
		Type:            engine.TypeEngineReady,
		MatchID:         "",
		RunID:           "",
		Sequence:        sendSeq,
		Payload: map[string]interface{}{
			"engineVersion": "0.3.0",
			"engineDigest":  "b67fd4613e5cc5d2d742b59c3f5d7846f83f348fb01fac5f521a05c57aef1cf6",
			"supportedProtocols": []string{
				protocolVersion,
			},
			"capabilities": map[string]interface{}{
				"headless":                 true,
				"deterministic":            true,
				"avian2d":                  true,
				"authoritative_commitment": true,
			},
		},
	}
	sendSeq++
	writeEnvelope(readyEnv)

	// Simulation state
	scanner := bufio.NewScanner(os.Stdin)
	// Support reading large input
	buf := make([]byte, 1024*1024)
	scanner.Buffer(buf, 2*1024*1024)

	var matchID string
	var runID string
	var seed int64
	var maxTicks int
	var currentTick int
	var lastStateHash string
	var players []string
	scores := make(map[string]int)
	alive := make(map[string]bool)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		var inEnv engine.Envelope
		if err := json.Unmarshal([]byte(line), &inEnv); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "[fake-engine] Error parsing input JSON: %v\n", err)
			continue
		}

		_, _ = fmt.Fprintf(os.Stderr, "[fake-engine] Received command: %s (seq=%d)\n", inEnv.Type, inEnv.Sequence)

		switch inEnv.Type {
		case engine.TypeInitializeMatch:
			if *mode == "engine_error" {
				errEnv := engine.Envelope{
					ProtocolVersion: engine.ProtocolVersion,
					Type:            engine.TypeEngineError,
					MatchID:         inEnv.MatchID,
					RunID:           inEnv.RunID,
					Sequence:        sendSeq,
					Payload: map[string]interface{}{
						"code":    "ERR_FAKE_SIMULATION_ABORT",
						"message": "Intentional error from fake engine",
						"fatal":   true,
					},
				}
				sendSeq++
				writeEnvelope(errEnv)
				continue
			}

			if *mode == "invalid_sequence" {
				sendSeq = 9999 // Send bad sequence
			}

			if *mode == "crash_during_match" {
				_, _ = fmt.Fprintf(os.Stderr, "[fake-engine] Crashing during match initialization\n")
				os.Exit(3)
			}

			var req engine.InitializeMatchRequest
			reqBytes, _ := json.Marshal(inEnv.Payload)
			_ = json.Unmarshal(reqBytes, &req)

			matchID = req.MatchID
			if matchID == "" {
				matchID = inEnv.MatchID
			}
			runID = inEnv.RunID
			if *mode == "mismatch_match_id" {
				matchID = "m-unexpected-fake-id"
			}
			seed = req.Seed
			maxTicks = req.MaxTicks
			if maxTicks <= 0 {
				maxTicks = 100
			}
			players = req.Players
			if len(players) == 0 {
				if slotsRaw, ok := inEnv.Payload["slots"].([]interface{}); ok {
					for _, s := range slotsRaw {
						if sm, ok := s.(map[string]interface{}); ok {
							if slotID, ok := sm["slotId"].(string); ok {
								players = append(players, slotID)
							}
						}
					}
				}
			}
			if len(players) == 0 {
				players = []string{"bot-1", "bot-2"}
			}

			for _, p := range players {
				scores[p] = 0
				alive[p] = true
			}

			initHash := calculateStateHash(seed, 0, scores)
			currentTick = 0
			lastStateHash = initHash
			perceptions := make(map[string]map[string]interface{})
			for i, p := range players {
				perceptions[p] = map[string]interface{}{
					"slot":  i + 1,
					"alive": true,
					"tick":  0,
				}
			}
			publicSnapshot := map[string]interface{}{
				"tick": 0, "fighters": []interface{}{}, "bullets": []interface{}{},
				"events": []string{"Match initialized by fake engine"}, "stateHash": initHash,
			}

			initEnv := engine.Envelope{
				ProtocolVersion: engine.ProtocolVersion,
				Type:            engine.TypeMatchInitialized,
				MatchID:         matchID,
				RunID:           runID,
				Sequence:        sendSeq,
				Payload: map[string]interface{}{
					"matchId":        matchID,
					"tick":           0,
					"initialTick":    0,
					"stateHash":      initHash,
					"events":         []string{"Match initialized by fake engine"},
					"perceptions":    perceptions,
					"observations":   perceptions,
					"publicSnapshot": publicSnapshot,
					"commitments": map[string]interface{}{
						"executionSpecDigest":          initHash,
						"actionBatchDigest":            initHash,
						"authoritativeStateCommitment": initHash,
						"publicSnapshotHash":           initHash,
						"replayChainDigest":            initHash,
					},
				},
			}
			sendSeq++
			writeEnvelope(initEnv)

		case engine.TypeAdvanceTick:
			if *mode == "timeout_on_tick" {
				_, _ = fmt.Fprintf(os.Stderr, "[fake-engine] Sleeping on tick advance to trigger timeout\n")
				time.Sleep(10 * time.Second)
				return
			}

			if *mode == "large_message" {
				hugeString := strings.Repeat("A", 2*1024*1024)
				bigEnv := engine.Envelope{
					ProtocolVersion: engine.ProtocolVersion,
					Type:            engine.TypeTickCompleted,
					MatchID:         matchID,
					RunID:           runID,
					Sequence:        sendSeq,
					Payload: map[string]interface{}{
						"tick":      1,
						"events":    []string{hugeString},
						"stateHash": "huge",
						"isOver":    false,
					},
				}
				sendSeq++
				writeEnvelope(bigEnv)
				continue
			}

			var tickReq engine.AdvanceTickRequest
			reqBytes, _ := json.Marshal(inEnv.Payload)
			_ = json.Unmarshal(reqBytes, &tickReq)

			actionTick := tickReq.Tick
			resultingTick := actionTick + 1
			rng := rand.New(rand.NewSource(seed + int64(actionTick)))

			events := make([]string, 0)
			for pID, act := range tickReq.Actions {
				if !alive[pID] {
					continue
				}

				if act.Status == engine.ActionStatusValid {
					gain := rng.Intn(15) + 5
					scores[pID] += gain
					events = append(events, fmt.Sprintf("%s submitted a valid action and gained %d points", pID, gain))
				} else {
					events = append(events, fmt.Sprintf("%s had non-valid action: %s", pID, act.Status))
				}
			}

			stateHash := calculateStateHash(seed, resultingTick, scores)
			currentTick = resultingTick
			lastStateHash = stateHash
			isOver := resultingTick >= maxTicks

			winner := ""
			if isOver {
				highestScore := -1
				for pID, sc := range scores {
					if sc > highestScore {
						highestScore = sc
						winner = pID
					}
				}
			}

			perceptions := make(map[string]map[string]interface{})
			for _, p := range players {
				perceptions[p] = map[string]interface{}{
					"tick":  resultingTick,
					"alive": alive[p],
					"score": scores[p],
				}
			}
			fighters := make([]map[string]interface{}, 0, len(players))
			for index, p := range players {
				fighters = append(fighters, map[string]interface{}{
					"playerId":     p,
					"position":     map[string]float64{"x": float64(index*200 - 100), "y": 0},
					"rotation":     0,
					"health":       100,
					"shieldActive": false,
				})
			}
			publicSnapshot := map[string]interface{}{
				"tick": resultingTick, "fighters": fighters, "bullets": []interface{}{},
				"events": events, "stateHash": stateHash,
			}

			tickEnv := engine.Envelope{
				ProtocolVersion: engine.ProtocolVersion,
				Type:            engine.TypeTickCompleted,
				MatchID:         matchID,
				RunID:           runID,
				Sequence:        sendSeq,
				Payload: map[string]interface{}{
					"tick":           resultingTick,
					"events":         events,
					"stateHash":      stateHash,
					"isOver":         isOver,
					"terminal":       isOver,
					"winner":         winner,
					"perceptions":    perceptions,
					"observations":   perceptions,
					"publicSnapshot": publicSnapshot,
					"commitments": map[string]interface{}{
						"executionSpecDigest":          stateHash,
						"actionBatchDigest":            stateHash,
						"authoritativeStateCommitment": stateHash,
						"publicSnapshotHash":           stateHash,
						"replayChainDigest":            stateHash,
					},
				},
			}
			sendSeq++
			writeEnvelope(tickEnv)

		case engine.TypeFinishMatch:
			reason := "score_limit"
			if r, ok := inEnv.Payload["reason"].(string); ok && r != "" {
				reason = r
			}
			winner := ""
			highestScore := -1
			for pID, sc := range scores {
				if sc > highestScore {
					highestScore = sc
					winner = pID
				}
			}

			rankings := make([]engine.PlayerRank, 0)
			for pID, sc := range scores {
				rank := 2
				if pID == winner {
					rank = 1
				}
				rankings = append(rankings, engine.PlayerRank{
					PlayerID: pID,
					Rank:     rank,
					Score:    sc,
				})
			}

			doneEnv := engine.Envelope{
				ProtocolVersion: engine.ProtocolVersion,
				Type:            engine.TypeMatchCompleted,
				MatchID:         matchID,
				RunID:           runID,
				Sequence:        sendSeq,
				Payload: map[string]interface{}{
					"finalTick":      currentTick,
					"reason":         reason,
					"winner":         winner,
					"scores":         scores,
					"rankings":       rankings,
					"finalStateHash": lastStateHash,
					"commitments": map[string]interface{}{
						"executionSpecDigest":          lastStateHash,
						"actionBatchDigest":            lastStateHash,
						"authoritativeStateCommitment": lastStateHash,
						"publicSnapshotHash":           lastStateHash,
						"replayChainDigest":            lastStateHash,
					},
				},
			}
			sendSeq++
			writeEnvelope(doneEnv)

		case engine.TypeShutdown:
			ackEnv := engine.Envelope{
				ProtocolVersion: engine.ProtocolVersion,
				Type:            engine.TypeShutdownAck,
				MatchID:         matchID,
				RunID:           runID,
				Sequence:        sendSeq,
				Payload: map[string]interface{}{
					"status": "ok",
				},
			}
			writeEnvelope(ackEnv)
			_, _ = fmt.Fprintf(os.Stderr, "[fake-engine] Clean shutdown complete\n")
			os.Exit(0)
		}
	}
}

func calculateStateHash(seed int64, tick int, scores map[string]int) string {
	h := sha256.New()
	_, _ = fmt.Fprintf(h, "seed:%d|tick:%d", seed, tick)
	keys := make([]string, 0, len(scores))
	for p := range scores {
		keys = append(keys, p)
	}
	sort.Strings(keys)
	for _, p := range keys {
		_, _ = fmt.Fprintf(h, "|%s:%d", p, scores[p])
	}
	return hex.EncodeToString(h.Sum(nil))
}

func writeEnvelope(env engine.Envelope) {
	data, err := json.Marshal(env)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "[fake-engine] Marshal error: %v\n", err)
		return
	}
	// Strict JSON Lines format: 1 single line terminated by newline
	_, _ = fmt.Fprintf(os.Stdout, "%s\n", string(data))
}
