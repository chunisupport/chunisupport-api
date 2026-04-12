package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/username"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestUserSecurityUsecase_GetUser(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	userCredentialUsecase := newTestUserCredentialUsecase(mockUserRepo, nil)

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
		userCredentialUsecase := newTestUserCredentialUsecase(mockUserRepo, playerRecordRepo)

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
		userCredentialUsecase := newTestUserCredentialUsecase(mockUserRepo, playerRecordRepo)

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
			tm, mockUserRepo, nil,
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
			tm, mockUserRepo, nil,
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
				NewUserCredentialUsecase(nil, &mockTransactionManager{}, userRepo, playerRecordRepo, masterCache)
			},
			message: "executor is nil",
		},
		{
			name: "transaction managerがnil",
			build: func() {
				NewUserCredentialUsecase(exec, nil, userRepo, playerRecordRepo, masterCache)
			},
			message: "transaction manager is nil",
		},
		{
			name: "user repositoryがnil",
			build: func() {
				NewUserCredentialUsecase(exec, &mockTransactionManager{}, nil, playerRecordRepo, masterCache)
			},
			message: "user repository is nil",
		},
		{
			name: "player record repositoryがnil",
			build: func() {
				NewUserCredentialUsecase(exec, &mockTransactionManager{}, userRepo, nil, masterCache)
			},
			message: "player record repository is nil",
		},
		{
			name: "master cacheがnil",
			build: func() {
				NewUserCredentialUsecase(exec, &mockTransactionManager{}, userRepo, playerRecordRepo, nil)
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
