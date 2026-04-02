package usecase

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/service"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/playername"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/score"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubOverpowerSummaryPlayerRepository struct {
	player *entity.Player
	err    error
}

func (s *stubOverpowerSummaryPlayerRepository) FindByID(ctx context.Context, exec repository.Executor, id int) (*entity.Player, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.player, nil
}

func (s *stubOverpowerSummaryPlayerRepository) FindByIDWithHonors(ctx context.Context, exec repository.Executor, id int) (*repository.PlayerWithHonors, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &repository.PlayerWithHonors{Player: s.player}, nil
}

func (s *stubOverpowerSummaryPlayerRepository) FindByUserID(ctx context.Context, exec repository.Executor, userID int) (*entity.Player, error) {
	return nil, nil
}

func (s *stubOverpowerSummaryPlayerRepository) FindHonorsByPlayerID(ctx context.Context, exec repository.Executor, playerID int) ([]*repository.PlayerHonor, error) {
	return nil, nil
}

func (s *stubOverpowerSummaryPlayerRepository) UpdateCalculatedRatings(ctx context.Context, exec repository.Executor, playerID int, calculatedRating, bestAverage, newAverage float64) error {
	return nil
}

func (s *stubOverpowerSummaryPlayerRepository) Save(ctx context.Context, exec repository.Executor, player *entity.Player) error {
	return nil
}

func (s *stubOverpowerSummaryPlayerRepository) DeleteByUserID(ctx context.Context, exec repository.Executor, userID int) error {
	return nil
}

type stubOverpowerSummarySongMasterProvider struct {
	masters *masterdata.SongMasters
}

func (s *stubOverpowerSummarySongMasterProvider) SongMasters() *masterdata.SongMasters {
	return s.masters
}

