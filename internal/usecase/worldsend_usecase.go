package usecase

import (
	"context"
	"errors"
	"log/slog"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
)

// WorldsendUsecase は WORLD'S END 楽曲に関するユースケースを提供します。
type WorldsendUsecase interface {
	// GetAllWorldsendSongs は全 WORLD'S END 楽曲を取得します。
	// includeDeleted が false の場合、削除済み楽曲は除外されます。
	GetAllWorldsendSongs(ctx context.Context, includeDeleted bool) ([]*repository.WorldsendSongWithChart, error)

	// GetWorldsendSongByDisplayID は指定された DisplayID の WORLD'S END 楽曲を取得します。
	GetWorldsendSongByDisplayID(ctx context.Context, displayID string) (*repository.WorldsendSongWithChart, error)

	// DeleteWorldsendSong は指定された DisplayID の WORLD'S END 楽曲を論理削除します。
	DeleteWorldsendSong(ctx context.Context, displayID string) error

	// RestoreWorldsendSong は指定された DisplayID の WORLD'S END 楽曲を復活させます。
	RestoreWorldsendSong(ctx context.Context, displayID string) error

	// UpdateWorldsendSongs は WORLD'S END 楽曲および譜面情報を一括更新します。
	UpdateWorldsendSongs(ctx context.Context, songs []*entity.Song, charts []*entity.WorldsendChart) error
}

// worldsendUsecase は WorldsendUsecase の実装です。
type worldsendUsecase struct {
	worldsendChartRepo repository.WorldsendChartRepository
	defaultExecutor    repository.Executor
	tm                 TransactionManager
}

// NewWorldsendUsecase は WorldsendUsecase の実装を生成します。
func NewWorldsendUsecase(worldsendChartRepo repository.WorldsendChartRepository, tm TransactionManager, defaultExecutor repository.Executor) WorldsendUsecase {
	return &worldsendUsecase{
		worldsendChartRepo: worldsendChartRepo,
		defaultExecutor:    defaultExecutor,
		tm:                 tm,
	}
}

// GetAllWorldsendSongs は全 WORLD'S END 楽曲を取得します。
func (s *worldsendUsecase) GetAllWorldsendSongs(ctx context.Context, includeDeleted bool) ([]*repository.WorldsendSongWithChart, error) {
	songsWithCharts, err := s.worldsendChartRepo.FindAll(ctx, s.defaultExecutor, includeDeleted)
	if err != nil {
		slog.Error("failed to find all worldsend songs", "error", err)
		return nil, err
	}

	return songsWithCharts, nil
}

// GetWorldsendSongByDisplayID は指定された DisplayID の WORLD'S END 楽曲を取得します。
func (s *worldsendUsecase) GetWorldsendSongByDisplayID(ctx context.Context, displayID string) (*repository.WorldsendSongWithChart, error) {
	songWithChart, err := s.worldsendChartRepo.FindByDisplayID(ctx, s.defaultExecutor, displayID)
	if err != nil {
		if errors.Is(err, repository.ErrSongNotFound) {
			return nil, repository.ErrSongNotFound
		}
		slog.Error("failed to find worldsend song by display_id", "display_id", displayID, "error", err)
		return nil, err
	}

	return songWithChart, nil
}

// DeleteWorldsendSong は指定された DisplayID の WORLD'S END 楽曲を論理削除します。
func (s *worldsendUsecase) DeleteWorldsendSong(ctx context.Context, displayID string) error {
	err := s.worldsendChartRepo.DeleteSong(ctx, s.defaultExecutor, displayID)
	if err != nil {
		if errors.Is(err, repository.ErrSongNotFound) {
			return repository.ErrSongNotFound
		}
		slog.Error("failed to delete worldsend song", "display_id", displayID, "error", err)
		return err
	}
	return nil
}

// RestoreWorldsendSong は指定された DisplayID の WORLD'S END 楽曲を復活させます。
func (s *worldsendUsecase) RestoreWorldsendSong(ctx context.Context, displayID string) error {
	err := s.worldsendChartRepo.RestoreSong(ctx, s.defaultExecutor, displayID)
	if err != nil {
		if errors.Is(err, repository.ErrSongNotFound) {
			return repository.ErrSongNotFound
		}
		slog.Error("failed to restore worldsend song", "display_id", displayID, "error", err)
		return err
	}
	return nil
}

// UpdateWorldsendSongs は WORLD'S END 楽曲および譜面情報を一括更新します。
func (s *worldsendUsecase) UpdateWorldsendSongs(ctx context.Context, songs []*entity.Song, charts []*entity.WorldsendChart) error {
	// バリデーション
	for i, chart := range charts {
		if err := chart.Validate(); err != nil {
			slog.Warn("worldsend chart validation failed", "index", i, "error", err)
			return err
		}
	}

	// リポジトリに委譲
	if err := s.tm.Transactional(ctx, func(tx repository.Executor) error {
		return s.worldsendChartRepo.UpdateSongs(ctx, tx, songs, charts)
	}); err != nil {
		slog.Error("failed to update worldsend songs", "error", err)
		return err
	}

	return nil
}
