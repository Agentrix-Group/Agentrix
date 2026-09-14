package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewConfiguration(t *testing.T) {
	r := require.New(t)

	cfg := NewConfiguration()
	r.NotNil(cfg)
	r.Equal("dev", cfg.Mode)
	r.Equal("8080", cfg.Server.Port)
	r.Equal("3306", cfg.Database.Port)
	r.Equal("agentrix", cfg.Database.Name)
}

func TestGetStringDBConnection(t *testing.T) {
	r := require.New(t)

	cfg := NewConfiguration()
	cfg.Database.Username = "testuser"
	cfg.Database.Password = "testpass"
	cfg.Database.Host = "localhost"
	cfg.Database.Port = "3306"
	cfg.Database.Name = "agentrix_test"

	connStr := cfg.GetStringDBConnection()
	r.Contains(connStr, "testuser:testpass@tcp(localhost:3306)/agentrix_test")

	// Test GCP Mode
	cfg.Mode = ModeGCP
	gcpConnStr := cfg.GetStringDBConnection()
	r.Contains(gcpConnStr, "testuser:testpass@unix(/cloudsql/localhost)/agentrix_test")
}

func TestEnvOverride(t *testing.T) {
	r := require.New(t)

	os.Setenv("PORT", "9090")
	os.Setenv("DB_NAME", "custom_agentrix")
	defer func() {
		os.Unsetenv("PORT")
		os.Unsetenv("DB_NAME")
	}()

	cfg := NewConfiguration()
	r.Equal("9090", cfg.Server.Port)
	r.Equal("custom_agentrix", cfg.Database.Name)
}
