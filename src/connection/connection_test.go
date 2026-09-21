package connection

import (
	"context"
	"testing"

	"github.com/Agentrix-Group/Agentrix/src/config"
	"github.com/stretchr/testify/require"
)

func TestOpenDatabaseFailsFastWhenUnreachable(t *testing.T) {
	cfg := &config.Config{Database: config.Database{Host: "127.0.0.1", Port: "1", Name: "x", Username: "x", SSLMode: "disable", MaxConns: 1}}
	_, err := OpenDatabase(context.Background(), cfg)
	require.Error(t, err)
}
