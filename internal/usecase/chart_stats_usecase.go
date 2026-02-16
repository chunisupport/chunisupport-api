package usecase

import (
	"context"
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
	// requesterAccountTypeIDがnilまたはEDITOR(2)未満の場合、削除済み楽曲はErrSongNotFoundを返します。
	GetSongStatsByDisplayID(ctx context.Context, displayID string, requesterAccountTypeID *int) (*entity.SongChartStats, error)

	// GetChartStatsByDisplayIDAndDifficulty は指定されたDisplayIDと難易度の譜面統計を取得します。
	// difficultyNameは大文字の難易度名（"BASIC", "ADVANCED", "EXPERT", "MASTER", "ULTIMA"）
	// または "WORLD'S END" である必要があります。
	// requesterAccountTypeIDがnilまたはEDITOR(2)未満の場合、削除済み楽曲はErrSongNotFoundを返します。
	GetChartStatsByDisplayIDAndDifficulty(ctx context.Context, displayID, difficultyName string, requesterAccountTypeID *int) (*entity.SingleChartStats, error)
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
// requesterAccountTypeIDがnilまたはEDITOR(2)未満の場合、削除済み楽曲はErrSongNotFoundを返します。
func (u *chartStatsUsecaseImpl) GetSongStatsByDisplayID(ctx context.Context, displayID string, requesterAccountTypeID *int) (*entity.SongChartStats, error) {
	song, err := u.songRepo.FindByDisplayID(ctx, u.defaultExecutor, displayID)
	if err != nil {
		return nil, err
	}

	// 削除済み楽曲の権限チェック
	if song.IsDeleted {
		// EDITOR以上の権限を持たない場合は404を返す
		if requesterAccountTypeID == nil || *requesterAccountTypeID < info.AccountTypeEditor {
			return nil, repository.ErrSongNotFound
		}
	}

	// rating_bandsはキャッシュから取得
	ratingBands := u.staticMasterCache.RatingBands

	entries, err := u.buildChartEntries(ctx, song)
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
		SongID: song.DisplayID,
		Charts: charts,
	}, nil
}

func (u *chartStatsUsecaseImpl) buildChartEntries(ctx context.Context, song *entity.Song) ([]chartEntry, error) {
	if song.IsWorldsend {
		worldsend, err := u.worldsendChartRepo.FindByDisplayID(ctx, u.defaultExecutor, song.DisplayID)
		if err != nil {
			if errors.Is(err, repository.ErrSongNotFound) {
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

	entries := make([]chartEntry, 0, len(song.Charts))
	for _, chart := range song.Charts {
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

// GetChartStatsByDisplayIDAndDifficulty は指定されたDisplayIDと難易度の譜面統計を取得します。
// requesterAccountTypeIDがnilまたはEDITOR(2)未満の場合、削除済み楽曲はErrSongNotFoundを返します。
func (u *chartStatsUsecaseImpl) GetChartStatsByDisplayIDAndDifficulty(ctx context.Context, displayID, difficultyName string, requesterAccountTypeID *int) (*entity.SingleChartStats, error) {
	song, err := u.songRepo.FindByDisplayID(ctx, u.defaultExecutor, displayID)
	if err != nil {
		return nil, err
	}

	// 削除済み楽曲の権限チェック
	if song.IsDeleted {
		// EDITOR以上の権限を持たない場合は404を返す
		if requesterAccountTypeID == nil || *requesterAccountTypeID < info.AccountTypeEditor {
			return nil, repository.ErrSongNotFound
		}
	}

	// rating_bandsはキャッシュから取得
	ratingBands := u.staticMasterCache.RatingBands

	// 指定難易度の譜面を検索
	entry, err := u.findChartEntryByDifficulty(ctx, song, difficultyName)
	if err != nil {
		return nil, err
	}

	// 譜面統計を取得
	statsRows, err := u.statsRepo.FindChartStatsByChartIDs(ctx, u.statsExecutor, []int{entry.id})
	if err != nil {
		return nil, err
	}

	// レーティング帯順でソート
	bandOrder := make(map[int]int, len(ratingBands))
	for _, band := range ratingBands {
		bandOrder[band.ID] = band.SortOrder
	}

	sort.Slice(statsRows, func(i, j int) bool {
		return bandOrder[statsRows[i].RatingBandID] < bandOrder[statsRows[j].RatingBandID]
	})

	return &entity.SingleChartStats{
		SongID:     song.DisplayID,
		Difficulty: entry.key,
		Stats:      statsRows,
	}, nil
}

// findChartEntryByDifficulty は指定難易度の譜面エントリを検索します。
func (u *chartStatsUsecaseImpl) findChartEntryByDifficulty(ctx context.Context, song *entity.Song, difficultyName string) (*chartEntry, error) {
	// WORLD'S END楽曲の場合
	if difficultyName == info.StatsDifficultyWorldsend {
		if !song.IsWorldsend {
			return nil, ErrChartNotFound
		}
		worldsend, err := u.worldsendChartRepo.FindByDisplayID(ctx, u.defaultExecutor, song.DisplayID)
		if err != nil {
			if errors.Is(err, repository.ErrSongNotFound) {
				return nil, ErrChartNotFound
			}
			return nil, err
		}
		return &chartEntry{
			id:  worldsend.Chart.ID,
			key: info.StatsDifficultyWorldsend,
		}, nil
	}

	// 通常楽曲の場合、WORLD'S ENDリクエストは無効
	if song.IsWorldsend {
		return nil, ErrChartNotFound
	}

	masters := u.masterCache.SongMasters()
	if masters == nil {
		return nil, fmt.Errorf("master cache is not initialized")
	}

	// 該当する難易度の譜面を検索
	// DifficultyNamesByIDを逆引きして、難易度名に一致する譜面を探す
	for _, chart := range song.Charts {
		name, ok := masters.DifficultyNamesByID[chart.DifficultyID]
		if ok && name == difficultyName {
			return &chartEntry{
				id:  chart.ID,
				key: difficultyName,
			}, nil
		}
	}

	// 指定された難易度の譜面が存在しない
	// 難易度名が有効かどうかをチェック
	validDifficulty := false
	for _, name := range masters.DifficultyNamesByID {
		if name == difficultyName {
			validDifficulty = true
			break
		}
	}
	if !validDifficulty {
		return nil, ErrInvalidDifficulty
	}

	return nil, ErrChartNotFound
}
