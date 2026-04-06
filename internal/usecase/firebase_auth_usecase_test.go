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

type mockTokenVerifier struct {
	mock.Mock
}

func (m *mockTokenVerifier) VerifyIDToken(ctx context.Context, idToken string) (string, error) {
	args := m.Called(ctx, idToken)
	return args.String(0), args.Error(1)
}

func TestFirebaseAuthUsecase_Authenticate(t *testing.T) {
	tests := []struct {
		name          string
		idToken       string
		setup         func(verifier *mockTokenVerifier, userRepo *MockUserRepository)
		wantUser      *entity.User
		wantErr       error
		wantErrText   string
		assertionFunc func(t *testing.T, verifier *mockTokenVerifier, userRepo *MockUserRepository)
	}{
		{
			name:    "有効なIDトークンの場合はユーザーを返す",
			idToken: "valid-token",
			setup: func(verifier *mockTokenVerifier, userRepo *MockUserRepository) {
				user := &entity.User{ID: 10}
				verifier.On("VerifyIDToken", mock.Anything, "valid-token").Return("firebase-uid", nil).Once()
				userRepo.On("FindByFirebaseUID", mock.Anything, mock.Anything, "firebase-uid").Return(user, nil).Once()
			},
			wantUser: &entity.User{ID: 10},
			assertionFunc: func(t *testing.T, verifier *mockTokenVerifier, userRepo *MockUserRepository) {
				verifier.AssertExpectations(t)
				userRepo.AssertExpectations(t)
			},
		},
		{
			name:    "空のIDトークンはErrInvalidIDTokenを返す",
			idToken: " \t ",
			setup:   func(verifier *mockTokenVerifier, userRepo *MockUserRepository) {},
			wantErr: ErrInvalidIDToken,
			assertionFunc: func(t *testing.T, verifier *mockTokenVerifier, userRepo *MockUserRepository) {
				verifier.AssertNotCalled(t, "VerifyIDToken", mock.Anything, mock.Anything)
				userRepo.AssertNotCalled(t, "FindByFirebaseUID", mock.Anything, mock.Anything, mock.Anything)
			},
		},
		{
			name:    "IDトークン検証に失敗した場合はErrInvalidIDTokenを返す",
			idToken: "invalid-token",
			setup: func(verifier *mockTokenVerifier, userRepo *MockUserRepository) {
				verifier.On("VerifyIDToken", mock.Anything, "invalid-token").Return("", errors.Join(ErrInvalidIDToken, errors.New("verify failed"))).Once()
			},
			wantErr: ErrInvalidIDToken,
			assertionFunc: func(t *testing.T, verifier *mockTokenVerifier, userRepo *MockUserRepository) {
				verifier.AssertExpectations(t)
				userRepo.AssertNotCalled(t, "FindByFirebaseUID", mock.Anything, mock.Anything, mock.Anything)
			},
		},
		{
			name:    "失効済みIDトークンの場合はErrInvalidIDTokenを返しDB参照しない",
			idToken: "revoked-token",
			setup: func(verifier *mockTokenVerifier, userRepo *MockUserRepository) {
				verifier.On("VerifyIDToken", mock.Anything, "revoked-token").Return("", errors.Join(ErrInvalidIDToken, errors.New("revoked token"))).Once()
			},
			wantErr: ErrInvalidIDToken,
			assertionFunc: func(t *testing.T, verifier *mockTokenVerifier, userRepo *MockUserRepository) {
				verifier.AssertExpectations(t)
				userRepo.AssertNotCalled(t, "FindByFirebaseUID", mock.Anything, mock.Anything, mock.Anything)
			},
		},
		{
			name:    "無効化済みユーザーのIDトークンの場合はErrInvalidIDTokenを返しDB参照しない",
			idToken: "disabled-token",
			setup: func(verifier *mockTokenVerifier, userRepo *MockUserRepository) {
				verifier.On("VerifyIDToken", mock.Anything, "disabled-token").Return("", errors.Join(ErrInvalidIDToken, errors.New("user disabled"))).Once()
			},
			wantErr: ErrInvalidIDToken,
			assertionFunc: func(t *testing.T, verifier *mockTokenVerifier, userRepo *MockUserRepository) {
				verifier.AssertExpectations(t)
				userRepo.AssertNotCalled(t, "FindByFirebaseUID", mock.Anything, mock.Anything, mock.Anything)
			},
		},
		{
			name:    "空のUIDが返された場合はErrInternalErrorを返す",
			idToken: "empty-uid-token",
			setup: func(verifier *mockTokenVerifier, userRepo *MockUserRepository) {
				verifier.On("VerifyIDToken", mock.Anything, "empty-uid-token").Return("  ", nil).Once()
			},
			wantErr: ErrInternalError,
			assertionFunc: func(t *testing.T, verifier *mockTokenVerifier, userRepo *MockUserRepository) {
				verifier.AssertExpectations(t)
				userRepo.AssertNotCalled(t, "FindByFirebaseUID", mock.Anything, mock.Anything, mock.Anything)
			},
		},
		{
			name:    "Firebase UIDに紐づくユーザーが存在しない場合はErrInvalidIDTokenを返す",
			idToken: "missing-user-token",
			setup: func(verifier *mockTokenVerifier, userRepo *MockUserRepository) {
				verifier.On("VerifyIDToken", mock.Anything, "missing-user-token").Return("firebase-uid", nil).Once()
				userRepo.On("FindByFirebaseUID", mock.Anything, mock.Anything, "firebase-uid").Return(nil, repository.ErrUserNotFound).Once()
			},
			wantErr: ErrInvalidIDToken,
			assertionFunc: func(t *testing.T, verifier *mockTokenVerifier, userRepo *MockUserRepository) {
				verifier.AssertExpectations(t)
				userRepo.AssertExpectations(t)
			},
		},
		{

			name:    "ユーザー取得がnilならErrInternalErrorを返す",
			idToken: "nil-user-token",
			setup: func(verifier *mockTokenVerifier, userRepo *MockUserRepository) {
				verifier.On("VerifyIDToken", mock.Anything, "nil-user-token").Return("firebase-uid", nil).Once()
				userRepo.On("FindByFirebaseUID", mock.Anything, mock.Anything, "firebase-uid").Return(nil, nil).Once()
			},
			wantErr: ErrInternalError,
			assertionFunc: func(t *testing.T, verifier *mockTokenVerifier, userRepo *MockUserRepository) {
				verifier.AssertExpectations(t)
				userRepo.AssertExpectations(t)
			},
		},
		{
			name:    "ユーザー取得で内部エラーが起きた場合はそのまま返す",
			idToken: "repo-error-token",
			setup: func(verifier *mockTokenVerifier, userRepo *MockUserRepository) {
				repoErr := errors.New("db error")
				verifier.On("VerifyIDToken", mock.Anything, "repo-error-token").Return("firebase-uid", nil).Once()
				userRepo.On("FindByFirebaseUID", mock.Anything, mock.Anything, "firebase-uid").Return(nil, repoErr).Once()
			},
			wantErrText: "db error",
			assertionFunc: func(t *testing.T, verifier *mockTokenVerifier, userRepo *MockUserRepository) {
				verifier.AssertExpectations(t)
				userRepo.AssertExpectations(t)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			verifier := new(mockTokenVerifier)
			userRepo := new(MockUserRepository)
			service := NewFirebaseAuthUsecase(nil, userRepo, verifier)
			tt.setup(verifier, userRepo)

			// When
			gotUser, err := service.Authenticate(context.Background(), tt.idToken)

			// Then
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, gotUser)
			} else if tt.wantErrText != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErrText)
				assert.Nil(t, gotUser)
			} else {
				require.NoError(t, err)
				require.NotNil(t, gotUser)
				assert.Equal(t, tt.wantUser.ID, gotUser.ID)
			}

			tt.assertionFunc(t, verifier, userRepo)
		})
	}
}

func TestFirebaseAuthUsecase_Authenticate_TokenVerifierがnilの場合はErrInternalErrorを返す(t *testing.T) {
	// Given
	userRepo := new(MockUserRepository)
	service := NewFirebaseAuthUsecase(nil, userRepo, nil)

	// When
	gotUser, err := service.Authenticate(context.Background(), "valid-token")

	// Then
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInternalError)
	assert.Nil(t, gotUser)
	userRepo.AssertNotCalled(t, "FindByFirebaseUID", mock.Anything, mock.Anything, mock.Anything)
}

var _ TokenVerifier = (*mockTokenVerifier)(nil)
