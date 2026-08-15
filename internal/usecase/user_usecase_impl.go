package usecase

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/service"
	"github.com/chunisupport/chunisupport-api/internal/info"
)

// userUsecase は UserUsecase の実装です。
type userUsecase struct {
	db                           repository.Executor
	userRepo                     repository.UserRepository
	playerRepo                   repository.PlayerRepository
	playerRecordRepo             repository.PlayerRecordRepository
	worldsendRecordRepo          repository.WorldsendRecordRepository
	courseRepo                   repository.CourseRepository
	songRepo                     repository.SongRepository
	worldsendChartRepo           repository.WorldsendChartRepository
	playerLockedSongRepo         repository.PlayerLockedSongRepository
	friendshipRepo               repository.FriendshipRepository
	overpowerDenominatorProvider repository.OverpowerDenominatorProvider
	userUpdatedAtQuery           repository.UserUpdatedAtQueryService
	recordCompletionSvc          *service.RecordCompletionService
	masterProvider               userMasterProvider
	firebaseDeleter              FirebaseUserDeleter
}

type userMasterProvider interface {
	repository.SongMasterProvider
	repository.AccountTypeMasterProvider
}

type userProfilePlayerRecords struct {
	all             []*entity.PlayerRecord
	slotMap         map[string][]*entity.PlayerRecord
	latestUpdatedAt time.Time
}

// NewUserUsecase は UserUsecase の実装を生成します。
func NewUserUsecase(db repository.Executor, userRepo repository.UserRepository, playerRepo repository.PlayerRepository, playerRecordRepo repository.PlayerRecordRepository, worldsendRecordRepo repository.WorldsendRecordRepository, songRepo repository.SongRepository, worldsendChartRepo repository.WorldsendChartRepository, masterProvider userMasterProvider) UserUsecase {
	return &userUsecase{
		db:                  db,
		userRepo:            userRepo,
		playerRepo:          playerRepo,
		playerRecordRepo:    playerRecordRepo,
		worldsendRecordRepo: worldsendRecordRepo,
		songRepo:            songRepo,
		worldsendChartRepo:  worldsendChartRepo,
		recordCompletionSvc: service.NewRecordCompletionService(),
		masterProvider:      masterProvider,
		firebaseDeleter:     noopFirebaseUserDeleter{},
	}
}

// NewUserUsecaseWithOverpowerDenominator はOVER POWER割合の随時計算Provider付きで UserUsecase を生成します。
func NewUserUsecaseWithOverpowerDenominator(db repository.Executor, userRepo repository.UserRepository, playerRepo repository.PlayerRepository, playerRecordRepo repository.PlayerRecordRepository, worldsendRecordRepo repository.WorldsendRecordRepository, songRepo repository.SongRepository, worldsendChartRepo repository.WorldsendChartRepository, masterProvider userMasterProvider, playerLockedSongRepo repository.PlayerLockedSongRepository, overpowerDenominatorProvider repository.OverpowerDenominatorProvider) UserUsecase {
	usecase := NewUserUsecase(db, userRepo, playerRepo, playerRecordRepo, worldsendRecordRepo, songRepo, worldsendChartRepo, masterProvider)
	impl, ok := usecase.(*userUsecase)
	if !ok {
		return usecase
	}
	impl.playerLockedSongRepo = playerLockedSongRepo
	impl.overpowerDenominatorProvider = overpowerDenominatorProvider
	return impl
}

// SetFriendshipRepository は非公開ユーザー閲覧時のフレンド判定リポジトリを設定します。
func (s *userUsecase) SetFriendshipRepository(friendshipRepo repository.FriendshipRepository) {
	s.friendshipRepo = friendshipRepo
}

// SetCourseRepository はユーザーレコードレスポンスへコースを統合します。
func (s *userUsecase) SetCourseRepository(courseRepo repository.CourseRepository) {
	s.courseRepo = courseRepo
}

// NewUserUsecaseWithFirebaseDeleter は Firebase 削除連携付きの UserUsecase を生成します。
func NewUserUsecaseWithFirebaseDeleter(db repository.Executor, userRepo repository.UserRepository, playerRepo repository.PlayerRepository, playerRecordRepo repository.PlayerRecordRepository, worldsendRecordRepo repository.WorldsendRecordRepository, songRepo repository.SongRepository, worldsendChartRepo repository.WorldsendChartRepository, masterProvider userMasterProvider, firebaseDeleter FirebaseUserDeleter) UserUsecase {
	usecase := NewUserUsecase(db, userRepo, playerRepo, playerRecordRepo, worldsendRecordRepo, songRepo, worldsendChartRepo, masterProvider)
	impl, ok := usecase.(*userUsecase)
	if !ok {
		return usecase
	}
	if firebaseDeleter != nil {
		impl.firebaseDeleter = firebaseDeleter
	}
	return impl
}

