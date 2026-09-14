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
	r.Equal("5432", cfg.Database.Port)
	r.Equal("postgres", cfg.Database.Username)
	r.Equal("pgx", cfg.Database.Driver)
	r.Equal("agentrix", cfg.Database.Name)
}

func TestGetStringDBConnection(t *testing.T) {
	r := require.New(t)

	cfg := NewConfiguration()
	cfg.Database.Username = "testuser"
	cfg.Database.Password = "testpass"
	cfg.Database.Host = "localhost"
	cfg.Database.Port = "5432"
	cfg.Database.Name = "agentrix_test"
	cfg.Database.SSLMode = "disable"

	connStr := cfg.GetStringDBConnection()
	r.Equal("postgres://testuser:testpass@localhost:5432/agentrix_test?sslmode=disable", connStr)

	// Test GCP Mode
	cfg.Mode = ModeGCP
	gcpConnStr := cfg.GetStringDBConnection()
	r.Equal("postgres://testuser:testpass@/cloudsql/localhost/agentrix_test?sslmode=disable", gcpConnStr)
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

func TestLoadDotEnv(t *testing.T) {
	r := require.New(t)

	// Create temporary env file
	tmpFile, err := os.CreateTemp("", ".env.test.*")
	r.NoError(err)
	defer os.Remove(tmpFile.Name())

	content := `
# This is a comment
TEST_KEY_ONE=hello_world
TEST_KEY_QUOTED="quoted_val"
TEST_KEY_SINGLE='single_val'
TEST_EXISTING=new_val
`
	_, err = tmpFile.WriteString(content)
	r.NoError(err)
	tmpFile.Close()

	// Pre-set TEST_EXISTING to ensure it is not overwritten
	os.Setenv("TEST_EXISTING", "original_val")
	defer func() {
		os.Unsetenv("TEST_KEY_ONE")
		os.Unsetenv("TEST_KEY_QUOTED")
		os.Unsetenv("TEST_KEY_SINGLE")
		os.Unsetenv("TEST_EXISTING")
	}()

	loadDotEnv(tmpFile.Name())

	r.Equal("hello_world", os.Getenv("TEST_KEY_ONE"))
	r.Equal("quoted_val", os.Getenv("TEST_KEY_QUOTED"))
	r.Equal("single_val", os.Getenv("TEST_KEY_SINGLE"))
	r.Equal("original_val", os.Getenv("TEST_EXISTING"))
}
