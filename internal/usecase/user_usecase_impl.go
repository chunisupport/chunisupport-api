package usecase

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/service"
	"github.com/chunisupport/chunisupport-api/internal/dto"
	"github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/info"
)

// userUsecase は UserUsecase の実装です。
type userUsecase struct {
	db                           repository.Executor
	userRepo                     repository.UserRepository
	playerRepo                   repository.PlayerRepository
	playerRecordRepo             repository.PlayerRecordRepository
	worldsendRecordRepo          repository.WorldsendRecordRepository
	songRepo                     repository.SongRepository
	worldsendChartRepo           repository.WorldsendChartRepository
	playerLockedSongRepo         repository.PlayerLockedSongRepository
	overpowerDenominatorProvider repository.OverpowerDenominatorProvider
	recordCompletionSvc          *service.RecordCompletionService
	masterProvider               userMasterProvider
	firebaseDeleter              FirebaseUserDeleter
}

type userMasterProvider interface {
	repository.SongMasterProvider
	repository.AccountTypeMasterProvider
}

type userProfilePlayerRecords struct {
	all             []*dto.PlayerRecordDTO
	slotMap         map[string][]*dto.PlayerRecordDTO
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
func NewUserUsecaseWithFirebaseDeleterAndOverpowerDenominator(db repository.Executor, userRepo repository.UserRepository, playerRepo repository.PlayerRepository, playerRecordRepo repository.PlayerRecordRepository, worldsendRecordRepo repository.WorldsendRecordRepository, songRepo repository.SongRepository, worldsendChartRepo repository.WorldsendChartRepository, masterProvider userMasterProvider, firebaseDeleter FirebaseUserDeleter, playerLockedSongRepo repository.PlayerLockedSongRepository, overpowerDenominatorProvider repository.OverpowerDenominatorProvider) UserUsecase {
	usecase := NewUserUsecaseWithFirebaseDeleter(db, userRepo, playerRepo, playerRecordRepo, worldsendRecordRepo, songRepo, worldsendChartRepo, masterProvider, firebaseDeleter)
	impl, ok := usecase.(*userUsecase)
	if !ok {
		return usecase
	}
	impl.playerLockedSongRepo = playerLockedSongRepo
	impl.overpowerDenominatorProvider = overpowerDenominatorProvider
	return impl
}

// GetUserProfile はユーザー名をキーにプロファイル（username + player）を軽量に取得します。
// 対象ユーザーが非公開設定の場合は、本人以外は ErrUserPrivate を返します。
func (s *userUsecase) GetUserProfile(ctx context.Context, username string, requester *entity.User) (*api_internal.UserProfileDTO, error) {
	user, err := s.getAccessibleUser(ctx, username, requester)
	if err != nil {
		return nil, err
	}
	player, err := s.getOptionalPlayer(ctx, user)
	if err != nil {
		return nil, err
	}
	return &api_internal.UserProfileDTO{
		Username: user.Username.String(),
		Player:   player,
	}, nil
}

// GetUserUpdatedAt はユーザーのプロフィールとレコードの updated_at のうち新しい方を返します。
func (s *userUsecase) GetUserUpdatedAt(ctx context.Context, username string, requester *entity.User) (*api_internal.UserUpdatedAtDTO, error) {
	user, err := s.getAccessibleUser(ctx, username, requester)
	if err != nil {
		return nil, err
	}
	player, err := s.getOptionalPlayer(ctx, user)
	if err != nil {
		return nil, err
	}
	if player == nil || user.PlayerID == nil {
		return &api_internal.UserUpdatedAtDTO{
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

	return &api_internal.UserUpdatedAtDTO{
		UpdatedAt: &latestUpdatedAt,
	}, nil
}

// GetUserProfileWithRecords はユーザー名をキーにプロファイルとレコードを一括取得します。
// 対象ユーザーが非公開設定の場合は、本人以外は ErrUserPrivate を返します。
func (s *userUsecase) GetUserProfileWithRecords(ctx context.Context, username string, requester *entity.User, includeNoPlay bool) (*api_internal.UserProfileWithRecordsDTO, error) {
	user, err := s.getAccessibleUser(ctx, username, requester)
	if err != nil {
		return nil, err
	}
	player, err := s.getOptionalPlayer(ctx, user)
	if err != nil {
		return nil, err
	}
	if player == nil {
		return &api_internal.UserProfileWithRecordsDTO{
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

	recordsUpdatedAt := latestUserRecordUpdatedAt(playerRecords.latestUpdatedAt, latestWorldsendRecordUpdatedAt(worldsendRecords))
	if recordsUpdatedAt.IsZero() {
		recordsUpdatedAt = player.UpdatedAt
	}
	recordsDTO := &dto.UserRecordResponseDTO{
		UpdatedAt:     recordsUpdatedAt,
		Best:          playerRecords.slotMap["best"],
		BestCandidate: playerRecords.slotMap["best_candidate"],
		New:           playerRecords.slotMap["new"],
		NewCandidate:  playerRecords.slotMap["new_candidate"],
		All:           playerRecords.all,
		WorldsEnd:     worldsendRecords,
	}

	return &api_internal.UserProfileWithRecordsDTO{
		UserID:    user.ID,
		Username:  user.Username.String(),
		Player:    player,
		Records:   recordsDTO,
		UpdatedAt: &player.UpdatedAt,
	}, nil
}

// GetUserProfileRatingView はユーザー名をキーにレーティング表示向けのプロファイルとレコードを取得します。
func (s *userUsecase) GetUserProfileRatingView(ctx context.Context, username string, requester *entity.User) (*api_internal.UserProfileRatingViewDTO, error) {
	user, err := s.getAccessibleUser(ctx, username, requester)
	if err != nil {
		return nil, err
	}
	player, err := s.getOptionalPlayer(ctx, user)
	if err != nil {
		return nil, err
	}
	if player == nil {
		return &api_internal.UserProfileRatingViewDTO{
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

	slotMap := initializeRatingSlotMap()
	var latestRecordUpdatedAt time.Time
	for _, record := range records {
		dtoRecord := dto.ToPlayerRecordDTO(record)

		slotKey := record.SlotKey()
		if slotKey != "" {
			slotMap[slotKey] = append(slotMap[slotKey], dtoRecord)
		}
		if record.UpdatedAt.After(latestRecordUpdatedAt) {
			latestRecordUpdatedAt = record.UpdatedAt
		}
	}

	recordsUpdatedAt := latestRecordUpdatedAt
	if recordsUpdatedAt.IsZero() {
		recordsUpdatedAt = player.UpdatedAt
	}
	recordsDTO := &api_internal.UserRatingRecordResponseDTO{
		UpdatedAt:     recordsUpdatedAt,
		Best:          slotMap["best"],
		BestCandidate: slotMap["best_candidate"],
		New:           slotMap["new"],
		NewCandidate:  slotMap["new_candidate"],
	}

	return &api_internal.UserProfileRatingViewDTO{
		Username:  user.Username.String(),
		Player:    player,
		Records:   recordsDTO,
		UpdatedAt: &player.UpdatedAt,
	}, nil
}

// GetAllUsersForAdmin はADMIN用にすべてのユーザー一覧を取得します。
// プライベート・削除済み・プレイヤー未紐付けアカウントを含みます。
func (s *userUsecase) GetAllUsersForAdmin(ctx context.Context, page int, limit int, name string) ([]api_internal.AdminUserListResponse, error) {
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

	responses := make([]api_internal.AdminUserListResponse, 0, len(users))
	for _, u := range users {
		accountTypeName := "UNKNOWN"
		if s.masterProvider != nil {
			accountTypeName = s.masterProvider.GetAccountTypeNameByID(u.User.AccountTypeID)
		}

		resp := api_internal.AdminUserListResponse{
			UserName:     u.User.Username.String(),
			AccountType:  accountTypeName,
			CreatedAt:    u.User.CreatedAt,
			UpdatedAt:    u.User.UpdatedAt,
			IsSuspicious: u.User.IsSuspicious,
			IsPrivate:    u.User.IsPrivate,
			FirebaseUID:  u.User.FirebaseUID,
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
	if err := s.ensureAdminPermission(requester); err != nil {
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

	if err := s.ensureNotLastAdminRemoval(ctx, user); err != nil {
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

// ChangeUserAccountType はADMIN操作としてユーザー権限を変更します。
// ハンドラの認可ミドルウェアだけに依存しないよう、ユースケースでもADMIN権限を検証します。
func (s *userUsecase) ChangeUserAccountType(ctx context.Context, requester *entity.User, username string, accountType string) (*entity.User, error) {
	if err := s.ensureAdminPermission(requester); err != nil {
		return nil, err
	}

	accountTypeID, ok := info.AccountTypeIDByName(accountType)
	if !ok {
		return nil, ErrInvalidAccountType
	}

	user, err := s.userRepo.FindByUsername(ctx, s.db, username)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrUserNotFound
		}
		slog.Error("failed to find user by username for account type change", "username", username, "error", err)
		return nil, err
	}

	if err := s.ensureNotLastAdminDemotion(ctx, user, accountTypeID); err != nil {
		return nil, err
	}

	if err := user.ChangeAccountType(accountTypeID); err != nil {
		return nil, err
	}

	if err := s.userRepo.Save(ctx, s.db, user); err != nil {
		slog.Error("failed to save user account type", "username", username, "error", err)
		return nil, err
	}

	return user, nil
}

func (s *userUsecase) ensureAdminPermission(requester *entity.User) error {
	if requester == nil || !info.HasRole(requester.AccountTypeID, info.AccountTypeAdmin) {
		return ErrAdminRequired
	}
	return nil
}

func (s *userUsecase) ensureNotLastAdminDemotion(ctx context.Context, user *entity.User, nextAccountTypeID int) error {
	if user.AccountTypeID != info.AccountTypeAdmin || nextAccountTypeID == info.AccountTypeAdmin {
		return nil
	}
	return s.ensureAdminCountIsNotOne(ctx)
}

func (s *userUsecase) ensureNotLastAdminRemoval(ctx context.Context, user *entity.User) error {
	if user.AccountTypeID != info.AccountTypeAdmin {
		return nil
	}
	return s.ensureAdminCountIsNotOne(ctx)
}

func (s *userUsecase) ensureAdminCountIsNotOne(ctx context.Context) error {
	// このアプリでは最初のADMINをDB直接編集で付与する運用を許容しており、
	// 万一ADMINが0人になってもDBから復旧できる規模のため、
	// TransactionManagerや行ロックまでは導入せず、通常操作での「1人→0人」だけを防ぎます。
	adminCount, err := s.userRepo.CountByAccountType(ctx, s.db, info.AccountTypeAdmin)
	if err != nil {
		slog.Error("failed to count admin users", "error", err)
		return err
	}
	if adminCount == 1 {
		return ErrLastAdminRequired
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
func (s *userUsecase) GetUserProfileRecordView(ctx context.Context, username string, requester *entity.User, includeNoPlay bool) (*api_internal.UserProfileRecordViewDTO, error) {
	user, err := s.getAccessibleUser(ctx, username, requester)
	if err != nil {
		return nil, err
	}
	player, err := s.getOptionalPlayer(ctx, user)
	if err != nil {
		return nil, err
	}
	if player == nil {
		return &api_internal.UserProfileRecordViewDTO{
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

	recordsUpdatedAt := latestUserRecordUpdatedAt(playerRecords.latestUpdatedAt, latestWorldsendRecordUpdatedAt(worldsendRecords))
	if recordsUpdatedAt.IsZero() {
		recordsUpdatedAt = player.UpdatedAt
	}

	return &api_internal.UserProfileRecordViewDTO{
		Username: user.Username.String(),
		Player:   player,
		Records: &api_internal.UserRecordViewResponseDTO{
			UpdatedAt: recordsUpdatedAt,
			All:       playerRecords.all,
			Worldsend: worldsendRecords,
		},
		UpdatedAt: &player.UpdatedAt,
	}, nil
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
	if includeNoPlay {
		allRecords, err = s.completePlayerRecords(ctx, playerID, records)
		if err != nil {
			return nil, err
		}
	}

	slotMap := initializeSlotMap()
	allRecordDTOs := make([]*dto.PlayerRecordDTO, 0, len(allRecords))
	for _, record := range allRecords {
		allRecordDTOs = append(allRecordDTOs, dto.ToPlayerRecordDTO(record))
	}

	for _, record := range records {
		dtoRecord := dto.ToPlayerRecordDTO(record)
		slotKey := record.SlotKey()
		if slotKey != "" {
			slotMap[slotKey] = append(slotMap[slotKey], dtoRecord)
		}
	}

	return &userProfilePlayerRecords{
		all:             allRecordDTOs,
		slotMap:         slotMap,
		latestUpdatedAt: latestPlayerRecordUpdatedAt(records),
	}, nil
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

func (s *userUsecase) getUserProfileWorldsendRecords(ctx context.Context, playerID int, includeNoPlay bool) ([]*dto.WorldsendRecordDTO, error) {
	if s.worldsendRecordRepo == nil {
		return []*dto.WorldsendRecordDTO{}, nil
	}

	records, err := s.worldsendRecordRepo.FindByPlayerID(ctx, s.db, playerID)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			slog.Warn("failed to find worldsend records due to context canceled", "player_id", playerID, "error", err)
		} else {
			slog.Error("failed to find worldsend records", "player_id", playerID, "error", err)
		}
		return []*dto.WorldsendRecordDTO{}, nil
	}

	if includeNoPlay {
		records, err = s.completeWorldsendRecords(ctx, playerID, records)
		if err != nil {
			return nil, err
		}
	}

	worldsendRecords := make([]*dto.WorldsendRecordDTO, len(records))
	for i, record := range records {
		worldsendRecords[i] = dto.ToWorldsendRecordDTO(record)
	}
	return worldsendRecords, nil
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

func latestWorldsendRecordUpdatedAt(records []*dto.WorldsendRecordDTO) time.Time {
	var latest time.Time
	for _, record := range records {
		if record == nil || record.UpdatedAt == nil {
			continue
		}
		if record.UpdatedAt.After(latest) {
			latest = *record.UpdatedAt
		}
	}
	return latest
}

func latestUserRecordUpdatedAt(playerRecordsUpdatedAt time.Time, worldsendRecordsUpdatedAt time.Time) time.Time {
	if worldsendRecordsUpdatedAt.After(playerRecordsUpdatedAt) {
		return worldsendRecordsUpdatedAt
	}
	return playerRecordsUpdatedAt
}

func buildPlayerDTO(playerWithHonors *repository.PlayerWithHonors) *dto.PlayerDTO {
	playerDTO := dto.ToPlayerDTO(playerWithHonors.Player)
	honors := make([]*dto.HonorDTO, len(playerWithHonors.Honors))
	for i, honor := range playerWithHonors.Honors {
		honors[i] = &dto.HonorDTO{
			Slot:     honor.Slot,
			Name:     honor.Name,
			TypeName: honor.TypeName,
			ImageURL: derefString(honor.ImageURL),
		}
	}
	playerDTO.Honors = honors
	return playerDTO
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

// initializeSlotMap はスロット別レコードを格納するmapを初期化します。
func initializeSlotMap() map[string][]*dto.PlayerRecordDTO {
	slots := []string{"best", "best_candidate", "new", "new_candidate"}
	result := make(map[string][]*dto.PlayerRecordDTO, len(slots))
	for _, slot := range slots {
		result[slot] = []*dto.PlayerRecordDTO{}
	}
	return result
}

func initializeRatingSlotMap() map[string][]*dto.PlayerRecordDTO {
	slots := []string{"best", "best_candidate", "new", "new_candidate"}
	result := make(map[string][]*dto.PlayerRecordDTO, len(slots))
	for _, slot := range slots {
		result[slot] = []*dto.PlayerRecordDTO{}
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

	if user.IsPrivate && (requester == nil || requester.ID != user.ID) {
		return nil, ErrUserPrivate
	}

	if !user.HasLinkedPlayer() {
		return user, nil
	}

	return user, nil
}

func (s *userUsecase) getOptionalPlayer(ctx context.Context, user *entity.User) (*dto.PlayerDTO, error) {
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

	player := buildPlayerDTO(playerWithHonors)
	if err := s.applyDynamicOverpowerPercent(ctx, player, *user.PlayerID); err != nil {
		return nil, err
	}
	return player, nil
}

func (s *userUsecase) applyDynamicOverpowerPercent(ctx context.Context, player *dto.PlayerDTO, playerID int) error {
	if player == nil || player.OverpowerValue == nil || s.overpowerDenominatorProvider == nil {
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

	percent := service.CalcOverpowerPercent(*player.OverpowerValue, denominator)
	player.OverpowerPercent = &percent
	return nil
}
