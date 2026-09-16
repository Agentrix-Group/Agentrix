package main

import (
	"testing"

	"github.com/F4nk1/Agentrix/src/config"
	"github.com/stretchr/testify/require"
)

func TestMainConfig(t *testing.T) {
	r := require.New(t)

	cfg := config.NewConfiguration()
	r.NotNil(cfg)
	r.NotEmpty(cfg.Server.Port)
}
