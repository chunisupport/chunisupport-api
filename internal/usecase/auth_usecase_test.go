package usecase

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/passwordhash"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/username"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/chunisupport/chunisupport-api/internal/utils"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestAuthUsecase_Register(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	mockSessionRepo := new(MockSessionRepository)
	authService := NewAuthService(nil, nil, mockUserRepo, mockSessionRepo, nil, nil, "test-secret", 24, 24, "test-pepper", newMockMasterCache())

	t.Run("Register_正常系_ユーザー登録が成功する", func(t *testing.T) {
		mockUserRepo.On("FindByUsername", mock.Anything, mock.Anything, "testuser").Return(nil, sql.ErrNoRows).Once()
		mockUserRepo.On("Create", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
		mockSessionRepo.On("Create", mock.Anything, mock.Anything, mock.AnythingOfType("*entity.Session")).Return(nil).Once()
		mockSessionRepo.On("DeleteOldestSessionsOverLimit", mock.Anything, mock.Anything, mock.Anything, info.MaxSessionsPerUser).Return(nil).Once()

		userDTO, token, err := authService.Register(context.Background(), "testuser", "password")
		assert.NoError(t, err)
		assert.NotNil(t, userDTO)
		assert.Equal(t, "testuser", userDTO.Username)
		assert.NotEmpty(t, token)
		mockUserRepo.AssertExpectations(t)
		mockSessionRepo.AssertExpectations(t)
	})

	t.Run("Register_異常系_ユーザーが既に存在する", func(t *testing.T) {
		mockUserRepo.On("FindByUsername", mock.Anything, mock.Anything, "existinguser").Return(&entity.User{}, nil).Once()

		_, _, err := authService.Register(context.Background(), "existinguser", "password")
		assert.ErrorIs(t, err, ErrUsernameTaken)
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("Register_正常系_セッション数制限が機能する", func(t *testing.T) {
		mockUserRepo.On("FindByUsername", mock.Anything, mock.Anything, "sessionlimituser").Return(nil, sql.ErrNoRows).Once()
		mockUserRepo.On("Create", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
		mockSessionRepo.On("Create", mock.Anything, mock.Anything, mock.AnythingOfType("*entity.Session")).Return(nil).Once()
		mockSessionRepo.On("DeleteOldestSessionsOverLimit", mock.Anything, mock.Anything, mock.Anything, info.MaxSessionsPerUser).Return(nil).Once()

		userDTO, token, err := authService.Register(context.Background(), "sessionlimituser", "password")
		assert.NoError(t, err)
		assert.NotNil(t, userDTO)
		assert.NotEmpty(t, token)
		mockUserRepo.AssertExpectations(t)
		mockSessionRepo.AssertExpectations(t)
	})
}

func TestAuthUsecase_Login(t *testing.T) {
	hashedPassword, _ := utils.HashPasswordWithPepper("password", "test-pepper")
	ph, _ := passwordhash.NewPasswordHash(hashedPassword)
	un, _ := username.NewUserName("testuser")
	mockUser := &entity.User{ID: 1, Username: un, PasswordHash: ph}

	t.Run("Login_正常系_ログインが成功する", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		mockSessionRepo := new(MockSessionRepository)
		authService := NewAuthService(nil, nil, mockUserRepo, mockSessionRepo, nil, nil, "test-secret", 24, 24, "test-pepper", newMockMasterCache())

		mockUserRepo.On("FindByUsername", mock.Anything, mock.Anything, "testuser").Return(mockUser, nil).Once()
		mockSessionRepo.On("Create", mock.Anything, mock.Anything, mock.AnythingOfType("*entity.Session")).Return(nil).Once()
		mockSessionRepo.On("DeleteOldestSessionsOverLimit", mock.Anything, mock.Anything, mock.Anything, info.MaxSessionsPerUser).Return(nil).Once()

		token, err := authService.Login(context.Background(), "testuser", "password")
		assert.NoError(t, err)
		assert.NotEmpty(t, token)
		mockUserRepo.AssertExpectations(t)
		mockSessionRepo.AssertExpectations(t)
	})

	t.Run("Login_異常系_論理削除されたユーザーはログインできない", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		mockSessionRepo := new(MockSessionRepository)
		authService := NewAuthService(nil, nil, mockUserRepo, mockSessionRepo, nil, nil, "test-secret", 24, 24, "test-pepper", newMockMasterCache())

		deletedUser := &entity.User{ID: 2, Username: un, PasswordHash: ph, IsDeleted: true}
		mockUserRepo.On("FindByUsername", mock.Anything, mock.Anything, "testuser").Return(deletedUser, nil).Once()

		token, err := authService.Login(context.Background(), "testuser", "password")
		assert.ErrorIs(t, err, ErrInvalidCredentials)
		assert.Empty(t, token)
		mockUserRepo.AssertExpectations(t)
		mockSessionRepo.AssertNotCalled(t, "Create", mock.Anything)
	})
}

func TestAuthUsecase_Logout(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	mockSessionRepo := new(MockSessionRepository)
	authService := NewAuthService(nil, nil, mockUserRepo, mockSessionRepo, nil, nil, "test-secret", 24, 24, "test-pepper", newMockMasterCache())

	t.Run("Logout_正常系_ログアウトが成功する", func(t *testing.T) {
		sessionID := uuid.New().String()
		mockSessionRepo.On("Delete", mock.Anything, mock.Anything, sessionID).Return(nil).Once()

		err := authService.Logout(context.Background(), sessionID)
		assert.NoError(t, err)
		mockSessionRepo.AssertExpectations(t)
	})
}

func TestAuthUsecase_Authenticate(t *testing.T) {
	un, _ := username.NewUserName("testuser")
	mockUser := &entity.User{ID: 1, Username: un}
	sessionID := uuid.New().String()
	mockSession := &entity.Session{ID: sessionID, UserID: mockUser.ID, ExpiresAt: time.Now().Add(1 * time.Hour)}

	t.Run("Authenticate_正常系_認証が成功する", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		mockSessionRepo := new(MockSessionRepository)
		authService := NewAuthService(nil, nil, mockUserRepo, mockSessionRepo, nil, nil, "test-secret", 24, 24, "test-pepper", newMockMasterCache())

		mockSessionRepo.On("FindByID", mock.Anything, mock.Anything, sessionID).Return(mockSession, nil).Once()
		mockUserRepo.On("FindByID", mock.Anything, mock.Anything, mockUser.ID).Return(mockUser, nil).Once()

		user, err := authService.Authenticate(context.Background(), mockUser.ID, sessionID)
		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, mockUser.ID, user.ID)
		mockSessionRepo.AssertExpectations(t)
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("Authenticate_異常系_セッションが見つからない", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		mockSessionRepo := new(MockSessionRepository)
		authService := NewAuthService(nil, nil, mockUserRepo, mockSessionRepo, nil, nil, "test-secret", 24, 24, "test-pepper", newMockMasterCache())

		mockSessionRepo.On("FindByID", mock.Anything, mock.Anything, "invalidsession").Return(nil, sql.ErrNoRows).Once()

		_, err := authService.Authenticate(context.Background(), mockUser.ID, "invalidsession")
		assert.ErrorIs(t, err, ErrInvalidSession)
		mockSessionRepo.AssertExpectations(t)
	})

	t.Run("Authenticate_異常系_セッションのユーザーIDが不一致", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		mockSessionRepo := new(MockSessionRepository)
		authService := NewAuthService(nil, nil, mockUserRepo, mockSessionRepo, nil, nil, "test-secret", 24, 24, "test-pepper", newMockMasterCache())
		invalidUserID := 999

		mockSessionRepo.On("FindByID", mock.Anything, mock.Anything, sessionID).Return(mockSession, nil).Once()

		_, err := authService.Authenticate(context.Background(), invalidUserID, sessionID)
		assert.ErrorIs(t, err, ErrUserIDMismatch)
		mockSessionRepo.AssertExpectations(t)
	})

	t.Run("Authenticate_異常系_セッションが期限切れ", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		mockSessionRepo := new(MockSessionRepository)
		authService := NewAuthService(nil, nil, mockUserRepo, mockSessionRepo, nil, nil, "test-secret", 24, 24, "test-pepper", newMockMasterCache())
		expiredSession := &entity.Session{ID: sessionID, UserID: mockUser.ID, ExpiresAt: time.Now().Add(-1 * time.Hour)}
		mockSessionRepo.On("FindByID", mock.Anything, mock.Anything, sessionID).Return(expiredSession, nil).Once()
		mockSessionRepo.On("Delete", mock.Anything, mock.Anything, sessionID).Return(nil).Once()

		_, err := authService.Authenticate(context.Background(), mockUser.ID, sessionID)
		assert.ErrorIs(t, err, ErrInvalidSession)
		mockSessionRepo.AssertExpectations(t)
	})

	t.Run("Authenticate_異常系_論理削除されたユーザー", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		mockSessionRepo := new(MockSessionRepository)
		authService := NewAuthService(nil, nil, mockUserRepo, mockSessionRepo, nil, nil, "test-secret", 24, 24, "test-pepper", newMockMasterCache())
		deletedUser := &entity.User{ID: mockUser.ID, Username: un, IsDeleted: true}

		mockSessionRepo.On("FindByID", mock.Anything, mock.Anything, sessionID).Return(mockSession, nil).Once()
		mockUserRepo.On("FindByID", mock.Anything, mock.Anything, mockUser.ID).Return(deletedUser, nil).Once()
		mockSessionRepo.On("Delete", mock.Anything, mock.Anything, sessionID).Return(nil).Once()

		_, err := authService.Authenticate(context.Background(), mockUser.ID, sessionID)
		assert.ErrorIs(t, err, ErrUserDeleted)
		mockSessionRepo.AssertExpectations(t)
		mockUserRepo.AssertExpectations(t)
	})
}
