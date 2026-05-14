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

type playerLockedSongRow struct {
	PlayerID int  `db:"player_id"`
	SongID   int  `db:"song_id"`
	IsUltima bool `db:"is_ultima"`
}

func (r *PlayerLockedSongRepository) ListByPlayerID(ctx context.Context, exec domainrepo.Executor, playerID int) ([]*entity.PlayerLockedSong, error) {
	const q = `SELECT player_id, song_id, is_ultima FROM player_locked_songs WHERE player_id = ? ORDER BY song_id ASC, is_ultima ASC`
	var rows []playerLockedSongRow
	if err := sqlx.SelectContext(ctx, exec, &rows, q, playerID); err != nil {
		return nil, wrapPlayerLockedSongRepositoryError("list by player id", err)
	}
	res := make([]*entity.PlayerLockedSong, 0, len(rows))
	for _, row := range rows {
		lockedSong, err := entity.NewPlayerLockedSong(row.PlayerID, row.SongID, row.IsUltima)
		if err != nil {
			return nil, wrapPlayerLockedSongRepositoryError("list by player id", err)
		}
		res = append(res, lockedSong)
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

func (r *PlayerLockedSongRepository) BulkCreate(ctx context.Context, exec domainrepo.Executor, lockedSongs []*entity.PlayerLockedSong) error {
	if len(lockedSongs) == 0 {
		return nil
	}
	for _, lockedSong := range lockedSongs {
		if err := lockedSong.Validate(); err != nil {
			return err
		}
	}
	const baseQuery = `INSERT INTO player_locked_songs (player_id, song_id, is_ultima) VALUES `
	const valueClause = `(?, ?, ?)`
	query := baseQuery
	args := make([]interface{}, 0, len(lockedSongs)*3)
	for i, lockedSong := range lockedSongs {
		if i > 0 {
			query += ", "
		}
		query += valueClause
		args = append(args, lockedSong.PlayerID, lockedSong.SongID, lockedSong.IsUltima)
	}
	query += ` ON DUPLICATE KEY UPDATE player_id = player_id`
	_, err := exec.ExecContext(ctx, query, args...)
	if err != nil {
		return wrapPlayerLockedSongRepositoryError("bulk create", err)
	}
	return nil
}

func (r *PlayerLockedSongRepository) BulkDelete(ctx context.Context, exec domainrepo.Executor, playerID int, songIDs []int, isUltimaFlags []bool) error {
	if len(songIDs) == 0 {
		return nil
	}
	if len(songIDs) != len(isUltimaFlags) {
		return wrapPlayerLockedSongRepositoryError("bulk delete", fmt.Errorf("songIDs and isUltimaFlags length mismatch"))
	}
	const baseQuery = `DELETE FROM player_locked_songs WHERE player_id = ? AND (`
	query := baseQuery
	args := []interface{}{playerID}
	for i := range songIDs {
		if i > 0 {
			query += " OR "
		}
		query += "(song_id = ? AND is_ultima = ?)"
		args = append(args, songIDs[i], isUltimaFlags[i])
	}
	query += ")"
	_, err := exec.ExecContext(ctx, query, args...)
	return wrapPlayerLockedSongRepositoryError("bulk delete", err)
}

func wrapPlayerLockedSongRepositoryError(operation string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %s: %v", domainrepo.ErrRepositoryOperationFailed, operation, err)
}
