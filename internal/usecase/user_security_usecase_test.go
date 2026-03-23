package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/passwordhash"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/username"
	"github.com/chunisupport/chunisupport-api/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestUserSecurityUsecase_ChangePassword(t *testing.T) {
	pepper := "test-pepper"
	errDB := errors.New("db error")

	tests := []struct {
		name            string
		userID          int
		currentPassword string
		newPassword     string
		setupMock       func(*MockUserRepository)
		wantErr         error
	}{
		{
			name:            "ChangePassword_正常系_パスワード変更が成功する",
			userID:          1,
			currentPassword: "old-password",
			newPassword:     "new-password",
			setupMock: func(m *MockUserRepository) {
				hashedPassword, _ := utils.HashPasswordWithPepper("old-password", pepper)
				ph, _ := passwordhash.NewPasswordHash(hashedPassword)
				un, _ := username.NewUserName("testuser")
				mockUser := &entity.User{ID: 1, Username: un, PasswordHash: ph}
				m.On("FindByID", mock.Anything, mock.Anything, 1).Return(mockUser, nil).Once()
				m.On("Save", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
			},
		},
		{
			name:            "ChangePassword_異常系_ユーザーが見つからない",
			userID:          2,
			currentPassword: "old-password",
			newPassword:     "new-password",
			setupMock: func(m *MockUserRepository) {
				m.On("FindByID", mock.Anything, mock.Anything, 2).Return(nil, repository.ErrUserNotFound).Once()
			},
			wantErr: ErrUserNotFound,
		},
		{
			name:            "ChangePassword_異常系_ユーザー検索時にデータベースエラー",
			userID:          1,
			currentPassword: "old-password",
			newPassword:     "new-password",
			setupMock: func(m *MockUserRepository) {
				m.On("FindByID", mock.Anything, mock.Anything, 1).Return(nil, errDB).Once()
			},
			wantErr: errDB,
		},
		{
			name:            "ChangePassword_異常系_現在のパスワードが間違っている",
			userID:          1,
			currentPassword: "wrong-password",
			newPassword:     "new-password",
			setupMock: func(m *MockUserRepository) {
				hashedPassword, _ := utils.HashPasswordWithPepper("old-password", pepper)
				ph, _ := passwordhash.NewPasswordHash(hashedPassword)
				un, _ := username.NewUserName("testuser")
				mockUser := &entity.User{ID: 1, Username: un, PasswordHash: ph}
				m.On("FindByID", mock.Anything, mock.Anything, 1).Return(mockUser, nil).Once()
			},
			wantErr: ErrIncorrectPassword,
		},
		{
			name:            "ChangePassword_異常系_パスワード更新時にデータベースエラー",
			userID:          1,
			currentPassword: "old-password",
			newPassword:     "new-password",
			setupMock: func(m *MockUserRepository) {
				hashedPassword, _ := utils.HashPasswordWithPepper("old-password", pepper)
				ph, _ := passwordhash.NewPasswordHash(hashedPassword)
				un, _ := username.NewUserName("testuser")
				mockUser := &entity.User{ID: 1, Username: un, PasswordHash: ph}
				m.On("FindByID", mock.Anything, mock.Anything, 1).Return(mockUser, nil).Once()
				m.On("Save", mock.Anything, mock.Anything, mock.Anything).Return(errDB).Once()
			},
			wantErr: errDB,
		},
		{
			name:            "ChangePassword_異常系_新しいパスワードが現在と同じ",
			userID:          1,
			currentPassword: "old-password",
			newPassword:     "old-password",
			setupMock: func(m *MockUserRepository) {
				hashedPassword, _ := utils.HashPasswordWithPepper("old-password", pepper)
				ph, _ := passwordhash.NewPasswordHash(hashedPassword)
				un, _ := username.NewUserName("testuser")
				mockUser := &entity.User{ID: 1, Username: un, PasswordHash: ph}
				m.On("FindByID", mock.Anything, mock.Anything, 1).Return(mockUser, nil).Once()
			},
			wantErr: ErrInvalidPassword,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockUserRepo := new(MockUserRepository)
			userCredentialUsecase := newTestUserCredentialUsecase(mockUserRepo, nil, pepper)

			tc.setupMock(mockUserRepo)
			err := userCredentialUsecase.ChangePassword(context.Background(), tc.userID, tc.currentPassword, tc.newPassword)

			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
			} else {
				assert.NoError(t, err)
			}
			mockUserRepo.AssertExpectations(t)
		})
	}
}

