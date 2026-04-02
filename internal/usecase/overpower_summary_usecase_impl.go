package usecase

import (
	"context"
	"database/sql"
	"log/slog"
	"math"
	"strconv"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/service"
	dtoapiinternal "github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
)

var overpowerSummaryGenreOrder = []string{
	"POPS & ANIME",
	"niconico",
	"東方Project",
	"VARIETY",
	"イロドリミドリ",
	"ゲキマイ",
	"ORIGINAL",
}

var overpowerSummaryDifficultyOrder = []string{
	"BASIC",
	"ADVANCED",
	"EXPERT",
	"MASTER",
	"ULTIMA",
}

var overpowerSummaryLevelOrder = []string{
	"10",
	"10+",
	"11",
	"11+",
	"12",
	"12+",
	"13",
	"13+",
	"14",
	"14+",
	"15",
	"15+",
}

const (
	overpowerSummaryTheoreticalScore  = 1010000
	overpowerSummaryAllJusticeComboID = 3
)

type overpowerSummaryUsecase struct {
	playerRepo       repository.PlayerRepository
	playerRecordRepo repository.PlayerRecordRepository
	songRepo         repository.SongRepository
	masterProvider   repository.SongMasterProvider
	defaultExecutor  repository.Executor
}

// NewOverpowerSummaryUsecase は OVER POWER 集計ユースケースを生成します。
func NewOverpowerSummaryUsecase(
	playerRepo repository.PlayerRepository,
	playerRecordRepo repository.PlayerRecordRepository,
	songRepo repository.SongRepository,
	masterProvider repository.SongMasterProvider,
	defaultExecutor repository.Executor,
) OverpowerSummaryUsecase {
	return &overpowerSummaryUsecase{
		playerRepo:       playerRepo,
		playerRecordRepo: playerRecordRepo,
		songRepo:         songRepo,
		masterProvider:   masterProvider,
		defaultExecutor:  defaultExecutor,
	}
}

type overpowerSummaryAccumulator struct {
	currentOP   float64
	maxOP       float64
	targetCount int
	playedCount int
}

func (u *overpowerSummaryUsecase) Get(ctx context.Context, user *entity.User) (*dtoapiinternal.OverpowerSummaryResponse, error) {
	if user == nil || user.PlayerID == nil {
		return nil, ErrPlayerNotLinked
	}

	player, err := u.playerRepo.FindByID(ctx, u.defaultExecutor, *user.PlayerID)
	if err != nil {
		if err == sql.ErrNoRows {
			// player_id は存在するが players 実体がないデータ不整合のため ErrPlayerNotFound を返す
			return nil, ErrPlayerNotFound
		}
		return nil, err
	}
	if player == nil {
		// 同上: nil は players 実体欠損を意味する
		return nil, ErrPlayerNotFound
	}

	records, err := u.playerRecordRepo.FindByPlayerID(ctx, u.defaultExecutor, player.ID)
	if err != nil {
		return nil, err
	}

	songs, err := u.songRepo.FindAllExcludingWorldsend(ctx, u.defaultExecutor, false)
	if err != nil {
		return nil, err
	}

	if u.masterProvider == nil {
		return nil, ErrInternalError
	}

	masters := u.masterProvider.SongMasters()
	if masters == nil {
		return nil, ErrInternalError
	}

	recordByChartID := make(map[int]*entity.PlayerRecord, len(records))
	for _, record := range records {
		recordByChartID[record.ChartID] = record
	}

	overall := &overpowerSummaryAccumulator{}
	genres := newAccumulatorMap(overpowerSummaryGenreOrder)
	difficulties := newAccumulatorMap(overpowerSummaryDifficultyOrder)
	levels := newAccumulatorMap(overpowerSummaryLevelOrder)

	for _, song := range songs {
		songCurrentOP := 0.0
		songMaxOP := 0.0
		songPlayed := false

		for _, chart := range song.Charts {
			chartMaxOP := service.CalcSingleOverpower(overpowerSummaryTheoreticalScore, float64(chart.Const), overpowerSummaryAllJusticeComboID)
			chartCurrentOP := 0.0
			played := false

			if record, ok := recordByChartID[chart.ID]; ok {
				chartCurrentOP = service.CalcSingleOverpower(uint32(record.Score), float64(chart.Const), record.ComboLampID)
				played = true
			}

			songCurrentOP = max(songCurrentOP, chartCurrentOP)
			songMaxOP = max(songMaxOP, chartMaxOP)
			songPlayed = songPlayed || played

			if acc := difficultyAccumulator(difficulties, masters, chart.DifficultyID); acc != nil {
				accumulateChartStats(acc, chartCurrentOP, chartMaxOP, played)
			}

			if levelName, ok := classifyOverpowerSummaryLevel(float64(chart.Const)); ok {
				if acc := levels[levelName]; acc != nil {
					accumulateChartStats(acc, chartCurrentOP, chartMaxOP, played)
				}
			}
		}

		accumulateSongStats(overall, songCurrentOP, songMaxOP, songPlayed)

		if song.GenreID != nil {
			if acc := genreAccumulator(genres, masters, *song.GenreID, song.DisplayID); acc != nil {
				accumulateSongStats(acc, songCurrentOP, songMaxOP, songPlayed)
			}
		}
	}

	return &dtoapiinternal.OverpowerSummaryResponse{
		UpdatedAt:    player.UpdatedAt,
		Overall:      toOverpowerSummaryItem(overall),
		Genres:       toOverpowerSummaryItems(genres, overpowerSummaryGenreOrder),
		Difficulties: toOverpowerSummaryItems(difficulties, overpowerSummaryDifficultyOrder),
		Levels:       toOverpowerSummaryItems(levels, overpowerSummaryLevelOrder),
	}, nil
}