// NewUserUsecaseWithFirebaseDeleterAndOverpowerDenominator はFirebase連携とOVER POWER随時計算Provider付きで UserUsecase を生成します。
func NewUserUsecaseWithFirebaseDeleterAndOverpowerDenominator(db repository.Executor, userRepo repository.UserRepository, playerRepo repository.PlayerRepository, playerRecordRepo repository.PlayerRecordRepository, worldsendRecordRepo repository.WorldsendRecordRepository, songRepo repository.SongRepository, worldsendChartRepo repository.WorldsendChartRepository, masterProvider userMasterProvider, firebaseDeleter FirebaseUserDeleter, playerLockedSongRepo repository.PlayerLockedSongRepository, overpowerDenominatorProvider repository.OverpowerDenominatorProvider, userUpdatedAtQuery repository.UserUpdatedAtQueryService) UserUsecase {
	usecase := NewUserUsecaseWithFirebaseDeleter(db, userRepo, playerRepo, playerRecordRepo, worldsendRecordRepo, songRepo, worldsendChartRepo, masterProvider, firebaseDeleter)
	impl, ok := usecase.(*userUsecase)
	if !ok {
		return usecase
	}
	impl.playerLockedSongRepo = playerLockedSongRepo
	impl.overpowerDenominatorProvider = overpowerDenominatorProvider
	impl.userUpdatedAtQuery = userUpdatedAtQuery
	return impl
}

// GetUserProfile はユーザー名をキーにプロファイル（username + player）を軽量に取得します。
// 対象ユーザーが非公開設定の場合は、本人または承認済みフレンド以外は ErrUserPrivate を返します。
func (s *userUsecase) GetUserProfile(ctx context.Context, username string, requester *entity.User) (*UserProfileOutput, error) {
	user, err := s.getAccessibleUser(ctx, username, requester)
	if err != nil {
		return nil, err
	}
	player, err := s.getOptionalPlayer(ctx, user)
	if err != nil {
		return nil, err
	}
	return &UserProfileOutput{
		Username: user.Username.String(),
		Player:   player,
	}, nil
}

// GetUserUpdatedAt はユーザーのプロフィールとレコードの updated_at のうち新しい方を返します。
func (s *userUsecase) GetUserUpdatedAt(ctx context.Context, username string, requester *entity.User) (*UserUpdatedAtOutput, error) {
	if s.userUpdatedAtQuery != nil {
		return s.getUserUpdatedAtByQuery(ctx, username, requester)
	}

	user, err := s.getAccessibleUser(ctx, username, requester)
	if err != nil {
		return nil, err
	}
	if user.PlayerID == nil {
		return &UserUpdatedAtOutput{UpdatedAt: nil}, nil
	}

	player, err := s.playerRepo.FindByID(ctx, s.db, *user.PlayerID)
	if err != nil {
		if errors.Is(err, repository.ErrPlayerNotFound) {
			return &UserUpdatedAtOutput{UpdatedAt: nil}, nil
		}
		return nil, err
	}
	if player == nil {
		return &UserUpdatedAtOutput{
			UpdatedAt: nil,
		}, nil
	}

	lastScoreUpdate, err := s.playerRecordRepo.GetLastScoreUpdate(ctx, s.db, *user.PlayerID)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			slog.Warn("failed to get last score update due to context canceled", "player_id", *user.PlayerID, "error", err)
		} else {
			slog.Error("failed to get last score update", "player_id", *user.PlayerID, "error", err)
		}
		return nil, err
	}

	latestUpdatedAt := player.UpdatedAt
	if lastScoreUpdate != nil && lastScoreUpdate.After(latestUpdatedAt) {
		latestUpdatedAt = *lastScoreUpdate
	}

	return &UserUpdatedAtOutput{
		UpdatedAt: &latestUpdatedAt,
	}, nil
}

func (s *userUsecase) getUserUpdatedAtByQuery(ctx context.Context, username string, requester *entity.User) (*UserUpdatedAtOutput, error) {
	result, err := s.userUpdatedAtQuery.FindByUsername(ctx, s.db, username)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	if result == nil || result.User == nil {
		return nil, ErrUserNotFound
	}
	accessible, err := canAccessPrivateUser(ctx, s.db, s.friendshipRepo, result.User, requester)
	if err != nil {
		return nil, err
	}
	if !accessible {
		return nil, ErrUserPrivate
	}
	if result.PlayerUpdatedAt == nil {
		return &UserUpdatedAtOutput{UpdatedAt: nil}, nil
	}

	latestUpdatedAt := *result.PlayerUpdatedAt
	if result.RecordsUpdatedAt != nil && result.RecordsUpdatedAt.After(latestUpdatedAt) {
		latestUpdatedAt = *result.RecordsUpdatedAt
	}

	return &UserUpdatedAtOutput{UpdatedAt: &latestUpdatedAt}, nil
}

