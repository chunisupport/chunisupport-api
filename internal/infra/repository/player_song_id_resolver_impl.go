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
