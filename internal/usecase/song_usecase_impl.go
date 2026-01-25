package usecase

import (
	"context"
	"fmt"

	"github.com/Qman110101/chunisupport-api/internal/domain/entity"
	"github.com/Qman110101/chunisupport-api/internal/domain/repository"
	"github.com/Qman110101/chunisupport-api/internal/dto/api_internal"
)

// songUsecaseImpl は SongUsecase の実装です。
type songUsecaseImpl struct {
	songRepo        repository.SongRepository
	statsRepo       repository.ChartStatisticsRepository
	masterCache     repository.SongMasterProvider
	tm              TransactionManager
	defaultExecutor repository.Executor
}

// NewSongService は新しい SongUsecase を生成します。
func NewSongService(
	songRepo repository.SongRepository,
	statsRepo repository.ChartStatisticsRepository,
	masterCache repository.SongMasterProvider,
	tm TransactionManager,
	defaultExecutor repository.Executor,
) SongUsecase {
	return &songUsecaseImpl{
		songRepo:        songRepo,
		statsRepo:       statsRepo,
		masterCache:     masterCache,
		tm:              tm,
		defaultExecutor: defaultExecutor,
	}
}

// GetAllSongsExcludingWorldsend はWORLD'S END以外の全楽曲を取得します。
// includeDeletedがfalseの場合、削除済み楽曲は除外されます。
func (s *songUsecaseImpl) GetAllSongsExcludingWorldsend(ctx context.Context, includeDeleted bool) ([]*repository.SongWithCharts, error) {
	return s.songRepo.FindAllExcludingWorldsend(ctx, s.defaultExecutor, includeDeleted)
}

// GetSongByDisplayID は指定されたDisplayIDの楽曲を取得します。
func (s *songUsecaseImpl) GetSongByDisplayID(ctx context.Context, displayID string) (*repository.SongWithCharts, error) {
	// リポジトリ側で既にErrSongNotFoundに変換済み
	return s.songRepo.FindByDisplayID(ctx, s.defaultExecutor, displayID)
}

// DeleteSong は指定されたDisplayIDの楽曲を論理削除します。
func (s *songUsecaseImpl) DeleteSong(ctx context.Context, displayID string) error {
	return s.tm.Transactional(ctx, func(tx repository.Executor) error {
		// 楽曲の存在確認
		_, err := s.songRepo.FindByDisplayID(ctx, tx, displayID)
		if err != nil {
			return err
		}

		return s.songRepo.DeleteSong(ctx, tx, displayID)
	})
}

// RestoreSong は指定されたDisplayIDの楽曲を復活させます。
func (s *songUsecaseImpl) RestoreSong(ctx context.Context, displayID string) error {
	return s.tm.Transactional(ctx, func(tx repository.Executor) error {
		// 楽曲の存在確認
		_, err := s.songRepo.FindByDisplayID(ctx, tx, displayID)
		if err != nil {
			return err
		}

		return s.songRepo.RestoreSong(ctx, tx, displayID)
	})
}

// UpdateSongs は楽曲および譜面情報を一括更新します。
func (s *songUsecaseImpl) UpdateSongs(ctx context.Context, requests []*api_internal.UpdateSongRequest) error {
	// マスターデータ検証
	masters := s.masterCache.SongMasters()
	if masters == nil {
		return fmt.Errorf("master cache is not initialized")
	}

	for _, req := range requests {
		// GenreID の検証
		if req.GenreID != nil {
			if _, ok := masters.GenreNamesByID[*req.GenreID]; !ok {
				return fmt.Errorf("invalid genre_id: %d (song: %s)", *req.GenreID, req.DisplayID)
			}
		}

		// DifficultyID の検証
		for _, chart := range req.Charts {
			if _, ok := masters.DifficultyNamesByID[chart.DifficultyID]; !ok {
				return fmt.Errorf("invalid difficulty_id: %d (song: %s)", chart.DifficultyID, req.DisplayID)
			}
		}
	}

	// トランザクション内でリポジトリに委譲
	return s.tm.Transactional(ctx, func(tx repository.Executor) error {
		return s.songRepo.UpdateSongs(ctx, tx, requests)
	})
}

// GetChartStatisticsByChartIDs は指定された譜面IDリストの統計を一括取得します。
// 譜面IDをキーとするマップで返します（統計が存在しない譜面は空のスライス）。
func (s *songUsecaseImpl) GetChartStatisticsByChartIDs(ctx context.Context, chartIDs []int) (map[int][]*entity.ChartStatistics, error) {
	if len(chartIDs) == 0 {
		return make(map[int][]*entity.ChartStatistics), nil
	}

	// 統計データを一括取得（N+1問題回避）
	statsList, err := s.statsRepo.FindByChartIDs(ctx, nil, chartIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get chart statistics: %w", err)
	}

	// 譜面IDをキーとするマップに変換
	statsMap := make(map[int][]*entity.ChartStatistics)
	for _, stats := range statsList {
		statsMap[stats.ChartID] = append(statsMap[stats.ChartID], stats)
	}

	return statsMap, nil
}
