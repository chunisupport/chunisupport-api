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

func TestFirebaseLinkUsecase_LinkFirebaseUID(t *testing.T) {
	tests := []struct {
		name        string
		userID      int
		idToken     string
		setup       func(verifier *mockTokenVerifier, userRepo *MockUserRepository)
		wantErr     error
		assertAfter func(t *testing.T, verifier *mockTokenVerifier, userRepo *MockUserRepository)
	}{
		{
			name:    "未連携ユーザーにUIDを紐付けできる",
			userID:  1,
			idToken: "valid-token",
			setup: func(verifier *mockTokenVerifier, userRepo *MockUserRepository) {
				user := &entity.User{ID: 1}
				verifier.On("VerifyIDToken", mock.Anything, "valid-token").Return("firebase-uid", nil).Once()
				userRepo.On("FindByFirebaseUID", mock.Anything, mock.Anything, "firebase-uid").Return(nil, repository.ErrUserNotFound).Once()
				userRepo.On("FindByIDForUpdate", mock.Anything, mock.Anything, 1).Return(user, nil).Once()
				userRepo.On("LinkFirebaseUID", mock.Anything, mock.Anything, 1, (*string)(nil), "firebase-uid", mock.AnythingOfType("time.Time")).Return(nil).Once()
			},
			assertAfter: func(t *testing.T, verifier *mockTokenVerifier, userRepo *MockUserRepository) {
				verifier.AssertExpectations(t)
				userRepo.AssertExpectations(t)
			},
		},
		{
			name:    "自分に同じUIDが既に紐付いていれば冪等成功する",
			userID:  10,
			idToken: "same-user-token",
			setup: func(verifier *mockTokenVerifier, userRepo *MockUserRepository) {
				verifier.On("VerifyIDToken", mock.Anything, "same-user-token").Return("firebase-uid", nil).Once()
				userRepo.On("FindByFirebaseUID", mock.Anything, mock.Anything, "firebase-uid").Return(&entity.User{ID: 10}, nil).Once()
			},
			assertAfter: func(t *testing.T, verifier *mockTokenVerifier, userRepo *MockUserRepository) {
				verifier.AssertExpectations(t)
				userRepo.AssertExpectations(t)
				userRepo.AssertNotCalled(t, "FindByIDForUpdate", mock.Anything, mock.Anything, mock.Anything)
				userRepo.AssertNotCalled(t, "LinkFirebaseUID", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
			},
		},
		{
			name:    "自分に別のUIDが既に紐付いていれば新しいUIDへ更新できる",
			userID:  10,
			idToken: "replace-user-token",
			setup: func(verifier *mockTokenVerifier, userRepo *MockUserRepository) {
				existingUID := "old-firebase-uid"
				user := &entity.User{ID: 10, FirebaseUID: &existingUID}
				verifier.On("VerifyIDToken", mock.Anything, "replace-user-token").Return("new-firebase-uid", nil).Once()
				userRepo.On("FindByFirebaseUID", mock.Anything, mock.Anything, "new-firebase-uid").Return(nil, repository.ErrUserNotFound).Once()
				userRepo.On("FindByIDForUpdate", mock.Anything, mock.Anything, 10).Return(user, nil).Once()
				userRepo.On("LinkFirebaseUID", mock.Anything, mock.Anything, 10, &existingUID, "new-firebase-uid", mock.AnythingOfType("time.Time")).Return(nil).Once()
			},
			assertAfter: func(t *testing.T, verifier *mockTokenVerifier, userRepo *MockUserRepository) {
				verifier.AssertExpectations(t)
				userRepo.AssertExpectations(t)
			},
		},
		{
			name:    "他ユーザーに紐付いているUIDは409相当のエラーを返す",
			userID:  11,
			idToken: "linked-token",
			setup: func(verifier *mockTokenVerifier, userRepo *MockUserRepository) {
				verifier.On("VerifyIDToken", mock.Anything, "linked-token").Return("firebase-uid", nil).Once()
				userRepo.On("FindByFirebaseUID", mock.Anything, mock.Anything, "firebase-uid").Return(&entity.User{ID: 99}, nil).Once()
			},
			wantErr: ErrFirebaseUIDAlreadyLinked,
			assertAfter: func(t *testing.T, verifier *mockTokenVerifier, userRepo *MockUserRepository) {
				verifier.AssertExpectations(t)
				userRepo.AssertExpectations(t)
				userRepo.AssertNotCalled(t, "FindByIDForUpdate", mock.Anything, mock.Anything, mock.Anything)
				userRepo.AssertNotCalled(t, "LinkFirebaseUID", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
			},
		},
		{
			name:    "削除済み他ユーザーに紐付いているUIDは再利用できない",
			userID:  12,
			idToken: "deleted-linked-token",
			setup: func(verifier *mockTokenVerifier, userRepo *MockUserRepository) {
				verifier.On("VerifyIDToken", mock.Anything, "deleted-linked-token").Return("firebase-uid", nil).Once()
				userRepo.On("FindByFirebaseUID", mock.Anything, mock.Anything, "firebase-uid").Return(&entity.User{ID: 99, IsDeleted: true}, nil).Once()
			},
			wantErr: ErrFirebaseUIDAlreadyLinked,
			assertAfter: func(t *testing.T, verifier *mockTokenVerifier, userRepo *MockUserRepository) {
				verifier.AssertExpectations(t)
				userRepo.AssertExpectations(t)
				userRepo.AssertNotCalled(t, "FindByIDForUpdate", mock.Anything, mock.Anything, mock.Anything)
				userRepo.AssertNotCalled(t, "LinkFirebaseUID", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
			},
		},
		{
			name:    "削除済みの自分に既存UIDがあっても連携成功にしない",
			userID:  10,
			idToken: "deleted-same-user-token",
			setup: func(verifier *mockTokenVerifier, userRepo *MockUserRepository) {
				verifier.On("VerifyIDToken", mock.Anything, "deleted-same-user-token").Return("firebase-uid", nil).Once()
				userRepo.On("FindByFirebaseUID", mock.Anything, mock.Anything, "firebase-uid").Return(&entity.User{ID: 10, IsDeleted: true}, nil).Once()
			},
			wantErr: ErrUserDeleted,
			assertAfter: func(t *testing.T, verifier *mockTokenVerifier, userRepo *MockUserRepository) {
				verifier.AssertExpectations(t)
				userRepo.AssertExpectations(t)
				userRepo.AssertNotCalled(t, "FindByIDForUpdate", mock.Anything, mock.Anything, mock.Anything)
				userRepo.AssertNotCalled(t, "LinkFirebaseUID", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
			},
		},
		{
			name:    "無効なトークンはErrInvalidIDTokenを返す",
			userID:  1,
			idToken: "invalid-token",
			setup: func(verifier *mockTokenVerifier, userRepo *MockUserRepository) {
				verifier.On("VerifyIDToken", mock.Anything, "invalid-token").Return("", errors.Join(ErrInvalidIDToken, errors.New("verify failed"))).Once()
			},
			wantErr: ErrInvalidIDToken,
			assertAfter: func(t *testing.T, verifier *mockTokenVerifier, userRepo *MockUserRepository) {
				verifier.AssertExpectations(t)
				userRepo.AssertNotCalled(t, "FindByFirebaseUID", mock.Anything, mock.Anything, mock.Anything)
			},
		},
		{
			name:    "削除済みユーザーには連携できない",
			userID:  1,
			idToken: "deleted-user-token",
			setup: func(verifier *mockTokenVerifier, userRepo *MockUserRepository) {
				verifier.On("VerifyIDToken", mock.Anything, "deleted-user-token").Return("firebase-uid", nil).Once()
				userRepo.On("FindByFirebaseUID", mock.Anything, mock.Anything, "firebase-uid").Return(nil, repository.ErrUserNotFound).Once()
				userRepo.On("FindByIDForUpdate", mock.Anything, mock.Anything, 1).Return(&entity.User{ID: 1, IsDeleted: true}, nil).Once()
			},
			wantErr: ErrUserDeleted,
			assertAfter: func(t *testing.T, verifier *mockTokenVerifier, userRepo *MockUserRepository) {
				verifier.AssertExpectations(t)
				userRepo.AssertExpectations(t)
				userRepo.AssertNotCalled(t, "LinkFirebaseUID", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
			},
		},
		{
			name:    "FindByFirebaseUIDがnilユーザーを返した場合は内部エラーにする",
			userID:  1,
			idToken: "nil-linked-user-token",
			setup: func(verifier *mockTokenVerifier, userRepo *MockUserRepository) {
				verifier.On("VerifyIDToken", mock.Anything, "nil-linked-user-token").Return("firebase-uid", nil).Once()
				userRepo.On("FindByFirebaseUID", mock.Anything, mock.Anything, "firebase-uid").Return(nil, nil).Once()
			},
			wantErr: ErrInternalError,
			assertAfter: func(t *testing.T, verifier *mockTokenVerifier, userRepo *MockUserRepository) {
				verifier.AssertExpectations(t)
				userRepo.AssertExpectations(t)
				userRepo.AssertNotCalled(t, "FindByIDForUpdate", mock.Anything, mock.Anything, mock.Anything)
				userRepo.AssertNotCalled(t, "LinkFirebaseUID", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
			},
		},
		{
			name:    "FindByIDがnilユーザーを返した場合は内部エラーにする",
			userID:  1,
			idToken: "nil-user-token",
			setup: func(verifier *mockTokenVerifier, userRepo *MockUserRepository) {
				verifier.On("VerifyIDToken", mock.Anything, "nil-user-token").Return("firebase-uid", nil).Once()
				userRepo.On("FindByFirebaseUID", mock.Anything, mock.Anything, "firebase-uid").Return(nil, repository.ErrUserNotFound).Once()
				userRepo.On("FindByIDForUpdate", mock.Anything, mock.Anything, 1).Return(nil, nil).Once()
			},
			wantErr: ErrInternalError,
			assertAfter: func(t *testing.T, verifier *mockTokenVerifier, userRepo *MockUserRepository) {
				verifier.AssertExpectations(t)
				userRepo.AssertExpectations(t)
				userRepo.AssertNotCalled(t, "LinkFirebaseUID", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
			},
		},
		{
			name:    "保存時のUNIQUE制約違反は409相当のエラーを返す",
			userID:  1,
			idToken: "duplicate-save-token",
			setup: func(verifier *mockTokenVerifier, userRepo *MockUserRepository) {
				user := &entity.User{ID: 1}
				verifier.On("VerifyIDToken", mock.Anything, "duplicate-save-token").Return("firebase-uid", nil).Once()
				userRepo.On("FindByFirebaseUID", mock.Anything, mock.Anything, "firebase-uid").Return(nil, repository.ErrUserNotFound).Once()
				userRepo.On("FindByIDForUpdate", mock.Anything, mock.Anything, 1).Return(user, nil).Once()
				userRepo.On("LinkFirebaseUID", mock.Anything, mock.Anything, 1, (*string)(nil), "firebase-uid", mock.AnythingOfType("time.Time")).Return(repository.ErrFirebaseUIDAlreadyLinked).Once()
			},
			wantErr: ErrFirebaseUIDAlreadyLinked,
			assertAfter: func(t *testing.T, verifier *mockTokenVerifier, userRepo *MockUserRepository) {
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
			tm := &mockTransactionManager{}
			service := NewFirebaseLinkUsecase(tm, userRepo, verifier)
			tt.setup(verifier, userRepo)

			// When
			err := service.LinkFirebaseUID(context.Background(), tt.userID, tt.idToken)

			// Then
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}

			tt.assertAfter(t, verifier, userRepo)
		})
	}
}
