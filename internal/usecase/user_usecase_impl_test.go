package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo/master"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/notes"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/playername"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/score"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/username"
)

type stubUserRepository struct {
	user            *entity.User
	usersWithPlayer []entity.UserWithPlayer
	err             error
	saveErr         error
	savedUser       *entity.User
}

func (s *stubUserRepository) FindByID(ctx context.Context, exec repository.Executor, id int) (*entity.User, error) {
	return nil, errors.New("not implemented")
}

func (s *stubUserRepository) FindByUsername(ctx context.Context, exec repository.Executor, username string) (*entity.User, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.user, nil
}

func (s *stubUserRepository) FindAllWithPlayer(ctx context.Context, exec repository.Executor, limit int, offset int, searchName string) ([]entity.UserWithPlayer, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.usersWithPlayer, nil
}

func (s *stubUserRepository) FindAllWithPlayerForAdmin(ctx context.Context, exec repository.Executor, limit int, offset int, searchName string) ([]entity.UserWithPlayer, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.usersWithPlayer, nil
}

func (s *stubUserRepository) Create(ctx context.Context, exec repository.Executor, user *entity.User) error {
	return errors.New("not implemented")
}

func (s *stubUserRepository) Save(ctx context.Context, exec repository.Executor, user *entity.User) error {
	if s.saveErr != nil {
		return s.saveErr
	}
	s.savedUser = user
	return nil
}

type stubPlayerRecordRepository struct {
	records         []*entity.PlayerRecord
	ratingRecords   []*entity.PlayerRecord
	lastScoreUpdate *time.Time
	err             error
}

func (s *stubPlayerRecordRepository) FindByPlayerID(ctx context.Context, exec repository.Executor, playerID int) ([]*entity.PlayerRecord, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.records, nil
}

func (s *stubPlayerRecordRepository) FindByPlayerIDForRating(ctx context.Context, exec repository.Executor, playerID int) ([]*entity.PlayerRecord, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.ratingRecords != nil {
		return s.ratingRecords, nil
	}
	return s.records, nil
}

func (s *stubPlayerRecordRepository) GetLastScoreUpdate(ctx context.Context, exec repository.Executor, playerID int) (*time.Time, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.lastScoreUpdate, nil
}

type stubPlayerRepository struct {
	playerWithHonors *repository.PlayerWithHonors
	err              error
}

func (s *stubPlayerRepository) FindByID(ctx context.Context, exec repository.Executor, id int) (*entity.Player, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.playerWithHonors == nil {
		return nil, nil
	}
	return s.playerWithHonors.Player, nil
}

func (s *stubPlayerRepository) FindByIDWithHonors(ctx context.Context, exec repository.Executor, id int) (*repository.PlayerWithHonors, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.playerWithHonors, nil
}

func (s *stubPlayerRepository) FindByUserID(ctx context.Context, exec repository.Executor, userID int) (*entity.Player, error) {
	return nil, errors.New("not implemented")
}

func (s *stubPlayerRepository) FindHonorsByPlayerID(ctx context.Context, exec repository.Executor, playerID int) ([]*repository.PlayerHonor, error) {
	return nil, errors.New("not implemented")
}

func (s *stubPlayerRepository) UpdateCalculatedRatings(ctx context.Context, exec repository.Executor, playerID int, calculatedRating, bestAverage, newAverage float64) error {
	return errors.New("not implemented")
}

func (s *stubPlayerRepository) Save(ctx context.Context, exec repository.Executor, player *entity.Player) error {
	return errors.New("not implemented")
}

func (s *stubPlayerRepository) DeleteByUserID(ctx context.Context, exec repository.Executor, userID int) error {
	return errors.New("not implemented")
}

type stubWorldsendRecordRepository struct {
	records []*entity.PlayerWorldsendRecord
	err     error
}

func (s *stubWorldsendRecordRepository) FindByPlayerID(ctx context.Context, exec repository.Executor, playerID int) ([]*entity.PlayerWorldsendRecord, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.records, nil
}

type stubSongRepository struct {
	songs []*entity.Song
	err   error
}

func (s *stubSongRepository) FindAllExcludingWorldsend(ctx context.Context, exec repository.Executor, includeDeleted bool) ([]*entity.Song, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.songs, nil
}

func (s *stubSongRepository) FindByDisplayID(ctx context.Context, exec repository.Executor, displayID string) (*entity.Song, error) {
	return nil, errors.New("not implemented")
}

func (s *stubSongRepository) FindByDisplayIDs(ctx context.Context, exec repository.Executor, displayIDs []string) ([]*entity.Song, error) {
	return nil, errors.New("not implemented")
}

