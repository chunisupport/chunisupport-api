package usecase

import (
	"context"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/stretchr/testify/mock"
)

// newMockMasterCache はテスト用のアカウント種別プロバイダを作成します。
func newMockMasterCache() AccountTypeProvider {
	return &stubAccountTypeProvider{
		nameByID: map[int]string{
			info.AccountTypePlayer: "PLAYER",
			info.AccountTypeEditor: "EDITOR",
			info.AccountTypeAdmin:  "ADMIN",
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

type mockRecentSignInVerifier struct {
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

func (m *mockRecentSignInVerifier) VerifyRecentSignIn(ctx context.Context, idToken string) (*RecentSignInInfo, error) {
	args := m.Called(ctx, idToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RecentSignInInfo), args.Error(1)
}

type mockTransactionManager struct {
	exec repository.Executor
}

func (m *mockTransactionManager) Transactional(ctx context.Context, f func(tx repository.Executor) error) error {
	return f(m.exec)
}

func newTestUserCredentialUsecase(userRepo repository.UserRepository, playerRecordRepo repository.PlayerRecordRepository) UserCredentialUsecase {
	if playerRecordRepo == nil {
		playerRecordRepo = &stubPlayerRecordRepository{}
	}
	return NewUserCredentialUsecase(
		&MockExecutor{},
		&mockTransactionManager{},
		userRepo,
		playerRecordRepo,
		newMockMasterCache(),
	)
}

func newTestUserCredentialUsecaseWithDeleteDependencies(
	tm TransactionManager,
	userRepo repository.UserRepository,
	playerRecordRepo repository.PlayerRecordRepository,
	recentSignInVerifier RecentSignInVerifier,
) UserCredentialUsecase {
	if playerRecordRepo == nil {
		playerRecordRepo = &stubPlayerRecordRepository{}
	}
	return NewUserCredentialUsecaseWithFirebaseServices(&MockExecutor{}, tm, userRepo, playerRecordRepo, recentSignInVerifier, nil, newMockMasterCache())
}