// GetUserProfileWithRecords はユーザー名をキーにプロファイルとレコードを一括取得します。
// 対象ユーザーが非公開設定の場合は、本人または承認済みフレンド以外は ErrUserPrivate を返します。
func (s *userUsecase) GetUserProfileWithRecords(ctx context.Context, username string, requester *entity.User, includeNoPlay bool) (*UserProfileWithRecordsOutput, error) {
	user, err := s.getAccessibleUser(ctx, username, requester)
	if err != nil {
		return nil, err
	}
	player, err := s.getOptionalPlayer(ctx, user)
	if err != nil {
		return nil, err
	}
	if player == nil {
		return &UserProfileWithRecordsOutput{
			UserID:    user.ID,
			Username:  user.Username.String(),
			Player:    nil,
			Records:   nil,
			UpdatedAt: nil,
		}, nil
	}

	playerRecords, err := s.getUserProfilePlayerRecords(ctx, *user.PlayerID, includeNoPlay)
	if err != nil {
		return nil, err
	}

	worldsendRecords, err := s.getUserProfileWorldsendRecords(ctx, *user.PlayerID, includeNoPlay)
	if err != nil {
		return nil, err
	}
	courseRecords, courseUpdatedAt, err := s.getUserProfileCourseRecords(ctx, *user.PlayerID, includeNoPlay)
	if err != nil {
		return nil, err
	}

	recordsUpdatedAt := latestUserRecordUpdatedAt(playerRecords.latestUpdatedAt, latestWorldsendRecordUpdatedAt(worldsendRecords), courseUpdatedAt)
	if recordsUpdatedAt.IsZero() {
		recordsUpdatedAt = player.Player.UpdatedAt
	}
	recordsDTO := &UserRecordOutput{
		UpdatedAt:     recordsUpdatedAt,
		Best:          toPlayerRecordOutputs(playerRecords.slotMap["best"]),
		BestCandidate: toPlayerRecordOutputs(playerRecords.slotMap["best_candidate"]),
		New:           toPlayerRecordOutputs(playerRecords.slotMap["new"]),
		NewCandidate:  toPlayerRecordOutputs(playerRecords.slotMap["new_candidate"]),
		All:           toPlayerRecordOutputs(playerRecords.all),
		WorldsEnd:     toWorldsendRecordOutputs(worldsendRecords),
		Courses:       courseRecords,
	}

	return &UserProfileWithRecordsOutput{
		UserID:    user.ID,
		Username:  user.Username.String(),
		Player:    player,
		Records:   recordsDTO,
		UpdatedAt: &player.Player.UpdatedAt,
	}, nil
}

// GetUserProfileRatingView はユーザー名をキーにレーティング表示向けのプロファイルとレコードを取得します。
func (s *userUsecase) GetUserProfileRatingView(ctx context.Context, username string, requester *entity.User) (*UserProfileRatingViewOutput, error) {
	user, err := s.getAccessibleUser(ctx, username, requester)
	if err != nil {
		return nil, err
	}
	player, err := s.getOptionalPlayer(ctx, user)
	if err != nil {
		return nil, err
	}
	if player == nil {
		return &UserProfileRatingViewOutput{
			Username:  user.Username.String(),
			Player:    nil,
			Records:   nil,
			UpdatedAt: nil,
		}, nil
	}

	records, err := s.playerRecordRepo.FindByPlayerIDForRating(ctx, s.db, *user.PlayerID)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			slog.Warn("failed to find player rating records due to context canceled", "player_id", *user.PlayerID, "error", err)
		} else {
			slog.Error("failed to find player rating records", "player_id", *user.PlayerID, "error", err)
		}
		return nil, err
	}
	opTargetSourceRecords, err := s.playerRecordRepo.FindByPlayerID(ctx, s.db, *user.PlayerID)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			slog.Warn("failed to find player records for OP target due to context canceled", "player_id", *user.PlayerID, "error", err)
		} else {
			slog.Error("failed to find player records for OP target", "player_id", *user.PlayerID, "error", err)
		}
		return nil, err
	}
	applyOPTargetFlags(records, calculateOPTargetChartIDs(opTargetSourceRecords))

	slotMap := initializeRatingSlotMap()
	var latestRecordUpdatedAt time.Time
	for _, record := range records {
		slotKey := record.SlotKey()
		if slotKey != "" {
			slotMap[slotKey] = append(slotMap[slotKey], record)
		}
		if record.UpdatedAt.After(latestRecordUpdatedAt) {
			latestRecordUpdatedAt = record.UpdatedAt
		}
	}

	recordsUpdatedAt := latestRecordUpdatedAt
	if recordsUpdatedAt.IsZero() {
		recordsUpdatedAt = player.Player.UpdatedAt
	}
	recordsDTO := &UserRatingRecordOutput{
		UpdatedAt:     recordsUpdatedAt,
		Best:          toPlayerRecordOutputs(slotMap["best"]),
		BestCandidate: toPlayerRecordOutputs(slotMap["best_candidate"]),
		New:           toPlayerRecordOutputs(slotMap["new"]),
		NewCandidate:  toPlayerRecordOutputs(slotMap["new_candidate"]),
	}

	return &UserProfileRatingViewOutput{
		Username:  user.Username.String(),
		Player:    player,
		Records:   recordsDTO,
		UpdatedAt: &player.Player.UpdatedAt,
	}, nil
}

