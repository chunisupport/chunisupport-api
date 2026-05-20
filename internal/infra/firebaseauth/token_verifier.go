package firebaseauth

import (
	"context"
	"errors"
	"strings"
	"time"

	firebase "firebase.google.com/go/v4"
	firebaseauthsdk "firebase.google.com/go/v4/auth"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
)

type authClient interface {
	VerifyIDToken(ctx context.Context, idToken string) (*firebaseauthsdk.Token, error)
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
	token, err := v.verifyTokenWithRevocationCheck(ctx, idToken)
	if err != nil {
		return "", err
	}

	return normalizeUID(token.UID), nil
}

// VerifyRecentSignIn は Firebase ID トークンを検証し、UID と auth_time を返します。

// VerifyIDTokenWithoutRevocationCheck は Firebase ID トークンを失効確認なしで検証し、UID を返します。
func (v *tokenVerifier) VerifyIDTokenWithoutRevocationCheck(ctx context.Context, idToken string) (string, error) {
	token, err := v.verifyTokenWithoutRevocationCheck(ctx, idToken)
	if err != nil {
		return "", err
	}

	return normalizeUID(token.UID), nil
}

func (v *tokenVerifier) VerifyRecentSignIn(ctx context.Context, idToken string) (*usecase.RecentSignInInfo, error) {
	token, err := v.verifyTokenWithRevocationCheck(ctx, idToken)
	if err != nil {
		return nil, err
	}
	if token.AuthTime == 0 {
		return nil, errors.Join(usecase.ErrRecentSignInAuthTimeMissing, errors.New("firebase token auth_time is empty"))
	}

	return &usecase.RecentSignInInfo{
		UID:      normalizeUID(token.UID),
		AuthTime: time.Unix(token.AuthTime, 0).UTC(),
	}, nil
}

func (v *tokenVerifier) verifyTokenWithRevocationCheck(ctx context.Context, idToken string) (*firebaseauthsdk.Token, error) {
	if v.client == nil {
		return nil, errors.Join(usecase.ErrInternalError, errors.New("firebase auth client is nil"))
	}

	token, err := v.client.VerifyIDTokenAndCheckRevoked(ctx, idToken)
	if err != nil {
		if isFirebaseInvalidIDToken(err) || isFirebaseIDTokenRevoked(err) || isFirebaseUserDisabled(err) {
			return nil, errors.Join(usecase.ErrInvalidIDToken, err)
		}

		return nil, errors.Join(usecase.ErrInternalError, err)
	}

	if token == nil || normalizeUID(token.UID) == "" {
		return nil, errors.Join(usecase.ErrInternalError, errors.New("firebase token uid is empty"))
	}

	return token, nil
}

func normalizeUID(uid string) string {
	return strings.TrimSpace(uid)
}

var _ usecase.TokenVerifier = (*tokenVerifier)(nil)
var _ usecase.RecentSignInVerifier = (*tokenVerifier)(nil)

func (v *tokenVerifier) verifyTokenWithoutRevocationCheck(ctx context.Context, idToken string) (*firebaseauthsdk.Token, error) {
	if v.client == nil {
		return nil, errors.Join(usecase.ErrInternalError, errors.New("firebase auth client is nil"))
	}

	token, err := v.client.VerifyIDToken(ctx, idToken)
	if err != nil {
		if isFirebaseInvalidIDToken(err) {
			return nil, errors.Join(usecase.ErrInvalidIDToken, err)
		}

		return nil, errors.Join(usecase.ErrInternalError, err)
	}

	if token == nil || normalizeUID(token.UID) == "" {
		return nil, errors.Join(usecase.ErrInternalError, errors.New("firebase token uid is empty"))
	}

	return token, nil
}
