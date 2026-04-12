package usecase

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
)

// TokenVerifier は外部IDプロバイダのIDトークン検証を抽象化します。
type TokenVerifier interface {
	// VerifyIDToken はIDトークンを検証し、紐づくUIDを返します。
	VerifyIDToken(ctx context.Context, idToken string) (string, error)
}

// FirebaseAuthUsecase はFirebase IDトークンによる認証を扱います。
type FirebaseAuthUsecase interface {
	// Authenticate はFirebase IDトークンを検証し、紐づく有効ユーザーを返します。
	Authenticate(ctx context.Context, idToken string) (*entity.User, error)
	// AuthenticateOptional はFirebase IDトークンを検証し、未登録ユーザーなら匿名扱いにします。
	AuthenticateOptional(ctx context.Context, idToken string) (*entity.User, error)
}

type firebaseAuthUsecase struct {
	db            repository.Executor
	userRepo      repository.UserRepository
	tokenVerifier TokenVerifier
}

// NewFirebaseAuthUsecase は新しいFirebaseAuthUsecaseを生成します。
func NewFirebaseAuthUsecase(db repository.Executor, userRepo repository.UserRepository, tokenVerifier TokenVerifier) FirebaseAuthUsecase {
	return &firebaseAuthUsecase{
		db:            db,
		userRepo:      userRepo,
		tokenVerifier: tokenVerifier,
	}
}

// Authenticate はFirebase IDトークンを検証し、紐づくユーザーを返します。
func (u *firebaseAuthUsecase) Authenticate(ctx context.Context, idToken string) (*entity.User, error) {
	return u.authenticate(ctx, idToken, false)
}

// AuthenticateOptional はFirebase IDトークンを検証し、未登録ユーザーなら匿名扱いにします。
func (u *firebaseAuthUsecase) AuthenticateOptional(ctx context.Context, idToken string) (*entity.User, error) {
	return u.authenticate(ctx, idToken, true)
}

func (u *firebaseAuthUsecase) authenticate(ctx context.Context, idToken string, allowMissingUser bool) (*entity.User, error) {
	idToken = strings.TrimSpace(idToken)
	if idToken == "" {
		return nil, ErrInvalidIDToken
	}
	if u.tokenVerifier == nil {
		return nil, errors.Join(ErrInternalError, errors.New("token verifier is nil"))
	}

	// Firebase 側で invalid / revoked / disabled と判定されたトークンは、
	// DB 参照に進む前に tokenVerifier が ErrInvalidIDToken として拒否します。
	uid, err := u.tokenVerifier.VerifyIDToken(ctx, idToken)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidIDToken):
			return nil, err
		case errors.Is(err, ErrInternalError):
			return nil, err
		default:
			return nil, errors.Join(ErrInternalError, err)
		}
	}
	uid = strings.TrimSpace(uid)
	if uid == "" {
		return nil, errors.Join(ErrInternalError, errors.New("firebase uid is empty"))
	}

	user, err := u.userRepo.FindByFirebaseUID(ctx, u.db, uid)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			if allowMissingUser {
				return nil, nil
			}

			return nil, ErrInvalidIDToken
		}

		slog.Error("failed to find user by firebase uid", "firebase_uid", uid, "error", err)
		return nil, err
	}
	if user == nil {
		return nil, errors.Join(ErrInternalError, errors.New("user repository returned nil user"))
	}

	return user, nil
}
