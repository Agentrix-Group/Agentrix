package connection

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/F4nk1/Agentrix/src/config"
)

type Connection struct {
	Db *sql.DB
}

func NewConnection(ctx context.Context, config *config.Config) (*Connection, error) {
	connectionString := config.GetStringDBConnection()

	db, err := sql.Open(config.Database.Driver, connectionString)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("cannot connect to database: %v", err)
	}

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