// GetAllUsersForAdmin はADMIN用にすべてのユーザー一覧を取得します。
// プライベート・削除済み・プレイヤー未紐付けアカウントを含みます。
func (s *userUsecase) GetAllUsersForAdmin(ctx context.Context, page int, limit int, name string) ([]AdminUserOutput, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 100 // default fallback if 0
	}
	offset := (page - 1) * limit

	users, err := s.userRepo.FindAllWithPlayerForAdmin(ctx, s.db, limit, offset, name)
	if err != nil {
		slog.Error("failed to fetch user list for admin", "error", err)
		return nil, err
	}

	responses := make([]AdminUserOutput, 0, len(users))
	for _, u := range users {
		accountTypeName := "UNKNOWN"
		if s.masterProvider != nil {
			accountTypeName = s.masterProvider.GetAccountTypeNameByID(u.User.AccountTypeID)
		}

		resp := AdminUserOutput{
			UserName:     u.User.Username.String(),
			AccountType:  accountTypeName,
			CreatedAt:    u.User.CreatedAt,
			UpdatedAt:    u.User.UpdatedAt,
			IsSuspicious: u.User.IsSuspicious,
			IsPrivate:    u.User.IsPrivate,
		}
		if u.Player != nil {
			playerName := u.Player.Name.String()
			resp.PlayerName = &playerName
			resp.Rating = u.Player.CalculatedRating
			resp.OverPowerValue = u.Player.OverpowerValue
		}
		responses = append(responses, resp)
	}

	return responses, nil
}

// DeleteUser はユーザーを物理削除します。
// 防御的深度: ハンドラ層のミドルウェアに加え、ユースケース層でもADMIN権限を検証します。
func (s *userUsecase) DeleteUser(ctx context.Context, requester *entity.User, username string) error {
	if err := s.ensureDeleteUserPermission(requester); err != nil {
		return err
	}

	// 1. ユーザーを取得
	user, err := s.userRepo.FindByUsername(ctx, s.db, username)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return ErrUserNotFound
		}
		slog.Error("failed to find user by username", "username", username, "error", err)
		return err
	}

	firebaseUID := ""
	if user.FirebaseUID != nil {
		firebaseUID = *user.FirebaseUID
	}

	if err := s.performPhysicalUserDeletion(ctx, user.ID, username); err != nil {
		return err
	}

	if firebaseUID != "" {
		if err := s.firebaseDeleter.DeleteUser(ctx, firebaseUID); err != nil {
			slog.Error("failed to delete firebase user after account deletion", "user_id", user.ID, "username", username, "firebase_uid", firebaseUID, "error", err)
		}
	}

	slog.Info("user deleted successfully", "username", username, "user_id", user.ID)
	return nil
}

func (s *userUsecase) ensureDeleteUserPermission(requester *entity.User) error {
	if requester == nil || !info.HasRole(requester.AccountTypeID, info.AccountTypeAdmin) {
		return ErrAdminRequired
	}
	return nil
}

func (s *userUsecase) performPhysicalUserDeletion(ctx context.Context, userID int, username string) error {
	if err := s.userRepo.DeleteByID(ctx, s.db, userID); err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return ErrUserNotFound
		}
		slog.Error("failed to delete user from database", "user_id", userID, "username", username, "error", err)
		return err
	}
	return nil
}