func (s *stubSongRepository) Save(ctx context.Context, exec repository.Executor, song *entity.Song) error {
	return errors.New("not implemented")
}

func (s *stubSongRepository) UpdateSongs(ctx context.Context, exec repository.Executor, songs []*entity.Song) error {
	return errors.New("not implemented")
}

type stubSongMasterProvider struct {
	masters *masterdata.SongMasters
}

func (s *stubSongMasterProvider) SongMasters() *masterdata.SongMasters {
	return s.masters
}

type stubWorldsendChartRepository struct {
	records []*repository.WorldsendSongWithChart
	err     error
}

func (s *stubWorldsendChartRepository) FindAll(ctx context.Context, exec repository.Executor, includeDeleted bool) ([]*repository.WorldsendSongWithChart, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.records, nil
}

func (s *stubWorldsendChartRepository) FindByDisplayID(ctx context.Context, exec repository.Executor, displayID string) (*repository.WorldsendSongWithChart, error) {
	return nil, errors.New("not implemented")
}

func (s *stubWorldsendChartRepository) SaveSong(ctx context.Context, exec repository.Executor, song *entity.Song) error {
	return errors.New("not implemented")
}

func (s *stubWorldsendChartRepository) UpdateSongs(ctx context.Context, exec repository.Executor, updates []*repository.WorldsendUpdate) error {
	return errors.New("not implemented")
}

func TestUserService_GetUserProfileWithRecords_UserNotFound(t *testing.T) {
	service := NewUserService(nil, &stubUserRepository{err: repository.ErrUserNotFound}, &stubPlayerRepository{}, &stubPlayerRecordRepository{}, nil, nil, nil, nil)

	_, err := service.GetUserProfileWithRecords(context.Background(), "missing", nil, false)
	require.ErrorIs(t, err, ErrUserNotFound)
}

func TestUserService_GetUserProfileWithRecords_PlayerNotLinked(t *testing.T) {
	user := &entity.User{ID: 1}
	service := NewUserService(nil, &stubUserRepository{user: user}, &stubPlayerRepository{}, &stubPlayerRecordRepository{}, nil, nil, nil, nil)

	_, err := service.GetUserProfileWithRecords(context.Background(), "no-player", nil, false)
	require.ErrorIs(t, err, ErrPlayerNotLinked)
}

func TestUserService_GetUserProfileWithRecords_PrivateSelf(t *testing.T) {
	now := time.Now()
	un, _ := username.NewUserName("selfuser")
	user := &entity.User{
		ID:        1,
		Username:  un,
		PlayerID:  intPointer(1),
		IsPrivate: true,
	}
	player := &entity.Player{
		ID:        1,
		Name:      playername.MustNewPlayerName("セルフプレイヤー"),
		Level:     1,
		UpdatedAt: now,
	}
	service := NewUserService(nil, &stubUserRepository{user: user}, &stubPlayerRepository{playerWithHonors: &repository.PlayerWithHonors{Player: player, Honors: []*repository.PlayerHonor{}}}, &stubPlayerRecordRepository{}, nil, nil, nil, nil)

	_, err := service.GetUserProfileWithRecords(context.Background(), "selfuser", &entity.User{ID: 1}, false)
	require.NoError(t, err)
}

func TestUserService_GetUserProfileWithRecords_PlayerRepositoryNoRows(t *testing.T) {
	un, _ := username.NewUserName("tester")
	user := &entity.User{
		ID:       1,
		Username: un,
		PlayerID: intPointer(1),
	}
	service := NewUserService(nil, &stubUserRepository{user: user}, &stubPlayerRepository{err: repository.ErrPlayerNotFound}, &stubPlayerRecordRepository{}, nil, nil, nil, nil)

	_, err := service.GetUserProfileWithRecords(context.Background(), "tester", nil, false)
	require.ErrorIs(t, err, ErrPlayerNotLinked)
}

