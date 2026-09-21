// Command migrate applies the canonical schema. It is the only component
// allowed to change the database structure; API and worker only verify it.
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
	if err := run(cmd); err != nil {
		fmt.Fprintf(os.Stderr, "migrate %s: %v\n", cmd, err)
		os.Exit(1)
	}
}

func run(cmd string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	db, err := connection.OpenDatabase(ctx, cfg)
	if err != nil {
		return err
	}
	defer db.Close()
	switch cmd {
	case "up":
		if err := database.Migrate(db); err != nil {
			return err
		}
		if err := database.CheckSchemaCompatible(ctx, db); err != nil {
			return fmt.Errorf("post-migration verification: %w", err)
		}
		fmt.Printf("schema at version %d and verified\n", database.TargetSchemaVersion)
	case "check":
		if err := database.CheckSchemaCompatible(ctx, db); err != nil {
			return err
		}
		fmt.Println("schema verified")
	case "status":
		return database.Status(db)
	default:
		return fmt.Errorf("unknown command (use up, check or status)")
	}
	return nil
}
