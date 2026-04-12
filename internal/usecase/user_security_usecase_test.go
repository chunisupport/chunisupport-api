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
			name:            "パスワード変更に成功する",
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
			name:            "ユーザーが見つからない場合はErrUserNotFoundを返す",
			userID:          2,
			currentPassword: "old-password",
			newPassword:     "new-password",
			setupMock: func(m *MockUserRepository) {
				m.On("FindByID", mock.Anything, mock.Anything, 2).Return(nil, repository.ErrUserNotFound).Once()
			},
			wantErr: ErrUserNotFound,
		},
		{
			name:            "ユーザー取得でDBエラーが発生した場合はそのまま返す",
			userID:          1,
			currentPassword: "old-password",
			newPassword:     "new-password",
			setupMock: func(m *MockUserRepository) {
				m.On("FindByID", mock.Anything, mock.Anything, 1).Return(nil, errDB).Once()
			},
			wantErr: errDB,
		},
		{
			name:            "現在のパスワードが誤っている場合はErrIncorrectPasswordを返す",
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
			name:            "保存時にDBエラーが発生した場合はそのまま返す",
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
			name:            "新しいパスワードが現在のものと同じ場合はErrInvalidPasswordを返す",
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

	t.Run("ユーザー取得に成功する", func(t *testing.T) {
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

	t.Run("PlayerIDがある場合は最終スコア更新日時を含める", func(t *testing.T) {
		playerID := 10
		lastScoreUpdate := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
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

	t.Run("PlayerIDがある場合に最終スコア更新日時取得が失敗したらエラーを返す", func(t *testing.T) {
		playerID := 11
		mockUserRepo := new(MockUserRepository)
		playerRecordRepo := &stubPlayerRecordRepository{err: errors.New("db error")}
		userCredentialUsecase := newTestUserCredentialUsecase(mockUserRepo, playerRecordRepo, "test-pepper")

		un, _ := username.NewUserName("playeruser2")
		mockUser := &entity.User{ID: 3, Username: un, PlayerID: &playerID}
		mockUserRepo.On("FindByID", mock.Anything, mock.Anything, 3).Return(mockUser, nil).Once()

		userDTO, err := userCredentialUsecase.GetUser(context.Background(), 3)
		assert.Error(t, err)
		assert.Nil(t, userDTO)
		mockUserRepo.AssertExpectations(t)
	})
}

func TestUserSecurityUsecase_DeleteUser(t *testing.T) {
	un, _ := username.NewUserName("testuser")

	t.Run("アカウント削除に成功する", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		tm := &mockTransactionManager{}
		userCredentialUsecase := newTestUserCredentialUsecaseWithDeleteDependencies(
			tm, mockUserRepo, nil, "test-pepper",
		)

		user := &entity.User{ID: 1, Username: un}
		mockUserRepo.On("FindByIDForUpdate", mock.Anything, mock.Anything, 1).Return(user, nil).Once()
		mockUserRepo.On("DeleteByID", mock.Anything, mock.Anything, 1).Return(nil).Once()

		err := userCredentialUsecase.DeleteOwnAccount(context.Background(), 1)
		assert.NoError(t, err)
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("物理削除時にDBエラーが発生した場合はエラーを返す", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		tm := &mockTransactionManager{}
		userCredentialUsecase := newTestUserCredentialUsecaseWithDeleteDependencies(
			tm, mockUserRepo, nil, "test-pepper",
		)

		user := &entity.User{ID: 2, Username: un}
		mockUserRepo.On("FindByIDForUpdate", mock.Anything, mock.Anything, 2).Return(user, nil).Once()
		mockUserRepo.On("DeleteByID", mock.Anything, mock.Anything, 2).Return(errors.New("db error")).Once()

		err := userCredentialUsecase.DeleteOwnAccount(context.Background(), 2)
		assert.Error(t, err)
		assert.Equal(t, "db error", err.Error())
		mockUserRepo.AssertExpectations(t)
	})
}

func TestNewUserCredentialUsecase_必須依存がnilの場合はpanicする(t *testing.T) {
	pepper := "test-pepper"
	userRepo := new(MockUserRepository)
	playerRecordRepo := &stubPlayerRecordRepository{}
	masterCache := newMockMasterCache()
	exec := &MockExecutor{}

	tests := []struct {
		name    string
		build   func()
		message string
	}{
		{
			name: "executorがnil",
			build: func() {
				NewUserCredentialUsecase(nil, &mockTransactionManager{}, userRepo, playerRecordRepo, pepper, masterCache)
			},
			message: "executor is nil",
		},
		{
			name: "transaction managerがnil",
			build: func() {
				NewUserCredentialUsecase(exec, nil, userRepo, playerRecordRepo, pepper, masterCache)
			},
			message: "transaction manager is nil",
		},
		{
			name: "user repositoryがnil",
			build: func() {
				NewUserCredentialUsecase(exec, &mockTransactionManager{}, nil, playerRecordRepo, pepper, masterCache)
			},
			message: "user repository is nil",
		},
		{
			name: "player record repositoryがnil",
			build: func() {
				NewUserCredentialUsecase(exec, &mockTransactionManager{}, userRepo, nil, pepper, masterCache)
			},
			message: "player record repository is nil",
		},
		{
			name: "master cacheがnil",
			build: func() {
				NewUserCredentialUsecase(exec, &mockTransactionManager{}, userRepo, playerRecordRepo, pepper, nil)
			},
			message: "master cache is nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.PanicsWithValue(t, tt.message, tt.build)
		})
	}
}