func TestUserService_GetUserProfileWithRecords_Success(t *testing.T) {
	now := time.Now()
	notesValue := notes.Notes(500)
	chartConst, _ := chartconstant.NewChartConstant(12.4)

	score1, _ := score.NewScore(1000000)
	score2, _ := score.NewScore(950000)

	records := []*entity.PlayerRecord{
		{
			PlayerID:    1,
			ChartID:     101,
			Score:       score1,
			ClearLampID: 1,
			ComboLampID: 1,
			FullChainID: 1,
			SlotID:      1,
			SlotOrder:   intPointer(1),
			UpdatedAt:   now,
			Chart: &entity.Chart{
				ID:             101,
				SongID:         1001,
				DifficultyID:   2,
				Const:          chartConst,
				IsConstUnknown: false,
				Notes:          &notesValue,
			},
			Song: &entity.Song{
				ID:        1001,
				DisplayID: "0000000000000001",
				Title:     "Song A",
				Artist:    "Artist A",
				Charts:    []*entity.Chart{},
			},
			ClearLamp:       &master.ClearLampType{ID: 1, Name: "FAILED"},
			ComboLamp:       &master.ComboLampType{ID: 1, Name: "NONE"},
			FullChain:       &master.FullChainType{ID: 1, Name: "NONE"},
			Slot:            &master.Slot{ID: 1, Name: "best"},
			ChartDifficulty: &master.ChartDifficulty{ID: 2, Name: "EXPERT"},
		},
		{
			PlayerID:    1,
			ChartID:     102,
			Score:       score2,
			ClearLampID: 2,
			ComboLampID: 2,
			FullChainID: 2,
			SlotID:      2,
			UpdatedAt:   now,
			Chart: &entity.Chart{
				ID:             102,
				SongID:         1002,
				DifficultyID:   3,
				Const:          chartConst,
				IsConstUnknown: true,
			},
			Song: &entity.Song{
				ID:        1002,
				DisplayID: "0000000000000002",
				Title:     "Song B",
				Artist:    "Artist B",
				Charts:    []*entity.Chart{},
			},
			ClearLamp:       &master.ClearLampType{ID: 2, Name: "CLEAR"},
			ComboLamp:       &master.ComboLampType{ID: 2, Name: "FC"},
			FullChain:       &master.FullChainType{ID: 2, Name: "FC"},
			Slot:            &master.Slot{ID: 2, Name: "new_candidate"},
			ChartDifficulty: &master.ChartDifficulty{ID: 3, Name: "MASTER"},
		},
	}

	playerUpdatedAt := now.Add(-time.Hour) // プレイヤーのupdated_atはレコードより前の時刻
	rating := 15.0
	player := &entity.Player{ID: 1, Name: playername.MustNewPlayerName("テストプレイヤー"), Level: 100, OfficialRating: &rating, UpdatedAt: playerUpdatedAt}
	user := &entity.User{ID: 1, PlayerID: intPointer(1)}
	service := NewUserService(nil, &stubUserRepository{user: user}, &stubPlayerRepository{playerWithHonors: &repository.PlayerWithHonors{Player: player, Honors: []*repository.PlayerHonor{}}}, &stubPlayerRecordRepository{records: records}, nil, nil, nil, nil)

	result, err := service.GetUserProfileWithRecords(context.Background(), "tester", nil, false)
	require.NoError(t, err)
	assert.Equal(t, 1, result.UserID)

	// updated_atの検証
	assert.True(t, result.Records.UpdatedAt.Equal(now))

	// 各スロットの長さを検証
	require.Len(t, result.Records.Best, 1)
	assert.Len(t, result.Records.NewCandidate, 1)
	assert.Empty(t, result.Records.BestCandidate)
	assert.Empty(t, result.Records.New)
	assert.Len(t, result.Records.All, 2)

	bestRecord := result.Records.Best[0]
	assert.Equal(t, chartConst, bestRecord.Const)
	require.NotNil(t, bestRecord.Slot)
	assert.Equal(t, "best", *bestRecord.Slot)
	assert.Equal(t, "EXPERT", bestRecord.Difficulty)
}

func TestUserService_GetUserProfileWithRecords_HonorsIsEmptySliceWhenNoHonors(t *testing.T) {
	now := time.Now()
	user := &entity.User{ID: 1, PlayerID: intPointer(1)}
	player := &entity.Player{ID: 1, Name: playername.MustNewPlayerName("テストプレイヤー"), Level: 10, UpdatedAt: now}
	service := NewUserService(nil, &stubUserRepository{user: user}, &stubPlayerRepository{playerWithHonors: &repository.PlayerWithHonors{Player: player, Honors: []*repository.PlayerHonor{}}}, &stubPlayerRecordRepository{}, nil, nil, nil, nil)

	result, err := service.GetUserProfileWithRecords(context.Background(), "tester", nil, false)
	require.NoError(t, err)
	require.NotNil(t, result.Player)
	require.NotNil(t, result.Player.Honors)
	assert.Empty(t, result.Player.Honors)
}

