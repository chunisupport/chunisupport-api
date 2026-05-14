package usecase

import (
	"context"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/displayid"
)

type PlayerLockedSongInput struct {
	DisplayID displayid.DisplayID
	IsUltima  bool
}

type PlayerLockedSongOutput struct {
	DisplayID string
	Title     string
	IsUltima  bool
}

type PlayerLockedSongUsecase interface {
	List(ctx context.Context, username string, requester *entity.User) ([]*PlayerLockedSongOutput, error)
	Lock(ctx context.Context, userID int, input *PlayerLockedSongInput) error
	Unlock(ctx context.Context, userID int, input *PlayerLockedSongInput) error
	Batch(ctx context.Context, userID int, input *PlayerLockedSongBatchInput) error
}

type PlayerLockedSongBatchInput struct {
	Add    []*PlayerLockedSongInput
	Delete []*PlayerLockedSongInput
}

type PlayerLockedSongReadModel struct {
	SongID    int
	DisplayID string
	Title     string
	IsUltima  bool
}

type PlayerLockedSongQueryService interface {
	ListWithSongDisplayIDAndTitleByPlayerID(ctx context.Context, exec repository.Executor, playerID int) ([]*PlayerLockedSongReadModel, error)
}

type PlayerSongIDResolver interface {
	ResolveSongIDByDisplayID(ctx context.Context, exec repository.Executor, displayID string) (*int, error)
}
