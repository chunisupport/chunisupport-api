package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestFirebaseRegisterUsecase_RegisterWithFirebase(t *testing.T) {
	tests := []struct {
		name        string
		idToken     string
		username    string
		setup       func(verifier *mockTokenVerifier, userRepo *MockUserRepository, sessionIssuer *authMockSessionIssuer)
		wantToken   string
		wantErr     error
		assertAfter func(t *testing.T, verifier *mockTokenVerifier, userRepo *MockUserRepository, sessionIssuer *authMockSessionIssuer)
	}{
		{
			name:     "正常系: 新規ユーザーを作成しトークンを返す",
			idToken:  "valid-token",
			username: "newuser",
			setup: func(verifier *mockTokenVerifier, userRepo *MockUserRepository, sessionIssuer *authMockSessionIssuer) {
				verifier.On("VerifyIDToken", mock.Anything, "valid-token").Return("firebase-uid", nil).Once()
				userRepo.On("Save", mock.Anything, mock.Anything, mock.AnythingOfType("*entity.User")).Return(nil).Once()
				sessionIssuer.On("IssueSession", mock.Anything, mock.AnythingOfType("*entity.User")).Return("jwt-token", nil).Once()
			},
			wantToken: "jwt-token",
			assertAfter: func(t *testing.T, verifier *mockTokenVerifier, userRepo *MockUserRepository, sessionIssuer *authMockSessionIssuer) {
				verifier.AssertExpectations(t)
				userRepo.AssertExpectations(t)
				sessionIssuer.AssertExpectations(t)
			},
		},
		{
			name:     "異常系: 空のIDトークンはErrInvalidIDTokenを返す",
			idToken:  "  ",
			username: "newuser",
			setup: func(verifier *mockTokenVerifier, userRepo *MockUserRepository, sessionIssuer *authMockSessionIssuer) {
			},
			wantErr: ErrInvalidIDToken,
			assertAfter: func(t *testing.T, verifier *mockTokenVerifier, userRepo *MockUserRepository, sessionIssuer *authMockSessionIssuer) {
				verifier.AssertNotCalled(t, "VerifyIDToken", mock.Anything, mock.Anything)
				userRepo.AssertNotCalled(t, "Save", mock.Anything, mock.Anything, mock.Anything)
			},
		},
		{
			name:     "異常系: IDトークン検証失敗はErrInvalidIDTokenを返す",
			idToken:  "bad-token",
			username: "newuser",
			setup: func(verifier *mockTokenVerifier, userRepo *MockUserRepository, sessionIssuer *authMockSessionIssuer) {
				verifier.On("VerifyIDToken", mock.Anything, "bad-token").Return("", errors.Join(ErrInvalidIDToken, errors.New("verify failed"))).Once()
			},
			wantErr: ErrInvalidIDToken,
			assertAfter: func(t *testing.T, verifier *mockTokenVerifier, userRepo *MockUserRepository, sessionIssuer *authMockSessionIssuer) {
				verifier.AssertExpectations(t)
				userRepo.AssertNotCalled(t, "Save", mock.Anything, mock.Anything, mock.Anything)
			},
		},
		{
			name:     "異常系: Firebase UIDが既存ユーザーに紐付いている場合はErrFirebaseUIDAlreadyLinkedを返す",
			idToken:  "linked-token",
			username: "newuser",
			setup: func(verifier *mockTokenVerifier, userRepo *MockUserRepository, sessionIssuer *authMockSessionIssuer) {
				verifier.On("VerifyIDToken", mock.Anything, "linked-token").Return("existing-uid", nil).Once()
				userRepo.On("Save", mock.Anything, mock.Anything, mock.AnythingOfType("*entity.User")).Return(repository.ErrFirebaseUIDAlreadyLinked).Once()
			},
			wantErr: ErrFirebaseUIDAlreadyLinked,
			assertAfter: func(t *testing.T, verifier *mockTokenVerifier, userRepo *MockUserRepository, sessionIssuer *authMockSessionIssuer) {
				verifier.AssertExpectations(t)
				userRepo.AssertExpectations(t)
				sessionIssuer.AssertNotCalled(t, "IssueSession", mock.Anything, mock.Anything)
			},
		},
		{
			name:     "異常系: ユーザー名が使用済みの場合はErrUsernameTakenを返す",
			idToken:  "valid-token",
			username: "takenuser",
			setup: func(verifier *mockTokenVerifier, userRepo *MockUserRepository, sessionIssuer *authMockSessionIssuer) {
				verifier.On("VerifyIDToken", mock.Anything, "valid-token").Return("firebase-uid", nil).Once()
				userRepo.On("Save", mock.Anything, mock.Anything, mock.AnythingOfType("*entity.User")).Return(repository.ErrDuplicateUsername).Once()
			},
			wantErr: ErrUsernameTaken,
			assertAfter: func(t *testing.T, verifier *mockTokenVerifier, userRepo *MockUserRepository, sessionIssuer *authMockSessionIssuer) {
				verifier.AssertExpectations(t)
				userRepo.AssertExpectations(t)
				sessionIssuer.AssertNotCalled(t, "IssueSession", mock.Anything, mock.Anything)
			},
		},
		{
			name:     "異常系: ユーザー名バリデーション失敗",
			idToken:  "valid-token",
			username: "ab",
			setup: func(verifier *mockTokenVerifier, userRepo *MockUserRepository, sessionIssuer *authMockSessionIssuer) {
				verifier.On("VerifyIDToken", mock.Anything, "valid-token").Return("firebase-uid", nil).Once()
			},
			wantErr: ErrUsernameTooShort,
			assertAfter: func(t *testing.T, verifier *mockTokenVerifier, userRepo *MockUserRepository, sessionIssuer *authMockSessionIssuer) {
				verifier.AssertExpectations(t)
				userRepo.AssertNotCalled(t, "Save", mock.Anything, mock.Anything, mock.Anything)
			},
		},
		{
			name:     "異常系: DB競合でFirebase UID重複が発生した場合はErrFirebaseUIDAlreadyLinkedを返す",
			idToken:  "valid-token",
			username: "newuser",
			setup: func(verifier *mockTokenVerifier, userRepo *MockUserRepository, sessionIssuer *authMockSessionIssuer) {
				verifier.On("VerifyIDToken", mock.Anything, "valid-token").Return("firebase-uid", nil).Once()
				userRepo.On("Save", mock.Anything, mock.Anything, mock.AnythingOfType("*entity.User")).Return(repository.ErrFirebaseUIDAlreadyLinked).Once()
			},
			wantErr: ErrFirebaseUIDAlreadyLinked,
			assertAfter: func(t *testing.T, verifier *mockTokenVerifier, userRepo *MockUserRepository, sessionIssuer *authMockSessionIssuer) {
				verifier.AssertExpectations(t)
				userRepo.AssertExpectations(t)
				sessionIssuer.AssertNotCalled(t, "IssueSession", mock.Anything, mock.Anything)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			verifier := new(mockTokenVerifier)
			userRepo := new(MockUserRepository)
			sessionIssuer := new(authMockSessionIssuer)
			tm := &mockTransactionManager{exec: &MockExecutor{}}
			svc := NewFirebaseRegisterUsecase(tm, userRepo, verifier, sessionIssuer)
			tt.setup(verifier, userRepo, sessionIssuer)

			// When
			token, err := svc.RegisterWithFirebase(context.Background(), tt.idToken, tt.username)

			// Then
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.True(t, errors.Is(err, tt.wantErr), "エラー種別不一致: got %v, want %v", err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantToken, token)
			}

			if tt.assertAfter != nil {
				tt.assertAfter(t, verifier, userRepo, sessionIssuer)
			}

			userRepo.AssertNotCalled(t, "FindByFirebaseUID", mock.Anything, mock.Anything, mock.Anything)
			userRepo.AssertNotCalled(t, "FindByUsername", mock.Anything, mock.Anything, mock.Anything)
		})
	}
}