func TestUserService_GetUserProfileWithRecords_IncludeNoPlay(t *testing.T) {
	now := time.Now()
	scorePlayed, _ := score.NewScore(1000000)
	chartConst, _ := chartconstant.NewChartConstant(12.4)

	user := &entity.User{ID: 1, PlayerID: intPointer(1)}
	player := &entity.Player{ID: 1, Name: playername.MustNewPlayerName("テストプレイヤー"), Level: 1, UpdatedAt: now.Add(-time.Hour)}
	playedSong := &entity.Song{ID: 10, DisplayID: "song10", Charts: []*entity.Chart{{ID: 1001, SongID: 10, DifficultyID: 3, Const: chartConst}}}
	unplayedSong := &entity.Song{ID: 20, DisplayID: "song20", Charts: []*entity.Chart{{ID: 2001, SongID: 20, DifficultyID: 4, Const: chartConst}}}
	weSong := &entity.Song{ID: 30, DisplayID: "we30"}
	weChart := &entity.WorldsendChart{ID: 3001, SongID: 30}

	service := NewUserService(
		nil,
		&stubUserRepository{user: user},
		&stubPlayerRepository{playerWithHonors: &repository.PlayerWithHonors{Player: player, Honors: []*repository.PlayerHonor{}}},
		&stubPlayerRecordRepository{records: []*entity.PlayerRecord{{
			ChartID:         1001,
			Score:           scorePlayed,
			UpdatedAt:       now,
			Chart:           playedSong.Charts[0],
			Song:            playedSong,
			ChartDifficulty: &master.ChartDifficulty{ID: 3, Name: "expert"},
		}}},
		&stubWorldsendRecordRepository{},
		&stubSongRepository{songs: []*entity.Song{playedSong, unplayedSong}},
		&stubWorldsendChartRepository{records: []*repository.WorldsendSongWithChart{{Song: weSong, Chart: weChart}}},
		&stubSongMasterProvider{masters: &masterdata.SongMasters{CommonMasters: masterdata.CommonMasters{DifficultyNamesByID: map[int]string{3: "EXPERT", 4: "MASTER"}}}},
	)

	result, err := service.GetUserProfileWithRecords(context.Background(), "tester", nil, true)
	require.NoError(t, err)

	require.Len(t, result.Records.All, 2)
	assert.True(t, result.Records.All[0].IsPlayed, "expected first record is played")
	assert.False(t, result.Records.All[1].IsPlayed, "expected second record is unplayed")
	assert.Empty(t, result.Records.Best)
	assert.Empty(t, result.Records.New)
	assert.Empty(t, result.Records.NewCandidate)
	assert.Empty(t, result.Records.BestCandidate)
	assert.Nil(t, result.Records.All[1].UpdatedAt, "expected unplayed updated_at nil")
	assert.Nil(t, result.Records.All[1].ClearLamp, "expected unplayed clear_lamp nil")
	assert.Equal(t, "EXPERT", result.Records.All[0].Difficulty)
	assert.Equal(t, "MASTER", result.Records.All[1].Difficulty)
	require.Len(t, result.Records.WorldsEnd, 1)
	assert.False(t, result.Records.WorldsEnd[0].IsPlayed, "expected worldsend completion record is unplayed")

	// include_noplay=true でも slot ベースの並びは補完前レコードに依存する
	assert.Nil(t, result.Records.All[0].Slot, "expected all record slot nil")
}

