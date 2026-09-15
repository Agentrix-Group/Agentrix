package config

import (
	"fmt"
	"os"
	"strings"
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

type Logging struct {
	Level  string `json:"level"`
	Format string `json:"format"`
	Color  string `json:"color"`
}

type Config struct {
	Mode      string    `json:"mode"`
	Server    Server    `json:"server"`
	Database  Database  `json:"database"`
	Artifacts Artifacts `json:"artifacts"`
	Auth      Auth      `json:"auth"`
	Logging   Logging   `json:"logging"`
}

// loadDotEnv reads key=value pairs from specified files and populates them into
// the environment only if they are not already set. It ignores comments and empty lines.
func loadDotEnv(filenames ...string) {
	for _, filename := range filenames {
		data, err := os.ReadFile(filename)
		if err != nil {
			continue
		}
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			if len(val) >= 2 && ((val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'')) {
				val = val[1 : len(val)-1]
			}
			if os.Getenv(key) == "" {
				_ = os.Setenv(key, val)
			}
		}
	}
}

func NewConfiguration() *Config {
	loadDotEnv(".env")

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

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	logFormat := os.Getenv("LOG_FORMAT")
	if logFormat == "" {
		if mode == ModeGCP || mode == ModeRailway {
			logFormat = "json"
		} else {
			logFormat = "console"
		}
	}

	logColor := os.Getenv("LOG_COLOR")
	if logColor == "" {
		logColor = "auto"
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
		Logging: Logging{
			Level:  logLevel,
			Format: logFormat,
			Color:  logColor,
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
