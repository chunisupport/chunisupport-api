package usecase

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
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
	playerUsecase       PlayerUsecase
}

// NewUserService は UserUsecase の実装を生成します。
func NewUserService(db repository.Executor, userRepo repository.UserRepository, playerRecordRepo repository.PlayerRecordRepository, playerUsecase PlayerUsecase) UserUsecase {
	return &userUsecase{
		db:               db,
		userRepo:         userRepo,
		playerRecordRepo: playerRecordRepo,
		playerUsecase:    playerUsecase,
	}
}

// SetWorldsendRecordRepository は WorldsendRecordRepository を設定します。
// DI の都合上、後から設定できるようにしています。
func (s *userUsecase) SetWorldsendRecordRepository(repo repository.WorldsendRecordRepository) {
	s.worldsendRecordRepo = repo
}

// GetUserProfileWithRecords はユーザー名をキーにプロファイルとレコードを一括取得します。
// 対象ユーザーが非公開設定の場合は、本人以外は ErrUserPrivate を返します。
// プレイヤーが紐付いていない場合は ErrPlayerNotLinked を返します。
//
// TODO: 最適化の余地あり - 現在はユーザー→プレイヤー→称号→レコードで4回クエリを発行している。
// PlayerRepository.FindByIDWithHonors() のようなJOINクエリを作成して3回に削減できる可能性がある。
func (s *userUsecase) GetUserProfileWithRecords(ctx context.Context, username string, requester *entity.User) (*api_internal.UserProfileWithRecordsDTO, error) {
	user, player, err := s.getUserAndPlayer(ctx, username, requester)
	if err != nil {
		return nil, err
	}

	// 3. レコードを取得
	records, err := s.playerRecordRepo.FindByPlayerID(ctx, s.db, *user.PlayerID)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			slog.Warn("failed to find player records due to context canceled", "player_id", *user.PlayerID, "error", err)
		} else {
			slog.Error("failed to find player records", "player_id", *user.PlayerID, "error", err)
		}
		return nil, err
	}

	// スロット別にグルーピング
	slotMap := initializeSlotMap()
	var latestRecordUpdatedAt time.Time
	for _, record := range records {
		dtoRecord := dto.ToPlayerRecordDTO(record)
		slotMap["all"] = append(slotMap["all"], dtoRecord)

		slotKey := record.SlotKey()
		if slotKey != "" {
			slotMap[slotKey] = append(slotMap[slotKey], dtoRecord)
		}
		if record.UpdatedAt.After(latestRecordUpdatedAt) {
			latestRecordUpdatedAt = record.UpdatedAt
		}
	}

	// 4. WORLD'S END レコードを取得（通常レコードとは完全に独立）
	var worldsendRecords []*dto.WorldsendRecordDTO
	if s.worldsendRecordRepo != nil {
		weRecords, err := s.worldsendRecordRepo.FindByPlayerID(ctx, s.db, *user.PlayerID)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				slog.Warn("failed to find worldsend records due to context canceled", "player_id", *user.PlayerID, "error", err)
			} else {
				slog.Error("failed to find worldsend records", "player_id", *user.PlayerID, "error", err)
			}
			// WORLD'S END レコードの取得失敗は致命的ではないため、空配列で続行
			worldsendRecords = []*dto.WorldsendRecordDTO{}
		} else {
			worldsendRecords = make([]*dto.WorldsendRecordDTO, len(weRecords))
			for i, r := range weRecords {
				worldsendRecords[i] = dto.ToWorldsendRecordDTO(r)
			}
		}
	} else {
		worldsendRecords = []*dto.WorldsendRecordDTO{}
	}

	recordsUpdatedAt := latestRecordUpdatedAt
	if recordsUpdatedAt.IsZero() {
		recordsUpdatedAt = player.UpdatedAt
	}
	recordsDTO := &dto.UserRecordResponseDTO{
		UpdatedAt:     recordsUpdatedAt,
		Best:          slotMap["best"],
		BestCandidate: slotMap["best_candidate"],
		New:           slotMap["new"],
		NewCandidate:  slotMap["new_candidate"],
		All:           slotMap["all"],
		WorldsEnd:     worldsendRecords,
	}

	return &api_internal.UserProfileWithRecordsDTO{
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
	if requester == nil || requester.AccountTypeID < info.AccountTypeAdmin {
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
	if requester == nil || requester.AccountTypeID < info.AccountTypeAdmin {
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