func TestUserSecurityUsecase_GetUser(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	userCredentialUsecase := newTestUserCredentialUsecase(mockUserRepo, nil, "test-pepper")

	t.Run("GetUser_正常系_ユーザー取得が成功する", func(t *testing.T) {
		un, _ := username.NewUserName("testuser")
		mockUser := &entity.User{ID: 1, Username: un, IsPrivate: false, AccountTypeID: 1}
		mockUserRepo.On("FindByID", mock.Anything, mock.Anything, 1).Return(mockUser, nil).Once()

		userDTO, err := userCredentialUsecase.GetUser(context.Background(), 1)
		assert.NoError(t, err)
		assert.NotNil(t, userDTO)
		assert.Equal(t, "testuser", userDTO.Username)
		assert.Equal(t, "PLAYER", userDTO.AccountType)
		assert.False(t, userDTO.IsPrivate)
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("GetUser_正常系_PlayerIDがある場合は最終スコア更新日時を含む", func(t *testing.T) {
		playerID := 10
		lastScoreUpdate := time.Now()
		mockUserRepo := new(MockUserRepository)
		playerRecordRepo := &stubPlayerRecordRepository{lastScoreUpdate: &lastScoreUpdate}
		userCredentialUsecase := newTestUserCredentialUsecase(mockUserRepo, playerRecordRepo, "test-pepper")

		un, _ := username.NewUserName("playeruser")
		mockUser := &entity.User{ID: 2, Username: un, IsPrivate: true, AccountTypeID: 1, PlayerID: &playerID}
		mockUserRepo.On("FindByID", mock.Anything, mock.Anything, 2).Return(mockUser, nil).Once()

		userDTO, err := userCredentialUsecase.GetUser(context.Background(), 2)
		assert.NoError(t, err)
		assert.NotNil(t, userDTO)
		assert.Equal(t, "playeruser", userDTO.Username)
		assert.True(t, userDTO.IsPrivate)
		if assert.NotNil(t, userDTO.LastScoreUpdate) {
			assert.True(t, userDTO.LastScoreUpdate.Equal(lastScoreUpdate))
		}
		mockUserRepo.AssertExpectations(t)
	})
}

func TestUserSecurityUsecase_DeleteUser(t *testing.T) {
	un, _ := username.NewUserName("testuser")

	t.Run("DeleteUser_正常系_論理削除が成功する", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		userCredentialUsecase := newTestUserCredentialUsecase(mockUserRepo, nil, "test-pepper")

		user := &entity.User{ID: 1, Username: un}
		mockUserRepo.On("FindByID", mock.Anything, mock.Anything, 1).Return(user, nil).Once()
		mockUserRepo.On("Save", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

		err := userCredentialUsecase.DeleteOwnAccount(context.Background(), 1)
		assert.NoError(t, err)
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("DeleteUser_異常系_リポジトリエラー", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		userCredentialUsecase := newTestUserCredentialUsecase(mockUserRepo, nil, "test-pepper")

		user := &entity.User{ID: 2, Username: un}
		mockUserRepo.On("FindByID", mock.Anything, mock.Anything, 2).Return(user, nil).Once()
		mockUserRepo.On("Save", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("db error")).Once()

		err := userCredentialUsecase.DeleteOwnAccount(context.Background(), 2)
		assert.Error(t, err)
		assert.Equal(t, "db error", err.Error())
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("DeleteUser_異常系_既に削除済みのユーザー", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		userCredentialUsecase := newTestUserCredentialUsecase(mockUserRepo, nil, "test-pepper")

		deletedUser := &entity.User{ID: 3, Username: un, IsDeleted: true}
		mockUserRepo.On("FindByID", mock.Anything, mock.Anything, 3).Return(deletedUser, nil).Once()

		err := userCredentialUsecase.DeleteOwnAccount(context.Background(), 3)
		assert.ErrorIs(t, err, ErrUserAlreadyDeleted)
		mockUserRepo.AssertExpectations(t)
	})
}