func TestUserService_GetUserProfileRatingView_Success(t *testing.T) {
	now := time.Now()
	notesValue := notes.Notes(500)
	chartConst, _ := chartconstant.NewChartConstant(12.4)

	score1, _ := score.NewScore(1000000)
	score2, _ := score.NewScore(950000)

	records := []*entity.PlayerRecord{
		{
			PlayerID:    1,
			ChartID:     101,
			Score:       score1,
			ClearLampID: 1,
			ComboLampID: 1,
			FullChainID: 1,
			SlotID:      1,
			SlotOrder:   intPointer(1),
			UpdatedAt:   now,
			Chart: &entity.Chart{
				ID:             101,
				SongID:         1001,
				DifficultyID:   2,
				Const:          chartConst,
				IsConstUnknown: false,
				Notes:          &notesValue,
			},
			Song: &entity.Song{
				ID:        1001,
				DisplayID: "0000000000000001",
				Title:     "Song A",
				Artist:    "Artist A",
				Charts:    []*entity.Chart{},
			},
			ClearLamp:       &master.ClearLampType{ID: 1, Name: "FAILED"},
			ComboLamp:       &master.ComboLampType{ID: 1, Name: "NONE"},
			FullChain:       &master.FullChainType{ID: 1, Name: "NONE"},
			Slot:            &master.Slot{ID: 1, Name: "best"},
			ChartDifficulty: &master.ChartDifficulty{ID: 2, Name: "EXPERT"},
		},
		{
			PlayerID:    1,
			ChartID:     102,
			Score:       score2,
			ClearLampID: 2,
			ComboLampID: 2,
			FullChainID: 2,
			SlotID:      2,
			UpdatedAt:   now,
			Chart: &entity.Chart{
				ID:             102,
				SongID:         1002,
				DifficultyID:   3,
				Const:          chartConst,
				IsConstUnknown: true,
			},
			Song: &entity.Song{
				ID:        1002,
				DisplayID: "0000000000000002",
				Title:     "Song B",
				Artist:    "Artist B",
				Charts:    []*entity.Chart{},
			},
			ClearLamp:       &master.ClearLampType{ID: 2, Name: "CLEAR"},
			ComboLamp:       &master.ComboLampType{ID: 2, Name: "FC"},
			FullChain:       &master.FullChainType{ID: 2, Name: "FC"},
			Slot:            &master.Slot{ID: 2, Name: "new_candidate"},
			ChartDifficulty: &master.ChartDifficulty{ID: 3, Name: "MASTER"},
		},
	}

	playerUpdatedAt := now.Add(-time.Hour)
	player := &entity.Player{ID: 1, Name: playername.MustNewPlayerName("テストプレイヤー"), Level: 100, UpdatedAt: playerUpdatedAt}
	user := &entity.User{ID: 1, PlayerID: intPointer(1)}
	service := NewUserService(nil, &stubUserRepository{user: user}, &stubPlayerRepository{playerWithHonors: &repository.PlayerWithHonors{Player: player, Honors: []*repository.PlayerHonor{}}}, &stubPlayerRecordRepository{ratingRecords: records}, nil, nil, nil, nil)

	result, err := service.GetUserProfileRatingView(context.Background(), "tester", nil)
	require.NoError(t, err)

	assert.True(t, result.Records.UpdatedAt.Equal(now))
	assert.Len(t, result.Records.Best, 1)
	assert.Len(t, result.Records.NewCandidate, 1)
	assert.Empty(t, result.Records.BestCandidate)
	assert.Empty(t, result.Records.New)
}

func TestUserService_GetUserProfileRecordView_IncludeNoPlay(t *testing.T) {
	now := time.Now()
	scorePlayed, _ := score.NewScore(1000000)
	chartConst, _ := chartconstant.NewChartConstant(12.4)

	user := &entity.User{ID: 1, PlayerID: intPointer(1)}
	player := &entity.Player{ID: 1, Name: playername.MustNewPlayerName("テストプレイヤー"), Level: 1, UpdatedAt: now.Add(-time.Hour)}
	playedSong := &entity.Song{ID: 10, DisplayID: "song10", Charts: []*entity.Chart{{ID: 1001, SongID: 10, DifficultyID: 3, Const: chartConst}}}
	unplayedSong := &entity.Song{ID: 20, DisplayID: "song20", Charts: []*entity.Chart{{ID: 2001, SongID: 20, DifficultyID: 4, Const: chartConst}}}
	weSong := &entity.Song{ID: 30, DisplayID: "we30"}
	weChart := &entity.WorldsendChart{ID: 3001, SongID: 30}

	service := NewUserService(
		nil,
		&stubUserRepository{user: user},
		&stubPlayerRepository{playerWithHonors: &repository.PlayerWithHonors{Player: player, Honors: []*repository.PlayerHonor{}}},
		&stubPlayerRecordRepository{records: []*entity.PlayerRecord{{
			ChartID:         1001,
			Score:           scorePlayed,
			UpdatedAt:       now,
			Chart:           playedSong.Charts[0],
			Song:            playedSong,
			ChartDifficulty: &master.ChartDifficulty{ID: 3, Name: "expert"},
		}}},
		&stubWorldsendRecordRepository{},
		&stubSongRepository{songs: []*entity.Song{playedSong, unplayedSong}},
		&stubWorldsendChartRepository{records: []*repository.WorldsendSongWithChart{{Song: weSong, Chart: weChart}}},
		&stubSongMasterProvider{masters: &masterdata.SongMasters{CommonMasters: masterdata.CommonMasters{DifficultyNamesByID: map[int]string{3: "EXPERT", 4: "MASTER"}}}},
	)

	result, err := service.GetUserProfileRecordView(context.Background(), "tester", nil, true)
	require.NoError(t, err)

	require.NotNil(t, result)
	require.NotNil(t, result.Records)
	require.Len(t, result.Records.All, 2)
	assert.True(t, result.Records.All[0].IsPlayed, "expected first record is played")
	assert.False(t, result.Records.All[1].IsPlayed, "expected second record is unplayed")
	assert.Nil(t, result.Records.All[1].UpdatedAt, "expected unplayed updated_at nil")
	assert.Nil(t, result.Records.All[1].ClearLamp, "expected unplayed clear_lamp nil")
	assert.Equal(t, "EXPERT", result.Records.All[0].Difficulty)
	assert.Equal(t, "MASTER", result.Records.All[1].Difficulty)

	require.Len(t, result.Records.Worldsend, 1)
	assert.False(t, result.Records.Worldsend[0].IsPlayed, "expected worldsend completion record is unplayed")
}

