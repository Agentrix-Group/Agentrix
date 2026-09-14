package connection

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/F4nk1/Agentrix/src/config"
	"github.com/F4nk1/Agentrix/src/tracer"
)

type Connection struct {
	Db *sql.DB
}

func NewConnection(ctx context.Context, config *config.Config) (*Connection, error) {
	tracer.Debug(ctx, "Initializing database connection")

	connectionString := config.GetStringDBConnection()

	db, err := sql.Open(config.Database.Driver, connectionString)
	if err != nil {
		tracer.Errorf(ctx, "Failed to open database connection: %s", err)
		return nil, err
	}

	if err := db.Ping(); err != nil {
		tracer.Errorf(ctx, "Failed to ping database: %s", err)
		return nil, fmt.Errorf("cannot connect to database: %v", err)
	}

	tracer.Debug(ctx, "Connected to database successfully")

	return &Connection{
		Db: db,
	}, nil
}

func (c *Connection) Close() error {
	if c.Db != nil {
		return c.Db.Close()
	}
	return nil
}