// GetUserProfileRecordView はユーザー名をキーにレコード表示向けのプロファイルとレコードを取得します。
func (s *userUsecase) GetUserProfileRecordView(ctx context.Context, username string, requester *entity.User, includeNoPlay bool) (*UserProfileRecordViewOutput, error) {
	user, err := s.getAccessibleUser(ctx, username, requester)
	if err != nil {
		return nil, err
	}
	player, err := s.getOptionalPlayer(ctx, user)
	if err != nil {
		return nil, err
	}
	if player == nil {
		return &UserProfileRecordViewOutput{
			Username:  user.Username.String(),
			Player:    nil,
			Records:   nil,
			UpdatedAt: nil,
		}, nil
	}

	playerRecords, err := s.getUserProfilePlayerRecords(ctx, *user.PlayerID, includeNoPlay)
	if err != nil {
		return nil, err
	}

	worldsendRecords, err := s.getUserProfileWorldsendRecords(ctx, *user.PlayerID, includeNoPlay)
	if err != nil {
		return nil, err
	}
	courseRecords, courseUpdatedAt, err := s.getUserProfileCourseRecords(ctx, *user.PlayerID, includeNoPlay)
	if err != nil {
		return nil, err
	}

	recordsUpdatedAt := latestUserRecordUpdatedAt(playerRecords.latestUpdatedAt, latestWorldsendRecordUpdatedAt(worldsendRecords), courseUpdatedAt)
	if recordsUpdatedAt.IsZero() {
		recordsUpdatedAt = player.Player.UpdatedAt
	}

	return &UserProfileRecordViewOutput{
		Username: user.Username.String(),
		Player:   player,
		Records: &UserRecordViewOutput{
			UpdatedAt: recordsUpdatedAt,
			All:       toPlayerRecordOutputs(playerRecords.all),
			Worldsend: toWorldsendRecordOutputs(worldsendRecords),
			Courses:   courseRecords,
		},
		UpdatedAt: &player.Player.UpdatedAt,
	}, nil
}

// GetUserSongRecord は指定した通常楽曲に属するレコードだけを返します。
func (s *userUsecase) GetUserSongRecord(ctx context.Context, username string, requester *entity.User, displayID string, includeNoPlay bool, difficulty string) (*UserSongRecordOutput, error) {
	user, err := s.getAccessibleUser(ctx, username, requester)
	if err != nil {
		return nil, err
	}

	song, err := s.songRepo.FindByDisplayID(ctx, s.db, displayID)
	if err != nil {
		if errors.Is(err, repository.ErrSongNotFound) {
			return nil, repository.ErrSongNotFound
		}
		return nil, err
	}
	if song == nil || song.IsDeleted || song.IsWorldsend {
		return nil, repository.ErrSongNotFound
	}

	difficultyID, err := s.resolveSongDifficulty(song, difficulty)
	if err != nil {
		return nil, err
	}

	response := &UserSongRecordOutput{Standard: []*PlayerRecordOutput{}, Meta: &UserSongRecordMetaOutput{}}
	if !user.HasLinkedPlayer() {
		return response, nil
	}

	records, err := s.playerRecordRepo.FindByPlayerIDAndSongDisplayID(ctx, s.db, *user.PlayerID, displayID)
	if err != nil {
		return nil, err
	}
	markOPTargetPlayerRecords(records)

	allRecords := records
	if includeNoPlay {
		difficultyNames, difficultySortOrders := s.songDifficultyMasters()
		allRecords = s.recordCompletionSvc.CompletePlayerRecords(records, []*entity.Song{song}, difficultyNames, difficultySortOrders)
	}
	if difficultyID != nil {
		allRecords = filterPlayerRecordsByDifficultyID(allRecords, *difficultyID)
	}

	response.Standard = toPlayerRecordOutputs(allRecords)
	if latest := latestPlayerRecordUpdatedAt(allRecords); !latest.IsZero() {
		response.UpdatedAt = &latest
		response.Meta.UpdatedAt = &latest
	}
	return response, nil
}

// GetUserWorldsendSongRecord は指定した WORLD'S END 楽曲のレコードを返します。
func (s *userUsecase) GetUserWorldsendSongRecord(ctx context.Context, username string, requester *entity.User, displayID string, includeNoPlay bool) (*UserWorldsendSongRecordOutput, error) {
	user, err := s.getAccessibleUser(ctx, username, requester)
	if err != nil {
		return nil, err
	}

	songChart, err := s.worldsendChartRepo.FindByDisplayID(ctx, s.db, displayID)
	if err != nil {
		if errors.Is(err, repository.ErrSongNotFound) {
			return nil, repository.ErrSongNotFound
		}
		return nil, err
	}
	if songChart == nil || songChart.Song == nil || songChart.Chart == nil ||
		songChart.Song.IsDeleted || !songChart.Song.IsWorldsend {
		return nil, repository.ErrSongNotFound
	}

	response := &UserWorldsendSongRecordOutput{Meta: &UserSongRecordMetaOutput{}}
	if !user.HasLinkedPlayer() {
		return response, nil
	}

	records, err := s.worldsendRecordRepo.FindByPlayerIDAndSongDisplayID(ctx, s.db, *user.PlayerID, displayID)
	if err != nil {
		return nil, err
	}
	if includeNoPlay {
		records = s.recordCompletionSvc.CompleteWorldsendRecords(records, []*entity.WorldsendSongWithChart{songChart})
	}
	if len(records) == 0 {
		return response, nil
	}

	response.Worldsend = toWorldsendRecordOutput(records[0])
	if !records[0].UpdatedAt.IsZero() {
		value := records[0].UpdatedAt
		response.UpdatedAt = &value
		response.Meta.UpdatedAt = &value
	}
	return response, nil
}

