package usecase

import (
	"context"
	"errors"
	"strings"

	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
)

// FirebaseLinkUsecase はログイン済みユーザーへの Firebase UID 連携を扱います。
type FirebaseLinkUsecase interface {
	LinkFirebaseUID(ctx context.Context, userID int, idToken string) error
}

type firebaseLinkUsecase struct {
	tm            TransactionManager
	userRepo      repository.UserRepository
	tokenVerifier TokenVerifier
}

// NewFirebaseLinkUsecase は Firebase 連携ユースケースを生成します。
func NewFirebaseLinkUsecase(tm TransactionManager, userRepo repository.UserRepository, tokenVerifier TokenVerifier) FirebaseLinkUsecase {
	return &firebaseLinkUsecase{
		tm:            tm,
		userRepo:      userRepo,
		tokenVerifier: tokenVerifier,
	}
}

func (u *firebaseLinkUsecase) LinkFirebaseUID(ctx context.Context, userID int, idToken string) error {
	idToken = strings.TrimSpace(idToken)
	if idToken == "" {
		return ErrInvalidIDToken
	}
	if u.tm == nil {
		return errors.Join(ErrInternalError, errors.New("transaction manager is nil"))
	}
	if u.tokenVerifier == nil {
		return errors.Join(ErrInternalError, errors.New("token verifier is nil"))
	}

	uid, err := u.tokenVerifier.VerifyIDToken(ctx, idToken)
	if err != nil {
		return err
	}
	uid = strings.TrimSpace(uid)
	if uid == "" {
		return errors.Join(ErrInternalError, errors.New("firebase uid is empty"))
	}

	return u.tm.Transactional(ctx, func(tx repository.Executor) error {
		linkedUser, err := u.userRepo.FindByFirebaseUID(ctx, tx, uid)
		if err == nil {
			if linkedUser == nil {
				return errors.Join(ErrInternalError, errors.New("user repository returned nil user"))
			}
			if linkedUser.ID == userID {
				return nil
			}
			return ErrFirebaseUIDAlreadyLinked
		}
		if !errors.Is(err, repository.ErrUserNotFound) {
			return err
		}

		user, err := u.userRepo.FindByIDForUpdate(ctx, tx, userID)
		if err != nil {
			if errors.Is(err, repository.ErrUserNotFound) {
				return ErrUserNotFound
			}
			return err
		}
		if user == nil {
			return errors.Join(ErrInternalError, errors.New("user repository returned nil user"))
		}

		currentFirebaseUID := user.FirebaseUID
		if currentFirebaseUID != nil && *currentFirebaseUID != uid {
			return ErrFirebaseUIDAlreadyLinked
		}
		user.LinkFirebaseUID(uid)
		if err := u.userRepo.LinkFirebaseUID(ctx, tx, user.ID, currentFirebaseUID, uid, user.UpdatedAt); err != nil {
			if errors.Is(err, repository.ErrFirebaseUIDAlreadyLinked) {
				return ErrFirebaseUIDAlreadyLinked
			}
			return err
		}

		return nil
	})
}

var _ FirebaseLinkUsecase = (*firebaseLinkUsecase)(nil)
