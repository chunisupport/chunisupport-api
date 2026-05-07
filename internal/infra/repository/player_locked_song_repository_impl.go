package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/jmoiron/sqlx"
)

type PlayerLockedSongRepository struct {
	db *sqlx.DB
}

func NewPlayerLockedSongRepository(db *sqlx.DB) *PlayerLockedSongRepository {
	return &PlayerLockedSongRepository{db: db}
}

func (r *PlayerLockedSongRepository) ListByPlayerID(ctx context.Context, exec domainrepo.Executor, playerID int) ([]*entity.PlayerLockedSong, error) {
	const q = `SELECT player_id, song_id, is_ultima FROM player_locked_songs WHERE player_id = ? ORDER BY song_id ASC, is_ultima ASC`
	var rows []entity.PlayerLockedSong
	if err := sqlx.SelectContext(ctx, exec, &rows, q, playerID); err != nil {
		return nil, wrapPlayerLockedSongRepositoryError("list by player id", err)
	}
	res := make([]*entity.PlayerLockedSong, 0, len(rows))
	for i := range rows {
		res = append(res, &rows[i])
	}
	return res, nil
}

func (r *PlayerLockedSongRepository) Create(ctx context.Context, exec domainrepo.Executor, lockedSong *entity.PlayerLockedSong) error {
	if err := lockedSong.Validate(); err != nil {
		return err
	}
	const q = `INSERT INTO player_locked_songs (player_id, song_id, is_ultima) VALUES (?, ?, ?)`
	_, err := exec.ExecContext(ctx, q, lockedSong.PlayerID, lockedSong.SongID, lockedSong.IsUltima)
	if err != nil && !isMySQLDuplicateEntryForKey(err, "PRIMARY") {
		return wrapPlayerLockedSongRepositoryError("create", err)
	}
	return nil
}

func (r *PlayerLockedSongRepository) Delete(ctx context.Context, exec domainrepo.Executor, playerID int, songID int, isUltima bool) error {
	const q = `DELETE FROM player_locked_songs WHERE player_id = ? AND song_id = ? AND is_ultima = ?`
	_, err := exec.ExecContext(ctx, q, playerID, songID, isUltima)
	return wrapPlayerLockedSongRepositoryError("delete", err)
}

type playerLockedSongReadModelRow struct {
	SongID    int    `db:"song_id"`
	DisplayID string `db:"display_id"`
	Title     string `db:"title"`
	IsUltima  bool   `db:"is_ultima"`
}

func (r *PlayerLockedSongRepository) ListWithSongDisplayIDAndTitleByPlayerID(ctx context.Context, exec domainrepo.Executor, playerID int) ([]*usecase.PlayerLockedSongReadModel, error) {
	const q = `SELECT pls.song_id, pls.is_ultima, s.display_id, s.title FROM player_locked_songs pls INNER JOIN songs s ON s.id = pls.song_id WHERE pls.player_id = ? AND s.is_deleted = FALSE AND s.is_worldsend = FALSE ORDER BY s.display_id ASC, pls.is_ultima ASC`
	var rows []playerLockedSongReadModelRow
	if err := sqlx.SelectContext(ctx, exec, &rows, q, playerID); err != nil {
		return nil, wrapPlayerLockedSongRepositoryError("list read model by player id", err)
	}
	res := make([]*usecase.PlayerLockedSongReadModel, 0, len(rows))
	for _, row := range rows {
		res = append(res, &usecase.PlayerLockedSongReadModel{SongID: row.SongID, DisplayID: row.DisplayID, Title: row.Title, IsUltima: row.IsUltima})
	}
	return res, nil
}

func (r *PlayerLockedSongRepository) ResolveSongIDByDisplayID(ctx context.Context, exec domainrepo.Executor, displayID string) (*int, error) {
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

func wrapPlayerLockedSongRepositoryError(operation string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %s: %v", domainrepo.ErrRepositoryOperationFailed, operation, err)
}
