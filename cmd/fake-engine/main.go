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

	"github.com/F4nk1/Agentrix/src/engine"
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
		Sequence:        sendSeq,
		Payload: map[string]interface{}{
			"engineVersion": "0.1.0-fake",
			"supportedProtocols": []string{
				protocolVersion,
			},
			"capabilities": map[string]interface{}{
				"headless":      true,
				"deterministic": true,
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
	var seed int64
	var maxTicks int
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
			seed = req.Seed
			maxTicks = req.MaxTicks
			if maxTicks <= 0 {
				maxTicks = 100
			}
			players = req.Players

			for _, p := range players {
				scores[p] = 0
				alive[p] = true
			}

			initHash := calculateStateHash(seed, 0, scores)
			perceptions := make(map[string]map[string]interface{})
			for i, p := range players {
				perceptions[p] = map[string]interface{}{
					"slot":  i + 1,
					"alive": true,
					"tick":  0,
				}
			}

			initEnv := engine.Envelope{
				ProtocolVersion: engine.ProtocolVersion,
				Type:            engine.TypeMatchInitialized,
				MatchID:         matchID,
				Sequence:        sendSeq,
				Payload: map[string]interface{}{
					"matchId":     matchID,
					"initialTick": 0,
					"stateHash":   initHash,
					"events":      []string{"Match initialized by fake engine"},
					"perceptions": perceptions,
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

			currentTick := tickReq.Tick
			rng := rand.New(rand.NewSource(seed + int64(currentTick)))

			events := make([]string, 0)
			for pID, act := range tickReq.Actions {
				if !alive[pID] {
					continue
				}

				if act.Status == engine.ActionStatusValid {
					gain := rng.Intn(15) + 5
					scores[pID] += gain
					events = append(events, fmt.Sprintf("%s performed %s and gained %d points", pID, act.ActionType, gain))
				} else {
					events = append(events, fmt.Sprintf("%s had non-valid action: %s", pID, act.Status))
				}
			}

			stateHash := calculateStateHash(seed, currentTick, scores)
			isOver := currentTick >= maxTicks

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
					"tick":  currentTick,
					"alive": alive[p],
					"score": scores[p],
				}
			}

			tickEnv := engine.Envelope{
				ProtocolVersion: engine.ProtocolVersion,
				Type:            engine.TypeTickCompleted,
				MatchID:         matchID,
				Sequence:        sendSeq,
				Payload: map[string]interface{}{
					"tick":        currentTick,
					"events":      events,
					"stateHash":   stateHash,
					"isOver":      isOver,
					"winner":      winner,
					"perceptions": perceptions,
				},
			}
			sendSeq++
			writeEnvelope(tickEnv)

		case engine.TypeFinishMatch:
			reason := "time_limit"
			if r, ok := inEnv.Payload["reason"].(string); ok && r != "" {
				reason = r
			}
			finishHash := calculateStateHash(seed, 9999, scores)

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
				Sequence:        sendSeq,
				Payload: map[string]interface{}{
					"finalTick":      maxTicks,
					"reason":         reason,
					"winner":         winner,
					"scores":         scores,
					"rankings":       rankings,
					"finalStateHash": finishHash,
				},
			}
			sendSeq++
			writeEnvelope(doneEnv)

		case engine.TypeShutdown:
			ackEnv := engine.Envelope{
				ProtocolVersion: engine.ProtocolVersion,
				Type:            engine.TypeShutdownAck,
				MatchID:         matchID,
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
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
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
