package usecase

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/notes"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/playername"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/score"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/username"
	"github.com/chunisupport/chunisupport-api/internal/dto"
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
	records       []*entity.PlayerRecord
	ratingRecords []*entity.PlayerRecord
	err           error
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
	return nil, nil
}

type stubPlayerService struct {
	player *dto.PlayerDTO
	err    error
}

func (s *stubPlayerService) CreatePlayer(ctx context.Context, name string) (*dto.PlayerDTO, error) {
	return nil, errors.New("not implemented")
}

func (s *stubPlayerService) GetPlayerByID(ctx context.Context, id int) (*dto.PlayerDTO, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.player, nil
}

func TestUserService_GetUserProfileWithRecords_UserNotFound(t *testing.T) {
	service := NewUserService(nil, &stubUserRepository{err: sql.ErrNoRows}, &stubPlayerRecordRepository{}, &stubPlayerService{})

	_, err := service.GetUserProfileWithRecords(context.Background(), "missing", nil)
	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestUserService_GetUserProfileWithRecords_PlayerNotLinked(t *testing.T) {
	user := &entity.User{ID: 1}
	service := NewUserService(nil, &stubUserRepository{user: user}, &stubPlayerRecordRepository{}, &stubPlayerService{})

	_, err := service.GetUserProfileWithRecords(context.Background(), "no-player", nil)
	if !errors.Is(err, ErrPlayerNotLinked) {
		t.Fatalf("expected ErrPlayerNotLinked, got %v", err)
	}
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
	player := &dto.PlayerDTO{
		Name:      "SelfPlayer",
		Level:     1,
		UpdatedAt: now,
	}
	service := NewUserService(nil, &stubUserRepository{user: user}, &stubPlayerRecordRepository{}, &stubPlayerService{player: player})

	_, err := service.GetUserProfileWithRecords(context.Background(), "selfuser", &entity.User{ID: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
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
			ClearLamp:       &entity.ClearLampType{ID: 1, Name: "FAILED"},
			ComboLamp:       &entity.ComboLampType{ID: 1, Name: "NONE"},
			FullChain:       &entity.FullChainType{ID: 1, Name: "NONE"},
			Slot:            &entity.Slot{ID: 1, Name: "best"},
			ChartDifficulty: &entity.ChartDifficulty{ID: 2, Name: "EXPERT"},
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
			ClearLamp:       &entity.ClearLampType{ID: 2, Name: "CLEAR"},
			ComboLamp:       &entity.ComboLampType{ID: 2, Name: "FC"},
			FullChain:       &entity.FullChainType{ID: 2, Name: "FC"},
			Slot:            &entity.Slot{ID: 2, Name: "new_candidate"},
			ChartDifficulty: &entity.ChartDifficulty{ID: 3, Name: "MASTER"},
		},
	}

	playerUpdatedAt := now.Add(-time.Hour) // プレイヤーのupdated_atはレコードより前の時刻
	rating := 15.0
	player := &dto.PlayerDTO{
		Name:      "TestPlayer",
		Level:     100,
		Rating:    &rating,
		UpdatedAt: playerUpdatedAt,
	}
	user := &entity.User{ID: 1, PlayerID: intPointer(1)}
	service := NewUserService(nil, &stubUserRepository{user: user}, &stubPlayerRecordRepository{records: records}, &stubPlayerService{player: player})

	result, err := service.GetUserProfileWithRecords(context.Background(), "tester", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// updated_atの検証
	expectedRecordUpdatedAt := now
	if !result.Records.UpdatedAt.Equal(expectedRecordUpdatedAt) {
		t.Fatalf("expected updated_at to be %v, got %v", expectedRecordUpdatedAt, result.Records.UpdatedAt)
	}

	// 各スロットの長さを検証
	if len(result.Records.Best) != 1 {
		t.Fatalf("expected 1 record for best, got %d", len(result.Records.Best))
	}
	if len(result.Records.NewCandidate) != 1 {
		t.Fatalf("expected 1 record for new_candidate, got %d", len(result.Records.NewCandidate))
	}
	if len(result.Records.BestCandidate) != 0 {
		t.Fatalf("expected 0 records for best_candidate, got %d", len(result.Records.BestCandidate))
	}
	if len(result.Records.New) != 0 {
		t.Fatalf("expected 0 records for new, got %d", len(result.Records.New))
	}
	if len(result.Records.All) != 2 {
		t.Fatalf("expected 2 records for all, got %d", len(result.Records.All))
	}

	bestRecord := result.Records.Best[0]
	if bestRecord.Const != chartConst {
		t.Fatalf("expected chart const to be %v, got %v", chartConst, bestRecord.Const)
	}
	if bestRecord.Slot == nil || *bestRecord.Slot != "best" {
		t.Fatalf("expected slot name best, got %v", bestRecord.Slot)
	}
	if bestRecord.Difficulty != "EXPERT" {
		t.Fatalf("expected difficulty EXPERT, got %v", bestRecord.Difficulty)
	}
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
			ClearLamp:       &entity.ClearLampType{ID: 1, Name: "FAILED"},
			ComboLamp:       &entity.ComboLampType{ID: 1, Name: "NONE"},
			FullChain:       &entity.FullChainType{ID: 1, Name: "NONE"},
			Slot:            &entity.Slot{ID: 1, Name: "best"},
			ChartDifficulty: &entity.ChartDifficulty{ID: 2, Name: "EXPERT"},
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
			ClearLamp:       &entity.ClearLampType{ID: 2, Name: "CLEAR"},
			ComboLamp:       &entity.ComboLampType{ID: 2, Name: "FC"},
			FullChain:       &entity.FullChainType{ID: 2, Name: "FC"},
			Slot:            &entity.Slot{ID: 2, Name: "new_candidate"},
			ChartDifficulty: &entity.ChartDifficulty{ID: 3, Name: "MASTER"},
		},
	}

	playerUpdatedAt := now.Add(-time.Hour)
	player := &dto.PlayerDTO{
		Name:      "TestPlayer",
		Level:     100,
		UpdatedAt: playerUpdatedAt,
	}
	user := &entity.User{ID: 1, PlayerID: intPointer(1)}
	service := NewUserService(nil, &stubUserRepository{user: user}, &stubPlayerRecordRepository{ratingRecords: records}, &stubPlayerService{player: player})

	result, err := service.GetUserProfileRatingView(context.Background(), "tester", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedRecordUpdatedAt := now
	if !result.Records.UpdatedAt.Equal(expectedRecordUpdatedAt) {
		t.Fatalf("expected updated_at to be %v, got %v", expectedRecordUpdatedAt, result.Records.UpdatedAt)
	}

	if len(result.Records.Best) != 1 {
		t.Fatalf("expected 1 record for best, got %d", len(result.Records.Best))
	}
	if len(result.Records.NewCandidate) != 1 {
		t.Fatalf("expected 1 record for new_candidate, got %d", len(result.Records.NewCandidate))
	}
	if len(result.Records.BestCandidate) != 0 {
		t.Fatalf("expected 0 records for best_candidate, got %d", len(result.Records.BestCandidate))
	}
	if len(result.Records.New) != 0 {
		t.Fatalf("expected 0 records for new, got %d", len(result.Records.New))
	}
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
	service := NewUserService(nil, repo, &stubPlayerRecordRepository{}, &stubPlayerService{})

	list, err := service.GetAllUsersForAdmin(context.Background(), 1, 10, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(list) != 2 {
		t.Fatalf("expected 2 users, got %d", len(list))
	}

	// Verify User 1
	if list[0].UserName != "user1" {
		t.Errorf("expected username user1, got %s", list[0].UserName)
	}
	if list[0].PlayerName != "プレイヤー１" {
		t.Errorf("expected player name プレイヤー１, got %s", list[0].PlayerName)
	}
	if list[0].Rating == nil || *list[0].Rating != 15.0 {
		t.Errorf("expected rating 15.0, got %v", list[0].Rating)
	}
	if list[0].OverPowerValue == nil || *list[0].OverPowerValue != 10.0 {
		t.Errorf("expected overpower 10.0, got %v", list[0].OverPowerValue)
	}

	// Verify User 2 (No player)
	if list[1].UserName != "user2" {
		t.Errorf("expected username user2, got %s", list[1].UserName)
	}
	if list[1].PlayerName != "" {
		t.Errorf("expected empty player name, got %s", list[1].PlayerName)
	}
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
	service := NewUserService(nil, repo, &stubPlayerRecordRepository{}, &stubPlayerService{})

	err := service.DeleteUser(context.Background(), adminRequester, "testuser")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Saveに渡されたエンティティの状態を検証
	if repo.savedUser == nil {
		t.Fatal("expected user to be saved")
	}
	if !repo.savedUser.IsDeleted {
		t.Error("expected user to be marked as deleted")
	}
}

func TestUserService_DeleteUser_UserNotFound(t *testing.T) {
	adminRequester := &entity.User{ID: 99, AccountTypeID: 3}
	repo := &stubUserRepository{err: sql.ErrNoRows}
	service := NewUserService(nil, repo, &stubPlayerRecordRepository{}, &stubPlayerService{})

	err := service.DeleteUser(context.Background(), adminRequester, "missing")
	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
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
	service := NewUserService(nil, repo, &stubPlayerRecordRepository{}, &stubPlayerService{})

	err := service.DeleteUser(context.Background(), adminRequester, "deleteduser")
	if !errors.Is(err, ErrUserAlreadyDeleted) {
		t.Fatalf("expected ErrUserAlreadyDeleted, got %v", err)
	}
}

func TestUserService_DeleteUser_AdminRequired(t *testing.T) {
	normalUser := &entity.User{ID: 1, AccountTypeID: 1}
	service := NewUserService(nil, &stubUserRepository{}, &stubPlayerRecordRepository{}, &stubPlayerService{})

	err := service.DeleteUser(context.Background(), normalUser, "testuser")
	if !errors.Is(err, ErrAdminRequired) {
		t.Fatalf("expected ErrAdminRequired, got %v", err)
	}
}

func TestUserService_DeleteUser_NilRequester(t *testing.T) {
	service := NewUserService(nil, &stubUserRepository{}, &stubPlayerRecordRepository{}, &stubPlayerService{})

	err := service.DeleteUser(context.Background(), nil, "testuser")
	if !errors.Is(err, ErrAdminRequired) {
		t.Fatalf("expected ErrAdminRequired, got %v", err)
	}
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
	service := NewUserService(nil, repo, &stubPlayerRecordRepository{}, &stubPlayerService{})

	err := service.RestoreUser(context.Background(), adminRequester, "deleteduser")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Saveに渡されたエンティティの状態を検証
	if repo.savedUser == nil {
		t.Fatal("expected user to be saved")
	}
	if repo.savedUser.IsDeleted {
		t.Error("expected user to be restored (not deleted)")
	}
}

func TestUserService_RestoreUser_UserNotFound(t *testing.T) {
	adminRequester := &entity.User{ID: 99, AccountTypeID: 3}
	repo := &stubUserRepository{err: sql.ErrNoRows}
	service := NewUserService(nil, repo, &stubPlayerRecordRepository{}, &stubPlayerService{})

	err := service.RestoreUser(context.Background(), adminRequester, "missing")
	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
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
	service := NewUserService(nil, repo, &stubPlayerRecordRepository{}, &stubPlayerService{})

	err := service.RestoreUser(context.Background(), adminRequester, "activeuser")
	if !errors.Is(err, ErrUserNotDeleted) {
		t.Fatalf("expected ErrUserNotDeleted, got %v", err)
	}
}

func TestUserService_RestoreUser_AdminRequired(t *testing.T) {
	normalUser := &entity.User{ID: 1, AccountTypeID: 1}
	service := NewUserService(nil, &stubUserRepository{}, &stubPlayerRecordRepository{}, &stubPlayerService{})

	err := service.RestoreUser(context.Background(), normalUser, "deleteduser")
	if !errors.Is(err, ErrAdminRequired) {
		t.Fatalf("expected ErrAdminRequired, got %v", err)
	}
}

func TestUserService_RestoreUser_NilRequester(t *testing.T) {
	service := NewUserService(nil, &stubUserRepository{}, &stubPlayerRecordRepository{}, &stubPlayerService{})

	err := service.RestoreUser(context.Background(), nil, "deleteduser")
	if !errors.Is(err, ErrAdminRequired) {
		t.Fatalf("expected ErrAdminRequired, got %v", err)
	}
}
