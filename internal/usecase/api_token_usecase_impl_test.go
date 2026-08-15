package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type apiTokenPassthroughTransactionManager struct{}

func (apiTokenPassthroughTransactionManager) Transactional(ctx context.Context, f func(repository.Executor) error) error {
	return f(nil)
}

type stubAPITokenRepository struct {
	tokens         map[uint64]*entity.APIToken
	nextID         uint64
	saveErr        error
	listErr        error
	findErr        error
	countOverride  *int
	deleteErr      error
	saveCalls      int
	forUpdateCalls int
	lockedToken    *entity.APIToken
}

func newStubAPITokenRepository() *stubAPITokenRepository {
	return &stubAPITokenRepository{tokens: make(map[uint64]*entity.APIToken), nextID: 1}
}

func (s *stubAPITokenRepository) Save(_ context.Context, _ repository.Executor, token *entity.APIToken) error {
	if s.saveErr != nil {
		return s.saveErr
	}
	s.saveCalls++
	if token.ID == 0 {
		token.ID = s.nextID
		s.nextID++
		token.CreatedAt = time.Date(2026, 7, 22, 12, 0, 0, 0, time.UTC)
	}
	s.tokens[token.ID] = cloneAPITokenForTest(token)
	return nil
}

func (s *stubAPITokenRepository) ListByUserID(_ context.Context, _ repository.Executor, userID int) ([]*entity.APIToken, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	tokens := make([]*entity.APIToken, 0)
	for _, token := range s.tokens {
		if token.UserID == userID {
			tokens = append(tokens, cloneAPITokenForTest(token))
		}
	}
	return tokens, nil
}

func (s *stubAPITokenRepository) FindByIDAndUserID(_ context.Context, _ repository.Executor, id uint64, userID int) (*entity.APIToken, error) {
	if s.findErr != nil {
		return nil, s.findErr
	}
	token, ok := s.tokens[id]
	if !ok || token.UserID != userID {
		return nil, repository.ErrAPITokenNotFound
	}
	return cloneAPITokenForTest(token), nil
}

func (s *stubAPITokenRepository) FindByIDAndUserIDForUpdate(ctx context.Context, exec repository.Executor, id uint64, userID int) (*entity.APIToken, error) {
	s.forUpdateCalls++
	return s.FindByIDAndUserID(ctx, exec, id, userID)
}

func (s *stubAPITokenRepository) FindByHashedToken(_ context.Context, _ repository.Executor, hashedToken string) (*entity.APIToken, error) {
	if s.findErr != nil {
		return nil, s.findErr
	}
	for _, token := range s.tokens {
		if token.HashedToken == hashedToken {
			return cloneAPITokenForTest(token), nil
		}
	}
	return nil, repository.ErrAPITokenNotFound
}

func (s *stubAPITokenRepository) FindByHashedTokenForUpdate(ctx context.Context, exec repository.Executor, hashedToken string) (*entity.APIToken, error) {
	s.forUpdateCalls++
	if s.lockedToken != nil && s.lockedToken.HashedToken == hashedToken {
		return cloneAPITokenForTest(s.lockedToken), nil
	}
	return s.FindByHashedToken(ctx, exec, hashedToken)
}

func (s *stubAPITokenRepository) CountByUserID(_ context.Context, _ repository.Executor, userID int) (int, error) {
	if s.countOverride != nil {
		return *s.countOverride, nil
	}
	count := 0
	for _, token := range s.tokens {
		if token.UserID == userID {
			count++
		}
	}
	return count, nil
}

func (s *stubAPITokenRepository) DeleteByIDAndUserID(_ context.Context, _ repository.Executor, id uint64, userID int) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	token, ok := s.tokens[id]
	if !ok || token.UserID != userID {
		return repository.ErrAPITokenNotFound
	}
	delete(s.tokens, id)
	return nil
}

type tokenStubUserRepository struct {
	user    *entity.User
	findErr error
	locked  bool
}

func (s *tokenStubUserRepository) FindByID(_ context.Context, _ repository.Executor, id int) (*entity.User, error) {
	if s.findErr != nil {
		return nil, s.findErr
	}
	if s.user == nil || s.user.ID != id {
		return nil, repository.ErrUserNotFound
	}
	copyUser := *s.user
	return &copyUser, nil
}

func (s *tokenStubUserRepository) FindByIDForUpdate(ctx context.Context, exec repository.Executor, id int) (*entity.User, error) {
	s.locked = true
	return s.FindByID(ctx, exec, id)
}

func (s *tokenStubUserRepository) FindByUsername(context.Context, repository.Executor, string) (*entity.User, error) {
	return nil, errors.New("not implemented")
}
func (s *tokenStubUserRepository) FindAllWithPlayer(context.Context, repository.Executor, int, int, string) ([]entity.UserWithPlayer, error) {
	return nil, errors.New("not implemented")
}
func (s *tokenStubUserRepository) FindAllWithPlayerForAdmin(context.Context, repository.Executor, int, int, string) ([]entity.UserWithPlayer, error) {
	return nil, errors.New("not implemented")
}
func (s *tokenStubUserRepository) Save(context.Context, repository.Executor, *entity.User) error {
	return errors.New("not implemented")
}
func (s *tokenStubUserRepository) LinkFirebaseUID(context.Context, repository.Executor, int, *string, string, time.Time) error {
	return errors.New("not implemented")
}
func (s *tokenStubUserRepository) FindByFirebaseUID(context.Context, repository.Executor, string) (*entity.User, error) {
	return nil, errors.New("not implemented")
}
func (s *tokenStubUserRepository) DeleteByID(context.Context, repository.Executor, int) error {
	return errors.New("not implemented")
}

