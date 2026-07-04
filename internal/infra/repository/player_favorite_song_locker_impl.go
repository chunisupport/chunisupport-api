package repository

import (
	"context"
	"database/sql"
	"errors"

	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
)

var (
	_ usecase.PlayerFavoriteSongQueryService = (*PlayerFavoriteSongQueryService)(nil)
	_ usecase.PlayerFavoriteSongLocker       = (*PlayerFavoriteSongLocker)(nil)
)

type PlayerFavoriteSongQueryService struct{}

func NewPlayerFavoriteSongQueryService() *PlayerFavoriteSongQueryService {
	return &PlayerFavoriteSongQueryService{}
}

type PlayerFavoriteSongLocker struct{}

func NewPlayerFavoriteSongLocker() *PlayerFavoriteSongLocker {
	return &PlayerFavoriteSongLocker{}
}

func (s *PlayerFavoriteSongLocker) LockPlayer(ctx context.Context, exec domainrepo.Executor, playerID int) error {
	const q = `SELECT id FROM players WHERE id = ? FOR UPDATE`
	var id int
	if err := exec.GetContext(ctx, &id, q, playerID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return wrapPlayerFavoriteSongRepositoryError("lock player", domainrepo.ErrPlayerNotFound)
		}
		return wrapPlayerFavoriteSongRepositoryError("lock player", err)
	}
	return nil
}
