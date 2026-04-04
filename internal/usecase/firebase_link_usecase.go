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
	db            repository.Executor
	userRepo      repository.UserRepository
	tokenVerifier TokenVerifier
}

// NewFirebaseLinkUsecase は Firebase 連携ユースケースを生成します。
func NewFirebaseLinkUsecase(db repository.Executor, userRepo repository.UserRepository, tokenVerifier TokenVerifier) FirebaseLinkUsecase {
	return &firebaseLinkUsecase{
		db:            db,
		userRepo:      userRepo,
		tokenVerifier: tokenVerifier,
	}
}

func (u *firebaseLinkUsecase) LinkFirebaseUID(ctx context.Context, userID int, idToken string) error {
	idToken = strings.TrimSpace(idToken)
	if idToken == "" {
		return ErrInvalidIDToken
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

	linkedUser, err := u.userRepo.FindByFirebaseUID(ctx, u.db, uid)
	if err == nil {
		if linkedUser == nil {
			return errors.Join(ErrInternalError, errors.New("user repository returned nil user"))
		}
		if linkedUser.ID == userID && !linkedUser.IsActive() {
			return ErrUserDeleted
		}
		if !linkedUser.IsActive() {
			return ErrFirebaseUIDAlreadyLinked
		}
		if linkedUser.ID == userID {
			return nil
		}
		return ErrFirebaseUIDAlreadyLinked
	}
	if !errors.Is(err, repository.ErrUserNotFound) {
		return err
	}

	user, err := u.userRepo.FindByID(ctx, u.db, userID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	if !user.IsActive() {
		return ErrUserDeleted
	}
	if user.FirebaseUID != nil && *user.FirebaseUID == uid {
		return nil
	}

	user.LinkFirebaseUID(uid)
	if err := u.userRepo.Save(ctx, u.db, user); err != nil {
		if errors.Is(err, repository.ErrFirebaseUIDAlreadyLinked) {
			return ErrFirebaseUIDAlreadyLinked
		}
		return err
	}

	return nil
}

var _ FirebaseLinkUsecase = (*firebaseLinkUsecase)(nil)
