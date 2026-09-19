package engine

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type mockEngineClient struct {
	startFn           func(ctx context.Context, cfg StartConfig) error
	initializeMatchFn func(ctx context.Context, req InitializeMatchRequest) (*MatchInitializedResult, error)
	advanceTickFn     func(ctx context.Context, req AdvanceTickRequest) (*TickResult, error)
	finishMatchFn     func(ctx context.Context, reason string) (*MatchResult, error)
	closeFn           func(ctx context.Context) error
}

func (m *mockEngineClient) Start(ctx context.Context, cfg StartConfig) error {
	if m.startFn != nil {
		return m.startFn(ctx, cfg)
	}
	return nil
}

func (m *mockEngineClient) InitializeMatch(ctx context.Context, req InitializeMatchRequest) (*MatchInitializedResult, error) {
	if m.initializeMatchFn != nil {
		return m.initializeMatchFn(ctx, req)
	}
	return &MatchInitializedResult{MatchID: req.MatchID, InitialTick: 0}, nil
}

func (m *mockEngineClient) AdvanceTick(ctx context.Context, req AdvanceTickRequest) (*TickResult, error) {
	if m.advanceTickFn != nil {
		return m.advanceTickFn(ctx, req)
	}
	return &TickResult{Tick: req.Tick + 1}, nil
}

func (m *mockEngineClient) FinishMatch(ctx context.Context, reason string) (*MatchResult, error) {
	if m.finishMatchFn != nil {
		return m.finishMatchFn(ctx, reason)
	}
	return &MatchResult{Reason: reason}, nil
}

func (m *mockEngineClient) Close(ctx context.Context) error {
	if m.closeFn != nil {
		return m.closeFn(ctx)
	}
	return nil
}

func TestEngineClientInterface(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	var client EngineClient = &mockEngineClient{}
	r.NotNil(client)

	cfg := StartConfig{
		BinaryPath:       "/bin/fake-engine",
		HandshakeTimeout: 2 * time.Second,
	}
	r.NoError(client.Start(ctx, cfg))

	initRes, err := client.InitializeMatch(ctx, InitializeMatchRequest{MatchID: "m-1"})
	r.NoError(err)
	r.Equal("m-1", initRes.MatchID)

	tickRes, err := client.AdvanceTick(ctx, AdvanceTickRequest{Tick: 0})
	r.NoError(err)
	r.Equal(1, tickRes.Tick)

	matchRes, err := client.FinishMatch(ctx, "completed")
	r.NoError(err)
	r.Equal("completed", matchRes.Reason)

	r.NoError(client.Close(ctx))
}
