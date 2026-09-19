package connection

import (
	"context"
	"testing"

	"github.com/Agentrix-Group/Agentrix/src/config"
	"github.com/stretchr/testify/require"
)

func TestConnectionCloseNil(t *testing.T) {
	r := require.New(t)

	conn := &Connection{Db: nil}
	r.NoError(conn.Close())
}

func TestNewConnectionUnreachable(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	cfg := &config.Config{
		Database: config.Database{
			Driver:   "pgx",
			Host:     "127.0.0.1",
			Port:     "54321", // unreachable port
			Username: "fake_user",
			Password: "fake_password",
			Name:     "fake_db",
			SSLMode:  "disable",
		},
	}

	conn, err := NewConnection(ctx, cfg)
	r.Error(err)
	r.Nil(conn)
}
