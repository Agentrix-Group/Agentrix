package executor

// Partidas reales con los bots de referencia de Starfighter (ADR-0013).

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/common"
	"github.com/Agentrix-Group/Agentrix/src/connection"
	"github.com/Agentrix-Group/Agentrix/src/engine"
	"github.com/Agentrix-Group/Agentrix/src/game"
	"github.com/Agentrix-Group/Agentrix/src/model"
	replaystream "github.com/Agentrix-Group/Agentrix/src/replay"
)

// realMatchStats resume una partida real a partir de su replay sellado.
type realMatchStats struct {
	fired, hitsTaken map[int]int
	kills            map[int]int
	destroyed        []string
	ranks            map[string]int
	reason           string
	ticks            int
	participants     []string
}

func runRealMatch(t *testing.T, bots []string, seed int64, maxTicks int) realMatchStats {
	engineBin := resolveTestPath("bin/starfighter-engine")
	_ = game.GetRegistry().LoadGamesFromDir(resolveTestPath("games"))
	manifest := game.GetRegistry().GetManifest("starfighter")
	prev := manifest.MaxTicks
	defer func() { manifest.MaxTicks = prev }()
	manifest.BinaryPath = engineBin
	manifest.MaxTicks = maxTicks

	ids := make([]string, len(bots))
	byID := map[string]string{}
	for i, b := range bots {
		ids[i] = fmt.Sprintf("%s-%d", strings.TrimSuffix(filepath.Base(b), ".py"), i)
		byID[ids[i]] = resolveTestPath(b)
	}
	replayPath := filepath.Join(t.TempDir(), "match.ndjson")
	st := realMatchStats{fired: map[int]int{}, hitsTaken: map[int]int{}, kills: map[int]int{}, ranks: map[string]int{}}
	svc := &mockExecutorService{
		getMatchFn: func(ctx context.Context, id string) (*model.Match, error) {
			return &model.Match{Id: id, GameId: "starfighter", Status: common.MatchStatusPending}, nil
		},
		getSubmissionFn: func(ctx context.Context, id string) (*model.Submission, error) {
			return &model.Submission{Id: id, AgentId: id, Language: "python", CodePath: byID[id], Status: common.SubmissionStatusReady, Active: true}, nil
		},
		openReplayFn: func(ctx context.Context, r *model.Replay, m model.ReplayMetadata) (replaystream.StreamWriter, error) {
			f, err := os.Create(replayPath)
			if err != nil {
				return nil, err
			}
			return replaystream.NewStreamWriter(f, m)
		},
		createResultFn: func(ctx context.Context, res *model.Result) error {
			st.ranks[res.SubmissionId] = res.Rank
			return nil
		},
	}
	factory := func(ctx context.Context, _ *connection.MatchJob) (engine.EngineClient, error) {
		c := engine.NewSubprocessClient()
		return c, c.Start(ctx, engine.StartConfig{BinaryPath: engineBin, HandshakeTimeout: 5 * time.Second})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	if err := NewMatchExecutor(svc, NewSandbox(500*time.Millisecond), factory).Execute(ctx, &connection.MatchJob{
		JobId: "job-real", MatchId: "match-real", GameId: "starfighter", SubmissionIds: ids, Seed: seed,
	}); err != nil {
		t.Fatalf("match failed: %v", err)
	}
	f, _ := os.Open(replayPath)
	defer f.Close()
	doc, err := replaystream.DecodeNDJSON(f)
	if err != nil {
		t.Fatal(err)
	}
	st.reason, st.ticks, st.participants = doc.Result.Reason, doc.Result.FinalTick, doc.Metadata.Participants
	for _, s := range doc.Snapshots {
		var pub struct {
			Events []struct {
				Type     string `json:"type"`
				PlayerID int    `json:"player_id"`
				Killer   *int   `json:"killer"`
			} `json:"events"`
		}
		_ = json.Unmarshal(s.PublicSnapshot, &pub)
		for _, e := range pub.Events {
			switch e.Type {
			case "fired":
				st.fired[e.PlayerID]++
			case "hit":
				st.hitsTaken[e.PlayerID]++
			case "destroyed":
				st.destroyed = append(st.destroyed, fmt.Sprintf("P%d@%d", e.PlayerID, s.Tick))
				if e.Killer != nil {
					st.kills[*e.Killer]++
				}
			}
		}
	}
	return st
}

func total(m map[int]int) int {
	n := 0
	for _, v := range m {
		n += v
	}
	return n
}

// Cinco copias de bot_ace.py pelean de verdad: la partida termina por
// eliminación, con bajas atribuidas y un único primer puesto.
func TestReferenceBots_FiveAcesFightToElimination(t *testing.T) {
	ace := "games/starfighter/examples/bot_ace.py"
	st := runRealMatch(t, []string{ace, ace, ace, ace, ace}, 1, 3600)
	if st.reason != "eliminated" {
		t.Fatalf("reason = %q after %d ticks, want eliminated", st.reason, st.ticks)
	}
	if len(st.destroyed) != 4 {
		t.Fatalf("destroyed = %v, want four ships out", st.destroyed)
	}
	if total(st.kills) < 2 {
		t.Fatalf("kills = %v, want at least two kills by bullets", st.kills)
	}
	first := 0
	for _, rank := range st.ranks {
		if rank == 1 {
			first++
		}
	}
	if first != 1 {
		t.Fatalf("ranks = %v, want a single winner", st.ranks)
	}
}

// bot_ace.py esquiva y ataca: en un duelo contra bot_hunter.py (que solo
// ataca) no recibe impactos y gana.
func TestReferenceBots_AceOutDuelsHunter(t *testing.T) {
	st := runRealMatch(t, []string{
		"games/starfighter/examples/bot_ace.py",
		"games/starfighter/examples/bot_hunter.py",
	}, 1, 3600)
	if st.reason != "eliminated" || st.ranks["bot_ace-0"] != 1 {
		t.Fatalf("reason=%q ranks=%v, want the ace to eliminate the hunter", st.reason, st.ranks)
	}
	if st.hitsTaken[0] != 0 || st.fired[1] == 0 {
		t.Fatalf("hits taken by ace = %d with hunter shots = %d, want every shot dodged", st.hitsTaken[0], st.fired[1])
	}
}
