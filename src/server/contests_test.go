package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/common"
	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/Agentrix-Group/Agentrix/src/service"
	"github.com/stretchr/testify/require"
)

type mockContestService struct {
	service.Service
	listPublicContestsFn func(ctx context.Context, filter model.PublicContestsFilter) ([]model.PublicContestSummary, error)
	getPublicContestFn   func(ctx context.Context, id string) (*model.Contest, error)
	enrollAgentFn        func(ctx context.Context, participantId, contestId, agentId string) (*model.Ranking, error)
	listContestAgentsFn  func(ctx context.Context, contestId string) ([]model.Ranking, error)
	hasPermissionFn      func(ctx context.Context, participantId, permission string) (bool, error)
}

func (m *mockContestService) ListPublicContests(ctx context.Context, filter model.PublicContestsFilter) ([]model.PublicContestSummary, error) {
	if m.listPublicContestsFn != nil {
		return m.listPublicContestsFn(ctx, filter)
	}
	return nil, nil
}

func (m *mockContestService) GetPublicContest(ctx context.Context, id string) (*model.Contest, error) {
	if m.getPublicContestFn != nil {
		return m.getPublicContestFn(ctx, id)
	}
	return nil, service.ErrContestNotFound
}

func (m *mockContestService) EnrollAgent(ctx context.Context, participantId, contestId, agentId string) (*model.Ranking, error) {
	if m.enrollAgentFn != nil {
		return m.enrollAgentFn(ctx, participantId, contestId, agentId)
	}
	return nil, nil
}

func (m *mockContestService) ListContestAgents(ctx context.Context, contestId string) ([]model.Ranking, error) {
	if m.listContestAgentsFn != nil {
		return m.listContestAgentsFn(ctx, contestId)
	}
	return nil, nil
}

func (m *mockContestService) HasPermission(ctx context.Context, participantId, permission string) (bool, error) {
	if m.hasPermissionFn != nil {
		return m.hasPermissionFn(ctx, participantId, permission)
	}
	return true, nil
}

