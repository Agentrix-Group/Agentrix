package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Agentrix-Group/Agentrix/src/engine"
	"github.com/Agentrix-Group/Agentrix/src/game"
	"github.com/Agentrix-Group/Agentrix/src/service"
)

// Admission runs a candidate bot in the real sandbox with the module's
// limits and admission perception; the bot must answer tick 0 with a JSON
// object action.
type Admission struct {
	runtime BotRuntime
}

func NewAdmission(runtime BotRuntime) *Admission { return &Admission{runtime: runtime} }

func (a *Admission) Admit(ctx context.Context, module *game.Module, codePath string) error {
	limits := module.ExecutionLimits()
	session, err := StartSession(ctx, a.runtime, "admission", module.ID, 1, limits,
		[]BotLaunch{{PlayerID: "candidate", CodePath: codePath, Runtime: module.Runtimes[0]}})
	if err != nil {
		return fmt.Errorf("admission sandbox: %w", err)
	}
	defer session.Close("", "admission_check")
	actions := session.Turn(ctx, 0, map[string]json.RawMessage{"candidate": module.AdmissionPerception()})
	if ctx.Err() != nil {
		return ctx.Err()
	}
	action := actions["candidate"]
	if action.Status == engine.ActionStatusValid {
		return nil
	}
	reason := "bot failed the admission tick: " + action.ErrorDetails
	if tail := strings.TrimSpace(session.StderrTail("candidate")); tail != "" {
		lines := strings.Split(tail, "\n")
		if len(lines) > 12 {
			lines = lines[len(lines)-12:]
		}
		reason += "\n" + strings.Join(lines, "\n")
	}
	return &service.AdmissionRejection{Reason: reason}
}
