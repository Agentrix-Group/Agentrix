package config

import (
	"fmt"
	"os"
)

const (
	ModeDev     = "dev"
	ModeGCP     = "gcp"
	ModeRailway = "railway"
)

type Server struct {
	Host string `json:"host"`
	Port string `json:"port"`
}

type Database struct {
	Driver   string `json:"driver"`
	Name     string `json:"name"`
	Username string `json:"username"`
	Password string `json:"password"`
	Protocol string `json:"protocol"`
	Host     string `json:"host"`
	Port     string `json:"port"`
	SSLMode  string `json:"sslmode"`
}

type Artifacts struct {
	Dir string `json:"dir"`
}

type Auth struct {
	AccessSecret  string `json:"access_secret"`
	RefreshSecret string `json:"refresh_secret"`
	SessionSecret string `json:"session_secret"`
}

type Config struct {
	Mode      string    `json:"mode"`
	Server    Server    `json:"server"`
	Database  Database  `json:"database"`
	Artifacts Artifacts `json:"artifacts"`
	Auth      Auth      `json:"auth"`
}

func NewConfiguration() *Config {
	mode := os.Getenv("MODE")
	if mode == "" {
		mode = ModeDev
	}

	serverPort := os.Getenv("PORT")
	if serverPort == "" {
		serverPort = "8080"
	}

	dbDriver := os.Getenv("DB_DRIVER")
	if dbDriver == "" {
		dbDriver = "pgx"
	}

	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		dbUser = "postgres"
	}

	dbPass := os.Getenv("DB_PASSWORD")

	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}

	dbPort := os.Getenv("DB_PORT")
	if dbPort == "" {
		dbPort = "5432"
	}

	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		dbName = "agentrix"
	}

	dbSSLMode := os.Getenv("DB_SSLMODE")
	if dbSSLMode == "" {
		dbSSLMode = "disable"
	}

	artifactsDir := os.Getenv("ARTIFACTS_DIR")
	if artifactsDir == "" {
		artifactsDir = "./artifacts"
	}

	accessSecret := os.Getenv("ACCESS_SECRET")
	if accessSecret == "" {
		accessSecret = "agentrix-access-secret-key-change-in-prod"
	}

	refreshSecret := os.Getenv("REFRESH_SECRET")
	if refreshSecret == "" {
		refreshSecret = "agentrix-refresh-secret-key-change-in-prod"
	}

	sessionSecret := os.Getenv("SESSION_SECRET")
	if sessionSecret == "" {
		sessionSecret = "agentrix-session-secret-key-change-in-prod"
	}

	return &Config{
		Mode: mode,
		Server: Server{
			Host: "0.0.0.0",
			Port: serverPort,
		},
		Database: Database{
			Driver:   dbDriver,
			Name:     dbName,
			Username: dbUser,
			Password: dbPass,
			Protocol: "tcp",
			Host:     dbHost,
			Port:     dbPort,
			SSLMode:  dbSSLMode,
		},
		Artifacts: Artifacts{
			Dir: artifactsDir,
		},
		Auth: Auth{
			AccessSecret:  accessSecret,
			RefreshSecret: refreshSecret,
			SessionSecret: sessionSecret,
		},
	}
}

func (c *Config) GetStringDBConnection() string {
	if c.Mode == ModeGCP {
		if c.Database.Password != "" {
			return fmt.Sprintf("postgres://%s:%s@/cloudsql/%s/%s?sslmode=%s",
				c.Database.Username,
				c.Database.Password,
				c.Database.Host,
				c.Database.Name,
				c.Database.SSLMode,
			)
		}
		return fmt.Sprintf("postgres://%s@/cloudsql/%s/%s?sslmode=%s",
			c.Database.Username,
			c.Database.Host,
			c.Database.Name,
			c.Database.SSLMode,
		)
	}

	if c.Database.Password != "" {
		return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
			c.Database.Username,
			c.Database.Password,
			c.Database.Host,
			c.Database.Port,
			c.Database.Name,
			c.Database.SSLMode,
		)
	}

	return fmt.Sprintf("postgres://%s@%s:%s/%s?sslmode=%s",
		c.Database.Username,
		c.Database.Host,
		c.Database.Port,
		c.Database.Name,
		c.Database.SSLMode,
	)
}
