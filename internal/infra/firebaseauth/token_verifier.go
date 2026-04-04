package firebaseauth

import (
	"context"
	"errors"

	firebase "firebase.google.com/go/v4"
	firebaseauthsdk "firebase.google.com/go/v4/auth"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
)

type authClient interface {
	VerifyIDTokenAndCheckRevoked(ctx context.Context, idToken string) (*firebaseauthsdk.Token, error)
}

var isFirebaseInvalidIDToken = firebaseauthsdk.IsIDTokenInvalid
var isFirebaseIDTokenRevoked = firebaseauthsdk.IsIDTokenRevoked
var isFirebaseUserDisabled = firebaseauthsdk.IsUserDisabled

type tokenVerifier struct {
	client authClient
}

// NewTokenVerifier は Firebase Admin SDK の auth.Client を使う TokenVerifier を生成します。
func NewTokenVerifier(client *firebaseauthsdk.Client) usecase.TokenVerifier {
	return &tokenVerifier{client: client}
}

// NewTokenVerifierFromApp は Firebase App から TokenVerifier を生成します。
func NewTokenVerifierFromApp(ctx context.Context, app *firebase.App) (usecase.TokenVerifier, error) {
	if app == nil {
		return nil, errors.New("firebase app is nil")
	}

	client, err := app.Auth(ctx)
	if err != nil {
		return nil, err
	}

	return &tokenVerifier{client: client}, nil
}

// VerifyIDToken は Firebase ID トークンを検証し、失効・無効化も含めて UID を返します。
func (v *tokenVerifier) VerifyIDToken(ctx context.Context, idToken string) (string, error) {
	if v.client == nil {
		return "", errors.Join(usecase.ErrInternalError, errors.New("firebase auth client is nil"))
	}

	token, err := v.client.VerifyIDTokenAndCheckRevoked(ctx, idToken)
	if err != nil {
		if isFirebaseInvalidIDToken(err) || isFirebaseIDTokenRevoked(err) || isFirebaseUserDisabled(err) {
			return "", errors.Join(usecase.ErrInvalidIDToken, err)
		}

		return "", errors.Join(usecase.ErrInternalError, err)
	}

	if token == nil || token.UID == "" {
		return "", errors.Join(usecase.ErrInternalError, errors.New("firebase token uid is empty"))
	}

	return token.UID, nil
}

var _ usecase.TokenVerifier = (*tokenVerifier)(nil)
