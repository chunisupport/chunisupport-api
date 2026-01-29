package usecase

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Qman110101/chunisupport-api/internal/domain/entity"
	"github.com/Qman110101/chunisupport-api/internal/domain/repository"
	"github.com/Qman110101/chunisupport-api/internal/domain/vo/passwordhash"
	"github.com/Qman110101/chunisupport-api/internal/domain/vo/username"
	"github.com/Qman110101/chunisupport-api/internal/info"
	"github.com/Qman110101/chunisupport-api/internal/infra/masterdata"
	"github.com/Qman110101/chunisupport-api/internal/utils"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// newMockMasterCache はテスト用のマスタデータキャッシュを作成します。
func newMockMasterCache() *masterdata.Cache {
	return &masterdata.Cache{
		AccountTypes: map[string]masterdata.Item{
			"PLAYER": {ID: 1, Name: "PLAYER"},
			"EDITOR": {ID: 2, Name: "EDITOR"},
			"ADMIN":  {ID: 3, Name: "ADMIN"},
		},
	}
}

// MockUserRepository はUserRepositoryのモックです。
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) FindByID(ctx context.Context, exec repository.Executor, id int) (*entity.User, error) {
	args := m.Called(ctx, exec, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.User), args.Error(1)
}

func (m *MockUserRepository) FindByUsername(ctx context.Context, exec repository.Executor, username string) (*entity.User, error) {
	args := m.Called(ctx, exec, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.User), args.Error(1)
}

func (m *MockUserRepository) FindAllWithPlayer(ctx context.Context, exec repository.Executor, limit int, offset int, searchName string) ([]entity.UserWithPlayer, error) {
	args := m.Called(ctx, exec, limit, offset, searchName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]entity.UserWithPlayer), args.Error(1)
}

func (m *MockUserRepository) FindAllWithPlayerForAdmin(ctx context.Context, exec repository.Executor, limit int, offset int, searchName string) ([]entity.UserWithPlayer, error) {
	args := m.Called(ctx, exec, limit, offset, searchName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]entity.UserWithPlayer), args.Error(1)
}

func (m *MockUserRepository) Create(ctx context.Context, exec repository.Executor, user *entity.User) error {
	args := m.Called(ctx, exec, user)
	return args.Error(0)
}

func (m *MockUserRepository) UpdatePrivacy(ctx context.Context, exec repository.Executor, userID int, isPrivate bool) error {
	args := m.Called(ctx, exec, userID, isPrivate)
	return args.Error(0)
}

func (m *MockUserRepository) SoftDelete(ctx context.Context, exec repository.Executor, userID int) error {
	args := m.Called(ctx, exec, userID)
	return args.Error(0)
}

func (m *MockUserRepository) Restore(ctx context.Context, exec repository.Executor, userID int) error {
	args := m.Called(ctx, exec, userID)
	return args.Error(0)
}

func (m *MockUserRepository) LinkPlayer(ctx context.Context, exec repository.Executor, userID int, playerID int) error {
	return nil
}

func (m *MockUserRepository) UpdatePassword(ctx context.Context, exec repository.Executor, userID int, passwordHash string) error {
	args := m.Called(ctx, exec, userID, passwordHash)
	return args.Error(0)
}

func (m *MockUserRepository) Save(ctx context.Context, exec repository.Executor, user *entity.User) error {
	args := m.Called(ctx, exec, user)
	return args.Error(0)
}

// MockSessionRepository はSessionRepositoryのモックです。
type MockSessionRepository struct {
	mock.Mock
}

func (m *MockSessionRepository) Create(ctx context.Context, exec repository.Executor, session *entity.Session) error {
	args := m.Called(ctx, exec, session)
	return args.Error(0)
}

func (m *MockSessionRepository) FindByID(ctx context.Context, exec repository.Executor, id string) (*entity.Session, error) {
	args := m.Called(ctx, exec, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Session), args.Error(1)
}

func (m *MockSessionRepository) Delete(ctx context.Context, exec repository.Executor, id string) error {
	args := m.Called(ctx, exec, id)
	return args.Error(0)
}

func (m *MockSessionRepository) CountByUserID(ctx context.Context, exec repository.Executor, userID int) (int, error) {
	args := m.Called(ctx, exec, userID)
	return args.Int(0), args.Error(1)
}

func (m *MockSessionRepository) DeleteByUserIDExcept(ctx context.Context, exec repository.Executor, userID int, excludeSessionID string) error {
	args := m.Called(ctx, exec, userID, excludeSessionID)
	return args.Error(0)
}

