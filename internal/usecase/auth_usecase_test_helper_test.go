package usecase

import (
	"context"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/stretchr/testify/mock"
)

// newMockMasterCache はテスト用のアカウント種別プロバイダを作成します。
func newMockMasterCache() AccountTypeProvider {
	return &stubAccountTypeProvider{
		nameByID: map[int]string{
			1: "PLAYER",
			2: "EDITOR",
			3: "ADMIN",
		},
	}
}

type stubAccountTypeProvider struct {
	nameByID map[int]string
}

func (s *stubAccountTypeProvider) GetAccountTypeNameByID(id int) string {
	if name, ok := s.nameByID[id]; ok {
		return name
	}

	return "UNKNOWN"
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

func (m *MockUserRepository) FindByIDForUpdate(ctx context.Context, exec repository.Executor, id int) (*entity.User, error) {
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

func (m *MockUserRepository) Save(ctx context.Context, exec repository.Executor, user *entity.User) error {
	args := m.Called(ctx, exec, user)
	return args.Error(0)
}

func (m *MockUserRepository) LinkFirebaseUID(ctx context.Context, exec repository.Executor, userID int, currentUID *string, newUID string, updatedAt time.Time) error {
	args := m.Called(ctx, exec, userID, currentUID, newUID, updatedAt)
	return args.Error(0)
}

func (m *MockUserRepository) FindByFirebaseUID(ctx context.Context, exec repository.Executor, uid string) (*entity.User, error) {
	args := m.Called(ctx, exec, uid)
	if u, ok := args.Get(0).(*entity.User); ok {
		return u, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserRepository) DeleteByID(ctx context.Context, exec repository.Executor, id int) error {
	args := m.Called(ctx, exec, id)
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

func (m *MockSessionRepository) DeleteByUserID(ctx context.Context, exec repository.Executor, userID int) error {
	args := m.Called(ctx, exec, userID)
	return args.Error(0)
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

// MockAPITokenRepository はAPITokenRepositoryのモックです。
type MockAPITokenRepository struct {
	mock.Mock
}

func (m *MockAPITokenRepository) CreateOrReplace(ctx context.Context, exec repository.Executor, token *entity.APIToken) error {
	args := m.Called(ctx, exec, token)
	return args.Error(0)
}

func (m *MockAPITokenRepository) FindByHashedToken(ctx context.Context, exec repository.Executor, hashedToken string) (*entity.APIToken, error) {
	args := m.Called(ctx, exec, hashedToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.APIToken), args.Error(1)
}

func (m *MockAPITokenRepository) DeleteByUserID(ctx context.Context, exec repository.Executor, userID int) error {
	args := m.Called(ctx, exec, userID)
	return args.Error(0)
}

type mockTransactionManager struct {
	exec repository.Executor
}

func (m *mockTransactionManager) Transactional(ctx context.Context, f func(tx repository.Executor) error) error {
	return f(m.exec)
}

type authMockSessionIssuer struct {
	mock.Mock
}

func (m *authMockSessionIssuer) IssueSession(ctx context.Context, user *entity.User) (string, error) {
	args := m.Called(ctx, user)
	return args.String(0), args.Error(1)
}

func newTestAuthUsecaseWithSessionIssuer(userRepo repository.UserRepository, sessionRepo repository.SessionRepository, sessionIssuer SessionIssuer, pepper string) AuthUsecase {
	return NewAuthUsecase(nil, userRepo, sessionRepo, sessionIssuer, pepper, newMockMasterCache())
}

func newTestAuthUsecase(userRepo repository.UserRepository, sessionRepo repository.SessionRepository, pepper string) AuthUsecase {
	sessionIssuer := NewSessionIssuer(nil, sessionRepo, "test-secret", 24, 24)
	return newTestAuthUsecaseWithSessionIssuer(userRepo, sessionRepo, sessionIssuer, pepper)
}

func newTestUserCredentialUsecase(userRepo repository.UserRepository, playerRecordRepo repository.PlayerRecordRepository, pepper string) UserCredentialUsecase {
	if playerRecordRepo == nil {
		playerRecordRepo = &stubPlayerRecordRepository{}
	}
	return NewUserCredentialUsecase(
		&MockExecutor{},
		&mockTransactionManager{},
		userRepo,
		playerRecordRepo,
		new(MockSessionRepository),
		new(MockAPITokenRepository),
		new(MockRecoveryCodeRepository),
		pepper,
		newMockMasterCache(),
	)
}

func newTestUserCredentialUsecaseWithDeleteDependencies(
	tm TransactionManager,
	userRepo repository.UserRepository,
	playerRecordRepo repository.PlayerRecordRepository,
	sessionRepo repository.SessionRepository,
	apiTokenRepo repository.APITokenRepository,
	recoveryCodeRepo repository.RecoveryCodeRepository,
	pepper string,
) UserCredentialUsecase {
	if playerRecordRepo == nil {
		playerRecordRepo = &stubPlayerRecordRepository{}
	}
	return NewUserCredentialUsecase(&MockExecutor{}, tm, userRepo, playerRecordRepo, sessionRepo, apiTokenRepo, recoveryCodeRepo, pepper, newMockMasterCache())
}

func newTestRecoveryUsecase(tm TransactionManager, userRepo repository.UserRepository, recoveryRepo repository.RecoveryCodeRepository, pepper string) RecoveryUsecase {
	return NewRecoveryUsecase(nil, tm, userRepo, recoveryRepo, pepper)
}