func TestOverpowerSummaryUsecaseGet_楽曲単位譜面単位レベル別を集計できる(t *testing.T) {
	// Given
	playerID := 99
	updatedAt := time.Date(2026, 3, 25, 12, 34, 56, 0, time.UTC)
	playerName, err := playername.NewPlayerName("テスト")
	require.NoError(t, err)

	user := &entity.User{ID: 1, PlayerID: &playerID}
	player := &entity.Player{ID: playerID, Name: playerName, UpdatedAt: updatedAt}

	const10_4, err := chartconstant.NewChartConstant(10.4)
	require.NoError(t, err)
	const14_9, err := chartconstant.NewChartConstant(14.9)
	require.NoError(t, err)
	const9_9, err := chartconstant.NewChartConstant(9.9)
	require.NoError(t, err)
	const15_7, err := chartconstant.NewChartConstant(15.7)
	require.NoError(t, err)

	scoreAJC, err := score.NewScore(1010000)
	require.NoError(t, err)
	scoreA, err := score.NewScore(900000)
	require.NoError(t, err)

	songs := []*entity.Song{
		{
			ID:        1,
			DisplayID: "song-1",
			GenreID:   intPointer(1),
			Charts: []*entity.Chart{
				{ID: 101, SongID: 1, DifficultyID: 1, Const: const10_4},
				{ID: 102, SongID: 1, DifficultyID: 4, Const: const14_9},
			},
		},
		{
			ID:        2,
			DisplayID: "song-2",
			GenreID:   intPointer(2),
			Charts: []*entity.Chart{
				{ID: 201, SongID: 2, DifficultyID: 3, Const: const9_9},
				{ID: 202, SongID: 2, DifficultyID: 5, Const: const15_7},
			},
		},
	}

	records := []*entity.PlayerRecord{
		{PlayerID: playerID, ChartID: 101, Score: scoreAJC, ComboLampID: 3},
		{PlayerID: playerID, ChartID: 102, Score: scoreA, ComboLampID: 1},
	}

	masters := &masterdata.SongMasters{
		CommonMasters: masterdata.CommonMasters{
			DifficultyNamesByID: map[int]string{
				1: "BASIC",
				3: "EXPERT",
				4: "MASTER",
				5: "ULTIMA",
			},
		},
		GenreNamesByID: map[int]string{
			1: "POPS & ANIME",
			2: "niconico",
		},
	}

	uc := NewOverpowerSummaryUsecase(
		&stubOverpowerSummaryPlayerRepository{player: player},
		&stubPlayerRecordRepository{records: records},
		&stubSongRepository{songs: songs},
		&stubOverpowerSummarySongMasterProvider{masters: masters},
		nil,
	)

	song1Current := service.CalcSingleOverpower(uint32(scoreAJC), 10.4, 3)
	song1MasterCurrent := service.CalcSingleOverpower(uint32(scoreA), 14.9, 1)
	song1Max := service.CalcSingleOverpower(1010000, 14.9, 3)
	song2ExpertMax := service.CalcSingleOverpower(1010000, 9.9, 3)
	song2UltimaMax := service.CalcSingleOverpower(1010000, 15.7, 3)

	// When
	resp, err := uc.Get(context.Background(), user)

	// Then
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, updatedAt.Equal(resp.UpdatedAt))

	assert.InDelta(t, song1Current, resp.Overall.CurrentOP, 0.0001)
	assert.InDelta(t, song1Max+song2UltimaMax, resp.Overall.MaxOP, 0.0001)
	assert.Equal(t, 2, resp.Overall.TargetCount)
	assert.Equal(t, 1, resp.Overall.PlayedCount)

	assert.InDelta(t, song1Current, resp.Genres["POPS & ANIME"].CurrentOP, 0.0001)
	assert.InDelta(t, song1Max, resp.Genres["POPS & ANIME"].MaxOP, 0.0001)
	assert.Equal(t, 1, resp.Genres["POPS & ANIME"].PlayedCount)
	assert.InDelta(t, 0.0, resp.Genres["niconico"].CurrentOP, 0.0001)
	assert.InDelta(t, song2UltimaMax, resp.Genres["niconico"].MaxOP, 0.0001)
	assert.Equal(t, 0, resp.Genres["niconico"].PlayedCount)

	assert.InDelta(t, song1Current, resp.Difficulties["BASIC"].CurrentOP, 0.0001)
	assert.InDelta(t, service.CalcSingleOverpower(1010000, 10.4, 3), resp.Difficulties["BASIC"].MaxOP, 0.0001)
	assert.Equal(t, 1, resp.Difficulties["BASIC"].PlayedCount)
	assert.InDelta(t, song1MasterCurrent, resp.Difficulties["MASTER"].CurrentOP, 0.0001)
	assert.InDelta(t, song1Max, resp.Difficulties["MASTER"].MaxOP, 0.0001)
	assert.Equal(t, 1, resp.Difficulties["MASTER"].PlayedCount)
	assert.InDelta(t, 0.0, resp.Difficulties["EXPERT"].CurrentOP, 0.0001)
	assert.InDelta(t, song2ExpertMax, resp.Difficulties["EXPERT"].MaxOP, 0.0001)
	assert.Equal(t, 0, resp.Difficulties["EXPERT"].PlayedCount)
	assert.InDelta(t, 0.0, resp.Difficulties["ULTIMA"].CurrentOP, 0.0001)
	assert.InDelta(t, song2UltimaMax, resp.Difficulties["ULTIMA"].MaxOP, 0.0001)

	assert.InDelta(t, song1Current, resp.Levels["10"].CurrentOP, 0.0001)
	assert.Equal(t, 1, resp.Levels["10"].PlayedCount)
	assert.InDelta(t, song1Max, resp.Levels["14+"].MaxOP, 0.0001)
	assert.InDelta(t, song1MasterCurrent, resp.Levels["14+"].CurrentOP, 0.0001)
	assert.InDelta(t, song2UltimaMax, resp.Levels["15+"].MaxOP, 0.0001)
	assert.Equal(t, 0, resp.Levels["15+"].PlayedCount)
	assert.Equal(t, 0, resp.Levels["10+"].TargetCount)
	assert.Equal(t, 0, resp.Levels["11"].TargetCount)
}

func TestOverpowerSummaryUsecaseGet_プレイヤー未連携ならErrPlayerNotLinkedを返す(t *testing.T) {
	uc := NewOverpowerSummaryUsecase(
		&stubOverpowerSummaryPlayerRepository{},
		&stubPlayerRecordRepository{},
		&stubSongRepository{},
		&stubOverpowerSummarySongMasterProvider{},
		nil,
	)

	resp, err := uc.Get(context.Background(), &entity.User{ID: 1})

	require.ErrorIs(t, err, ErrPlayerNotLinked)
	assert.Nil(t, resp)
}

func TestOverpowerSummaryUsecaseGet_紐付け先プレイヤー実体が存在しないならErrPlayerNotFoundを返す(t *testing.T) {
	playerID := 99

	uc := NewOverpowerSummaryUsecase(
		&stubOverpowerSummaryPlayerRepository{err: sql.ErrNoRows},
		&stubPlayerRecordRepository{},
		&stubSongRepository{},
		&stubOverpowerSummarySongMasterProvider{},
		nil,
	)

	resp, err := uc.Get(context.Background(), &entity.User{ID: 1, PlayerID: &playerID})

	require.ErrorIs(t, err, ErrPlayerNotFound)
	assert.Nil(t, resp)
}
