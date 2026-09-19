package server

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/Agentrix-Group/Agentrix/src/service"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

type mockAgentsService struct {
	service.Service
	listAgentsFn              func(ctx context.Context) ([]model.Agent, error)
	listAgentsByParticipantFn func(ctx context.Context, participantId string) ([]model.Agent, error)
	getAgentFn                func(ctx context.Context, id string) (*model.Agent, error)
	createAgentFn             func(ctx context.Context, agent *model.Agent) error
	updateAgentFn             func(ctx context.Context, agent *model.Agent) error
	activateAgentFn           func(ctx context.Context, id string, isActive bool) error
}

func (m *mockAgentsService) ListAgents(ctx context.Context) ([]model.Agent, error) {
	if m.listAgentsFn != nil {
		return m.listAgentsFn(ctx)
	}
	return nil, nil
}

func (m *mockAgentsService) ListAgentsByParticipant(ctx context.Context, participantId string) ([]model.Agent, error) {
	if m.listAgentsByParticipantFn != nil {
		return m.listAgentsByParticipantFn(ctx, participantId)
	}
	return nil, nil
}

func (m *mockAgentsService) GetAgent(ctx context.Context, id string) (*model.Agent, error) {
	if m.getAgentFn != nil {
		return m.getAgentFn(ctx, id)
	}
	return &model.Agent{Id: id, Name: "Agent1"}, nil
}

func (m *mockAgentsService) CreateAgent(ctx context.Context, agent *model.Agent) error {
	if m.createAgentFn != nil {
		return m.createAgentFn(ctx, agent)
	}
	return nil
}

func (m *mockAgentsService) UpdateAgent(ctx context.Context, agent *model.Agent) error {
	if m.updateAgentFn != nil {
		return m.updateAgentFn(ctx, agent)
	}
	return nil
}

func (m *mockAgentsService) ActivateAgent(ctx context.Context, id string, isActive bool) error {
	if m.activateAgentFn != nil {
		return m.activateAgentFn(ctx, id, isActive)
	}
	return nil
}

func TestServerAgentsHandlers(t *testing.T) {
	r := require.New(t)

	mockSvc := &mockAgentsService{
		listAgentsFn: func(ctx context.Context) ([]model.Agent, error) {
			return []model.Agent{{Id: "a1", Name: "Agent1"}}, nil
		},
		listAgentsByParticipantFn: func(ctx context.Context, participantId string) ([]model.Agent, error) {
			return []model.Agent{{Id: "a1", ParticipantId: participantId}}, nil
		},
		getAgentFn: func(ctx context.Context, id string) (*model.Agent, error) {
			if id == "not-found" {
				return nil, sql.ErrNoRows
			}
			return &model.Agent{Id: id, Name: "Agent1"}, nil
		},
	}

	server := &Server{Service: mockSvc}

	// 1. listAgents (all)
	req := httptest.NewRequest(http.MethodGet, "/agents", nil)
	rec := httptest.NewRecorder()
	server.listAgents(rec, req)
	r.Equal(http.StatusOK, rec.Code)

	// 2. listAgents (by participant)
	req = httptest.NewRequest(http.MethodGet, "/agents?participant_id=p1", nil)
	rec = httptest.NewRecorder()
	server.listAgents(rec, req)
	r.Equal(http.StatusOK, rec.Code)

	// 3. getAgent (found)
	req = httptest.NewRequest(http.MethodGet, "/agents/a1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "a1"})
	rec = httptest.NewRecorder()
	server.getAgent(rec, req)
	r.Equal(http.StatusOK, rec.Code)

	// 4. getAgent (not found)
	req = httptest.NewRequest(http.MethodGet, "/agents/not-found", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "not-found"})
	rec = httptest.NewRecorder()
	server.getAgent(rec, req)
	r.Equal(http.StatusNotFound, rec.Code)

	// 5. createAgent (valid)
	body, _ := json.Marshal(model.Agent{Name: "NewAgent", ParticipantId: "p1", GameId: "starfighter"})
	req = httptest.NewRequest(http.MethodPost, "/agents", bytes.NewReader(body))
	rec = httptest.NewRecorder()
	server.createAgent(rec, req)
	r.Equal(http.StatusCreated, rec.Code)

	// 6. createAgent (invalid json)
	req = httptest.NewRequest(http.MethodPost, "/agents", bytes.NewReader([]byte("invalid")))
	rec = httptest.NewRecorder()
	server.createAgent(rec, req)
	r.Equal(http.StatusBadRequest, rec.Code)

	// 7. createAgent (missing fields)
	body, _ = json.Marshal(model.Agent{Name: ""})
	req = httptest.NewRequest(http.MethodPost, "/agents", bytes.NewReader(body))
	rec = httptest.NewRecorder()
	server.createAgent(rec, req)
	r.Equal(http.StatusBadRequest, rec.Code)

	// 8. updateAgent
	body, _ = json.Marshal(model.Agent{Name: "UpdatedAgent"})
	req = httptest.NewRequest(http.MethodPut, "/agents/a1", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"id": "a1"})
	rec = httptest.NewRecorder()
	server.updateAgent(rec, req)
	r.Equal(http.StatusOK, rec.Code)

	// 9. activateAgent
	req = httptest.NewRequest(http.MethodPatch, "/agents/a1?status=true", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "a1"})
	rec = httptest.NewRecorder()
	server.activateAgent(rec, req)
	r.Equal(http.StatusOK, rec.Code)
}
