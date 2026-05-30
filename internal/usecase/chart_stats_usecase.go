package usecase

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/info"
)

// ChartStatsUsecase は譜面統計の取得ユースケースを提供します。
type ChartStatsUsecase interface {
	// GetSongStatsByDisplayID は指定されたDisplayIDの譜面統計を取得します。
	// requesterAccountTypeIDがnilまたはEDITOR権限を満たさない場合、削除済み楽曲はErrSongNotFoundを返します。
	GetSongStatsByDisplayID(ctx context.Context, displayID string, requesterAccountTypeID *int) (*entity.SongChartStats, error)

	// GetChartStatsByDisplayIDAndDifficulty は指定されたDisplayIDと難易度の譜面統計を取得します。
	// difficultyNameは大文字の難易度名（"BASIC", "ADVANCED", "EXPERT", "MASTER", "ULTIMA"）
	// または "WORLD'S END" である必要があります。
	// requesterAccountTypeIDがnilまたはEDITOR権限を満たさない場合、削除済み楽曲はErrSongNotFoundを返します。
	GetChartStatsByDisplayIDAndDifficulty(ctx context.Context, displayID, difficultyName string, requesterAccountTypeID *int) (*entity.SingleChartStats, error)
}

type chartStatsUsecaseImpl struct {
	songRepo           repository.SongRepository
	worldsendChartRepo repository.WorldsendChartRepository
	statsRepo          repository.ChartStatsRepository
	masterCache        repository.SongMasterProvider
	masterProvider     repository.ChartStatsMasterProvider
	defaultExecutor    repository.Executor
	statsExecutor      repository.Executor
}

// NewChartStatsUsecase は ChartStatsUsecase の実装を生成します。
func NewChartStatsUsecase(
	songRepo repository.SongRepository,
	worldsendChartRepo repository.WorldsendChartRepository,
	statsRepo repository.ChartStatsRepository,
	masterCache repository.SongMasterProvider,
	masterProvider repository.ChartStatsMasterProvider,
	defaultExecutor repository.Executor,
	statsExecutor repository.Executor,
) ChartStatsUsecase {
	return &chartStatsUsecaseImpl{
		songRepo:           songRepo,
		worldsendChartRepo: worldsendChartRepo,
		statsRepo:          statsRepo,
		masterCache:        masterCache,
		masterProvider:     masterProvider,
		defaultExecutor:    defaultExecutor,
		statsExecutor:      statsExecutor,
	}
}

type chartEntry struct {
	id  int
	key string
}

// GetSongStatsByDisplayID は指定されたDisplayIDの譜面統計を取得します。
// requesterAccountTypeIDがnilまたはEDITOR権限を満たさない場合、削除済み楽曲はErrSongNotFoundを返します。
func (u *chartStatsUsecaseImpl) GetSongStatsByDisplayID(ctx context.Context, displayID string, requesterAccountTypeID *int) (*entity.SongChartStats, error) {
	song, err := u.songRepo.FindByDisplayID(ctx, u.defaultExecutor, displayID)
	if err != nil {
		return nil, err
	}

	// 削除済み楽曲は権限に応じて公開可否を制御する。
	if !song.IsActive() {
		// EDITOR以上の権限を持たない場合は404を返す。
		if requesterAccountTypeID == nil || !info.HasRole(*requesterAccountTypeID, info.AccountTypeEditor) {
			return nil, repository.ErrSongNotFound
		}
	}

	ratingBands := u.masterProvider.RatingBands()

	entries, err := u.buildChartEntries(ctx, song)
	if err != nil {
		return nil, err
	}

	chartIDs := make([]int, 0, len(entries))
	for _, entry := range entries {
		chartIDs = append(chartIDs, entry.id)
	}

	statsRows, err := u.findStatsRows(ctx, song.IsWorldsend, chartIDs)
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
		slices.SortFunc(stats, func(a, b *entity.ChartStatsByRatingBand) int {
			return cmp.Compare(bandOrder[a.RatingBandID], bandOrder[b.RatingBandID])
		})
		charts[entry.key] = stats
	}

	return &entity.SongChartStats{SongID: song.DisplayID, Charts: charts}, nil
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

		return []chartEntry{{id: worldsend.Chart.ID, key: info.StatsDifficultyWorldsend}}, nil
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
		entries = append(entries, chartEntry{id: chart.ID, key: name})
	}

	return entries, nil
}

func (u *chartStatsUsecaseImpl) findStatsRows(ctx context.Context, isWorldsend bool, chartIDs []int) ([]*entity.ChartStatsByRatingBand, error) {
	if isWorldsend {
		return u.statsRepo.FindWorldsendChartStatsByChartIDs(ctx, u.statsExecutor, chartIDs)
	}

	return u.statsRepo.FindChartStatsByChartIDs(ctx, u.statsExecutor, chartIDs)
}

