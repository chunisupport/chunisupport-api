package usecase

import (
	"context"
	"database/sql"
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
	db                  repository.Executor
	userRepo            repository.UserRepository
	playerRecordRepo    repository.PlayerRecordRepository
	worldsendRecordRepo repository.WorldsendRecordRepository
	songRepo            repository.SongRepository
	worldsendChartRepo  repository.WorldsendChartRepository
	recordCompletionSvc *service.RecordCompletionService
	songMasterProvider  repository.SongMasterProvider
	playerUsecase       PlayerUsecase
}

type userProfilePlayerRecords struct {
	all             []*dto.PlayerRecordDTO
	slotMap         map[string][]*dto.PlayerRecordDTO
	latestUpdatedAt time.Time
}

// NewUserService は UserUsecase の実装を生成します。
func NewUserService(db repository.Executor, userRepo repository.UserRepository, playerRecordRepo repository.PlayerRecordRepository, worldsendRecordRepo repository.WorldsendRecordRepository, playerUsecase PlayerUsecase, songRepo repository.SongRepository, worldsendChartRepo repository.WorldsendChartRepository, songMasterProvider repository.SongMasterProvider) UserUsecase {
	return &userUsecase{
		db:                  db,
		userRepo:            userRepo,
		playerRecordRepo:    playerRecordRepo,
		worldsendRecordRepo: worldsendRecordRepo,
		songRepo:            songRepo,
		worldsendChartRepo:  worldsendChartRepo,
		recordCompletionSvc: service.NewRecordCompletionService(),
		songMasterProvider:  songMasterProvider,
		playerUsecase:       playerUsecase,
	}
}

