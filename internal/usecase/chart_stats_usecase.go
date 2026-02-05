package usecase

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/chunisupport/chunisupport-api/internal/infra/masterdata"
)

// ChartStatsUsecase は譜面統計の取得ユースケースを提供します。
type ChartStatsUsecase interface {
	// GetSongStatsByDisplayID は指定されたDisplayIDの譜面統計を取得します。
	GetSongStatsByDisplayID(ctx context.Context, displayID string) (*entity.SongChartStats, error)
}

type chartStatsUsecaseImpl struct {
	songRepo           repository.SongRepository
	worldsendChartRepo repository.WorldsendChartRepository
	statsRepo          repository.ChartStatsRepository
	masterCache        repository.SongMasterProvider
	staticMasterCache  *masterdata.StaticCache
	defaultExecutor    repository.Executor
	statsExecutor      repository.Executor
}

// NewChartStatsUsecase は ChartStatsUsecase の実装を生成します。
func NewChartStatsUsecase(
	songRepo repository.SongRepository,
	worldsendChartRepo repository.WorldsendChartRepository,
	statsRepo repository.ChartStatsRepository,
	masterCache repository.SongMasterProvider,
	staticMasterCache *masterdata.StaticCache,
	defaultExecutor repository.Executor,
	statsExecutor repository.Executor,
) ChartStatsUsecase {
	return &chartStatsUsecaseImpl{
		songRepo:           songRepo,
		worldsendChartRepo: worldsendChartRepo,
		statsRepo:          statsRepo,
		masterCache:        masterCache,
		staticMasterCache:  staticMasterCache,
		defaultExecutor:    defaultExecutor,
		statsExecutor:      statsExecutor,
	}
}

type chartEntry struct {
	id  int
	key string
}

// GetSongStatsByDisplayID は指定されたDisplayIDの譜面統計を取得します。
func (u *chartStatsUsecaseImpl) GetSongStatsByDisplayID(ctx context.Context, displayID string) (*entity.SongChartStats, error) {
	songWithCharts, err := u.songRepo.FindByDisplayID(ctx, u.defaultExecutor, displayID)
	if err != nil {
		return nil, err
	}

	// rating_bandsはキャッシュから取得
	ratingBands := u.staticMasterCache.RatingBands

	entries, err := u.buildChartEntries(ctx, songWithCharts)
	if err != nil {
		return nil, err
	}

	chartIDs := make([]int, 0, len(entries))
	for _, entry := range entries {
		chartIDs = append(chartIDs, entry.id)
	}

	statsRows, err := u.statsRepo.FindChartStatsByChartIDs(ctx, u.statsExecutor, chartIDs)
	if err != nil {
		return nil, err
	}

	statsByChartID := make(map[int][]*entity.ChartStatsByRatingBand)
	for _, row := range statsRows {
		statsByChartID[row.ChartID] = append(statsByChartID[row.ChartID], row)
	}

	bandOrder := make(map[int]int, len(ratingBands))
	for _, band := range ratingBands {
		bandOrder[band.ID] = band.SortOrder
	}

	charts := make(map[string][]*entity.ChartStatsByRatingBand, len(entries))
	for _, entry := range entries {
		stats := statsByChartID[entry.id]
		sort.Slice(stats, func(i, j int) bool {
			return bandOrder[stats[i].RatingBandID] < bandOrder[stats[j].RatingBandID]
		})
		charts[entry.key] = stats
	}

	return &entity.SongChartStats{
		SongID: songWithCharts.Song.DisplayID,
		Charts: charts,
	}, nil
}

func (u *chartStatsUsecaseImpl) buildChartEntries(ctx context.Context, songWithCharts *repository.SongWithCharts) ([]chartEntry, error) {
	if songWithCharts.Song.IsWorldsend {
		worldsend, err := u.worldsendChartRepo.FindByDisplayID(ctx, u.defaultExecutor, songWithCharts.Song.DisplayID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, repository.ErrSongNotFound
			}
			return nil, err
		}

		return []chartEntry{
			{
				id:  worldsend.Chart.ID,
				key: info.StatsDifficultyWorldsend,
			},
		}, nil
	}

	masters := u.masterCache.SongMasters()
	if masters == nil {
		return nil, fmt.Errorf("master cache is not initialized")
	}

	entries := make([]chartEntry, 0, len(songWithCharts.Charts))
	for _, chart := range songWithCharts.Charts {
		name, ok := masters.DifficultyNamesByID[chart.DifficultyID]
		if !ok {
			return nil, fmt.Errorf("difficulty not found: %d", chart.DifficultyID)
		}
		entries = append(entries, chartEntry{
			id:  chart.ID,
			key: name,
		})
	}

	return entries, nil
}