func (s *userUsecase) resolveSongDifficulty(song *entity.Song, difficulty string) (*int, error) {
	if difficulty == "" {
		return nil, nil
	}
	normalized := strings.ToUpper(difficulty)
	switch normalized {
	case "BASIC", "ADVANCED", "EXPERT", "MASTER", "ULTIMA":
	default:
		return nil, ErrInvalidDifficulty
	}

	names, _ := s.songDifficultyMasters()
	for difficultyID, name := range names {
		if strings.ToUpper(name) == normalized && song.HasDifficultyChart(difficultyID) {
			return &difficultyID, nil
		}
	}
	return nil, ErrInvalidDifficulty
}

func (s *userUsecase) songDifficultyMasters() (map[int]string, map[int]int) {
	if s.masterProvider == nil || s.masterProvider.SongMasters() == nil {
		return nil, nil
	}
	masters := s.masterProvider.SongMasters()
	return masters.DifficultyNamesByID, masters.DifficultySortOrderByID()
}

func filterPlayerRecordsByDifficultyID(records []*entity.PlayerRecord, difficultyID int) []*entity.PlayerRecord {
	filtered := make([]*entity.PlayerRecord, 0, 1)
	for _, record := range records {
		if record != nil && record.Chart != nil && record.Chart.DifficultyID == difficultyID {
			filtered = append(filtered, record)
		}
	}
	return filtered
}

func (s *userUsecase) getUserProfilePlayerRecords(ctx context.Context, playerID int, includeNoPlay bool) (*userProfilePlayerRecords, error) {
	records, err := s.playerRecordRepo.FindByPlayerID(ctx, s.db, playerID)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			slog.Warn("failed to find player records due to context canceled", "player_id", playerID, "error", err)
		} else {
			slog.Error("failed to find player records", "player_id", playerID, "error", err)
		}
		return nil, err
	}

	allRecords := records
	markOPTargetPlayerRecords(records)
	if includeNoPlay {
		allRecords, err = s.completePlayerRecords(ctx, playerID, records)
		if err != nil {
			return nil, err
		}
	}

	slotMap := initializeSlotMap()
	allRecordDTOs := allRecords

	for _, record := range records {
		slotKey := record.SlotKey()
		if slotKey != "" {
			slotMap[slotKey] = append(slotMap[slotKey], record)
		}
	}

	return &userProfilePlayerRecords{
		all:             allRecordDTOs,
		slotMap:         slotMap,
		latestUpdatedAt: latestPlayerRecordUpdatedAt(records),
	}, nil
}

type opTargetCandidate struct {
	record       *entity.PlayerRecord
	overpower    float64
	difficultyID int
}

func markOPTargetPlayerRecords(records []*entity.PlayerRecord) {
	applyOPTargetFlags(records, calculateOPTargetChartIDs(records))
}

func applyOPTargetFlags(records []*entity.PlayerRecord, targetChartIDs map[int]struct{}) {
	for _, record := range records {
		if record == nil {
			continue
		}
		chartID := playerRecordChartID(record)
		_, record.IsOPTarget = targetChartIDs[chartID]
		if chartID == 0 {
			record.IsOPTarget = false
		}
	}
}

func calculateOPTargetChartIDs(records []*entity.PlayerRecord) map[int]struct{} {
	bestBySongID := make(map[int]opTargetCandidate, len(records))
	for _, record := range records {
		if record == nil {
			continue
		}
		if record.UpdatedAt.IsZero() || record.Song == nil || record.Chart == nil {
			continue
		}

		overpower := service.CalcSingleOverpower(uint32(record.Score), record.Chart.Const.Float64(), record.ComboLampID)
		candidate := opTargetCandidate{
			record:       record,
			overpower:    overpower,
			difficultyID: record.Chart.DifficultyID,
		}
		current, exists := bestBySongID[record.Song.ID]
		if !exists || candidate.overpower > current.overpower ||
			(candidate.overpower == current.overpower && candidate.difficultyID > current.difficultyID) {
			bestBySongID[record.Song.ID] = candidate
		}
	}

	targetChartIDs := make(map[int]struct{}, len(bestBySongID))
	for _, candidate := range bestBySongID {
		if chartID := playerRecordChartID(candidate.record); chartID != 0 {
			targetChartIDs[chartID] = struct{}{}
		}
	}
	return targetChartIDs
}

