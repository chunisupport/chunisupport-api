package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestSignupUsecase_Signup(t *testing.T) {
	tests := []struct {
		name        string
		idToken     string
		username    string
		turnstile   string
		remoteIP    string
		setup       func(verifier *mockTokenVerifier, turnstileVerifier *mockTurnstileVerifier, userRepo *MockUserRepository)
		wantUser    string
		wantErr     error
		wantErrText string
	}{
		{
			name:      "有効なFirebaseトークンならユーザーを作成してDTOを返す",
			idToken:   "valid-token",
			username:  "newuser",
			turnstile: "turnstile-token",
			remoteIP:  "203.0.113.1",
			setup: func(verifier *mockTokenVerifier, turnstileVerifier *mockTurnstileVerifier, userRepo *MockUserRepository) {
				turnstileVerifier.On("VerifyTurnstile", mock.Anything, "turnstile-token", "203.0.113.1").Return(nil).Once()
				verifier.On("VerifyIDToken", mock.Anything, "valid-token").Return("firebase-uid", nil).Once()
				userRepo.On("Save", mock.Anything, mock.Anything, mock.MatchedBy(func(user *entity.User) bool {
					user.ID = 99
					return user.Username.String() == "newuser" && user.FirebaseUID != nil && *user.FirebaseUID == "firebase-uid"
				})).Return(nil).Once()
			},
			wantUser: "newuser",
		},
		{
			name:      "空のIDトークンはErrInvalidIDTokenを返す",
			idToken:   "  ",
			username:  "newuser",
			turnstile: "turnstile-token",
			setup: func(verifier *mockTokenVerifier, turnstileVerifier *mockTurnstileVerifier, userRepo *MockUserRepository) {
				turnstileVerifier.On("VerifyTurnstile", mock.Anything, "turnstile-token", "").Return(nil).Once()
			},
			wantErr: ErrInvalidIDToken,
		},
		{
			name:      "空のTurnstileトークンはErrInvalidTurnstileTokenを返す",
			idToken:   "valid-token",
			username:  "newuser",
			turnstile: "  ",
			setup: func(verifier *mockTokenVerifier, turnstileVerifier *mockTurnstileVerifier, userRepo *MockUserRepository) {
			},
			wantErr: ErrInvalidTurnstileToken,
		},
		{
			name:      "Turnstile検証に失敗した場合はErrInvalidTurnstileTokenを返す",
			idToken:   "valid-token",
			username:  "newuser",
			turnstile: "invalid-turnstile-token",
			setup: func(verifier *mockTokenVerifier, turnstileVerifier *mockTurnstileVerifier, userRepo *MockUserRepository) {
				turnstileVerifier.On("VerifyTurnstile", mock.Anything, "invalid-turnstile-token", "").Return(ErrInvalidTurnstileToken).Once()
			},
			wantErr: ErrInvalidTurnstileToken,
		},
		{
			name:      "無効なIDトークンはErrInvalidIDTokenを返す",
			idToken:   "invalid-token",
			username:  "newuser",
			turnstile: "turnstile-token",
			setup: func(verifier *mockTokenVerifier, turnstileVerifier *mockTurnstileVerifier, userRepo *MockUserRepository) {
				turnstileVerifier.On("VerifyTurnstile", mock.Anything, "turnstile-token", "").Return(nil).Once()
				verifier.On("VerifyIDToken", mock.Anything, "invalid-token").Return("", errors.Join(ErrInvalidIDToken, errors.New("invalid token"))).Once()
			},
			wantErr: ErrInvalidIDToken,
		},
		{
			name:      "Firebase UIDが既存ユーザーに紐付いていればErrFirebaseUIDAlreadyLinkedを返す",
			idToken:   "linked-token",
			username:  "newuser",
			turnstile: "turnstile-token",
			setup: func(verifier *mockTokenVerifier, turnstileVerifier *mockTurnstileVerifier, userRepo *MockUserRepository) {
				turnstileVerifier.On("VerifyTurnstile", mock.Anything, "turnstile-token", "").Return(nil).Once()
				verifier.On("VerifyIDToken", mock.Anything, "linked-token").Return("firebase-uid", nil).Once()
				userRepo.On("Save", mock.Anything, mock.Anything, mock.AnythingOfType("*entity.User")).Return(repository.ErrFirebaseUIDAlreadyLinked).Once()
			},
			wantErr: ErrFirebaseUIDAlreadyLinked,
		},
		{
			name:      "ユーザー名が既に使われていればErrUsernameTakenを返す",
			idToken:   "valid-token",
			username:  "newuser",
			turnstile: "turnstile-token",
			setup: func(verifier *mockTokenVerifier, turnstileVerifier *mockTurnstileVerifier, userRepo *MockUserRepository) {
				turnstileVerifier.On("VerifyTurnstile", mock.Anything, "turnstile-token", "").Return(nil).Once()
				verifier.On("VerifyIDToken", mock.Anything, "valid-token").Return("firebase-uid", nil).Once()
				userRepo.On("Save", mock.Anything, mock.Anything, mock.AnythingOfType("*entity.User")).Return(repository.ErrDuplicateUsername).Once()
			},
			wantErr: ErrUsernameTaken,
		},
		{
			name:      "空のFirebase UIDはErrInternalErrorを返す",
			idToken:   "empty-uid-token",
			username:  "newuser",
			turnstile: "turnstile-token",
			setup: func(verifier *mockTokenVerifier, turnstileVerifier *mockTurnstileVerifier, userRepo *MockUserRepository) {
				turnstileVerifier.On("VerifyTurnstile", mock.Anything, "turnstile-token", "").Return(nil).Once()
				verifier.On("VerifyIDToken", mock.Anything, "empty-uid-token").Return("   ", nil).Once()
			},
			wantErr: ErrInternalError,
		},
		{
			name:      "想定外の検証失敗はErrInternalErrorにまとめる",
			idToken:   "error-token",
			username:  "newuser",
			turnstile: "turnstile-token",
			setup: func(verifier *mockTokenVerifier, turnstileVerifier *mockTurnstileVerifier, userRepo *MockUserRepository) {
				turnstileVerifier.On("VerifyTurnstile", mock.Anything, "turnstile-token", "").Return(nil).Once()
				verifier.On("VerifyIDToken", mock.Anything, "error-token").Return("", errors.New("boom")).Once()
			},
			wantErr: ErrInternalError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verifier := new(mockTokenVerifier)
			turnstileVerifier := new(mockTurnstileVerifier)
			userRepo := new(MockUserRepository)
			service := NewSignupUsecase(&mockTransactionManager{}, userRepo, verifier, turnstileVerifier, newMockMasterCache())
			tt.setup(verifier, turnstileVerifier, userRepo)

			got, err := service.Signup(context.Background(), tt.idToken, tt.username, tt.turnstile, tt.remoteIP)

			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, got)
			} else if tt.wantErrText != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErrText)
				assert.Nil(t, got)
			} else {
				require.NoError(t, err)
				require.NotNil(t, got)
				assert.Equal(t, tt.wantUser, got.Username)
				assert.Equal(t, "PLAYER", got.AccountType)
			}

			verifier.AssertExpectations(t)
			turnstileVerifier.AssertExpectations(t)
			userRepo.AssertExpectations(t)
			userRepo.AssertNotCalled(t, "FindByFirebaseUID", mock.Anything, mock.Anything, mock.Anything)
			userRepo.AssertNotCalled(t, "FindByUsername", mock.Anything, mock.Anything, mock.Anything)
		})
	}
}
