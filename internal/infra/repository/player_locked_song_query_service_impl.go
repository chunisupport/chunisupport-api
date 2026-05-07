package repository

import (
	"context"

	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/jmoiron/sqlx"
)

type playerLockedSongReadModelRow struct {
	SongID    int    `db:"song_id"`
	DisplayID string `db:"display_id"`
	Title     string `db:"title"`
	IsUltima  bool   `db:"is_ultima"`
}

func (s *PlayerLockedSongQueryService) ListWithSongDisplayIDAndTitleByPlayerID(ctx context.Context, exec domainrepo.Executor, playerID int) ([]*usecase.PlayerLockedSongReadModel, error) {
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