func playerRecordChartID(record *entity.PlayerRecord) int {
	if record == nil {
		return 0
	}
	if record.ChartID != 0 {
		return record.ChartID
	}
	if record.Chart != nil {
		return record.Chart.ID
	}
	return 0
}

func (s *userUsecase) completePlayerRecords(ctx context.Context, playerID int, records []*entity.PlayerRecord) ([]*entity.PlayerRecord, error) {
	if s.songRepo == nil {
		return records, nil
	}

	songs, err := s.songRepo.FindAllExcludingWorldsend(ctx, s.db, false)
	if err != nil {
		slog.Error("failed to find songs for no-play completion", "player_id", playerID, "error", err)
		return nil, err
	}

	var difficultyNamesByID map[int]string
	var difficultySortOrderByID map[int]int
	if s.masterProvider != nil {
		masters := s.masterProvider.SongMasters()
		if masters != nil {
			difficultyNamesByID = masters.DifficultyNamesByID
			difficultySortOrderByID = masters.DifficultySortOrderByID()
		}
	}

	return s.recordCompletionSvc.CompletePlayerRecords(records, songs, difficultyNamesByID, difficultySortOrderByID), nil
}

func (s *userUsecase) getUserProfileWorldsendRecords(ctx context.Context, playerID int, includeNoPlay bool) ([]*entity.PlayerWorldsendRecord, error) {
	if s.worldsendRecordRepo == nil {
		return []*entity.PlayerWorldsendRecord{}, nil
	}

	records, err := s.worldsendRecordRepo.FindByPlayerID(ctx, s.db, playerID)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			slog.Warn("failed to find worldsend records due to context canceled", "player_id", playerID, "error", err)
		} else {
			slog.Error("failed to find worldsend records", "player_id", playerID, "error", err)
		}
		return nil, err
	}

	if includeNoPlay {
		records, err = s.completeWorldsendRecords(ctx, playerID, records)
		if err != nil {
			return nil, err
		}
	}

	return records, nil
}

func (s *userUsecase) completeWorldsendRecords(ctx context.Context, playerID int, records []*entity.PlayerWorldsendRecord) ([]*entity.PlayerWorldsendRecord, error) {
	if s.worldsendChartRepo == nil {
		return records, nil
	}

	worldsendSongs, err := s.worldsendChartRepo.FindAll(ctx, s.db, false)
	if err != nil {
		slog.Error("failed to find worldsend songs for no-play completion", "player_id", playerID, "error", err)
		return nil, err
	}

	return s.recordCompletionSvc.CompleteWorldsendRecords(records, worldsendSongs), nil
}

func latestPlayerRecordUpdatedAt(records []*entity.PlayerRecord) time.Time {
	var latest time.Time
	for _, record := range records {
		if record != nil && record.UpdatedAt.After(latest) {
			latest = record.UpdatedAt
		}
	}
	return latest
}

func latestWorldsendRecordUpdatedAt(records []*entity.PlayerWorldsendRecord) time.Time {
	var latest time.Time
	for _, record := range records {
		if record == nil || record.UpdatedAt.IsZero() {
			continue
		}
		if record.UpdatedAt.After(latest) {
			latest = record.UpdatedAt
		}
	}
	return latest
}

func latestUserRecordUpdatedAt(values ...time.Time) time.Time {
	var latest time.Time
	for _, value := range values {
		if value.After(latest) {
			latest = value
		}
	}
	return latest
}

func (s *userUsecase) getUserProfileCourseRecords(ctx context.Context, playerID int, includeNoPlay bool) ([]*CourseRecordOutput, time.Time, error) {
	if s.courseRepo == nil {
		return []*CourseRecordOutput{}, time.Time{}, nil
	}
	records, err := s.courseRepo.FindRecordsByPlayerID(ctx, s.db, playerID, false, includeNoPlay)
	if err != nil {
		return nil, time.Time{}, err
	}
	result := make([]*CourseRecordOutput, 0, len(records))
	var latest time.Time
	for _, record := range records {
		result = append(result, toCourseRecordOutput(record))
		if record.UpdatedAt.After(latest) {
			latest = record.UpdatedAt
		}
	}
	return result, latest, nil
}

func buildPlayerOutput(playerWithHonors *repository.PlayerWithHonors) *UserPlayerOutput {
	return &UserPlayerOutput{Player: playerWithHonors.Player, Honors: playerWithHonors.Honors}
}