func accumulateChartStats(acc *overpowerSummaryAccumulator, chartCurrentOP, chartMaxOP float64, played bool) {
	acc.currentOP += chartCurrentOP
	acc.maxOP += chartMaxOP
	acc.targetCount++
	if played {
		acc.playedCount++
	}
}

func accumulateSongStats(acc *overpowerSummaryAccumulator, songCurrentOP, songMaxOP float64, songPlayed bool) {
	acc.currentOP += songCurrentOP
	acc.maxOP += songMaxOP
	acc.targetCount++
	if songPlayed {
		acc.playedCount++
	}
}

func newAccumulatorMap(keys []string) map[string]*overpowerSummaryAccumulator {
	result := make(map[string]*overpowerSummaryAccumulator, len(keys))
	for _, key := range keys {
		result[key] = &overpowerSummaryAccumulator{}
	}
	return result
}

func difficultyAccumulator(accumulators map[string]*overpowerSummaryAccumulator, masters *masterdata.SongMasters, difficultyID int) *overpowerSummaryAccumulator {
	difficultyName, ok := masters.DifficultyNamesByID[difficultyID]
	if !ok {
		slog.Warn("difficulty name not found for overpower summary", "difficulty_id", difficultyID)
		return nil
	}

	return findAccumulatorByName(
		accumulators,
		difficultyName,
		"difficulty accumulator not found for overpower summary",
		"difficulty_id",
		difficultyID,
	)
}

func genreAccumulator(accumulators map[string]*overpowerSummaryAccumulator, masters *masterdata.SongMasters, genreID int, songDisplayID string) *overpowerSummaryAccumulator {
	genreName, ok := masters.GenreNamesByID[genreID]
	if !ok {
		slog.Warn("genre name not found for overpower summary", "genre_id", genreID, "song_display_id", songDisplayID)
		return nil
	}

	return findAccumulatorByName(
		accumulators,
		genreName,
		"genre accumulator not found for overpower summary",
		"genre_id",
		genreID,
		"song_display_id",
		songDisplayID,
	)
}

func findAccumulatorByName(accumulators map[string]*overpowerSummaryAccumulator, name string, message string, args ...any) *overpowerSummaryAccumulator {
	acc, ok := accumulators[name]
	if !ok {
		logArgs := append([]any{"name", name}, args...)
		slog.Warn(message, logArgs...)
		return nil
	}

	return acc
}

func toOverpowerSummaryItems(items map[string]*overpowerSummaryAccumulator, order []string) map[string]dtoapiinternal.OverpowerSummaryItem {
	result := make(map[string]dtoapiinternal.OverpowerSummaryItem, len(order))
	for _, key := range order {
		result[key] = toOverpowerSummaryItem(items[key])
	}
	return result
}

func toOverpowerSummaryItem(acc *overpowerSummaryAccumulator) dtoapiinternal.OverpowerSummaryItem {
	if acc == nil {
		return dtoapiinternal.OverpowerSummaryItem{}
	}

	percent := 0.0
	if acc.maxOP > 0 {
		percent = acc.currentOP / acc.maxOP * 100
	}

	return dtoapiinternal.OverpowerSummaryItem{
		CurrentOP:   acc.currentOP,
		MaxOP:       acc.maxOP,
		Percent:     percent,
		TargetCount: acc.targetCount,
		PlayedCount: acc.playedCount,
	}
}

func classifyOverpowerSummaryLevel(chartConst float64) (string, bool) {
	if chartConst < 10 {
		return "", false
	}

	bucket := math.Floor(chartConst*2) / 2
	if bucket >= 15.5 {
		return "15+", true
	}

	level := int(bucket)
	if bucket-float64(level) >= 0.5 {
		return strconv.Itoa(level) + "+", true
	}

	return strconv.Itoa(level), true
}
