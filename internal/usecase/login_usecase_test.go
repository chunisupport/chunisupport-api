package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/username"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestLoginUsecase_Login(t *testing.T) {
	tests := []struct {
		name      string
		idToken   string
		turnstile string
		remoteIP  string
		setup     func(verifier *mockTokenVerifier, turnstileVerifier *mockTurnstileVerifier, userRepo *MockUserRepository)
		wantUser  string
		wantErr   error
	}{
		{
			name:      "TurnstileとFirebaseトークンが有効ならユーザーDTOを返す",
			idToken:   "valid-token",
			turnstile: "turnstile-token",
			remoteIP:  "203.0.113.1",
			setup: func(verifier *mockTokenVerifier, turnstileVerifier *mockTurnstileVerifier, userRepo *MockUserRepository) {
				un := username.MustNewUserName("loginuser")
				user := &entity.User{ID: 10, Username: un, AccountTypeID: info.AccountTypePlayer}
				turnstileVerifier.On("VerifyTurnstile", mock.Anything, "turnstile-token", "203.0.113.1").Return(nil).Once()
				verifier.On("VerifyIDToken", mock.Anything, "valid-token").Return("firebase-uid", nil).Once()
				userRepo.On("FindByFirebaseUID", mock.Anything, mock.Anything, "firebase-uid").Return(user, nil).Once()
			},
			wantUser: "loginuser",
		},
		{
			name:      "Turnstileトークンが空ならFirebase検証に進まない",
			idToken:   "valid-token",
			turnstile: " ",
			setup: func(verifier *mockTokenVerifier, turnstileVerifier *mockTurnstileVerifier, userRepo *MockUserRepository) {
			},
			wantErr: ErrInvalidTurnstileToken,
		},
		{
			name:      "Turnstile検証に失敗したらErrInvalidTurnstileTokenを返す",
			idToken:   "valid-token",
			turnstile: "invalid-turnstile-token",
			setup: func(verifier *mockTokenVerifier, turnstileVerifier *mockTurnstileVerifier, userRepo *MockUserRepository) {
				turnstileVerifier.On("VerifyTurnstile", mock.Anything, "invalid-turnstile-token", "").Return(ErrInvalidTurnstileToken).Once()
			},
			wantErr: ErrInvalidTurnstileToken,
		},
		{
			name:      "Firebaseトークンが無効ならErrInvalidIDTokenを返す",
			idToken:   "invalid-token",
			turnstile: "turnstile-token",
			setup: func(verifier *mockTokenVerifier, turnstileVerifier *mockTurnstileVerifier, userRepo *MockUserRepository) {
				turnstileVerifier.On("VerifyTurnstile", mock.Anything, "turnstile-token", "").Return(nil).Once()
				verifier.On("VerifyIDToken", mock.Anything, "invalid-token").Return("", errors.Join(ErrInvalidIDToken, errors.New("invalid token"))).Once()
			},
			wantErr: ErrInvalidIDToken,
		},
		{
			name:      "未登録Firebase UIDならErrInvalidIDTokenを返す",
			idToken:   "missing-user-token",
			turnstile: "turnstile-token",
			setup: func(verifier *mockTokenVerifier, turnstileVerifier *mockTurnstileVerifier, userRepo *MockUserRepository) {
				turnstileVerifier.On("VerifyTurnstile", mock.Anything, "turnstile-token", "").Return(nil).Once()
				verifier.On("VerifyIDToken", mock.Anything, "missing-user-token").Return("firebase-uid", nil).Once()
				userRepo.On("FindByFirebaseUID", mock.Anything, mock.Anything, "firebase-uid").Return(nil, repository.ErrUserNotFound).Once()
			},
			wantErr: ErrInvalidIDToken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verifier := new(mockTokenVerifier)
			turnstileVerifier := new(mockTurnstileVerifier)
			userRepo := new(MockUserRepository)
			service := NewLoginUsecase(nil, userRepo, verifier, turnstileVerifier, newMockMasterCache())
			tt.setup(verifier, turnstileVerifier, userRepo)

			got, err := service.Login(context.Background(), tt.idToken, tt.turnstile, tt.remoteIP)

			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, got)
			} else {
				require.NoError(t, err)
				require.NotNil(t, got)
				assert.Equal(t, tt.wantUser, got.Username)
				assert.Equal(t, "PLAYER", got.AccountType)
			}

			turnstileVerifier.AssertExpectations(t)
			verifier.AssertExpectations(t)
			userRepo.AssertExpectations(t)
		})
	}
}