func TestUserService_GetUserProfileRecordView_RecordsUpdatedAtFallsBackToPlayerUpdatedAt(t *testing.T) {
	now := time.Now()

	user := &entity.User{ID: 1, PlayerID: intPointer(1)}
	player := &entity.Player{ID: 1, Name: playername.MustNewPlayerName("テストプレイヤー"), Level: 1, UpdatedAt: now}

	service := NewUserService(
		nil,
		&stubUserRepository{user: user},
		&stubPlayerRepository{playerWithHonors: &repository.PlayerWithHonors{Player: player, Honors: []*repository.PlayerHonor{}}},
		&stubPlayerRecordRepository{records: []*entity.PlayerRecord{}},
		&stubWorldsendRecordRepository{},
		nil,
		nil,
		nil,
	)

	result, err := service.GetUserProfileRecordView(context.Background(), "tester", nil, false)
	require.NoError(t, err)

	assert.True(t, result.Records.UpdatedAt.Equal(now))
}

func TestUserService_GetUserProfileWithRecords_RecordsUpdatedAtUsesWorldsendLatest(t *testing.T) {
	playerUpdatedAt := time.Now().Add(-2 * time.Hour)
	worldsendUpdatedAt := playerUpdatedAt.Add(time.Hour)
	scorePlayed, err := score.NewScore(1000000)
	require.NoError(t, err)

	user := &entity.User{ID: 1, PlayerID: intPointer(1)}
	player := &entity.Player{ID: 1, Name: playername.MustNewPlayerName("テストプレイヤー"), Level: 1, UpdatedAt: playerUpdatedAt}
	worldsendRecord := &entity.PlayerWorldsendRecord{
		PlayerID:         1,
		WorldsendChartID: 3001,
		Score:            scorePlayed,
		UpdatedAt:        worldsendUpdatedAt,
		Song:             &entity.Song{ID: 30, DisplayID: "we30", Title: "WE Song", Artist: "WE Artist"},
		WorldsendChart:   &entity.WorldsendChart{ID: 3001, SongID: 30},
	}

	service := NewUserService(
		nil,
		&stubUserRepository{user: user},
		&stubPlayerRepository{playerWithHonors: &repository.PlayerWithHonors{Player: player, Honors: []*repository.PlayerHonor{}}},
		&stubPlayerRecordRepository{records: []*entity.PlayerRecord{}},
		&stubWorldsendRecordRepository{records: []*entity.PlayerWorldsendRecord{worldsendRecord}},
		nil,
		nil,
		nil,
	)

	result, err := service.GetUserProfileWithRecords(context.Background(), "tester", nil, false)
	require.NoError(t, err)
	assert.True(t, result.Records.UpdatedAt.Equal(worldsendUpdatedAt))
}

func TestUserService_GetUserProfileRecordView_RecordsUpdatedAtUsesWorldsendLatest(t *testing.T) {
	playerUpdatedAt := time.Now().Add(-2 * time.Hour)
	worldsendUpdatedAt := playerUpdatedAt.Add(time.Hour)
	scorePlayed, err := score.NewScore(1000000)
	require.NoError(t, err)

	user := &entity.User{ID: 1, PlayerID: intPointer(1)}
	player := &entity.Player{ID: 1, Name: playername.MustNewPlayerName("テストプレイヤー"), Level: 1, UpdatedAt: playerUpdatedAt}
	worldsendRecord := &entity.PlayerWorldsendRecord{
		PlayerID:         1,
		WorldsendChartID: 3001,
		Score:            scorePlayed,
		UpdatedAt:        worldsendUpdatedAt,
		Song:             &entity.Song{ID: 30, DisplayID: "we30", Title: "WE Song", Artist: "WE Artist"},
		WorldsendChart:   &entity.WorldsendChart{ID: 3001, SongID: 30},
	}

	service := NewUserService(
		nil,
		&stubUserRepository{user: user},
		&stubPlayerRepository{playerWithHonors: &repository.PlayerWithHonors{Player: player, Honors: []*repository.PlayerHonor{}}},
		&stubPlayerRecordRepository{records: []*entity.PlayerRecord{}},
		&stubWorldsendRecordRepository{records: []*entity.PlayerWorldsendRecord{worldsendRecord}},
		nil,
		nil,
		nil,
	)

	result, err := service.GetUserProfileRecordView(context.Background(), "tester", nil, false)
	require.NoError(t, err)
	assert.True(t, result.Records.UpdatedAt.Equal(worldsendUpdatedAt))
}

