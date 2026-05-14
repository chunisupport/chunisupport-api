package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

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
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(displayIDs)), ",")
	args := make([]any, 0, len(displayIDs))
	for _, displayID := range displayIDs {
		args = append(args, displayID)
	}
	query := fmt.Sprintf("SELECT display_id, id FROM songs WHERE display_id IN (%s)", placeholders)
	rows, err := exec.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, wrapPlayerLockedSongRepositoryError("resolve song ids by display ids", err)
	}
	defer rows.Close()
	resolved := make(map[string]int, len(displayIDs))
	for rows.Next() {
		var displayID string
		var songID int
		if err := rows.Scan(&displayID, &songID); err != nil {
			return nil, wrapPlayerLockedSongRepositoryError("resolve song ids by display ids", err)
		}
		resolved[displayID] = songID
	}
	if err := rows.Err(); err != nil {
		return nil, wrapPlayerLockedSongRepositoryError("resolve song ids by display ids", err)
	}
	return resolved, nil
}