// GetUserProfileWithRecords はユーザー名をキーにプロファイルとレコードを一括取得します。
// 対象ユーザーが非公開設定の場合は、本人以外は ErrUserPrivate を返します。
// プレイヤーが紐付いていない場合は ErrPlayerNotLinked を返します。
//
// TODO: 最適化の余地あり - 現在はユーザー→プレイヤー→称号→レコードで4回クエリを発行している。
// PlayerRepository.FindByIDWithHonors() のようなJOINクエリを作成して3回に削減できる可能性がある。
func (s *userUsecase) GetUserProfileWithRecords(ctx context.Context, username string, requester *entity.User, includeNoPlay bool) (*api_internal.UserProfileWithRecordsDTO, error) {
	user, player, err := s.getUserAndPlayer(ctx, username, requester)
	if err != nil {
		return nil, err
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
	user, player, err := s.getUserAndPlayer(ctx, username, requester)
	if err != nil {
		return nil, err
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

// GetUserUpdatedAt はユーザー名をキーにプレイヤーデータの updated_at のみを取得します。
func (s *userUsecase) GetUserUpdatedAt(ctx context.Context, username string, requester *entity.User) (*api_internal.UserUpdatedAtDTO, error) {
	_, player, err := s.getUserAndPlayer(ctx, username, requester)
	if err != nil {
		return nil, err
	}

	return &api_internal.UserUpdatedAtDTO{
		UpdatedAt: player.UpdatedAt,
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
		resp := api_internal.AdminUserListResponse{
			UserName:  u.User.Username.String(),
			IsPrivate: u.User.IsPrivate,
			IsDeleted: u.User.IsDeleted,
		}
		if u.Player != nil {
			resp.PlayerName = u.Player.Name.String()
			resp.Rating = u.Player.OfficialRating
			resp.OverPowerValue = u.Player.OverpowerValue
		}
		responses = append(responses, resp)
	}

	return responses, nil
}

// DeleteUser はユーザーを論理削除します。
// 防御的深度: ハンドラ層のミドルウェアに加え、ユースケース層でもADMIN権限を検証します。
func (s *userUsecase) DeleteUser(ctx context.Context, requester *entity.User, username string) error {
	// 認可チェック: ADMIN権限が必要
	if requester == nil || !info.HasRole(requester.AccountTypeID, info.AccountTypeAdmin) {
		return ErrAdminRequired
	}

	// 1. ユーザーを取得
	user, err := s.userRepo.FindByUsername(ctx, s.db, username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrUserNotFound
		}
		slog.Error("failed to find user by username", "username", username, "error", err)
		return err
	}

	// 2. 既に削除済みかチェック
	if user.IsDeleted {
		return ErrUserAlreadyDeleted
	}

	// 3. 論理削除を実行
	user.Delete()
	if err := s.userRepo.Save(ctx, s.db, user); err != nil {
		slog.Error("failed to delete user", "user_id", user.ID, "error", err)
		return err
	}

	slog.Info("user deleted successfully", "username", username, "user_id", user.ID)
	return nil
}

// RestoreUser はユーザーを復活させます。
// 防御的深度: ハンドラ層のミドルウェアに加え、ユースケース層でもADMIN権限を検証します。
func (s *userUsecase) RestoreUser(ctx context.Context, requester *entity.User, username string) error {
	// 認可チェック: ADMIN権限が必要
	if requester == nil || !info.HasRole(requester.AccountTypeID, info.AccountTypeAdmin) {
		return ErrAdminRequired
	}

	// 1. ユーザーを取得
	user, err := s.userRepo.FindByUsername(ctx, s.db, username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrUserNotFound
		}
		slog.Error("failed to find user by username", "username", username, "error", err)
		return err
	}

	// 2. 削除されていない場合はエラー
	if !user.IsDeleted {
		return ErrUserNotDeleted
	}

	// 3. 復活を実行
	user.Restore()
	if err := s.userRepo.Save(ctx, s.db, user); err != nil {
		slog.Error("failed to restore user", "user_id", user.ID, "error", err)
		return err
	}

	slog.Info("user restored successfully", "username", username, "user_id", user.ID)
	return nil
}

// GetUserProfileRecordView はユーザー名をキーにレコード表示向けのプロファイルとレコードを取得します。
func (s *userUsecase) GetUserProfileRecordView(ctx context.Context, username string, requester *entity.User, includeNoPlay bool) (*api_internal.UserProfileRecordViewDTO, error) {
	user, player, err := s.getUserAndPlayer(ctx, username, requester)
	if err != nil {
		return nil, err
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
		dtoRecord := dto.ToPlayerRecordDTO(record)
		allRecordDTOs = append(allRecordDTOs, dtoRecord)
		slotMap["all"] = append(slotMap["all"], dtoRecord)
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
	if s.songMasterProvider != nil {
		masters := s.songMasterProvider.SongMasters()
		if masters != nil {
			difficultyNamesByID = masters.DifficultyNamesByID
		}
	}

	return s.recordCompletionSvc.CompletePlayerRecords(records, songs, difficultyNamesByID), nil
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

	pairs := make([]*service.WorldsendSongChartPair, 0, len(worldsendSongs))
	for _, worldsendSong := range worldsendSongs {
		if worldsendSong == nil {
			continue
		}
		pairs = append(pairs, &service.WorldsendSongChartPair{
			Song:  worldsendSong.Song,
			Chart: worldsendSong.Chart,
		})
	}

	return s.recordCompletionSvc.CompleteWorldsendRecords(records, pairs), nil
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

// initializeSlotMap はスロット別レコードを格納するmapを初期化します。
func initializeSlotMap() map[string][]*dto.PlayerRecordDTO {
	slots := []string{"best", "best_candidate", "new", "new_candidate"}
	result := make(map[string][]*dto.PlayerRecordDTO, len(slots)+1)
	for _, slot := range slots {
		result[slot] = []*dto.PlayerRecordDTO{}
	}
	result["all"] = []*dto.PlayerRecordDTO{}
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

func (s *userUsecase) getUserAndPlayer(ctx context.Context, username string, requester *entity.User) (*entity.User, *dto.PlayerDTO, error) {
	user, err := s.userRepo.FindByUsername(ctx, s.db, username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, ErrUserNotFound
		}
		slog.Error("failed to find user by username", "username", username, "error", err)
		return nil, nil, err
	}

	if user == nil || !user.IsActive() {
		return nil, nil, ErrUserNotFound
	}

	if user.IsPrivate && (requester == nil || requester.ID != user.ID) {
		return nil, nil, ErrUserPrivate
	}

	if !user.HasLinkedPlayer() {
		return nil, nil, ErrPlayerNotLinked
	}

	player, err := s.playerUsecase.GetPlayerByID(ctx, *user.PlayerID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, ErrPlayerNotLinked
		}
		if errors.Is(err, context.Canceled) {
			slog.Warn("failed to find player due to context canceled", "player_id", *user.PlayerID, "error", err)
		} else {
			slog.Error("failed to find player", "player_id", *user.PlayerID, "error", err)
		}
		return nil, nil, err
	}

	return user, player, nil
}
