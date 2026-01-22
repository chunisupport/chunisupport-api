package usecase

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"github.com/Qman110101/chunisupport-api/internal/domain/entity"
	"github.com/Qman110101/chunisupport-api/internal/domain/repository"
	"github.com/Qman110101/chunisupport-api/internal/dto"
)

// WorldsendUsecase は WORLD'S END 楽曲に関するユースケースを提供します。
type WorldsendUsecase interface {
	// GetAllWorldsendSongs は全 WORLD'S END 楽曲を取得します。
	// includeDeleted が false の場合、削除済み楽曲は除外されます。
	GetAllWorldsendSongs(ctx context.Context, includeDeleted bool) ([]*dto.WorldsendSongDTO, error)

	// GetWorldsendSongByDisplayID は指定された DisplayID の WORLD'S END 楽曲を取得します。
	GetWorldsendSongByDisplayID(ctx context.Context, displayID string) (*dto.WorldsendSongDTO, error)

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
}

// NewWorldsendUsecase は WorldsendUsecase の実装を生成します。
func NewWorldsendUsecase(worldsendChartRepo repository.WorldsendChartRepository) WorldsendUsecase {
	return &worldsendUsecase{
		worldsendChartRepo: worldsendChartRepo,
	}
}

// GetAllWorldsendSongs は全 WORLD'S END 楽曲を取得します。
func (s *worldsendUsecase) GetAllWorldsendSongs(ctx context.Context, includeDeleted bool) ([]*dto.WorldsendSongDTO, error) {
	songsWithCharts, err := s.worldsendChartRepo.FindAll(ctx, includeDeleted)
	if err != nil {
		slog.Error("failed to find all worldsend songs", "error", err)
		return nil, err
	}

	results := make([]*dto.WorldsendSongDTO, len(songsWithCharts))
	for i, swc := range songsWithCharts {
		results[i] = toWorldsendSongDTO(swc)
	}

	return results, nil
}

// GetWorldsendSongByDisplayID は指定された DisplayID の WORLD'S END 楽曲を取得します。
func (s *worldsendUsecase) GetWorldsendSongByDisplayID(ctx context.Context, displayID string) (*dto.WorldsendSongDTO, error) {
	songWithChart, err := s.worldsendChartRepo.FindByDisplayID(ctx, displayID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrSongNotFound
		}
		slog.Error("failed to find worldsend song by display_id", "display_id", displayID, "error", err)
		return nil, err
	}

	return toWorldsendSongDTO(songWithChart), nil
}

// DeleteWorldsendSong は指定された DisplayID の WORLD'S END 楽曲を論理削除します。
func (s *worldsendUsecase) DeleteWorldsendSong(ctx context.Context, displayID string) error {
	err := s.worldsendChartRepo.DeleteSong(ctx, displayID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return repository.ErrSongNotFound
		}
		slog.Error("failed to delete worldsend song", "display_id", displayID, "error", err)
		return err
	}
	return nil
}

// RestoreWorldsendSong は指定された DisplayID の WORLD'S END 楽曲を復活させます。
func (s *worldsendUsecase) RestoreWorldsendSong(ctx context.Context, displayID string) error {
	err := s.worldsendChartRepo.RestoreSong(ctx, displayID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
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
	err := s.worldsendChartRepo.UpdateSongs(ctx, songs, charts)
	if err != nil {
		slog.Error("failed to update worldsend songs", "error", err)
		return err
	}

	return nil
}

// toWorldsendSongDTO は WorldsendSongWithChart を WorldsendSongDTO に変換します。
func toWorldsendSongDTO(swc *repository.WorldsendSongWithChart) *dto.WorldsendSongDTO {
	if swc == nil {
		return nil
	}

	result := &dto.WorldsendSongDTO{
		IsDeleted: false,
	}

	if swc.Song != nil {
		result.ID = swc.Song.DisplayID
		result.Title = swc.Song.Title
		result.Artist = swc.Song.Artist
		result.GenreID = swc.Song.GenreID
		result.BPM = swc.Song.BPM
		result.ReleasedAt = swc.Song.ReleasedAt
		result.OfficialIdx = swc.Song.OfficialIdx
		result.Jacket = swc.Song.Jacket
		result.IsDeleted = swc.Song.IsDeleted
	}

	if swc.Chart != nil {
		result.WeStar = swc.Chart.WeStar
		result.WeKanji = swc.Chart.WeKanji
		result.Notes = dto.ToNotesIntPtr(swc.Chart.Notes)
	}

	return result
}
