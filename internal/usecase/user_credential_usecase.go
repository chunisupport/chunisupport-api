package usecase

import (
	"context"
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
	DeleteOwnAccount(ctx context.Context, userID int) error
}

type userCredentialUsecaseImpl struct {
	db               repository.Executor
	tm               TransactionManager
	userRepo         repository.UserRepository
	playerRecordRepo repository.PlayerRecordRepository
	firebaseDeleter  FirebaseUserDeleter
	pepper           string
	masterCache      AccountTypeProvider
}

func NewUserCredentialUsecase(
	db repository.Executor,
	tm TransactionManager,
	userRepo repository.UserRepository,
	playerRecordRepo repository.PlayerRecordRepository,
	pepper string,
	masterCache AccountTypeProvider,
) UserCredentialUsecase {
	if db == nil {
		panic("executor is nil")
	}
	if tm == nil {
		panic("transaction manager is nil")
	}
	if userRepo == nil {
		panic("user repository is nil")
	}
	if playerRecordRepo == nil {
		panic("player record repository is nil")
	}
	if masterCache == nil {
		panic("master cache is nil")
	}

	return &userCredentialUsecaseImpl{
		db:               db,
		tm:               tm,
		userRepo:         userRepo,
		playerRecordRepo: playerRecordRepo,
		firebaseDeleter:  noopFirebaseUserDeleter{},
		pepper:           pepper,
		masterCache:      masterCache,
	}
}

// NewUserCredentialUsecaseWithFirebaseDeleter は Firebase 削除連携付きの UserCredentialUsecase を生成します。
func NewUserCredentialUsecaseWithFirebaseDeleter(
	db repository.Executor,
	tm TransactionManager,
	userRepo repository.UserRepository,
	playerRecordRepo repository.PlayerRecordRepository,
	firebaseDeleter FirebaseUserDeleter,
	pepper string,
	masterCache AccountTypeProvider,
) UserCredentialUsecase {
	usecase := NewUserCredentialUsecase(db, tm, userRepo, playerRecordRepo, pepper, masterCache)
	impl, ok := usecase.(*userCredentialUsecaseImpl)
	if !ok {
		return usecase
	}
	if firebaseDeleter != nil {
		impl.firebaseDeleter = firebaseDeleter
	}
	return impl
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
	var deletedUserID int
	var deletedUsername string
	var deletedFirebaseUID string

	if err := s.tm.Transactional(ctx, func(tx repository.Executor) error {
		user, err := s.userRepo.FindByIDForUpdate(ctx, tx, userID)
		if err != nil {
			if errors.Is(err, repository.ErrUserNotFound) {
				return ErrUserNotFound
			}
			return err
		}
		deletedUserID = user.ID
		deletedUsername = user.Username.String()
		if user.FirebaseUID != nil {
			deletedFirebaseUID = *user.FirebaseUID
		}

		return s.userRepo.DeleteByID(ctx, tx, userID)
	}); err != nil {
		return err
	}

	if deletedFirebaseUID != "" {
		if err := s.firebaseDeleter.DeleteUser(ctx, deletedFirebaseUID); err != nil {
			slog.Error("failed to delete firebase user after account deletion", "user_id", deletedUserID, "username", deletedUsername, "firebase_uid", deletedFirebaseUID, "error", err)
		}
	}

	return nil
}