func TestAPITokenUsecase_Generate_ExistingTokenRemainsAvailable(t *testing.T) {
	// Given
	repo := newStubAPITokenRepository()
	legacy := newAPITokenForTest(t, 1, 10, "既存のトークン", "legacy-token", nil, nil)
	repo.tokens[legacy.ID] = legacy
	repo.nextID = 2
	users := &tokenStubUserRepository{user: &entity.User{ID: 10}}
	uc := newAPITokenUsecaseWithClock(nil, apiTokenPassthroughTransactionManager{}, repo, users, time.Now)

	// When
	generated, err := uc.Generate(context.Background(), 10, "  Discord Bot  ")

	// Then
	require.NoError(t, err)
	require.NotNil(t, generated)
	assert.NotEmpty(t, generated.Token)
	assert.Equal(t, "Discord Bot", generated.Metadata.Name)
	require.NotNil(t, generated.Metadata.TokenPrefix)
	assert.Equal(t, generated.Token[:info.APITokenPrefixLength], *generated.Metadata.TokenPrefix)
	assert.Equal(t, hashToken(generated.Token), repo.tokens[generated.Metadata.ID].HashedToken)
	assert.Contains(t, repo.tokens, legacy.ID)
	assert.True(t, users.locked)
}

func TestAPITokenUsecase_Generate_RejectsLimitAndDuplicateName(t *testing.T) {
	tests := []struct {
		name    string
		repo    *stubAPITokenRepository
		wantErr error
	}{
		{
			name: "10個発行済み",
			repo: func() *stubAPITokenRepository {
				repo := newStubAPITokenRepository()
				count := info.APITokenMaxPerUser
				repo.countOverride = &count
				return repo
			}(),
			wantErr: ErrAPITokenLimitExceeded,
		},
		{
			name: "同名トークンが存在する",
			repo: func() *stubAPITokenRepository {
				repo := newStubAPITokenRepository()
				repo.saveErr = repository.ErrAPITokenConflict
				return repo
			}(),
			wantErr: ErrAPITokenNameConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := NewAPITokenUsecase(nil, apiTokenPassthroughTransactionManager{}, tt.repo, &tokenStubUserRepository{user: &entity.User{ID: 10}})

			_, err := uc.Generate(context.Background(), 10, "CLI")

			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestAPITokenUsecase_Generate_RejectsInvalidName(t *testing.T) {
	uc := NewAPITokenUsecase(nil, apiTokenPassthroughTransactionManager{}, newStubAPITokenRepository(), &tokenStubUserRepository{user: &entity.User{ID: 10}})

	_, err := uc.Generate(context.Background(), 10, " ")

	assert.ErrorIs(t, err, ErrInvalidAPITokenName)
}

func TestAPITokenUsecase_ListAndRename(t *testing.T) {
	repo := newStubAPITokenRepository()
	token := newAPITokenForTest(t, 5, 10, "CLI", "plain-token", stringPointer("plain"), nil)
	repo.tokens[token.ID] = token
	uc := NewAPITokenUsecase(nil, apiTokenPassthroughTransactionManager{}, repo, &tokenStubUserRepository{})

	tokens, err := uc.List(context.Background(), 10)
	require.NoError(t, err)
	require.Len(t, tokens, 1)
	assert.Equal(t, "CLI", tokens[0].Name)

	renamed, err := uc.Rename(context.Background(), 10, "5", "  Batch  ")
	require.NoError(t, err)
	assert.Equal(t, "Batch", renamed.Name)
	assert.Equal(t, "Batch", repo.tokens[5].Name.String())
}

func TestAPITokenUsecase_Validate_UpdatesLastUsedAtWithoutDerivingLegacyPrefix(t *testing.T) {
	now := time.Date(2026, 7, 22, 15, 0, 0, 0, time.UTC)
	repo := newStubAPITokenRepository()
	legacy := newAPITokenForTest(t, 1, 10, "既存のトークン", "legacy-token", nil, nil)
	repo.tokens[legacy.ID] = legacy
	uc := newAPITokenUsecaseWithClock(nil, apiTokenPassthroughTransactionManager{}, repo, &tokenStubUserRepository{user: &entity.User{ID: 10}}, func() time.Time { return now })

	user, token, err := uc.Validate(context.Background(), "legacy-token")

	require.NoError(t, err)
	assert.Equal(t, 10, user.ID)
	assert.Nil(t, token.TokenPrefix)
	require.NotNil(t, token.LastUsedAt)
	assert.Equal(t, now, *token.LastUsedAt)
	assert.Equal(t, 1, repo.saveCalls)
	assert.Equal(t, 1, repo.forUpdateCalls)
}

func TestAPITokenUsecase_Validate_DoesNotUpdateRecentLastUsedAt(t *testing.T) {
	now := time.Date(2026, 7, 22, 15, 0, 0, 0, time.UTC)
	lastUsedAt := now.Add(-30 * time.Minute)
	repo := newStubAPITokenRepository()
	token := newAPITokenForTest(t, 1, 10, "CLI", "plain-token", stringPointer("plain"), &lastUsedAt)
	repo.tokens[token.ID] = token
	uc := newAPITokenUsecaseWithClock(nil, apiTokenPassthroughTransactionManager{}, repo, &tokenStubUserRepository{user: &entity.User{ID: 10}}, func() time.Time { return now })

	_, _, err := uc.Validate(context.Background(), "plain-token")

	require.NoError(t, err)
	assert.Zero(t, repo.saveCalls)
}

func TestAPITokenUsecase_Validate_RechecksLastUsedAtAfterLock(t *testing.T) {
	// Given: ロック待ち中に別リクエストが最終利用日時を更新した状態
	now := time.Date(2026, 7, 22, 15, 0, 0, 0, time.UTC)
	recentlyUsedAt := now.Add(-time.Minute)
	repo := newStubAPITokenRepository()
	repo.tokens[1] = newAPITokenForTest(t, 1, 10, "CLI", "plain-token", stringPointer("plain"), nil)
	repo.lockedToken = newAPITokenForTest(t, 1, 10, "CLI", "plain-token", stringPointer("plain"), &recentlyUsedAt)
	uc := newAPITokenUsecaseWithClock(nil, apiTokenPassthroughTransactionManager{}, repo, &tokenStubUserRepository{user: &entity.User{ID: 10}}, func() time.Time { return now })

	// When
	_, token, err := uc.Validate(context.Background(), "plain-token")

	// Then
	require.NoError(t, err)
	require.NotNil(t, token.LastUsedAt)
	assert.Equal(t, recentlyUsedAt, *token.LastUsedAt)
	assert.Zero(t, repo.saveCalls)
	assert.Equal(t, 1, repo.forUpdateCalls)
}

func TestAPITokenUsecase_Validate_InvalidCases(t *testing.T) {
	tests := []struct {
		name  string
		raw   string
		repo  *stubAPITokenRepository
		users *tokenStubUserRepository
	}{
		{name: "空トークン", raw: "", repo: newStubAPITokenRepository(), users: &tokenStubUserRepository{}},
		{name: "トークン不明", raw: "unknown", repo: newStubAPITokenRepository(), users: &tokenStubUserRepository{}},
		{name: "ユーザー不明", raw: "valid", repo: func() *stubAPITokenRepository {
			repo := newStubAPITokenRepository()
			repo.tokens[1] = newAPITokenForTest(t, 1, 10, "CLI", "valid", stringPointer("valid"), nil)
			return repo
		}(), users: &tokenStubUserRepository{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := NewAPITokenUsecase(nil, apiTokenPassthroughTransactionManager{}, tt.repo, tt.users)
			_, _, err := uc.Validate(context.Background(), tt.raw)
			assert.ErrorIs(t, err, ErrInvalidAPIToken)
		})
	}
}

func TestAPITokenUsecase_Delete_IsScopedToOwner(t *testing.T) {
	repo := newStubAPITokenRepository()
	repo.tokens[1] = newAPITokenForTest(t, 1, 10, "CLI", "plain-token", stringPointer("plain"), nil)
	uc := NewAPITokenUsecase(nil, apiTokenPassthroughTransactionManager{}, repo, &tokenStubUserRepository{})

	err := uc.Delete(context.Background(), 11, "1")
	assert.ErrorIs(t, err, ErrAPITokenNotFound)
	assert.Contains(t, repo.tokens, uint64(1))

	require.NoError(t, uc.Delete(context.Background(), 10, "1"))
	assert.NotContains(t, repo.tokens, uint64(1))
}

func TestAPITokenUsecase_Delete_RejectsInvalidID(t *testing.T) {
	uc := NewAPITokenUsecase(nil, apiTokenPassthroughTransactionManager{}, newStubAPITokenRepository(), &tokenStubUserRepository{})

	err := uc.Delete(context.Background(), 10, "invalid")

	assert.ErrorIs(t, err, ErrInvalidAPITokenID)
}

func newAPITokenForTest(t *testing.T, id uint64, userID int, name string, raw string, prefix *string, lastUsedAt *time.Time) *entity.APIToken {
	t.Helper()
	token, err := entity.RestoreAPIToken(id, userID, name, hashToken(raw), prefix, lastUsedAt, time.Date(2026, 7, 22, 12, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	return token
}

func cloneAPITokenForTest(token *entity.APIToken) *entity.APIToken {
	cloned, err := entity.RestoreAPIToken(token.ID, token.UserID, token.Name.String(), token.HashedToken, token.TokenPrefix, token.LastUsedAt, token.CreatedAt)
	if err != nil {
		panic(err)
	}
	return cloned
}

func stringPointer(value string) *string {
	return &value
}
