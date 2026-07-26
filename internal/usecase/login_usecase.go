package usecase

import (
	"context"
	"errors"

	"github.com/chunisupport/chunisupport-api/internal/info"
)

// LoginUsecase はTurnstile検証後のFirebaseログインを扱います。
type LoginUsecase interface {
	// Login はTurnstileとFirebase IDトークンを検証し、紐づくユーザーを返します。
	Login(ctx context.Context, idToken string, turnstileToken string, remoteIP string) (*UserOutput, error)
}

type loginUsecase struct {
	authUsecase         FirebaseAuthUsecase
	turnstileVerifier   TurnstileVerifier
	accountTypeProvider AccountTypeProvider
	maintenanceProvider MaintenanceStateProvider
}

// NewLoginUsecase はログイン用ユースケースを生成します。
func NewLoginUsecase(
	authUsecase FirebaseAuthUsecase,
	turnstileVerifier TurnstileVerifier,
	accountTypeProvider AccountTypeProvider,
	maintenanceProvider MaintenanceStateProvider,
) LoginUsecase {
	if authUsecase == nil {
		panic("loginUsecase: FirebaseAuthUsecase is nil")
	}
	if turnstileVerifier == nil {
		panic("loginUsecase: TurnstileVerifier is nil")
	}
	if accountTypeProvider == nil {
		panic("loginUsecase: AccountTypeProvider is nil")
	}
	if maintenanceProvider == nil {
		panic("loginUsecase: MaintenanceStateProvider is nil")
	}

	return &loginUsecase{
		authUsecase:         authUsecase,
		turnstileVerifier:   turnstileVerifier,
		accountTypeProvider: accountTypeProvider,
		maintenanceProvider: maintenanceProvider,
	}
}

func (u *loginUsecase) Login(ctx context.Context, idToken string, turnstileToken string, remoteIP string) (*UserOutput, error) {
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
	if u.maintenanceProvider.Current().Enabled && !info.HasRole(user.AccountTypeID, info.AccountTypeEditor) {
		return nil, ErrMaintenanceMode
	}

	accountTypeName := u.accountTypeProvider.GetAccountTypeNameByID(user.AccountTypeID)
	return &UserOutput{Username: user.Username.String(), AccountType: accountTypeName, IsPrivate: user.IsPrivate}, nil
}

var _ LoginUsecase = (*loginUsecase)(nil)
