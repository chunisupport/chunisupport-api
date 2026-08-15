package repository

import (
	"context"
	"fmt"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/jmoiron/sqlx"
)

var _ domainrepo.PlayerFavoriteSongRepository = (*PlayerFavoriteSongRepository)(nil)

type PlayerFavoriteSongRepository struct{}

func NewPlayerFavoriteSongRepository() *PlayerFavoriteSongRepository {
	return &PlayerFavoriteSongRepository{}
}

func (r *PlayerFavoriteSongRepository) CountByPlayerID(ctx context.Context, exec domainrepo.Executor, playerID int) (int, error) {
	const q = `SELECT COUNT(*) FROM player_favorite_songs WHERE player_id = ?`
	var count int
	if err := sqlx.GetContext(ctx, exec, &count, q, playerID); err != nil {
		return 0, wrapPlayerFavoriteSongRepositoryError("count by player id", err)
	}
	return count, nil
}

func (r *PlayerFavoriteSongRepository) Exists(ctx context.Context, exec domainrepo.Executor, playerID int, songID int) (bool, error) {
	const q = `SELECT EXISTS(SELECT 1 FROM player_favorite_songs WHERE player_id = ? AND song_id = ?)`
	var exists bool
	if err := sqlx.GetContext(ctx, exec, &exists, q, playerID, songID); err != nil {
		return false, wrapPlayerFavoriteSongRepositoryError("exists", err)
	}
	return exists, nil
}

func (r *PlayerFavoriteSongRepository) Save(ctx context.Context, exec domainrepo.Executor, favorite *entity.PlayerFavoriteSong) error {
	if err := favorite.Validate(); err != nil {
		return err
	}
	const q = `INSERT INTO player_favorite_songs (player_id, song_id, created_at) VALUES (?, ?, ?)`
	_, err := exec.ExecContext(ctx, q, favorite.PlayerID, favorite.SongID, favorite.CreatedAt)
	if err != nil && !isMySQLDuplicateEntryForKey(err, "PRIMARY") {
		return wrapPlayerFavoriteSongRepositoryError("save", err)
	}
	return nil
}

func (r *PlayerFavoriteSongRepository) Delete(ctx context.Context, exec domainrepo.Executor, playerID int, songID int) error {
	const q = `DELETE FROM player_favorite_songs WHERE player_id = ? AND song_id = ?`
	_, err := exec.ExecContext(ctx, q, playerID, songID)
	return wrapPlayerFavoriteSongRepositoryError("delete", err)
}

func (r *PlayerFavoriteSongRepository) DeleteBySongID(ctx context.Context, exec domainrepo.Executor, songID int) error {
	const q = `DELETE FROM player_favorite_songs WHERE song_id = ?`
	_, err := exec.ExecContext(ctx, q, songID)
	return wrapPlayerFavoriteSongRepositoryError("delete by song id", err)
}

func wrapPlayerFavoriteSongRepositoryError(operation string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %s: %v", domainrepo.ErrRepositoryOperationFailed, operation, err)
}