func (m *MockSessionRepository) DeleteOldestSessionsOverLimit(ctx context.Context, exec repository.Executor, userID int, maxCount int) error {
	args := m.Called(ctx, exec, userID, maxCount)
	return args.Error(0)
}

// MockRecoveryCodeRepository はRecoveryCodeRepositoryのモックです。
type MockRecoveryCodeRepository struct {
	mock.Mock
}

func (m *MockRecoveryCodeRepository) CreateBatch(ctx context.Context, exec repository.Executor, codes []*entity.RecoveryCode) error {
	args := m.Called(ctx, exec, codes)
	return args.Error(0)
}

func (m *MockRecoveryCodeRepository) DeleteByUserID(ctx context.Context, exec repository.Executor, userID int) error {
	args := m.Called(ctx, exec, userID)
	return args.Error(0)
}

func (m *MockRecoveryCodeRepository) DeleteByID(ctx context.Context, exec repository.Executor, id uint32) error {
	args := m.Called(ctx, exec, id)
	return args.Error(0)
}

func (m *MockRecoveryCodeRepository) FindByHash(ctx context.Context, exec repository.Executor, codeHash []byte) (*entity.RecoveryCode, error) {
	args := m.Called(ctx, exec, codeHash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.RecoveryCode), args.Error(1)
}

func (m *MockRecoveryCodeRepository) FindByHashForUpdate(ctx context.Context, exec repository.Executor, codeHash []byte) (*entity.RecoveryCode, error) {
	args := m.Called(ctx, exec, codeHash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.RecoveryCode), args.Error(1)
}

type mockTransactionManager struct {
	exec repository.Executor
}

func (m *mockTransactionManager) Transactional(ctx context.Context, f func(tx repository.Executor) error) error {
	return f(m.exec)
}

