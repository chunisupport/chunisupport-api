package usecase

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/username"
	"github.com/chunisupport/chunisupport-api/internal/info"
)

// FirebaseRegisterUsecase は Firebase ID トークンを使った新規ユーザー登録を扱います。
type FirebaseRegisterUsecase interface {
	// RegisterWithFirebase は Firebase ID トークンと希望ユーザー名で新規ユーザーを登録し、
	// 自動ログイン後の JWT トークンを返します。
	RegisterWithFirebase(ctx context.Context, idToken string, usernameStr string) (string, error)
}

type firebaseRegisterUsecase struct {
	tm            TransactionManager
	userRepo      repository.UserRepository
	tokenVerifier TokenVerifier
	sessionIssuer SessionIssuer
}

// NewFirebaseRegisterUsecase は Firebase 登録ユースケースを生成します。
func NewFirebaseRegisterUsecase(tm TransactionManager, userRepo repository.UserRepository, tokenVerifier TokenVerifier, sessionIssuer SessionIssuer) FirebaseRegisterUsecase {
	return &firebaseRegisterUsecase{
		tm:            tm,
		userRepo:      userRepo,
		tokenVerifier: tokenVerifier,
		sessionIssuer: sessionIssuer,
	}
}

func (u *firebaseRegisterUsecase) RegisterWithFirebase(ctx context.Context, idToken string, usernameStr string) (string, error) {
	idToken = strings.TrimSpace(idToken)
	if idToken == "" {
		return "", ErrInvalidIDToken
	}
	if u.tm == nil {
		return "", errors.Join(ErrInternalError, errors.New("transaction manager is nil"))
	}
	if u.tokenVerifier == nil {
		return "", errors.Join(ErrInternalError, errors.New("token verifier is nil"))
	}
	if u.sessionIssuer == nil {
		return "", errors.Join(ErrInternalError, errors.New("session issuer is nil"))
	}

	uid, err := u.tokenVerifier.VerifyIDToken(ctx, idToken)
	if err != nil {
		return "", err
	}
	uid = strings.TrimSpace(uid)
	if uid == "" {
		return "", errors.Join(ErrInternalError, errors.New("firebase uid is empty"))
	}

	un, err := username.NewUserName(usernameStr)
	if err != nil {
		return "", convertUsernameError(err)
	}

	var newUser *entity.User
	if err := u.tm.Transactional(ctx, func(tx repository.Executor) error {
		// Firebase UID が既存ユーザーに紐付いていないか確認
		if _, err := u.userRepo.FindByFirebaseUID(ctx, tx, uid); err == nil {
			return ErrFirebaseUIDAlreadyLinked
		} else if !errors.Is(err, repository.ErrUserNotFound) {
			return err
		}

		// ユーザー名が使用済みでないか確認
		if _, err := u.userRepo.FindByUsername(ctx, tx, un.String()); err == nil {
			return ErrUsernameTaken
		} else if !errors.Is(err, repository.ErrUserNotFound) {
			return err
		}

		newUser = entity.NewFirebaseUser(un, uid, info.AccountTypePlayer)
		if err := u.userRepo.Save(ctx, tx, newUser); err != nil {
			if errors.Is(err, repository.ErrDuplicateUsername) {
				return ErrUsernameTaken
			}
			if errors.Is(err, repository.ErrFirebaseUIDAlreadyLinked) {
				return ErrFirebaseUIDAlreadyLinked
			}
			return err
		}

		return nil
	}); err != nil {
		return "", err
	}

	token, err := u.sessionIssuer.IssueSession(ctx, newUser)
	if err != nil {
		slog.Error("Firebase登録後のセッション発行に失敗しました", "user_id", newUser.ID, "error", err)
		return "", err
	}

	return token, nil
}

var _ FirebaseRegisterUsecase = (*firebaseRegisterUsecase)(nil)
