package repository

import (
	"context"
	"database/sql"

	"github.com/F4nk1/Agentrix/src/model"
	"github.com/F4nk1/Agentrix/src/tracer"
)

func (r *repository) ListGames(ctx context.Context) ([]model.Game, error) {
	db, err := r.getDb()
	if err != nil {
		return nil, err
	}

	tracer.Debugf(ctx, "Executing database query to fetch active games")
	query := `SELECT id, name, description, manifest_path, min_players, max_players, active, created_at FROM games WHERE active = TRUE ORDER BY created_at DESC`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		tracer.Errorf(ctx, "Database query failed for listing games: %s", err)
		return nil, err
	}
	defer rows.Close()

	var games []model.Game
	for rows.Next() {
		var g model.Game
		if err := rows.Scan(&g.Id, &g.Name, &g.Description, &g.ManifestPath, &g.MinPlayers, &g.MaxPlayers, &g.Active, &g.CreatedAt); err != nil {
			tracer.Errorf(ctx, "Database row scan failed for games: %s", err)
			return nil, err
		}
		games = append(games, g)
	}

	tracer.Debugf(ctx, "Retrieved %d games from database", len(games))
	return games, nil
}

func (r *repository) GetGame(ctx context.Context, id string) (*model.Game, error) {
	db, err := r.getDb()
	if err != nil {
		return nil, err
	}

	tracer.Debugf(ctx, "Querying database for game %s", id)
	query := `SELECT id, name, description, manifest_path, min_players, max_players, active, created_at FROM games WHERE id = $1 AND active = TRUE`

	var g model.Game
	err = db.QueryRowContext(ctx, query, id).Scan(
		&g.Id, &g.Name, &g.Description, &g.ManifestPath, &g.MinPlayers, &g.MaxPlayers, &g.Active, &g.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			tracer.Warnf(ctx, "Game %s not found", id)
			return nil, err
		}
		tracer.Errorf(ctx, "Database query failed for game %s: %s", id, err)
		return nil, err
	}

	tracer.Debugf(ctx, "Successfully retrieved game %s", id)
	return &g, nil
}

func (r *repository) CreateGame(ctx context.Context, game *model.Game) error {
	db, err := r.getDb()
	if err != nil {
		return err
	}

	tracer.Debugf(ctx, "Creating game '%s'", game.Id)
	query := `INSERT INTO games (id, name, description, manifest_path, min_players, max_players, active, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err = db.ExecContext(ctx, query,
		game.Id, game.Name, game.Description, game.ManifestPath,
		game.MinPlayers, game.MaxPlayers, game.Active, game.CreatedAt,
	)
	if err != nil {
		tracer.Errorf(ctx, "Database insert failed for game %s: %s", game.Id, err)
		return err
	}

	tracer.Debugf(ctx, "Successfully created game %s", game.Id)
	return nil
}

func (r *repository) UpdateGame(ctx context.Context, game *model.Game) error {
	db, err := r.getDb()
	if err != nil {
		return err
	}

	tracer.Debugf(ctx, "Updating game '%s'", game.Id)
	query := `UPDATE games SET name = $1, description = $2, manifest_path = $3, min_players = $4, max_players = $5, active = $6 WHERE id = $7`

	_, err = db.ExecContext(ctx, query,
		game.Name, game.Description, game.ManifestPath, game.MinPlayers, game.MaxPlayers, game.Active, game.Id,
	)
	if err != nil {
		tracer.Errorf(ctx, "Database update failed for game %s: %s", game.Id, err)
		return err
	}

	tracer.Debugf(ctx, "Successfully updated game %s", game.Id)
	return nil
}

func (r *repository) ActivateGame(ctx context.Context, id string, isActive bool) error {
	db, err := r.getDb()
	if err != nil {
		return err
	}

	tracer.Debugf(ctx, "Setting game '%s' active status to %t", id, isActive)
	query := `UPDATE games SET active = $1 WHERE id = $2`

	_, err = db.ExecContext(ctx, query, isActive, id)
	return err
}
