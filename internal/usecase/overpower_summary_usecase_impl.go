package usecase

import (
	"context"
	"database/sql"
	"math"
	"strconv"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
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
			return nil, ErrPlayerNotLinked
		}
		return nil, err
	}
	if player == nil {
		return nil, ErrPlayerNotLinked
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

	overall := overpowerSummaryAccumulator{}
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

			if difficultyName, ok := masters.DifficultyNamesByID[chart.DifficultyID]; ok {
				acc := difficulties[difficultyName]
				acc.currentOP += chartCurrentOP
				acc.maxOP += chartMaxOP
				acc.targetCount++
				if played {
					acc.playedCount++
				}
				difficulties[difficultyName] = acc
			}

			if levelName, ok := classifyOverpowerSummaryLevel(float64(chart.Const)); ok {
				acc := levels[levelName]
				acc.currentOP += chartCurrentOP
				acc.maxOP += chartMaxOP
				acc.targetCount++
				if played {
					acc.playedCount++
				}
				levels[levelName] = acc
			}
		}

		overall.currentOP += songCurrentOP
		overall.maxOP += songMaxOP
		overall.targetCount++
		if songPlayed {
			overall.playedCount++
		}

		if song.GenreID != nil {
			if genreName, ok := masters.GenreNamesByID[*song.GenreID]; ok {
				acc := genres[genreName]
				acc.currentOP += songCurrentOP
				acc.maxOP += songMaxOP
				acc.targetCount++
				if songPlayed {
					acc.playedCount++
				}
				genres[genreName] = acc
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

func newAccumulatorMap(keys []string) map[string]overpowerSummaryAccumulator {
	result := make(map[string]overpowerSummaryAccumulator, len(keys))
	for _, key := range keys {
		result[key] = overpowerSummaryAccumulator{}
	}
	return result
}

func toOverpowerSummaryItems(items map[string]overpowerSummaryAccumulator, order []string) map[string]dtoapiinternal.OverpowerSummaryItem {
	result := make(map[string]dtoapiinternal.OverpowerSummaryItem, len(order))
	for _, key := range order {
		result[key] = toOverpowerSummaryItem(items[key])
	}
	return result
}

func toOverpowerSummaryItem(acc overpowerSummaryAccumulator) dtoapiinternal.OverpowerSummaryItem {
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
	if level < 10 {
		return "", false
	}

	if bucket-float64(level) >= 0.5 {
		return strconv.Itoa(level) + "+", true
	}

	return strconv.Itoa(level), true
}
