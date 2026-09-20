package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/config"
	"github.com/Agentrix-Group/Agentrix/src/connection"
	"github.com/Agentrix-Group/Agentrix/src/database"
)

func main() {
	cmd := "up"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cfg := config.NewConfiguration()
	conn, err := connection.NewConnection(ctx, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot connect to database at %s:%s/%s: %v\n",
			cfg.Database.Host, cfg.Database.Port, cfg.Database.Name, err)
		os.Exit(1)
	}
	defer conn.Close()

	switch cmd {
	case "up", "migrate":
		fmt.Printf("=== Agentrix Database Migration ===\n")
		fmt.Printf("Target: %s@%s:%s/%s\n", cfg.Database.Username, cfg.Database.Host, cfg.Database.Port, cfg.Database.Name)
		if err := database.Migrate(conn.Db); err != nil {
			fmt.Fprintf(os.Stderr, "Migration failed: %v\n", err)
			os.Exit(1)
		}
		ver, _ := database.GetCurrentVersion(conn.Db)
		fmt.Printf("Migrations applied successfully. Current schema version: %d (Target: %d)\n",
			ver, database.TargetSchemaVersion)

	case "status":
		fmt.Printf("=== Agentrix Database Status ===\n")
		fmt.Printf("Host: %s:%s\n", cfg.Database.Host, cfg.Database.Port)
		fmt.Printf("Database: %s\n", cfg.Database.Name)
		ver, err := database.GetCurrentVersion(conn.Db)
		if err != nil {
			fmt.Printf("Status: Schema not initialized (no version table found: %v)\n", err)
		} else {
			fmt.Printf("Current schema version: %d (Target: %d)\n", ver, database.TargetSchemaVersion)
		}
		fmt.Printf("\nDetailed Migration Status:\n")
		if err := database.Status(conn.Db); err != nil {
			fmt.Fprintf(os.Stderr, "Error reading migration status: %v\n", err)
			os.Exit(1)
		}

	case "check":
		if err := database.CheckSchemaCompatible(ctx, conn.Db); err != nil {
			fmt.Fprintf(os.Stderr, "Schema check failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Schema check passed: database is fully compatible and up-to-date.")

	case "version":
		ver, err := database.GetCurrentVersion(conn.Db)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not determine schema version: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Current: %d | Target: %d\n", ver, database.TargetSchemaVersion)

	default:
		fmt.Fprintf(os.Stderr, "Unknown command %q. Usage: migrate [up|status|check|version]\n", cmd)
		os.Exit(1)
	}
}
