package repository

import (
	"context"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

type PlayerLockedSongRepository interface {
	ListByPlayerID(ctx context.Context, exec Executor, playerID int) ([]*entity.PlayerLockedSong, error)
	Create(ctx context.Context, exec Executor, lockedSong *entity.PlayerLockedSong) error
	Delete(ctx context.Context, exec Executor, playerID int, songID int, isUltima bool) error
}
