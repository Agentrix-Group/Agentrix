package repository

import (
	"context"
	"database/sql"

	"github.com/Agentrix-Group/Agentrix/src/model"
)

func (r *repository) GetGame(ctx context.Context, id string) (*model.Game, error) {
	db, err := r.getDb()
	if err != nil {
		return nil, err
	}

	query := `SELECT id, name, COALESCE(description, ''), COALESCE(manifest_path, ''), COALESCE(min_players, 2), COALESCE(max_players, 4), active, created_at FROM games WHERE id = $1 AND active = TRUE`

	var g model.Game
	err = db.QueryRowContext(ctx, query, id).Scan(
		&g.Id, &g.Name, &g.Description, &g.ManifestPath, &g.MinPlayers, &g.MaxPlayers, &g.Active, &g.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		return nil, err
	}

	return &g, nil
}
