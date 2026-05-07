package repository

import (
	"context"
	"fmt"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/jmoiron/sqlx"
)

var (
	_ domainrepo.PlayerLockedSongRepository = (*PlayerLockedSongRepository)(nil)
	_ usecase.PlayerLockedSongQueryService  = (*PlayerLockedSongQueryService)(nil)
	_ usecase.PlayerSongIDResolver          = (*PlayerSongIDResolver)(nil)
)

type PlayerLockedSongRepository struct{}

func NewPlayerLockedSongRepository() *PlayerLockedSongRepository {
	return &PlayerLockedSongRepository{}
}

type PlayerLockedSongQueryService struct{}

func NewPlayerLockedSongQueryService() *PlayerLockedSongQueryService {
	return &PlayerLockedSongQueryService{}
}

type PlayerSongIDResolver struct{}

func NewPlayerSongIDResolver() *PlayerSongIDResolver {
	return &PlayerSongIDResolver{}
}

func (r *PlayerLockedSongRepository) ListByPlayerID(ctx context.Context, exec domainrepo.Executor, playerID int) ([]*entity.PlayerLockedSong, error) {
	const q = `SELECT player_id, song_id, is_ultima FROM player_locked_songs WHERE player_id = ? ORDER BY song_id ASC, is_ultima ASC`
	var res []*entity.PlayerLockedSong
	if err := sqlx.SelectContext(ctx, exec, &res, q, playerID); err != nil {
		return nil, wrapPlayerLockedSongRepositoryError("list by player id", err)
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

func wrapPlayerLockedSongRepositoryError(operation string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %s: %v", domainrepo.ErrRepositoryOperationFailed, operation, err)
}