func TestListPublicContests_Endpoint(t *testing.T) {
	r := require.New(t)

	t.Run("Anonymous request returns 200 OK with PublicContestSummary list and X-Request-Id", func(t *testing.T) {
		now := time.Now().UTC().Truncate(time.Second)
		mockSvc := &mockContestService{
			listPublicContestsFn: func(ctx context.Context, filter model.PublicContestsFilter) ([]model.PublicContestSummary, error) {
				r.Empty(filter.State)
				r.False(filter.IncludeArchived)
				return []model.PublicContestSummary{
					{
						Id:          "test-contest-uuid-1",
						Name:        "Primavera 2026",
						Description: "Torneo demo de agentes",
						State:       model.ContestStateRegistrationOpen,
						StartsAt:    &now,
						EndsAt:      nil,
					},
				}, nil
			},
		}

		srv := NewServer(mockSvc)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/contests", nil)
		rec := httptest.NewRecorder()

		srv.Handler.ServeHTTP(rec, req)

		r.Equal(http.StatusOK, rec.Code)
		r.NotEmpty(rec.Header().Get("X-Request-Id"))
		r.Contains(rec.Header().Get("Content-Type"), "application/json")

		var res []model.PublicContestSummary
		err := json.Unmarshal(rec.Body.Bytes(), &res)
		r.NoError(err)
		r.Len(res, 1)
		r.Equal("test-contest-uuid-1", res[0].Id)
		r.Equal("Primavera 2026", res[0].Name)
		r.Equal(model.ContestStateRegistrationOpen, res[0].State)
	})

	t.Run("Empty results serialize as empty JSON array [] not null", func(t *testing.T) {
		mockSvc := &mockContestService{
			listPublicContestsFn: func(ctx context.Context, filter model.PublicContestsFilter) ([]model.PublicContestSummary, error) {
				return []model.PublicContestSummary{}, nil
			},
		}

		srv := NewServer(mockSvc)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/contests", nil)
		rec := httptest.NewRecorder()

		srv.Handler.ServeHTTP(rec, req)

		r.Equal(http.StatusOK, rec.Code)
		bodyStr := rec.Body.String()
		r.JSONEq("[]", bodyStr)
	})

	t.Run("Valid state query param passes to service", func(t *testing.T) {
		called := false
		mockSvc := &mockContestService{
			listPublicContestsFn: func(ctx context.Context, filter model.PublicContestsFilter) ([]model.PublicContestSummary, error) {
				called = true
				r.Equal(model.ContestStatePublished, filter.State)
				r.True(filter.IncludeArchived)
				return []model.PublicContestSummary{}, nil
			},
		}

		srv := NewServer(mockSvc)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/contests?state=published&include_archived=true", nil)
		rec := httptest.NewRecorder()

		srv.Handler.ServeHTTP(rec, req)

		r.Equal(http.StatusOK, rec.Code)
		r.True(called)
	})

	t.Run("Draft state query param returns 400 Bad Request", func(t *testing.T) {
		mockSvc := &mockContestService{
			listPublicContestsFn: func(ctx context.Context, filter model.PublicContestsFilter) ([]model.PublicContestSummary, error) {
				return nil, service.ErrPrivateStateFilter
			},
		}

		srv := NewServer(mockSvc)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/contests?state=draft", nil)
		rec := httptest.NewRecorder()

		srv.Handler.ServeHTTP(rec, req)

		r.Equal(http.StatusBadRequest, rec.Code)
		var errResp common.Response
		err := json.Unmarshal(rec.Body.Bytes(), &errResp)
		r.NoError(err)
		r.Equal(http.StatusBadRequest, errResp.HttpStatusCode)
		r.Equal(common.INVALID_REQUEST_ERROR, errResp.ErrorCode)
		r.Contains(errResp.Message, "private")
	})

	t.Run("Invalid include_archived returns 400 Bad Request", func(t *testing.T) {
		mockSvc := &mockContestService{}
		srv := NewServer(mockSvc)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/contests?include_archived=invalid_bool", nil)
		rec := httptest.NewRecorder()

		srv.Handler.ServeHTTP(rec, req)

		r.Equal(http.StatusBadRequest, rec.Code)
		var errResp common.Response
		err := json.Unmarshal(rec.Body.Bytes(), &errResp)
		r.NoError(err)
		r.Equal(common.INVALID_REQUEST_ERROR, errResp.ErrorCode)
	})

	t.Run("Internal database error returns 500 Internal Server Error", func(t *testing.T) {
		mockSvc := &mockContestService{
			listPublicContestsFn: func(ctx context.Context, filter model.PublicContestsFilter) ([]model.PublicContestSummary, error) {
				return nil, errors.New("connection reset by peer")
			},
		}

		srv := NewServer(mockSvc)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/contests", nil)
		rec := httptest.NewRecorder()

		srv.Handler.ServeHTTP(rec, req)

		r.Equal(http.StatusInternalServerError, rec.Code)
		var errResp common.Response
		err := json.Unmarshal(rec.Body.Bytes(), &errResp)
		r.NoError(err)
		r.Equal(common.DATABASE_ERROR, errResp.ErrorCode)
	})
}

func TestGetPublicContest_Endpoint(t *testing.T) {
	r := require.New(t)

	t.Run("Public contest returns 200 OK and contest details", func(t *testing.T) {
		mockSvc := &mockContestService{
			getPublicContestFn: func(ctx context.Context, id string) (*model.Contest, error) {
				return &model.Contest{
					Id:     id,
					Name:   "Spring Arena 2026",
					State:  model.ContestStateRegistrationOpen,
					Active: true,
				}, nil
			},
		}

		srv := NewServer(mockSvc)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/contests/c-123", nil)
		rec := httptest.NewRecorder()

		srv.Handler.ServeHTTP(rec, req)

		r.Equal(http.StatusOK, rec.Code)
		var contest model.Contest
		err := json.Unmarshal(rec.Body.Bytes(), &contest)
		r.NoError(err)
		r.Equal("c-123", contest.Id)
		r.Equal("Spring Arena 2026", contest.Name)
	})

	t.Run("Draft contest returns 404 Not Found to public", func(t *testing.T) {
		mockSvc := &mockContestService{
			getPublicContestFn: func(ctx context.Context, id string) (*model.Contest, error) {
				return nil, service.ErrContestNotFound
			},
		}

		srv := NewServer(mockSvc)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/contests/draft-cup", nil)
		rec := httptest.NewRecorder()

		srv.Handler.ServeHTTP(rec, req)

		r.Equal(http.StatusNotFound, rec.Code)
	})
}

