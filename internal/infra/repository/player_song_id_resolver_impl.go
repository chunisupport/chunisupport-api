package repository

import (
	"context"
	"database/sql"
	"errors"

	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/jmoiron/sqlx"
)

func (r *PlayerSongIDResolver) ResolveSongIDByDisplayID(ctx context.Context, exec domainrepo.Executor, displayID string) (*int, error) {
	const q = `SELECT id FROM songs WHERE display_id = ? LIMIT 1`
	var id int
	if err := sqlx.GetContext(ctx, exec, &id, q, displayID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, wrapPlayerLockedSongRepositoryError("resolve song id by display id", err)
	}
	return &id, nil
}

func (r *PlayerSongIDResolver) ResolveSongIDsByDisplayIDs(ctx context.Context, exec domainrepo.Executor, displayIDs []string) (map[string]int, error) {
	if len(displayIDs) == 0 {
		return map[string]int{}, nil
	}
	query, args, err := sqlx.In("SELECT display_id, id FROM songs WHERE display_id IN (?)", displayIDs)
	if err != nil {
		return nil, wrapPlayerLockedSongRepositoryError("resolve song ids by display ids", err)
	}

	type songIDResult struct {
		DisplayID string `db:"display_id"`
		ID        int    `db:"id"`
	}
	results := make([]songIDResult, 0, len(displayIDs))
	if err := sqlx.SelectContext(ctx, exec, &results, query, args...); err != nil {
		return nil, wrapPlayerLockedSongRepositoryError("resolve song ids by display ids", err)
	}

	resolved := make(map[string]int, len(results))
	for _, result := range results {
		resolved[result.DisplayID] = result.ID
	}
	return resolved, nil
}
