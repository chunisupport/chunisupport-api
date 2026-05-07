package usecase

import (
	"context"
	"errors"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	domainservice "github.com/chunisupport/chunisupport-api/internal/domain/service"
)

var errPlayerLockedSongInputRequired = errors.New("input is required")

type playerLockedSongUsecase struct {
	db           repository.Executor
	playerRepo   repository.PlayerRepository
	songRepo     repository.SongRepository
	lockedRepo   repository.PlayerLockedSongRepository
	queryService PlayerLockedSongQueryService
	resolver     PlayerSongIDResolver
}

func NewPlayerLockedSongUsecase(db repository.Executor, playerRepo repository.PlayerRepository, songRepo repository.SongRepository, lockedRepo repository.PlayerLockedSongRepository, queryService PlayerLockedSongQueryService, resolver PlayerSongIDResolver) PlayerLockedSongUsecase {
	return &playerLockedSongUsecase{db: db, playerRepo: playerRepo, songRepo: songRepo, lockedRepo: lockedRepo, queryService: queryService, resolver: resolver}
}

func (u *playerLockedSongUsecase) List(ctx context.Context, userID int) ([]*PlayerLockedSongOutput, error) {
	player, err := u.playerRepo.FindByUserID(ctx, u.db, userID)
	if err != nil {
		return nil, err
	}
	if player == nil {
		return nil, ErrPlayerNotLinked
	}
	rows, err := u.queryService.ListWithSongDisplayIDAndTitleByPlayerID(ctx, u.db, player.ID)
	if err != nil {
		return nil, err
	}
	items := make([]*PlayerLockedSongOutput, 0, len(rows))
	for _, row := range rows {
		items = append(items, &PlayerLockedSongOutput{DisplayID: row.DisplayID, Title: row.Title, IsUltima: row.IsUltima})
	}
	return items, nil
}

func (u *playerLockedSongUsecase) Lock(ctx context.Context, userID int, input *PlayerLockedSongInput) error {
	if input == nil {
		return errPlayerLockedSongInputRequired
	}
	player, err := u.playerRepo.FindByUserID(ctx, u.db, userID)
	if err != nil {
		return err
	}
	if player == nil {
		return ErrPlayerNotLinked
	}
	song, err := u.songRepo.FindByDisplayID(ctx, u.db, input.DisplayID.String())
	if err != nil {
		if errors.Is(err, repository.ErrSongNotFound) {
			return repository.ErrSongNotFound
		}
		return err
	}
	if song == nil || song.IsDeleted || song.IsWorldsend {
		return repository.ErrSongNotFound
	}
	if input.IsUltima {
		if !song.HasDifficultyChart(domainservice.DifficultyIDUltima) {
			return ErrChartNotFound
		}
	}
	lockedSong, err := entity.NewPlayerLockedSong(player.ID, song.ID, input.IsUltima)
	if err != nil {
		return err
	}
	return u.lockedRepo.Create(ctx, u.db, lockedSong)
}

func (u *playerLockedSongUsecase) Unlock(ctx context.Context, userID int, input *PlayerLockedSongInput) error {
	if input == nil {
		return errPlayerLockedSongInputRequired
	}
	player, err := u.playerRepo.FindByUserID(ctx, u.db, userID)
	if err != nil {
		return err
	}
	if player == nil {
		return ErrPlayerNotLinked
	}
	songID, err := u.resolver.ResolveSongIDByDisplayID(ctx, u.db, input.DisplayID.String())
	if err != nil {
		return err
	}
	if songID == nil {
		return nil
	}
	return u.lockedRepo.Delete(ctx, u.db, player.ID, *songID, input.IsUltima)
}
