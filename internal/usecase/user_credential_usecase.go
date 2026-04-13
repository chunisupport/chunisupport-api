package usecase

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/info"
)

// UserCredentialUsecase は認証済みユーザー自身の資格情報・プロフィール設定管理を扱います。
type UserCredentialUsecase interface {
	GetUser(ctx context.Context, id int) (*api_internal.UserDTO, error)
	UpdatePrivacy(ctx context.Context, userID int, isPrivate bool) error
	DeleteOwnAccount(ctx context.Context, userID int, reauthToken string) error
}

type userCredentialUsecaseImpl struct {
	db                   repository.Executor
	tm                   TransactionManager
	userRepo             repository.UserRepository
	playerRecordRepo     repository.PlayerRecordRepository
	recentSignInVerifier RecentSignInVerifier
	firebaseDeleter      FirebaseUserDeleter
	masterCache          AccountTypeProvider
}

func NewUserCredentialUsecase(
	db repository.Executor,
	tm TransactionManager,
	userRepo repository.UserRepository,
	playerRecordRepo repository.PlayerRecordRepository,
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
		db:                   db,
		tm:                   tm,
		userRepo:             userRepo,
		playerRecordRepo:     playerRecordRepo,
		recentSignInVerifier: nil,
		firebaseDeleter:      noopFirebaseUserDeleter{},
		masterCache:          masterCache,
	}
}

// NewUserCredentialUsecaseWithFirebaseDeleter は Firebase 削除連携付きの UserCredentialUsecase を生成します。
func NewUserCredentialUsecaseWithFirebaseDeleter(
	db repository.Executor,
	tm TransactionManager,
	userRepo repository.UserRepository,
	playerRecordRepo repository.PlayerRecordRepository,
	firebaseDeleter FirebaseUserDeleter,
	masterCache AccountTypeProvider,
) UserCredentialUsecase {
	return NewUserCredentialUsecaseWithFirebaseServices(db, tm, userRepo, playerRecordRepo, nil, firebaseDeleter, masterCache)
}

// NewUserCredentialUsecaseWithFirebaseServices は recent sign-in 検証と Firebase 削除連携付きの UserCredentialUsecase を生成します。
func NewUserCredentialUsecaseWithFirebaseServices(
	db repository.Executor,
	tm TransactionManager,
	userRepo repository.UserRepository,
	playerRecordRepo repository.PlayerRecordRepository,
	recentSignInVerifier RecentSignInVerifier,
	firebaseDeleter FirebaseUserDeleter,
	masterCache AccountTypeProvider,
) UserCredentialUsecase {
	usecase := NewUserCredentialUsecase(db, tm, userRepo, playerRecordRepo, masterCache)
	impl, ok := usecase.(*userCredentialUsecaseImpl)
	if !ok {
		return usecase
	}
	if recentSignInVerifier != nil {
		impl.recentSignInVerifier = recentSignInVerifier
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

func (s *userCredentialUsecaseImpl) DeleteOwnAccount(ctx context.Context, userID int, reauthToken string) error {
	reauthInfo, err := s.verifyRecentSignIn(ctx, reauthToken)
	if err != nil {
		return err
	}

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

		firebaseUID := strings.TrimSpace(derefString(user.FirebaseUID))
		if firebaseUID == "" {
			return errors.Join(ErrRecentSignInRequired, ErrFirebaseUIDNotLinked)
		}
		if firebaseUID != reauthInfo.UID {
			return errors.Join(ErrRecentSignInRequired, ErrReauthUIDMismatch)
		}

		deletedUserID = user.ID
		deletedUsername = user.Username.String()
		deletedFirebaseUID = firebaseUID

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

func (s *userCredentialUsecaseImpl) verifyRecentSignIn(ctx context.Context, reauthToken string) (*RecentSignInInfo, error) {
	reauthToken = strings.TrimSpace(reauthToken)
	if reauthToken == "" {
		return nil, ErrRecentSignInRequired
	}
	if s.recentSignInVerifier == nil {
		return nil, errors.Join(ErrInternalError, errors.New("recent sign-in verifier is nil"))
	}

	reauthInfo, err := s.recentSignInVerifier.VerifyRecentSignIn(ctx, reauthToken)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidIDToken):
			return nil, errors.Join(ErrRecentSignInRequired, err)
		case errors.Is(err, ErrInternalError):
			return nil, err
		default:
			return nil, errors.Join(ErrInternalError, err)
		}
	}
	if reauthInfo == nil {
		return nil, errors.Join(ErrInternalError, errors.New("recent sign-in verifier returned nil info"))
	}

	reauthInfo.UID = strings.TrimSpace(reauthInfo.UID)
	if reauthInfo.UID == "" {
		return nil, errors.Join(ErrInternalError, errors.New("reauth token uid is empty"))
	}
	if reauthInfo.AuthTime.IsZero() {
		return nil, errors.Join(ErrRecentSignInRequired, errors.New("reauth token auth_time is empty"))
	}
	if reauthInfo.AuthTime.After(time.Now().Add(1 * time.Minute)) {
		return nil, errors.Join(ErrRecentSignInRequired, errors.New("reauth token auth_time is in the future"))
	}
	if time.Since(reauthInfo.AuthTime) > info.RecentSignInMaxAge {
		return nil, errors.Join(ErrRecentSignInRequired, ErrRecentSignInExpired)
	}

	return reauthInfo, nil
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}
