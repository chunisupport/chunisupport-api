package usecase

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/passwordhash"
	"github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/chunisupport/chunisupport-api/internal/utils"
)

// UserCredentialUsecase は認証済みユーザー自身の資格情報・プロフィール設定管理を扱います。
type UserCredentialUsecase interface {
	GetUser(ctx context.Context, id int) (*api_internal.UserDTO, error)
	UpdatePrivacy(ctx context.Context, userID int, isPrivate bool) error
	ChangePassword(ctx context.Context, userID int, currentPassword, newPassword string) error
	DeleteUser(ctx context.Context, userID int) error
}

type userCredentialUsecaseImpl struct {
	db               repository.Executor
	userRepo         repository.UserRepository
	playerRecordRepo repository.PlayerRecordRepository
	pepper           string
	masterCache      repository.AccountTypeMasterProvider
}

func NewUserCredentialUsecase(db repository.Executor, userRepo repository.UserRepository, playerRecordRepo repository.PlayerRecordRepository, pepper string, masterCache repository.AccountTypeMasterProvider) UserCredentialUsecase {
	return &userCredentialUsecaseImpl{db: db, userRepo: userRepo, playerRecordRepo: playerRecordRepo, pepper: pepper, masterCache: masterCache}
}

func (s *userCredentialUsecaseImpl) GetUser(ctx context.Context, id int) (*api_internal.UserDTO, error) {
	user, err := s.userRepo.FindByID(ctx, s.db, id)
	if err != nil {
		return nil, err
	}
	var lastScoreUpdate *time.Time
	if user.PlayerID != nil {
		lastScoreUpdate, err = s.playerRecordRepo.GetLastScoreUpdate(ctx, s.db, *user.PlayerID)
		if err != nil {
			slog.Error("failed to get last score update", "player_id", *user.PlayerID, "error", err)
		}
	}
	accountTypeName := s.masterCache.GetAccountTypeNameByID(user.AccountTypeID)
	return api_internal.ToUserDTO(user, accountTypeName, user.IsPrivate, lastScoreUpdate), nil
}

func (s *userCredentialUsecaseImpl) UpdatePrivacy(ctx context.Context, userID int, isPrivate bool) error {
	user, err := s.userRepo.FindByID(ctx, s.db, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrUserNotFound
		}
		return err
	}
	user.ChangePrivacy(isPrivate)
	return s.userRepo.Save(ctx, s.db, user)
}

func (s *userCredentialUsecaseImpl) ChangePassword(ctx context.Context, userID int, currentPassword, newPassword string) error {
	if len(newPassword) < info.PasswordMinLength {
		return ErrPasswordTooShort
	}
	if len(newPassword) > info.PasswordMaxLength {
		return ErrPasswordTooLong
	}
	user, err := s.userRepo.FindByID(ctx, s.db, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrUserNotFound
		}
		return err
	}
	if !utils.CheckPasswordHashWithPepper(currentPassword, s.pepper, user.PasswordHash.String()) {
		return ErrIncorrectPassword
	}
	if utils.CheckPasswordHashWithPepper(newPassword, s.pepper, user.PasswordHash.String()) {
		return ErrInvalidPassword
	}
	hashed, err := utils.HashPasswordWithPepper(newPassword, s.pepper)
	if err != nil {
		return err
	}
	newHash, err := passwordhash.NewPasswordHash(hashed)
	if err != nil {
		return err
	}
	user.ChangePassword(newHash)
	return s.userRepo.Save(ctx, s.db, user)
}

func (s *userCredentialUsecaseImpl) DeleteUser(ctx context.Context, userID int) error {
	user, err := s.userRepo.FindByID(ctx, s.db, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrUserNotFound
		}
		return err
	}
	user.Delete()
	return s.userRepo.Save(ctx, s.db, user)
}
