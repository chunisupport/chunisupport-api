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
		setup       func(verifier *mockTokenVerifier, userRepo *MockUserRepository)
		wantUser    string
		wantErr     error
		wantErrText string
	}{
		{
			name:     "有効なFirebaseトークンならユーザーを作成してDTOを返す",
			idToken:  "valid-token",
			username: "newuser",
			setup: func(verifier *mockTokenVerifier, userRepo *MockUserRepository) {
				verifier.On("VerifyIDToken", mock.Anything, "valid-token").Return("firebase-uid", nil).Once()
				userRepo.On("FindByFirebaseUID", mock.Anything, mock.Anything, "firebase-uid").Return(nil, repository.ErrUserNotFound).Once()
				userRepo.On("FindByUsername", mock.Anything, mock.Anything, "newuser").Return(nil, repository.ErrUserNotFound).Once()
				userRepo.On("Save", mock.Anything, mock.Anything, mock.MatchedBy(func(user *entity.User) bool {
					user.ID = 99
					return user.Username.String() == "newuser" && user.FirebaseUID != nil && *user.FirebaseUID == "firebase-uid"
				})).Return(nil).Once()
			},
			wantUser: "newuser",
		},
		{
			name:     "空のIDトークンはErrInvalidIDTokenを返す",
			idToken:  "  ",
			username: "newuser",
			setup:    func(verifier *mockTokenVerifier, userRepo *MockUserRepository) {},
			wantErr:  ErrInvalidIDToken,
		},
		{
			name:     "無効なIDトークンはErrInvalidIDTokenを返す",
			idToken:  "invalid-token",
			username: "newuser",
			setup: func(verifier *mockTokenVerifier, userRepo *MockUserRepository) {
				verifier.On("VerifyIDToken", mock.Anything, "invalid-token").Return("", errors.Join(ErrInvalidIDToken, errors.New("invalid token"))).Once()
			},
			wantErr: ErrInvalidIDToken,
		},
		{
			name:     "Firebase UIDが既存ユーザーに紐付いていればErrFirebaseUIDAlreadyLinkedを返す",
			idToken:  "linked-token",
			username: "newuser",
			setup: func(verifier *mockTokenVerifier, userRepo *MockUserRepository) {
				verifier.On("VerifyIDToken", mock.Anything, "linked-token").Return("firebase-uid", nil).Once()
				userRepo.On("FindByFirebaseUID", mock.Anything, mock.Anything, "firebase-uid").Return(&entity.User{ID: 10}, nil).Once()
			},
			wantErr: ErrFirebaseUIDAlreadyLinked,
		},
		{
			name:     "ユーザー名が既に使われていればErrUsernameTakenを返す",
			idToken:  "valid-token",
			username: "newuser",
			setup: func(verifier *mockTokenVerifier, userRepo *MockUserRepository) {
				verifier.On("VerifyIDToken", mock.Anything, "valid-token").Return("firebase-uid", nil).Once()
				userRepo.On("FindByFirebaseUID", mock.Anything, mock.Anything, "firebase-uid").Return(nil, repository.ErrUserNotFound).Once()
				userRepo.On("FindByUsername", mock.Anything, mock.Anything, "newuser").Return(&entity.User{ID: 1}, nil).Once()
			},
			wantErr: ErrUsernameTaken,
		},
		{
			name:     "空のFirebase UIDはErrInternalErrorを返す",
			idToken:  "empty-uid-token",
			username: "newuser",
			setup: func(verifier *mockTokenVerifier, userRepo *MockUserRepository) {
				verifier.On("VerifyIDToken", mock.Anything, "empty-uid-token").Return("   ", nil).Once()
			},
			wantErr: ErrInternalError,
		},
		{
			name:     "想定外の検証失敗はErrInternalErrorにまとめる",
			idToken:  "error-token",
			username: "newuser",
			setup: func(verifier *mockTokenVerifier, userRepo *MockUserRepository) {
				verifier.On("VerifyIDToken", mock.Anything, "error-token").Return("", errors.New("boom")).Once()
			},
			wantErr: ErrInternalError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verifier := new(mockTokenVerifier)
			userRepo := new(MockUserRepository)
			service := NewSignupUsecase(&mockTransactionManager{}, userRepo, verifier, newMockMasterCache())
			tt.setup(verifier, userRepo)

			got, err := service.Signup(context.Background(), tt.idToken, tt.username)

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
			userRepo.AssertExpectations(t)
		})
	}
}
