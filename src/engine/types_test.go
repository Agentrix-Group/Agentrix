package engine

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEnvelopeSerialization(t *testing.T) {
	r := require.New(t)

	env := Envelope{
		ProtocolVersion: ProtocolVersion,
		Type:            TypeAdvanceTick,
		MatchID:         "m-123",
		Sequence:        1,
		Payload: map[string]interface{}{
			"tick": 1,
		},
	}

	data, err := json.Marshal(env)
	r.NoError(err)
	r.Contains(string(data), `"protocolVersion":"agentrix-engine/1"`)
	r.Contains(string(data), `"type":"advance_tick"`)
	r.Contains(string(data), `"sequence":1`)

	var parsed Envelope
	err = json.Unmarshal(data, &parsed)
	r.NoError(err)
	r.Equal(ProtocolVersion, parsed.ProtocolVersion)
	r.Equal(TypeAdvanceTick, parsed.Type)
	r.Equal("m-123", parsed.MatchID)
	r.Equal(uint64(1), parsed.Sequence)
}

func TestInitializeAndTickDTOs(t *testing.T) {
	r := require.New(t)

	req := InitializeMatchRequest{
		MatchID:         "m-100",
		GameID:          "arena-basica",
		Seed:            42,
		FixedTimestepMs: 50,
		MaxTicks:        100,
		Players:         []string{"bot-1", "bot-2"},
	}

	data, err := json.Marshal(req)
	r.NoError(err)
	r.Contains(string(data), `"seed":42`)

	var parsedReq InitializeMatchRequest
	err = json.Unmarshal(data, &parsedReq)
	r.NoError(err)
	r.Equal("m-100", parsedReq.MatchID)
	r.Len(parsedReq.Players, 2)

	tickReq := AdvanceTickRequest{
		Tick: 1,
		Actions: map[string]PlayerActionInput{
			"bot-1": {
				Status:     ActionStatusValid,
				ActionType: "MOVE",
				Payload:    map[string]interface{}{"direction": "UP"},
			},
			"bot-2": {
				Status:       ActionStatusTimeout,
				ErrorDetails: "Execution timed out",
			},
		},
	}

	tickData, err := json.Marshal(tickReq)
	r.NoError(err)
	r.Contains(string(tickData), `"status":"valid"`)
	r.Contains(string(tickData), `"status":"timeout"`)

	res := MatchResult{
		FinalTick:      100,
		Reason:         "victory",
		Winner:         "bot-1",
		Scores:         map[string]int{"bot-1": 150, "bot-2": 50},
		Rankings:       []PlayerRank{{PlayerID: "bot-1", Rank: 1, Score: 150}},
		FinalStateHash: "sha256:hash123",
	}
	resData, err := json.Marshal(res)
	r.NoError(err)
	r.Contains(string(resData), `"reason":"victory"`)
}

func TestContractExamplesValidation(t *testing.T) {
	r := require.New(t)

	validDir := "../../protocol/engine/v1/examples/valid"
	files, err := os.ReadDir(validDir)
	if err != nil {
		validDir = "protocol/engine/v1/examples/valid"
		files, err = os.ReadDir(validDir)
	}
	r.NoError(err)
	r.NotEmpty(files)

	for _, f := range files {
		if !strings.HasSuffix(f.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(validDir, f.Name()))
		r.NoError(err, "Failed reading %s", f.Name())

		var env Envelope
		err = json.Unmarshal(data, &env)
		r.NoError(err, "Failed unmarshaling %s", f.Name())
		r.Equal(ProtocolVersion, env.ProtocolVersion, "Wrong protocol version in %s", f.Name())
		r.True(env.Sequence >= 1, "Sequence must be >= 1 in %s", f.Name())
		r.NotEmpty(env.Type, "Type must not be empty in %s", f.Name())
		r.NotNil(env.Payload, "Payload must not be nil in %s", f.Name())

		// Test payload mapping
		payloadBytes, _ := json.Marshal(env.Payload)
		switch env.Type {
		case TypeInitializeMatch:
			var p InitializeMatchRequest
			r.NoError(json.Unmarshal(payloadBytes, &p))
			r.NotEmpty(p.MatchID)
			r.NotEmpty(p.Players)
		case TypeAdvanceTick:
			var p AdvanceTickRequest
			r.NoError(json.Unmarshal(payloadBytes, &p))
			r.True(p.Tick > 0)
			r.NotEmpty(p.Actions)
		case TypeTickCompleted:
			var p TickResult
			r.NoError(json.Unmarshal(payloadBytes, &p))
			r.True(p.Tick > 0)
			r.NotEmpty(p.StateHash)
		case TypeMatchCompleted:
			var p MatchResult
			r.NoError(json.Unmarshal(payloadBytes, &p))
			r.NotEmpty(p.Reason)
			r.NotEmpty(p.FinalStateHash)
		case TypeEngineReady:
			var p EngineReadyPayload
			r.NoError(json.Unmarshal(payloadBytes, &p))
			r.NotEmpty(p.EngineVersion)
			r.NotEmpty(p.SupportedProtocols)
		case TypeMatchInitialized:
			var p MatchInitializedResult
			r.NoError(json.Unmarshal(payloadBytes, &p))
			r.NotEmpty(p.MatchID)
			r.NotEmpty(p.StateHash)
		case TypeEngineError:
			var p EngineErrorPayload
			r.NoError(json.Unmarshal(payloadBytes, &p))
			r.NotEmpty(p.Code)
		}
	}

	// Test invalid examples
	invalidDir := "../../protocol/engine/v1/examples/invalid"
	invFiles, err := os.ReadDir(invalidDir)
	if err != nil {
		invalidDir = "protocol/engine/v1/examples/invalid"
		invFiles, err = os.ReadDir(invalidDir)
	}
	r.NoError(err)
	r.NotEmpty(invFiles)

	for _, f := range invFiles {
		if !strings.HasSuffix(f.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(invalidDir, f.Name()))
		r.NoError(err)

		var env Envelope
		_ = json.Unmarshal(data, &env)

		isKnownType := false
		switch env.Type {
		case TypeInitializeMatch, TypeAdvanceTick, TypeFinishMatch, TypeShutdown,
			TypeEngineReady, TypeMatchInitialized, TypeTickCompleted, TypeMatchCompleted,
			TypeEngineError, TypeShutdownAck:
			isKnownType = true
		}

		// Each invalid file must violate at least one requirement
		isInvalid := env.ProtocolVersion != ProtocolVersion ||
			env.Sequence == 0 ||
			!isKnownType

		if !isInvalid {
			// Check payload invalidity
			payloadBytes, _ := json.Marshal(env.Payload)
			switch env.Type {
			case TypeAdvanceTick:
				var p AdvanceTickRequest
				_ = json.Unmarshal(payloadBytes, &p)
				if p.Tick == 0 {
					isInvalid = true
				}
			case TypeInitializeMatch:
				var p InitializeMatchRequest
				_ = json.Unmarshal(payloadBytes, &p)
				if len(p.Players) == 0 {
					isInvalid = true
				}
			}
		}

		r.True(isInvalid, "File %s should be identified as invalid", f.Name())
	}
}