// initializeSlotMap はスロット別レコードを格納するmapを初期化します。
func initializeSlotMap() map[string][]*entity.PlayerRecord {
	slots := []string{"best", "best_candidate", "new", "new_candidate"}
	result := make(map[string][]*entity.PlayerRecord, len(slots))
	for _, slot := range slots {
		result[slot] = []*entity.PlayerRecord{}
	}
	return result
}

func initializeRatingSlotMap() map[string][]*entity.PlayerRecord {
	slots := []string{"best", "best_candidate", "new", "new_candidate"}
	result := make(map[string][]*entity.PlayerRecord, len(slots))
	for _, slot := range slots {
		result[slot] = []*entity.PlayerRecord{}
	}
	return result
}

func (s *userUsecase) getAccessibleUser(ctx context.Context, username string, requester *entity.User) (*entity.User, error) {
	user, err := s.userRepo.FindByUsername(ctx, s.db, username)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrUserNotFound
		}
		slog.Error("failed to find user by username", "username", username, "error", err)
		return nil, err
	}

	if user == nil {
		return nil, ErrUserNotFound
	}

	accessible, err := canAccessPrivateUser(ctx, s.db, s.friendshipRepo, user, requester)
	if err != nil {
		return nil, err
	}
	if !accessible {
		return nil, ErrUserPrivate
	}

	if !user.HasLinkedPlayer() {
		return user, nil
	}

	return user, nil
}

func (s *userUsecase) getOptionalPlayer(ctx context.Context, user *entity.User) (*UserPlayerOutput, error) {
	if user == nil || !user.HasLinkedPlayer() {
		return nil, nil
	}

	playerWithHonors, err := s.playerRepo.FindByIDWithHonors(ctx, s.db, *user.PlayerID)
	if err != nil {
		if errors.Is(err, repository.ErrPlayerNotFound) {
			return nil, nil
		}
		if errors.Is(err, context.Canceled) {
			slog.Warn("failed to find player due to context canceled", "player_id", *user.PlayerID, "error", err)
		} else {
			slog.Error("failed to find player", "player_id", *user.PlayerID, "error", err)
		}
		return nil, err
	}

	if playerWithHonors == nil || playerWithHonors.Player == nil {
		return nil, nil
	}

	player := buildPlayerOutput(playerWithHonors)
	if err := s.applyDynamicOverpowerPercent(ctx, player, *user.PlayerID); err != nil {
		return nil, err
	}
	return player, nil
}

func (s *userUsecase) applyDynamicOverpowerPercent(ctx context.Context, player *UserPlayerOutput, playerID int) error {
	if player == nil || player.Player == nil || player.Player.OverpowerValue == nil || s.overpowerDenominatorProvider == nil {
		return nil
	}

	snapshot, err := s.overpowerDenominatorProvider.Snapshot(ctx)
	if err != nil {
		return err
	}

	denominator := snapshot.GlobalTotal
	if s.playerLockedSongRepo != nil {
		lockedSongs, err := s.playerLockedSongRepo.ListByPlayerID(ctx, s.db, playerID)
		if err != nil {
			return err
		}
		type lockState struct {
			normal bool
			ultima bool
		}
		lockedMap := make(map[int]lockState, len(lockedSongs))
		for _, lockedSong := range lockedSongs {
			if lockedSong == nil {
				continue
			}
			state := lockedMap[lockedSong.SongID]
			if lockedSong.IsUltima {
				state.ultima = true
			} else {
				state.normal = true
			}
			lockedMap[lockedSong.SongID] = state
		}
		for songID, state := range lockedMap {
			if state.normal {
				maxOP, ok := snapshot.SongMaxOP[songID]
				if !ok {
					slog.Warn("applyDynamicOverpowerPercent: locked song not found in snapshot", "song_id", songID, "player_id", playerID)
					continue
				}
				denominator -= maxOP
			} else if state.ultima {
				maxOP, ok := snapshot.SongMaxOP[songID]
				if !ok {
					slog.Warn("applyDynamicOverpowerPercent: locked Ultima song not found in snapshot.SongMaxOP", "song_id", songID, "player_id", playerID)
					continue
				}
				maxOPWithoutUltima, ok := snapshot.SongMaxOPWithoutUltima[songID]
				if !ok {
					slog.Warn("applyDynamicOverpowerPercent: locked Ultima song not found in snapshot.SongMaxOPWithoutUltima", "song_id", songID, "player_id", playerID)
					continue
				}
				denominator -= maxOP - maxOPWithoutUltima
			}
		}
	}

	percent := service.CalcOverpowerPercent(*player.Player.OverpowerValue, denominator)
	player.Player.OverpowerPercent = &percent
	return nil
}
