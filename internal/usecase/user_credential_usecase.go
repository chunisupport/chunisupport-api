package usecase

import (
	"context"
	"errors"
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
	DeleteOwnAccount(ctx context.Context, userID int) error
}

type userCredentialUsecaseImpl struct {
	db               repository.Executor
	tm               TransactionManager
	userRepo         repository.UserRepository
	playerRecordRepo repository.PlayerRecordRepository
	sessionRepo      repository.SessionRepository
	apiTokenRepo     repository.APITokenRepository
	recoveryCodeRepo repository.RecoveryCodeRepository
	pepper           string
	masterCache      AccountTypeProvider
}

func NewUserCredentialUsecase(
	db repository.Executor,
	tm TransactionManager,
	userRepo repository.UserRepository,
	playerRecordRepo repository.PlayerRecordRepository,
	sessionRepo repository.SessionRepository,
	apiTokenRepo repository.APITokenRepository,
	recoveryCodeRepo repository.RecoveryCodeRepository,
	pepper string,
	masterCache AccountTypeProvider,
) UserCredentialUsecase {
	if tm == nil {
		panic("transaction manager is nil")
	}
	if userRepo == nil {
		panic("user repository is nil")
	}
	if playerRecordRepo == nil {
		panic("player record repository is nil")
	}
	if sessionRepo == nil {
		panic("session repository is nil")
	}
	if apiTokenRepo == nil {
		panic("api token repository is nil")
	}
	if recoveryCodeRepo == nil {
		panic("recovery code repository is nil")
	}
	if masterCache == nil {
		panic("master cache is nil")
	}

	return &userCredentialUsecaseImpl{
		db:               db,
		tm:               tm,
		userRepo:         userRepo,
		playerRecordRepo: playerRecordRepo,
		sessionRepo:      sessionRepo,
		apiTokenRepo:     apiTokenRepo,
		recoveryCodeRepo: recoveryCodeRepo,
		pepper:           pepper,
		masterCache:      masterCache,
	}
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
			return nil, err
		}
	}
	accountTypeName := s.masterCache.GetAccountTypeNameByID(user.AccountTypeID)
	return api_internal.ToUserDTO(user, accountTypeName, user.IsPrivate, lastScoreUpdate), nil
}

func (s *userCredentialUsecaseImpl) UpdatePrivacy(ctx context.Context, userID int, isPrivate bool) error {
	user, err := s.userRepo.FindByID(ctx, s.db, userID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
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
		if errors.Is(err, repository.ErrUserNotFound) {
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

func (s *userCredentialUsecaseImpl) DeleteOwnAccount(ctx context.Context, userID int) error {
	return s.tm.Transactional(ctx, func(tx repository.Executor) error {
		user, err := s.userRepo.FindByIDForUpdate(ctx, tx, userID)
		if err != nil {
			if errors.Is(err, repository.ErrUserNotFound) {
				return ErrUserNotFound
			}
			return err
		}
		if !user.IsActive() {
			return ErrUserAlreadyDeleted
		}

		user.Delete()
		if err := s.userRepo.Save(ctx, tx, user); err != nil {
			return err
		}
		if err := s.sessionRepo.DeleteByUserID(ctx, tx, userID); err != nil {
			return err
		}
		if err := s.apiTokenRepo.DeleteByUserID(ctx, tx, userID); err != nil {
			return err
		}

		return s.recoveryCodeRepo.DeleteByUserID(ctx, tx, userID)
	})
}
