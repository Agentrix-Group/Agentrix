package integration

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/Agentrix-Group/Agentrix/src/auth"
	"github.com/Agentrix-Group/Agentrix/src/config"
	"github.com/Agentrix-Group/Agentrix/src/connection"
	"github.com/Agentrix-Group/Agentrix/src/executor"
	"github.com/Agentrix-Group/Agentrix/src/repository"
	"github.com/Agentrix-Group/Agentrix/src/server"
	"github.com/Agentrix-Group/Agentrix/src/service"
	"github.com/stretchr/testify/require"
)

func getTestPostgresConfig(dbName string) *config.Config {
	host := os.Getenv("DB_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("DB_PORT")
	if port == "" {
		port = "5432"
	}
	user := os.Getenv("DB_USER")
	if user == "" {
		user = "postgres"
	}
	pass := os.Getenv("DB_PASSWORD")

	cfg := config.NewConfiguration()
	cfg.Mode = config.ModeDev
	cfg.Database.Driver = "pgx"
	cfg.Database.Host = host
	cfg.Database.Port = port
	cfg.Database.Username = user
	cfg.Database.Password = pass
	cfg.Database.Name = dbName
	cfg.Database.SSLMode = "disable"
	return cfg
}

func connectAdminDB(t *testing.T) *sql.DB {
	cfg := getTestPostgresConfig("postgres")
	connStr := cfg.GetStringDBConnection()
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		t.Skipf("Skipping postgres integration test: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Skipf("Postgres server is not running on %s:%s, skipping: %v", cfg.Database.Host, cfg.Database.Port, err)
	}
	return db
}

func createIsolatedDB(t *testing.T, dbName string) (*connection.Connection, func()) {
	adminDB := connectAdminDB(t)
	defer adminDB.Close()

	_, err := adminDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE);", dbName))
	require.NoError(t, err)

	_, err = adminDB.Exec(fmt.Sprintf("CREATE DATABASE %s;", dbName))
	require.NoError(t, err)

	cfg := getTestPostgresConfig(dbName)
	conn, err := connection.NewConnection(context.Background(), cfg)
	require.NoError(t, err)

	cleanup := func() {
		_ = conn.Close()
		adminDB2 := connectAdminDB(t)
		defer adminDB2.Close()
		_, _ = adminDB2.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE);", dbName))
	}

	return conn, cleanup
}

func execSQLFile(t *testing.T, db *sql.DB, relPath string) {
	rootPath, err := filepath.Abs("../../" + relPath)
	require.NoError(t, err)
	content, err := os.ReadFile(rootPath)
	require.NoError(t, err)
	_, err = db.Exec(string(content))
	require.NoError(t, err)
}

func setupTestServer(conn *connection.Connection) (*server.Server, *auth.Auth) {
	repo := repository.NewRepository(conn)
	queue := connection.NewJobQueue(10)
	sandbox := executor.NewSandbox(0)
	tempDir, _ := os.MkdirTemp("", "agentrix-test-artifacts-*")
	cfg := &config.Config{
		Artifacts: config.Artifacts{Dir: tempDir},
	}
	artifacts, _ := connection.NewArtifactStore(context.Background(), cfg)
	svc := service.NewService(repo, artifacts, queue, sandbox)
	srv := server.NewServer(svc)
	authModule := auth.NewAuth("test-access-secret", "test-refresh-secret")
	srv.Auth = authModule
	return srv, authModule
}
