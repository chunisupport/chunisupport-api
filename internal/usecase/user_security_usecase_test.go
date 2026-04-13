package usecase

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/reauthtoken"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/username"
	"github.com/chunisupport/chunisupport-api/internal/info"
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
	currentTime := time.Date(2026, 4, 13, 12, 0, 0, 0, time.UTC)
	recentAuthTime := currentTime.Add(-1 * time.Minute)

	t.Run("アカウント削除に成功する", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		recentSignInVerifier := new(mockRecentSignInVerifier)
		tm := &mockTransactionManager{}
		userCredentialUsecase := newTestUserCredentialUsecaseWithDeleteDependencies(
			tm, mockUserRepo, nil, recentSignInVerifier, currentTime,
		)

		user := &entity.User{ID: 1, Username: un}
		recentSignInVerifier.On("VerifyRecentSignIn", mock.Anything, "reauth-token").Return(&RecentSignInInfo{UID: "firebase-uid", AuthTime: recentAuthTime}, nil).Once()
		user.FirebaseUID = ptrString("firebase-uid")
		mockUserRepo.On("FindByIDForUpdate", mock.Anything, mock.Anything, 1).Return(user, nil).Once()
		mockUserRepo.On("DeleteByID", mock.Anything, mock.Anything, 1).Return(nil).Once()

		err := userCredentialUsecase.DeleteOwnAccount(context.Background(), 1, reauthtoken.MustNew("reauth-token"))
		assert.NoError(t, err)
		mockUserRepo.AssertExpectations(t)
		recentSignInVerifier.AssertExpectations(t)
	})

	t.Run("物理削除時にDBエラーが発生した場合はエラーを返す", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		recentSignInVerifier := new(mockRecentSignInVerifier)
		tm := &mockTransactionManager{}
		userCredentialUsecase := newTestUserCredentialUsecaseWithDeleteDependencies(
			tm, mockUserRepo, nil, recentSignInVerifier, currentTime,
		)

		user := &entity.User{ID: 2, Username: un}
		recentSignInVerifier.On("VerifyRecentSignIn", mock.Anything, "reauth-token").Return(&RecentSignInInfo{UID: "firebase-uid", AuthTime: recentAuthTime}, nil).Once()
		user.FirebaseUID = ptrString("firebase-uid")
		mockUserRepo.On("FindByIDForUpdate", mock.Anything, mock.Anything, 2).Return(user, nil).Once()
		mockUserRepo.On("DeleteByID", mock.Anything, mock.Anything, 2).Return(errors.New("db error")).Once()

		err := userCredentialUsecase.DeleteOwnAccount(context.Background(), 2, reauthtoken.MustNew("reauth-token"))
		assert.Error(t, err)
		assert.Equal(t, "db error", err.Error())
		mockUserRepo.AssertExpectations(t)
		recentSignInVerifier.AssertExpectations(t)
	})

	t.Run("再認証トークンが空なら recent sign-in required を返す", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)

		_, err := reauthtoken.New("   ")
		assert.ErrorIs(t, err, reauthtoken.ErrEmpty)
		mockUserRepo.AssertNotCalled(t, "FindByIDForUpdate", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("不正な再認証トークンなら recent sign-in required を返す", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		recentSignInVerifier := new(mockRecentSignInVerifier)
		userCredentialUsecase := newTestUserCredentialUsecaseWithDeleteDependencies(
			&mockTransactionManager{}, mockUserRepo, nil, recentSignInVerifier, currentTime,
		)

		recentSignInVerifier.On("VerifyRecentSignIn", mock.Anything, "invalid-token").Return(nil, errors.Join(ErrInvalidIDToken, errors.New("invalid token"))).Once()

		err := userCredentialUsecase.DeleteOwnAccount(context.Background(), 1, reauthtoken.MustNew("invalid-token"))
		assert.ErrorIs(t, err, ErrRecentSignInRequired)
		mockUserRepo.AssertNotCalled(t, "FindByIDForUpdate", mock.Anything, mock.Anything, mock.Anything)
		recentSignInVerifier.AssertExpectations(t)
	})

	t.Run("再認証トークンのauth_timeが欠落しているなら recent sign-in required を返す", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		recentSignInVerifier := new(mockRecentSignInVerifier)
		userCredentialUsecase := newTestUserCredentialUsecaseWithDeleteDependencies(
			&mockTransactionManager{}, mockUserRepo, nil, recentSignInVerifier, currentTime,
		)

		recentSignInVerifier.On("VerifyRecentSignIn", mock.Anything, "missing-auth-time-token").Return(nil, errors.Join(ErrRecentSignInAuthTimeMissing, errors.New("firebase token auth_time is empty"))).Once()

		err := userCredentialUsecase.DeleteOwnAccount(context.Background(), 1, reauthtoken.MustNew("missing-auth-time-token"))
		assert.ErrorIs(t, err, ErrRecentSignInRequired)
		assert.ErrorIs(t, err, ErrRecentSignInAuthTimeMissing)
		mockUserRepo.AssertNotCalled(t, "FindByIDForUpdate", mock.Anything, mock.Anything, mock.Anything)
		recentSignInVerifier.AssertExpectations(t)
	})

	t.Run("recent sign-in が期限切れなら削除しない", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		recentSignInVerifier := new(mockRecentSignInVerifier)
		userCredentialUsecase := newTestUserCredentialUsecaseWithDeleteDependencies(
			&mockTransactionManager{}, mockUserRepo, nil, recentSignInVerifier, currentTime,
		)

		recentSignInVerifier.On("VerifyRecentSignIn", mock.Anything, "expired-token").Return(&RecentSignInInfo{UID: "firebase-uid", AuthTime: currentTime.Add(-info.RecentSignInMaxAge - time.Second)}, nil).Once()

		err := userCredentialUsecase.DeleteOwnAccount(context.Background(), 1, reauthtoken.MustNew("expired-token"))
		assert.ErrorIs(t, err, ErrRecentSignInRequired)
		assert.ErrorIs(t, err, ErrRecentSignInExpired)
		mockUserRepo.AssertNotCalled(t, "FindByIDForUpdate", mock.Anything, mock.Anything, mock.Anything)
		recentSignInVerifier.AssertExpectations(t)
	})

	t.Run("recent sign-in の auth_time が1分以内の未来なら許容する", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		recentSignInVerifier := new(mockRecentSignInVerifier)
		tm := &mockTransactionManager{}
		userCredentialUsecase := newTestUserCredentialUsecaseWithDeleteDependencies(
			tm, mockUserRepo, nil, recentSignInVerifier, currentTime,
		)

		user := &entity.User{ID: 3, Username: un, FirebaseUID: ptrString("firebase-uid")}
		recentSignInVerifier.On("VerifyRecentSignIn", mock.Anything, "slightly-future-token").Return(&RecentSignInInfo{UID: "firebase-uid", AuthTime: currentTime.Add(info.RecentSignInFutureAllowance - 30*time.Second)}, nil).Once()
		mockUserRepo.On("FindByIDForUpdate", mock.Anything, mock.Anything, 3).Return(user, nil).Once()
		mockUserRepo.On("DeleteByID", mock.Anything, mock.Anything, 3).Return(nil).Once()

		err := userCredentialUsecase.DeleteOwnAccount(context.Background(), 3, reauthtoken.MustNew("slightly-future-token"))
		assert.NoError(t, err)
		mockUserRepo.AssertExpectations(t)
		recentSignInVerifier.AssertExpectations(t)
	})

	t.Run("recent sign-in の auth_time が1分を超えて未来なら削除しない", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		recentSignInVerifier := new(mockRecentSignInVerifier)
		userCredentialUsecase := newTestUserCredentialUsecaseWithDeleteDependencies(
			&mockTransactionManager{}, mockUserRepo, nil, recentSignInVerifier, currentTime,
		)

		recentSignInVerifier.On("VerifyRecentSignIn", mock.Anything, "future-token").Return(&RecentSignInInfo{UID: "firebase-uid", AuthTime: currentTime.Add(info.RecentSignInFutureAllowance + time.Second)}, nil).Once()

		err := userCredentialUsecase.DeleteOwnAccount(context.Background(), 1, reauthtoken.MustNew("future-token"))
		assert.ErrorIs(t, err, ErrRecentSignInRequired)
		mockUserRepo.AssertNotCalled(t, "FindByIDForUpdate", mock.Anything, mock.Anything, mock.Anything)
		recentSignInVerifier.AssertExpectations(t)
	})

	t.Run("ユーザーのFirebase UIDと再認証UIDが不一致なら削除しない", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		recentSignInVerifier := new(mockRecentSignInVerifier)
		userCredentialUsecase := newTestUserCredentialUsecaseWithDeleteDependencies(
			&mockTransactionManager{}, mockUserRepo, nil, recentSignInVerifier, currentTime,
		)
		logBuffer := captureDefaultSlog(t)

		recentSignInVerifier.On("VerifyRecentSignIn", mock.Anything, "reauth-token").Return(&RecentSignInInfo{UID: "firebase-uid-a", AuthTime: recentAuthTime}, nil).Once()
		mockUserRepo.On("FindByIDForUpdate", mock.Anything, mock.Anything, 1).Return(&entity.User{ID: 1, Username: un, FirebaseUID: ptrString("firebase-uid-b")}, nil).Once()

		err := userCredentialUsecase.DeleteOwnAccount(context.Background(), 1, reauthtoken.MustNew("reauth-token"))
		assert.ErrorIs(t, err, ErrInvalidCredentials)
		assert.Contains(t, logBuffer.String(), "delete_account_reauth_uid_mismatch")
		assert.Contains(t, logBuffer.String(), "firebase-uid-a")
		assert.Contains(t, logBuffer.String(), "firebase-uid-b")
		mockUserRepo.AssertNotCalled(t, "DeleteByID", mock.Anything, mock.Anything, mock.Anything)
		mockUserRepo.AssertExpectations(t)
		recentSignInVerifier.AssertExpectations(t)
	})

	t.Run("ユーザーにFirebase UIDがなければ削除しない", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		recentSignInVerifier := new(mockRecentSignInVerifier)
		userCredentialUsecase := newTestUserCredentialUsecaseWithDeleteDependencies(
			&mockTransactionManager{}, mockUserRepo, nil, recentSignInVerifier, currentTime,
		)
		logBuffer := captureDefaultSlog(t)

		recentSignInVerifier.On("VerifyRecentSignIn", mock.Anything, "reauth-token").Return(&RecentSignInInfo{UID: "firebase-uid", AuthTime: recentAuthTime}, nil).Once()
		mockUserRepo.On("FindByIDForUpdate", mock.Anything, mock.Anything, 1).Return(&entity.User{ID: 1, Username: un}, nil).Once()

		err := userCredentialUsecase.DeleteOwnAccount(context.Background(), 1, reauthtoken.MustNew("reauth-token"))
		assert.ErrorIs(t, err, ErrInvalidCredentials)
		assert.Contains(t, logBuffer.String(), "delete_account_firebase_uid_not_linked")
		assert.Contains(t, logBuffer.String(), "firebase-uid")
		mockUserRepo.AssertNotCalled(t, "DeleteByID", mock.Anything, mock.Anything, mock.Anything)
		mockUserRepo.AssertExpectations(t)
		recentSignInVerifier.AssertExpectations(t)
	})

	t.Run("ユーザーのFirebase UIDが空文字列なら削除しない", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		recentSignInVerifier := new(mockRecentSignInVerifier)
		userCredentialUsecase := newTestUserCredentialUsecaseWithDeleteDependencies(
			&mockTransactionManager{}, mockUserRepo, nil, recentSignInVerifier, currentTime,
		)
		logBuffer := captureDefaultSlog(t)

		recentSignInVerifier.On("VerifyRecentSignIn", mock.Anything, "reauth-token").Return(&RecentSignInInfo{UID: "firebase-uid", AuthTime: recentAuthTime}, nil).Once()
		mockUserRepo.On("FindByIDForUpdate", mock.Anything, mock.Anything, 1).Return(&entity.User{ID: 1, Username: un, FirebaseUID: ptrString("   ")}, nil).Once()

		err := userCredentialUsecase.DeleteOwnAccount(context.Background(), 1, reauthtoken.MustNew("reauth-token"))
		assert.ErrorIs(t, err, ErrInvalidCredentials)
		assert.Contains(t, logBuffer.String(), "delete_account_firebase_uid_not_linked")
		assert.Contains(t, logBuffer.String(), "firebase-uid")
		mockUserRepo.AssertNotCalled(t, "DeleteByID", mock.Anything, mock.Anything, mock.Anything)
		mockUserRepo.AssertExpectations(t)
		recentSignInVerifier.AssertExpectations(t)
	})
}

func captureDefaultSlog(t *testing.T) *strings.Builder {
	t.Helper()

	var buffer strings.Builder
	original := slog.Default()
	logger := slog.New(slog.NewTextHandler(&buffer, nil))
	slog.SetDefault(logger)
	t.Cleanup(func() {
		slog.SetDefault(original)
	})

	return &buffer
}

func ptrString(value string) *string {
	return &value
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