func TestUserService_GetAllUsersForAdmin(t *testing.T) {
	un1, _ := username.NewUserName("user1")
	pn1, _ := playername.NewPlayerName("プレイヤー１")
	rating1 := 15.0
	op1 := 10.0

	un2, _ := username.NewUserName("user2")

	usersWithPlayer := []entity.UserWithPlayer{
		{
			User: entity.User{
				ID:       1,
				Username: un1,
				PlayerID: intPointer(1),
			},
			Player: &entity.Player{
				ID:             1,
				Name:           pn1,
				OfficialRating: &rating1,
				OverpowerValue: &op1,
			},
		},
		{
			User: entity.User{
				ID:       2,
				Username: un2,
				PlayerID: nil,
			},
			Player: nil,
		},
	}

	repo := &stubUserRepository{
		usersWithPlayer: usersWithPlayer,
	}
	service := NewUserService(nil, repo, &stubPlayerRepository{}, &stubPlayerRecordRepository{}, nil, nil, nil, nil)

	list, err := service.GetAllUsersForAdmin(context.Background(), 1, 10, "")
	require.NoError(t, err)

	require.Len(t, list, 2)

	// Verify User 1
	assert.Equal(t, "user1", list[0].UserName)
	assert.Equal(t, "プレイヤー１", list[0].PlayerName)
	require.NotNil(t, list[0].Rating)
	assert.Equal(t, 15.0, *list[0].Rating)
	require.NotNil(t, list[0].OverPowerValue)
	assert.Equal(t, 10.0, *list[0].OverPowerValue)

	// Verify User 2 (No player)
	assert.Equal(t, "user2", list[1].UserName)
	assert.Equal(t, "", list[1].PlayerName)
}

func intPointer(v int) *int {
	return &v
}

func TestUserService_DeleteUser_Success(t *testing.T) {
	un, _ := username.NewUserName("testuser")
	user := &entity.User{
		ID:        1,
		Username:  un,
		IsDeleted: false,
	}
	adminRequester := &entity.User{ID: 99, AccountTypeID: 3}
	repo := &stubUserRepository{user: user}
	service := NewUserService(nil, repo, &stubPlayerRepository{}, &stubPlayerRecordRepository{}, nil, nil, nil, nil)

	err := service.DeleteUser(context.Background(), adminRequester, "testuser")
	require.NoError(t, err)

	// Saveに渡されたエンティティの状態を検証
	require.NotNil(t, repo.savedUser, "expected user to be saved")
	assert.True(t, repo.savedUser.IsDeleted, "expected user to be marked as deleted")
}

func TestUserService_DeleteUser_UserNotFound(t *testing.T) {
	adminRequester := &entity.User{ID: 99, AccountTypeID: 3}
	repo := &stubUserRepository{err: repository.ErrUserNotFound}
	service := NewUserService(nil, repo, &stubPlayerRepository{}, &stubPlayerRecordRepository{}, nil, nil, nil, nil)

	err := service.DeleteUser(context.Background(), adminRequester, "missing")
	require.ErrorIs(t, err, ErrUserNotFound)
}

func TestUserService_DeleteUser_AlreadyDeleted(t *testing.T) {
	un, _ := username.NewUserName("deleteduser")
	user := &entity.User{
		ID:        1,
		Username:  un,
		IsDeleted: true,
	}
	adminRequester := &entity.User{ID: 99, AccountTypeID: 3}
	repo := &stubUserRepository{user: user}
	service := NewUserService(nil, repo, &stubPlayerRepository{}, &stubPlayerRecordRepository{}, nil, nil, nil, nil)

	err := service.DeleteUser(context.Background(), adminRequester, "deleteduser")
	require.ErrorIs(t, err, ErrUserAlreadyDeleted)
}

