package usecase

import (
	"context"
	"errors"
)

// FirebaseLoginUsecase は Firebase ID トークンによるログインを扱います。
type FirebaseLoginUsecase interface {
	LoginWithFirebase(ctx context.Context, idToken string) (string, error)
}

type firebaseLoginUsecase struct {
	firebaseAuthUsecase FirebaseAuthUsecase
	sessionIssuer       SessionIssuer
}

// NewFirebaseLoginUsecase は Firebase ログインユースケースを生成します。
func NewFirebaseLoginUsecase(firebaseAuthUsecase FirebaseAuthUsecase, sessionIssuer SessionIssuer) FirebaseLoginUsecase {
	return &firebaseLoginUsecase{
		firebaseAuthUsecase: firebaseAuthUsecase,
		sessionIssuer:       sessionIssuer,
	}
}

func (u *firebaseLoginUsecase) LoginWithFirebase(ctx context.Context, idToken string) (string, error) {
	if u.firebaseAuthUsecase == nil {
		return "", errors.Join(ErrInternalError, errors.New("firebase auth usecase is nil"))
	}
	if u.sessionIssuer == nil {
		return "", errors.Join(ErrInternalError, errors.New("session issuer is nil"))
	}

	user, err := u.firebaseAuthUsecase.Authenticate(ctx, idToken)
	if err != nil {
		return "", err
	}

	return u.sessionIssuer.IssueSession(ctx, user)
}

var _ FirebaseLoginUsecase = (*firebaseLoginUsecase)(nil)