func TestEnrollAgent_Endpoint(t *testing.T) {
	r := require.New(t)

	t.Run("Enrolling agent with valid Bearer token returns 201 Created", func(t *testing.T) {
		mockSvc := &mockContestService{
			enrollAgentFn: func(ctx context.Context, participantId, contestId, agentId string) (*model.Ranking, error) {
				return &model.Ranking{
					Id:            "rank-1",
					ContestId:     contestId,
					AgentId:       agentId,
					ParticipantId: participantId,
					Score:         0,
					Rank:          1,
				}, nil
			},
			hasPermissionFn: func(ctx context.Context, participantId, permission string) (bool, error) {
				return true, nil
			},
		}

		srv := NewServer(mockSvc)
		token, err := srv.Auth.GenerateAuthToken("part-1", "participant")
		r.NoError(err)

		body, _ := json.Marshal(model.EnrollAgentRequest{
			AgentId: "agent-99",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/contests/c-123/agents", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token.AccessToken)
		rec := httptest.NewRecorder()

		srv.Handler.ServeHTTP(rec, req)

		r.Equal(http.StatusCreated, rec.Code)
		var resp model.EnrollAgentResponse
		err = json.Unmarshal(rec.Body.Bytes(), &resp)
		r.NoError(err)
		r.Equal(http.StatusCreated, resp.HttpStatusCode)
		r.NotNil(resp.Ranking)
		r.Equal("c-123", resp.Ranking.ContestId)
		r.Equal("agent-99", resp.Ranking.AgentId)
	})

	t.Run("Enrolling without token returns 401 Unauthorized", func(t *testing.T) {
		srv := NewServer(&mockContestService{})

		body, _ := json.Marshal(model.EnrollAgentRequest{
			AgentId: "agent-99",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/contests/c-123/agents", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		srv.Handler.ServeHTTP(rec, req)

		r.Equal(http.StatusUnauthorized, rec.Code)
	})

	t.Run("Enrolling when registration closed returns 400 Bad Request", func(t *testing.T) {
		mockSvc := &mockContestService{
			enrollAgentFn: func(ctx context.Context, participantId, contestId, agentId string) (*model.Ranking, error) {
				return nil, service.ErrRegistrationClosed
			},
			hasPermissionFn: func(ctx context.Context, participantId, permission string) (bool, error) {
				return true, nil
			},
		}

		srv := NewServer(mockSvc)
		token, _ := srv.Auth.GenerateAuthToken("part-1", "participant")

		body, _ := json.Marshal(model.EnrollAgentRequest{
			AgentId: "agent-99",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/contests/c-123/agents", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token.AccessToken)
		rec := httptest.NewRecorder()

		srv.Handler.ServeHTTP(rec, req)

		r.Equal(http.StatusBadRequest, rec.Code)
	})

	t.Run("Enrolling duplicate agent returns 409 Conflict", func(t *testing.T) {
		mockSvc := &mockContestService{
			enrollAgentFn: func(ctx context.Context, participantId, contestId, agentId string) (*model.Ranking, error) {
				return nil, service.ErrAgentAlreadyEnrolled
			},
			hasPermissionFn: func(ctx context.Context, participantId, permission string) (bool, error) {
				return true, nil
			},
		}

		srv := NewServer(mockSvc)
		token, _ := srv.Auth.GenerateAuthToken("part-1", "participant")

		body, _ := json.Marshal(model.EnrollAgentRequest{
			AgentId: "agent-99",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/contests/c-123/agents", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token.AccessToken)
		rec := httptest.NewRecorder()

		srv.Handler.ServeHTTP(rec, req)

		r.Equal(http.StatusConflict, rec.Code)
	})
}

func TestListContestAgents_Endpoint(t *testing.T) {
	r := require.New(t)

	t.Run("Returns 200 OK and list of enrolled rankings", func(t *testing.T) {
		mockSvc := &mockContestService{
			listContestAgentsFn: func(ctx context.Context, contestId string) ([]model.Ranking, error) {
				return []model.Ranking{
					{
						Id:            "r-1",
						ContestId:     contestId,
						AgentId:       "agent-1",
						ParticipantId: "part-1",
						Score:         10,
						Rank:          1,
					},
				}, nil
			},
		}

		srv := NewServer(mockSvc)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/contests/c-100/agents", nil)
		rec := httptest.NewRecorder()

		srv.Handler.ServeHTTP(rec, req)

		r.Equal(http.StatusOK, rec.Code)
		var rankings []model.Ranking
		err := json.Unmarshal(rec.Body.Bytes(), &rankings)
		r.NoError(err)
		r.Len(rankings, 1)
		r.Equal("r-1", rankings[0].Id)
		r.Equal("agent-1", rankings[0].AgentId)
	})
}