func TestUserService_DeleteUser_AdminRequired(t *testing.T) {
	normalUser := &entity.User{ID: 1, AccountTypeID: 1}
	service := NewUserService(nil, &stubUserRepository{}, &stubPlayerRepository{}, &stubPlayerRecordRepository{}, nil, nil, nil, nil)

	err := service.DeleteUser(context.Background(), normalUser, "testuser")
	require.ErrorIs(t, err, ErrAdminRequired)
}

func TestUserService_DeleteUser_UnknownRoleRejected(t *testing.T) {
	unknownRoleUser := &entity.User{ID: 1, AccountTypeID: 4}
	service := NewUserService(nil, &stubUserRepository{}, &stubPlayerRepository{}, &stubPlayerRecordRepository{}, nil, nil, nil, nil)

	err := service.DeleteUser(context.Background(), unknownRoleUser, "testuser")
	require.ErrorIs(t, err, ErrAdminRequired)
}

func TestUserService_DeleteUser_NilRequester(t *testing.T) {
	service := NewUserService(nil, &stubUserRepository{}, &stubPlayerRepository{}, &stubPlayerRecordRepository{}, nil, nil, nil, nil)

	err := service.DeleteUser(context.Background(), nil, "testuser")
	require.ErrorIs(t, err, ErrAdminRequired)
}

func TestUserService_RestoreUser_Success(t *testing.T) {
	un, _ := username.NewUserName("deleteduser")
	user := &entity.User{
		ID:        1,
		Username:  un,
		IsDeleted: true,
	}
	adminRequester := &entity.User{ID: 99, AccountTypeID: 3}
	repo := &stubUserRepository{user: user}
	service := NewUserService(nil, repo, &stubPlayerRepository{}, &stubPlayerRecordRepository{}, nil, nil, nil, nil)

	err := service.RestoreUser(context.Background(), adminRequester, "deleteduser")
	require.NoError(t, err)

	// Saveに渡されたエンティティの状態を検証
	require.NotNil(t, repo.savedUser, "expected user to be saved")
	assert.False(t, repo.savedUser.IsDeleted, "expected user to be restored (not deleted)")
}

func TestUserService_RestoreUser_UserNotFound(t *testing.T) {
	adminRequester := &entity.User{ID: 99, AccountTypeID: 3}
	repo := &stubUserRepository{err: repository.ErrUserNotFound}
	service := NewUserService(nil, repo, &stubPlayerRepository{}, &stubPlayerRecordRepository{}, nil, nil, nil, nil)

	err := service.RestoreUser(context.Background(), adminRequester, "missing")
	require.ErrorIs(t, err, ErrUserNotFound)
}

func TestUserService_RestoreUser_NotDeleted(t *testing.T) {
	un, _ := username.NewUserName("activeuser")
	user := &entity.User{
		ID:        1,
		Username:  un,
		IsDeleted: false,
	}
	adminRequester := &entity.User{ID: 99, AccountTypeID: 3}
	repo := &stubUserRepository{user: user}
	service := NewUserService(nil, repo, &stubPlayerRepository{}, &stubPlayerRecordRepository{}, nil, nil, nil, nil)

	err := service.RestoreUser(context.Background(), adminRequester, "activeuser")
	require.ErrorIs(t, err, ErrUserNotDeleted)
}

func TestUserService_RestoreUser_AdminRequired(t *testing.T) {
	normalUser := &entity.User{ID: 1, AccountTypeID: 1}
	service := NewUserService(nil, &stubUserRepository{}, &stubPlayerRepository{}, &stubPlayerRecordRepository{}, nil, nil, nil, nil)

	err := service.RestoreUser(context.Background(), normalUser, "deleteduser")
	require.ErrorIs(t, err, ErrAdminRequired)
}

func TestUserService_RestoreUser_UnknownRoleRejected(t *testing.T) {
	unknownRoleUser := &entity.User{ID: 1, AccountTypeID: 4}
	service := NewUserService(nil, &stubUserRepository{}, &stubPlayerRepository{}, &stubPlayerRecordRepository{}, nil, nil, nil, nil)

	err := service.RestoreUser(context.Background(), unknownRoleUser, "deleteduser")
	require.ErrorIs(t, err, ErrAdminRequired)
}

func TestUserService_RestoreUser_NilRequester(t *testing.T) {
	service := NewUserService(nil, &stubUserRepository{}, &stubPlayerRepository{}, &stubPlayerRecordRepository{}, nil, nil, nil, nil)

	err := service.RestoreUser(context.Background(), nil, "deleteduser")
	require.ErrorIs(t, err, ErrAdminRequired)
}
