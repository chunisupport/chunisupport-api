package firebaseauth

import (
	"context"
	"errors"
	"testing"
	"time"

	firebaseauthsdk "firebase.google.com/go/v4/auth"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubAuthClient struct {
	token         *firebaseauthsdk.Token
	err           error
	receivedToken string
}

func (s *stubAuthClient) VerifyIDTokenAndCheckRevoked(_ context.Context, idToken string) (*firebaseauthsdk.Token, error) {
	s.receivedToken = idToken
	return s.token, s.err
}

func TestTokenVerifier_VerifyIDToken(t *testing.T) {
	originalIsFirebaseInvalidIDToken := isFirebaseInvalidIDToken
	originalIsFirebaseIDTokenRevoked := isFirebaseIDTokenRevoked
	originalIsFirebaseUserDisabled := isFirebaseUserDisabled
	t.Cleanup(func() {
		isFirebaseInvalidIDToken = originalIsFirebaseInvalidIDToken
		isFirebaseIDTokenRevoked = originalIsFirebaseIDTokenRevoked
		isFirebaseUserDisabled = originalIsFirebaseUserDisabled
	})

	invalidErr := errors.New("invalid token from firebase sdk")
	revokedErr := errors.New("revoked token from firebase sdk")
	disabledErr := errors.New("disabled user from firebase sdk")
	isFirebaseInvalidIDToken = func(err error) bool {
		return errors.Is(err, invalidErr)
	}
	isFirebaseIDTokenRevoked = func(err error) bool {
		return errors.Is(err, revokedErr)
	}
	isFirebaseUserDisabled = func(err error) bool {
		return errors.Is(err, disabledErr)
	}

	tests := []struct {
		name      string
		client    authClient
		idToken   string
		wantUID   string
		wantErr   error
		wantErrIn string
	}{
		{
			name:    "UID を返せる場合はそのまま返す",
			client:  &stubAuthClient{token: &firebaseauthsdk.Token{UID: "firebase-uid"}},
			idToken: "valid-token",
			wantUID: "firebase-uid",
		},
		{
			name:    "SDK が不正トークンエラーを返す場合は ErrInvalidIDToken を返す",
			client:  &stubAuthClient{err: invalidErr},
			idToken: "invalid-token",
			wantErr: usecase.ErrInvalidIDToken,
		},
		{
			name:    "SDK が失効済みトークンエラーを返す場合は ErrInvalidIDToken を返す",
			client:  &stubAuthClient{err: revokedErr},
			idToken: "revoked-token",
			wantErr: usecase.ErrInvalidIDToken,
		},
		{
			name:    "SDK が無効化済みユーザーエラーを返す場合は ErrInvalidIDToken を返す",
			client:  &stubAuthClient{err: disabledErr},
			idToken: "disabled-user-token",
			wantErr: usecase.ErrInvalidIDToken,
		},
		{
			name:      "SDK の内部エラーは ErrInternalError で返す",
			client:    &stubAuthClient{err: errors.New("verify failed")},
			idToken:   "internal-error-token",
			wantErr:   usecase.ErrInternalError,
			wantErrIn: "verify failed",
		},
		{
			name:      "UID が空なら ErrInternalError を返す",
			client:    &stubAuthClient{token: &firebaseauthsdk.Token{}},
			idToken:   "empty-uid-token",
			wantErr:   usecase.ErrInternalError,
			wantErrIn: "firebase token uid is empty",
		},
		{
			name:      "クライアントが nil なら ErrInternalError を返す",
			client:    nil,
			idToken:   "any-token",
			wantErr:   usecase.ErrInternalError,
			wantErrIn: "firebase auth client is nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			verifier := &tokenVerifier{client: tt.client}

			// When
			uid, err := verifier.VerifyIDToken(context.Background(), tt.idToken)

			// Then
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				if tt.wantErrIn != "" {
					assert.ErrorContains(t, err, tt.wantErrIn)
				}
				assert.Empty(t, uid)
				if client, ok := tt.client.(*stubAuthClient); ok {
					assert.Equal(t, tt.idToken, client.receivedToken)
				}
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantUID, uid)
			if client, ok := tt.client.(*stubAuthClient); ok {
				assert.Equal(t, tt.idToken, client.receivedToken)
			}
		})
	}
}

func TestTokenVerifier_VerifyRecentSignIn(t *testing.T) {
	originalIsFirebaseInvalidIDToken := isFirebaseInvalidIDToken
	originalIsFirebaseIDTokenRevoked := isFirebaseIDTokenRevoked
	originalIsFirebaseUserDisabled := isFirebaseUserDisabled
	t.Cleanup(func() {
		isFirebaseInvalidIDToken = originalIsFirebaseInvalidIDToken
		isFirebaseIDTokenRevoked = originalIsFirebaseIDTokenRevoked
		isFirebaseUserDisabled = originalIsFirebaseUserDisabled
	})

	invalidErr := errors.New("invalid token from firebase sdk")
	isFirebaseInvalidIDToken = func(err error) bool {
		return errors.Is(err, invalidErr)
	}
	isFirebaseIDTokenRevoked = func(err error) bool {
		return false
	}
	isFirebaseUserDisabled = func(err error) bool {
		return false
	}

	tests := []struct {
		name         string
		client       authClient
		idToken      string
		wantUID      string
		wantAuthTime int64
		wantErr      error
		wantErrIn    string
	}{
		{
			name:         "AuthTime を返せる場合は recent sign-in 情報を返す",
			client:       &stubAuthClient{token: &firebaseauthsdk.Token{UID: "firebase-uid", AuthTime: 1704067200}},
			idToken:      "valid-token",
			wantUID:      "firebase-uid",
			wantAuthTime: 1704067200,
		},
		{
			name:      "AuthTime が空なら ErrInvalidIDToken を返す",
			client:    &stubAuthClient{token: &firebaseauthsdk.Token{UID: "firebase-uid"}},
			idToken:   "missing-auth-time-token",
			wantErr:   usecase.ErrInvalidIDToken,
			wantErrIn: "firebase token auth_time is empty",
		},
		{
			name:      "SDK が不正トークンエラーを返す場合は ErrInvalidIDToken を返す",
			client:    &stubAuthClient{err: invalidErr},
			idToken:   "invalid-token",
			wantErr:   usecase.ErrInvalidIDToken,
			wantErrIn: "invalid token from firebase sdk",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verifier := &tokenVerifier{client: tt.client}

			info, err := verifier.VerifyRecentSignIn(context.Background(), tt.idToken)

			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				if tt.wantErrIn != "" {
					assert.ErrorContains(t, err, tt.wantErrIn)
				}
				assert.Nil(t, info)
				return
			}

			require.NoError(t, err)
			if assert.NotNil(t, info) {
				assert.Equal(t, tt.wantUID, info.UID)
				assert.Equal(t, time.Unix(tt.wantAuthTime, 0).UTC(), info.AuthTime)
			}
		})
	}
}
