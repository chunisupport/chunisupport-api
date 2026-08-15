package repository

import (
	"context"
	"time"

	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/jmoiron/sqlx"
)

type playerFavoriteSongReadModelRow struct {
	DisplayID   string    `db:"display_id"`
	Title       string    `db:"title"`
	Jacket      *string   `db:"jacket"`
	FavoritedAt time.Time `db:"favorited_at"`
}

func (s *PlayerFavoriteSongQueryService) ListWithSongDetailsByPlayerID(ctx context.Context, exec domainrepo.Executor, playerID int) ([]*usecase.PlayerFavoriteSongReadModel, error) {
	const q = `
		SELECT s.display_id, s.title, s.jacket, pfs.created_at AS favorited_at
		FROM player_favorite_songs pfs
		INNER JOIN songs s ON s.id = pfs.song_id
		WHERE pfs.player_id = ?
		  AND s.is_deleted = FALSE
		  AND s.is_worldsend = FALSE
		ORDER BY pfs.created_at DESC, s.display_id ASC
	`
	var rows []playerFavoriteSongReadModelRow
	if err := sqlx.SelectContext(ctx, exec, &rows, q, playerID); err != nil {
		return nil, wrapPlayerFavoriteSongRepositoryError("list with song details by player id", err)
	}
	res := make([]*usecase.PlayerFavoriteSongReadModel, 0, len(rows))
	for _, row := range rows {
		res = append(res, &usecase.PlayerFavoriteSongReadModel{
			DisplayID:   row.DisplayID,
			Title:       row.Title,
			Jacket:      row.Jacket,
			FavoritedAt: row.FavoritedAt,
		})
	}
	return res, nil
}
