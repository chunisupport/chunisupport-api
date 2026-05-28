package usecase

import (
	"context"
	"errors"

	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	dto_internal "github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
)

// LoginUsecase はTurnstile検証後のFirebaseログインを扱います。
type LoginUsecase interface {
	// Login はTurnstileとFirebase IDトークンを検証し、紐づくユーザーを返します。
	Login(ctx context.Context, idToken string, turnstileToken string, remoteIP string) (*dto_internal.UserDTO, error)
}

type loginUsecase struct {
	authUsecase         FirebaseAuthUsecase
	turnstileVerifier   TurnstileVerifier
	accountTypeProvider AccountTypeProvider
}

// NewLoginUsecase はログイン用ユースケースを生成します。
func NewLoginUsecase(
	db repository.Executor,
	userRepo repository.UserRepository,
	tokenVerifier TokenVerifier,
	turnstileVerifier TurnstileVerifier,
	accountTypeProvider AccountTypeProvider,
) LoginUsecase {
	if userRepo == nil {
		panic("loginUsecase: UserRepository is nil")
	}
	if tokenVerifier == nil {
		panic("loginUsecase: TokenVerifier is nil")
	}
	if turnstileVerifier == nil {
		panic("loginUsecase: TurnstileVerifier is nil")
	}
	if accountTypeProvider == nil {
		panic("loginUsecase: AccountTypeProvider is nil")
	}

	return &loginUsecase{
		authUsecase:         NewFirebaseAuthUsecase(db, userRepo, tokenVerifier),
		turnstileVerifier:   turnstileVerifier,
		accountTypeProvider: accountTypeProvider,
	}
}

func (u *loginUsecase) Login(ctx context.Context, idToken string, turnstileToken string, remoteIP string) (*dto_internal.UserDTO, error) {
	if err := verifyTurnstile(ctx, u.turnstileVerifier, turnstileToken, remoteIP); err != nil {
		return nil, err
	}

	user, err := u.authUsecase.Authenticate(ctx, idToken)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.Join(ErrInternalError, errors.New("login auth usecase returned nil user"))
	}

	accountTypeName := u.accountTypeProvider.GetAccountTypeNameByID(user.AccountTypeID)
	return dto_internal.ToUserDTO(user, accountTypeName, user.IsPrivate, nil), nil
}

var _ LoginUsecase = (*loginUsecase)(nil)