// GetChartStatsByDisplayIDAndDifficulty は指定されたDisplayIDと難易度の譜面統計を取得します。
// requesterAccountTypeIDがnilまたはEDITOR権限を満たさない場合、削除済み楽曲はErrSongNotFoundを返します。
func (u *chartStatsUsecaseImpl) GetChartStatsByDisplayIDAndDifficulty(ctx context.Context, displayID, difficultyName string, requesterAccountTypeID *int) (*entity.SingleChartStats, error) {
	// WORLD'S ENDはsongRepo.FindByDisplayIDで取得できないため、専用フローで処理する。
	if difficultyName == info.StatsDifficultyWorldsend {
		return u.getWorldsendSingleChartStats(ctx, displayID, requesterAccountTypeID)
	}

	song, err := u.songRepo.FindByDisplayID(ctx, u.defaultExecutor, displayID)
	if err != nil {
		return nil, err
	}

	if !song.IsActive() {
		if requesterAccountTypeID == nil || !info.HasRole(*requesterAccountTypeID, info.AccountTypeEditor) {
			return nil, repository.ErrSongNotFound
		}
	}

	ratingBands := u.masterProvider.RatingBands()

	entry, err := u.findChartEntryByDifficulty(song, difficultyName)
	if err != nil {
		return nil, err
	}

	statsRows, err := u.statsRepo.FindChartStatsByChartIDs(ctx, u.statsExecutor, []int{entry.id})
	if err != nil {
		return nil, err
	}

	bandOrder := make(map[int]int, len(ratingBands))
	for _, band := range ratingBands {
		bandOrder[band.ID] = band.SortOrder
	}

	slices.SortFunc(statsRows, func(a, b *entity.ChartStatsByRatingBand) int {
		return cmp.Compare(bandOrder[a.RatingBandID], bandOrder[b.RatingBandID])
	})

	return &entity.SingleChartStats{SongID: song.DisplayID, Difficulty: entry.key, Stats: statsRows}, nil
}

// getWorldsendSingleChartStats はWORLD'S END譜面の統計をworldsendChartRepo経由で取得します。
// songRepo.FindByDisplayIDがWORLD'S END楽曲を除外するため、専用メソッドで処理します。
func (u *chartStatsUsecaseImpl) getWorldsendSingleChartStats(ctx context.Context, displayID string, requesterAccountTypeID *int) (*entity.SingleChartStats, error) {
	worldsend, err := u.worldsendChartRepo.FindByDisplayID(ctx, u.defaultExecutor, displayID)
	if err != nil {
		return nil, err
	}

	// 削除済み楽曲は権限に応じて公開可否を制御する。
	if !worldsend.Song.IsActive() {
		if requesterAccountTypeID == nil || !info.HasRole(*requesterAccountTypeID, info.AccountTypeEditor) {
			return nil, repository.ErrSongNotFound
		}
	}

	ratingBands := u.masterProvider.RatingBands()

	statsRows, err := u.statsRepo.FindWorldsendChartStatsByChartIDs(ctx, u.statsExecutor, []int{worldsend.Chart.ID})
	if err != nil {
		return nil, err
	}

	bandOrder := make(map[int]int, len(ratingBands))
	for _, band := range ratingBands {
		bandOrder[band.ID] = band.SortOrder
	}

	slices.SortFunc(statsRows, func(a, b *entity.ChartStatsByRatingBand) int {
		return cmp.Compare(bandOrder[a.RatingBandID], bandOrder[b.RatingBandID])
	})

	return &entity.SingleChartStats{SongID: worldsend.Song.DisplayID, Difficulty: info.StatsDifficultyWorldsend, Stats: statsRows}, nil
}

// findChartEntryByDifficulty は通常難易度（WORLD'S END以外）の譜面エントリを検索します。
// WORLD'S ENDはGetChartStatsByDisplayIDAndDifficultyで事前に処理されるため、ここには到達しません。
func (u *chartStatsUsecaseImpl) findChartEntryByDifficulty(song *entity.Song, difficultyName string) (*chartEntry, error) {
	masters := u.masterCache.SongMasters()
	if masters == nil {
		return nil, fmt.Errorf("master cache is not initialized")
	}

	for _, chart := range song.Charts {
		name, ok := masters.DifficultyNamesByID[chart.DifficultyID]
		if ok && name == difficultyName {
			return &chartEntry{id: chart.ID, key: difficultyName}, nil
		}
	}

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