// TestAuthService_Register はRegisterメソッドのテストです。
func TestAuthService_Register(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	mockSessionRepo := new(MockSessionRepository) // 使わないがNewAuthServiceに必要
	authService := NewAuthService(nil, nil, mockUserRepo, mockSessionRepo, nil, nil, "test-secret", 24, 24, "test-pepper", newMockMasterCache())

	t.Run("正常系: ユーザー登録が成功する", func(t *testing.T) {
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

	t.Run("異常系: ユーザーが既に存在する", func(t *testing.T) {
		mockUserRepo.On("FindByUsername", mock.Anything, mock.Anything, "existinguser").Return(&entity.User{}, nil).Once()

		_, _, err := authService.Register(context.Background(), "existinguser", "password")
		assert.ErrorIs(t, err, ErrUsernameTaken)
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("正常系: セッション数制限が機能する", func(t *testing.T) {
		// ユーザー登録時にセッション作成を行い、上限を超えた場合に古いセッションが削除されることを確認
		mockUserRepo.On("FindByUsername", mock.Anything, mock.Anything, "sessionlimituser").Return(nil, sql.ErrNoRows).Once()
		mockUserRepo.On("Create", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
		mockSessionRepo.On("Create", mock.Anything, mock.Anything, mock.AnythingOfType("*entity.Session")).Return(nil).Once()
		// DeleteOldestSessionsOverLimitが呼ばれ、MaxSessionsPerUserが渡されることを確認
		mockSessionRepo.On("DeleteOldestSessionsOverLimit", mock.Anything, mock.Anything, mock.Anything, info.MaxSessionsPerUser).Return(nil).Once()

		userDTO, token, err := authService.Register(context.Background(), "sessionlimituser", "password")
		assert.NoError(t, err)
		assert.NotNil(t, userDTO)
		assert.NotEmpty(t, token)
		mockUserRepo.AssertExpectations(t)
		mockSessionRepo.AssertExpectations(t)
	})
}

// TestAuthService_ChangePassword はChangePasswordメソッドのテストです。
func TestAuthService_ChangePassword(t *testing.T) {
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
			name:            "正常系: パスワード変更が成功する",
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
			wantErr: nil,
		},
		{
			name:            "異常系: ユーザーが見つからない",
			userID:          2,
			currentPassword: "old-password",
			newPassword:     "new-password",
			setupMock: func(m *MockUserRepository) {
				m.On("FindByID", mock.Anything, mock.Anything, 2).Return(nil, sql.ErrNoRows).Once()
			},
			wantErr: ErrUserNotFound,
		},
		{
			name:            "異常系: ユーザー検索時にデータベースエラー",
			userID:          1,
			currentPassword: "old-password",
			newPassword:     "new-password",
			setupMock: func(m *MockUserRepository) {
				m.On("FindByID", mock.Anything, mock.Anything, 1).Return(nil, errDB).Once()
			},
			wantErr: errDB,
		},
		{
			name:            "異常系: 現在のパスワードが間違っている",
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
			name:            "異常系: パスワード更新時にデータベースエラー",
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
			name:            "異常系: 新しいパスワードが現在のパスワードと同じ",
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
			wantErr: ErrInvalidPassword, // セキュリティ強化: 汎用エラー
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockUserRepo := new(MockUserRepository)
			mockSessionRepo := new(MockSessionRepository)
			authService := NewAuthService(nil, nil, mockUserRepo, mockSessionRepo, nil, nil, "test-secret", 24, 24, pepper, newMockMasterCache())

			tc.setupMock(mockUserRepo)

			err := authService.ChangePassword(context.Background(), tc.userID, tc.currentPassword, tc.newPassword)

			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
			} else {
				assert.NoError(t, err)
			}

			mockUserRepo.AssertExpectations(t)
		})
	}
}

// TestAuthService_Login はLoginメソッドのテストです。
func TestAuthService_Login(t *testing.T) {
	hashedPassword, _ := utils.HashPasswordWithPepper("password", "test-pepper")
	ph, _ := passwordhash.NewPasswordHash(hashedPassword)
	un, _ := username.NewUserName("testuser")
	mockUser := &entity.User{ID: 1, Username: un, PasswordHash: ph}

	t.Run("正常系: ログインが成功する", func(t *testing.T) {
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

	t.Run("異常系: 論理削除されたユーザーはログインできない", func(t *testing.T) {
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

// TestAuthService_Logout はLogoutメソッドのテストです。
func TestAuthService_Logout(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	mockSessionRepo := new(MockSessionRepository)
	authService := NewAuthService(nil, nil, mockUserRepo, mockSessionRepo, nil, nil, "test-secret", 24, 24, "test-pepper", newMockMasterCache())

	t.Run("正常系: ログアウトが成功する", func(t *testing.T) {
		sessionID := uuid.New().String()
		mockSessionRepo.On("Delete", mock.Anything, mock.Anything, sessionID).Return(nil).Once()

		err := authService.Logout(context.Background(), sessionID)
		assert.NoError(t, err)
		mockSessionRepo.AssertExpectations(t)
	})
}

// TestAuthService_Authenticate はAuthenticateメソッドのテストです。
func TestAuthService_Authenticate(t *testing.T) {
	un, _ := username.NewUserName("testuser")
	mockUser := &entity.User{ID: 1, Username: un}
	sessionID := uuid.New().String()
	mockSession := &entity.Session{
		ID:        sessionID,
		UserID:    mockUser.ID,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	t.Run("正常系: 認証が成功する", func(t *testing.T) {
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

	t.Run("異常系: セッションが見つからない", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		mockSessionRepo := new(MockSessionRepository)
		authService := NewAuthService(nil, nil, mockUserRepo, mockSessionRepo, nil, nil, "test-secret", 24, 24, "test-pepper", newMockMasterCache())

		mockSessionRepo.On("FindByID", mock.Anything, mock.Anything, "invalidsession").Return(nil, sql.ErrNoRows).Once()

		_, err := authService.Authenticate(context.Background(), mockUser.ID, "invalidsession")
		assert.ErrorIs(t, err, ErrInvalidSession) // セキュリティ強化: 統合エラー
		mockSessionRepo.AssertExpectations(t)
	})

	t.Run("異常系: セッションのユーザーIDが不一致", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		mockSessionRepo := new(MockSessionRepository)
		authService := NewAuthService(nil, nil, mockUserRepo, mockSessionRepo, nil, nil, "test-secret", 24, 24, "test-pepper", newMockMasterCache())
		invalidUserID := 999

		mockSessionRepo.On("FindByID", mock.Anything, mock.Anything, sessionID).Return(mockSession, nil).Once()

		_, err := authService.Authenticate(context.Background(), invalidUserID, sessionID)
		assert.ErrorIs(t, err, ErrUserIDMismatch)
		mockSessionRepo.AssertExpectations(t)
	})

	t.Run("異常系: セッションが期限切れ", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		mockSessionRepo := new(MockSessionRepository)
		authService := NewAuthService(nil, nil, mockUserRepo, mockSessionRepo, nil, nil, "test-secret", 24, 24, "test-pepper", newMockMasterCache())
		expiredSession := &entity.Session{
			ID:        sessionID,
			UserID:    mockUser.ID,
			ExpiresAt: time.Now().Add(-1 * time.Hour),
		}
		mockSessionRepo.On("FindByID", mock.Anything, mock.Anything, sessionID).Return(expiredSession, nil).Once()
		mockSessionRepo.On("Delete", mock.Anything, mock.Anything, sessionID).Return(nil).Once() // 期限切れセッションは削除される

		_, err := authService.Authenticate(context.Background(), mockUser.ID, sessionID)
		assert.ErrorIs(t, err, ErrInvalidSession) // セキュリティ強化: 統合エラー
		mockSessionRepo.AssertExpectations(t)
	})

	t.Run("異常系: 論理削除されたユーザー", func(t *testing.T) {
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

// TestAuthService_GetUser はGetUserメソッドのテストです。
func TestAuthService_GetUser(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	mockSessionRepo := new(MockSessionRepository)
	authService := NewAuthService(nil, nil, mockUserRepo, mockSessionRepo, nil, nil, "test-secret", 24, 24, "test-pepper", newMockMasterCache())

	t.Run("正常系: ユーザー取得が成功する", func(t *testing.T) {
		un, _ := username.NewUserName("testuser")
		mockUser := &entity.User{ID: 1, Username: un}
		mockUserRepo.On("FindByID", mock.Anything, mock.Anything, 1).Return(mockUser, nil).Once()

		userDTO, err := authService.GetUser(context.Background(), 1)
		assert.NoError(t, err)
		assert.NotNil(t, userDTO)
		assert.Equal(t, "testuser", userDTO.Username)
		mockUserRepo.AssertExpectations(t)
	})
}

// TestAuthService_DeleteUser はDeleteUserメソッドのテストです。
func TestAuthService_DeleteUser(t *testing.T) {
	un, _ := username.NewUserName("testuser")
	mockUser := &entity.User{ID: 1, Username: un}

	t.Run("正常系: 論理削除が成功する", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		mockSessionRepo := new(MockSessionRepository)
		authService := NewAuthService(nil, nil, mockUserRepo, mockSessionRepo, nil, nil, "test-secret", 24, 24, "test-pepper", newMockMasterCache())

		mockUserRepo.On("FindByID", mock.Anything, mock.Anything, 1).Return(mockUser, nil).Once()
		mockUserRepo.On("Save", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

		err := authService.DeleteUser(context.Background(), 1)
		assert.NoError(t, err)
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("異常系: リポジトリエラー", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		mockSessionRepo := new(MockSessionRepository)
		authService := NewAuthService(nil, nil, mockUserRepo, mockSessionRepo, nil, nil, "test-secret", 24, 24, "test-pepper", newMockMasterCache())

		mockUserRepo.On("FindByID", mock.Anything, mock.Anything, 2).Return(mockUser, nil).Once()
		mockUserRepo.On("Save", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("db error")).Once()

		err := authService.DeleteUser(context.Background(), 2)
		assert.Error(t, err)
		assert.Equal(t, "db error", err.Error())
		mockUserRepo.AssertExpectations(t)
	})
}

// TestAuthService_IssueRecoveryCodes はIssueRecoveryCodesメソッドのテストです。
func TestAuthService_IssueRecoveryCodes(t *testing.T) {
	t.Run("正常系: リカバリーコード再発行が成功する", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		mockSessionRepo := new(MockSessionRepository)
		mockRecoveryRepo := new(MockRecoveryCodeRepository)
		tm := &mockTransactionManager{}
		authService := NewAuthService(nil, tm, mockUserRepo, mockSessionRepo, mockRecoveryRepo, nil, "test-secret", 24, 24, "test-pepper", newMockMasterCache())

		un, _ := username.NewUserName("testuser")
		mockUser := &entity.User{ID: 1, Username: un}
		mockUserRepo.On("FindByID", mock.Anything, mock.Anything, 1).Return(mockUser, nil).Once()
		mockRecoveryRepo.On("DeleteByUserID", mock.Anything, mock.Anything, 1).Return(nil).Once()
		mockRecoveryRepo.On("CreateBatch", mock.Anything, mock.Anything, mock.MatchedBy(func(codes []*entity.RecoveryCode) bool {
			return len(codes) == info.RecoveryCodeCount
		})).Return(nil).Once()

		codes, err := authService.IssueRecoveryCodes(context.Background(), 1)
		assert.NoError(t, err)
		assert.Len(t, codes, info.RecoveryCodeCount)
		for _, code := range codes {
			segments := strings.Split(code, "-")
			assert.Len(t, segments, info.RecoveryCodeSegmentCount)
			for _, segment := range segments {
				assert.Len(t, segment, info.RecoveryCodeSegmentLength)
			}
		}

		mockUserRepo.AssertExpectations(t)
		mockRecoveryRepo.AssertExpectations(t)
	})

	t.Run("異常系: ユーザーが存在しない", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		mockSessionRepo := new(MockSessionRepository)
		mockRecoveryRepo := new(MockRecoveryCodeRepository)
		tm := &mockTransactionManager{}
		authService := NewAuthService(nil, tm, mockUserRepo, mockSessionRepo, mockRecoveryRepo, nil, "test-secret", 24, 24, "test-pepper", newMockMasterCache())

		mockUserRepo.On("FindByID", mock.Anything, mock.Anything, 1).Return(nil, sql.ErrNoRows).Once()

		_, err := authService.IssueRecoveryCodes(context.Background(), 1)
		assert.ErrorIs(t, err, ErrUserNotFound)
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("異常系: リカバリーコード削除に失敗する", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		mockSessionRepo := new(MockSessionRepository)
		mockRecoveryRepo := new(MockRecoveryCodeRepository)
		tm := &mockTransactionManager{}
		authService := NewAuthService(nil, tm, mockUserRepo, mockSessionRepo, mockRecoveryRepo, nil, "test-secret", 24, 24, "test-pepper", newMockMasterCache())

		un, _ := username.NewUserName("testuser")
		mockUser := &entity.User{ID: 1, Username: un}
		mockUserRepo.On("FindByID", mock.Anything, mock.Anything, 1).Return(mockUser, nil).Once()
		mockRecoveryRepo.On("DeleteByUserID", mock.Anything, mock.Anything, 1).Return(errors.New("delete error")).Once()

		_, err := authService.IssueRecoveryCodes(context.Background(), 1)
		assert.Error(t, err)
		mockUserRepo.AssertExpectations(t)
		mockRecoveryRepo.AssertExpectations(t)
	})
}

// TestAuthService_RecoverWithRecoveryCode はRecoverWithRecoveryCodeメソッドのテストです。
func TestAuthService_RecoverWithRecoveryCode(t *testing.T) {
	pepper := "test-pepper"
	errDB := errors.New("db error")
	recoveryCode := "ABCD-EFGH-IJKL"
	normalized := normalizeRecoveryCode(recoveryCode)
	hash := hashRecoveryCode(normalized)
	un, _ := username.NewUserName("testuser")
	oldHash, _ := utils.HashPasswordWithPepper("old-password", pepper)
	ph, _ := passwordhash.NewPasswordHash(oldHash)
	samePasswordHash, _ := utils.HashPasswordWithPepper("same-password", pepper)
	samePasswordPH, _ := passwordhash.NewPasswordHash(samePasswordHash)
	newTestCode := func() *entity.RecoveryCode {
		return &entity.RecoveryCode{
			UserID:   1,
			CodeHash: hash,
		}
	}
	newActiveUser := func() *entity.User {
		return &entity.User{ID: 1, Username: un, PasswordHash: ph}
	}
	newDeletedUser := func() *entity.User {
		return &entity.User{ID: 1, Username: un, PasswordHash: ph, IsDeleted: true}
	}
	newSamePasswordUser := func() *entity.User {
		return &entity.User{ID: 1, Username: un, PasswordHash: samePasswordPH}
	}

	tests := []struct {
		name        string
		newPassword string
		setupMock   func(*MockUserRepository, *MockRecoveryCodeRepository)
		wantErr     error
	}{
		{
			name:        "正常系: リカバリーコードで復旧できる",
			newPassword: "new-password",
			setupMock: func(userRepo *MockUserRepository, recoveryRepo *MockRecoveryCodeRepository) {
				recoveryRepo.On("FindByHashForUpdate", mock.Anything, mock.Anything, hash).Return(newTestCode(), nil).Once()
				userRepo.On("FindByID", mock.Anything, mock.Anything, 1).Return(newActiveUser(), nil).Once()
				userRepo.On("Save", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
				recoveryRepo.On("DeleteByID", mock.Anything, mock.Anything, mock.AnythingOfType("uint32")).Return(nil).Once()
			},
			wantErr: nil,
		},
		{
			name:        "異常系: リカバリーコードが見つからない",
			newPassword: "new-password",
			setupMock: func(userRepo *MockUserRepository, recoveryRepo *MockRecoveryCodeRepository) {
				recoveryRepo.On("FindByHashForUpdate", mock.Anything, mock.Anything, hash).Return(nil, sql.ErrNoRows).Once()
			},
			wantErr: ErrInvalidRecoveryCredentials,
		},
		{
			name:        "異常系: ユーザーが見つからない",
			newPassword: "new-password",
			setupMock: func(userRepo *MockUserRepository, recoveryRepo *MockRecoveryCodeRepository) {
				recoveryRepo.On("FindByHashForUpdate", mock.Anything, mock.Anything, hash).Return(newTestCode(), nil).Once()
				userRepo.On("FindByID", mock.Anything, mock.Anything, 1).Return(nil, sql.ErrNoRows).Once()
			},
			wantErr: ErrInvalidRecoveryCredentials,
		},
		{
			name:        "異常系: 非アクティブユーザー",
			newPassword: "new-password",
			setupMock: func(userRepo *MockUserRepository, recoveryRepo *MockRecoveryCodeRepository) {
				recoveryRepo.On("FindByHashForUpdate", mock.Anything, mock.Anything, hash).Return(newTestCode(), nil).Once()
				userRepo.On("FindByID", mock.Anything, mock.Anything, 1).Return(newDeletedUser(), nil).Once()
			},
			wantErr: ErrInvalidRecoveryCredentials,
		},
		{
			name:        "異常系: 新しいパスワードが現在のパスワードと同じ",
			newPassword: "same-password",
			setupMock: func(userRepo *MockUserRepository, recoveryRepo *MockRecoveryCodeRepository) {
				recoveryRepo.On("FindByHashForUpdate", mock.Anything, mock.Anything, hash).Return(newTestCode(), nil).Once()
				userRepo.On("FindByID", mock.Anything, mock.Anything, 1).Return(newSamePasswordUser(), nil).Once()
			},
			wantErr: ErrInvalidPassword,
		},
		{
			name:        "異常系: ユーザー更新に失敗する",
			newPassword: "new-password",
			setupMock: func(userRepo *MockUserRepository, recoveryRepo *MockRecoveryCodeRepository) {
				recoveryRepo.On("FindByHashForUpdate", mock.Anything, mock.Anything, hash).Return(newTestCode(), nil).Once()
				userRepo.On("FindByID", mock.Anything, mock.Anything, 1).Return(newActiveUser(), nil).Once()
				userRepo.On("Save", mock.Anything, mock.Anything, mock.Anything).Return(errDB).Once()
			},
			wantErr: errDB,
		},
		{
			name:        "異常系: リカバリーコード削除に失敗する",
			newPassword: "new-password",
			setupMock: func(userRepo *MockUserRepository, recoveryRepo *MockRecoveryCodeRepository) {
				recoveryRepo.On("FindByHashForUpdate", mock.Anything, mock.Anything, hash).Return(newTestCode(), nil).Once()
				userRepo.On("FindByID", mock.Anything, mock.Anything, 1).Return(newActiveUser(), nil).Once()
				userRepo.On("Save", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
				recoveryRepo.On("DeleteByID", mock.Anything, mock.Anything, mock.AnythingOfType("uint32")).Return(errDB).Once()
			},
			wantErr: errDB,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockUserRepo := new(MockUserRepository)
			mockSessionRepo := new(MockSessionRepository)
			mockRecoveryRepo := new(MockRecoveryCodeRepository)
			tm := &mockTransactionManager{}
			authService := NewAuthService(nil, tm, mockUserRepo, mockSessionRepo, mockRecoveryRepo, nil, "test-secret", 24, 24, pepper, newMockMasterCache())

			tc.setupMock(mockUserRepo, mockRecoveryRepo)

			err := authService.RecoverWithRecoveryCode(context.Background(), recoveryCode, tc.newPassword)
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
			} else {
				assert.NoError(t, err)
			}

			mockUserRepo.AssertExpectations(t)
			mockRecoveryRepo.AssertExpectations(t)
		})
	}
}
