package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
)

type stubAPITokenRepository struct {
	savedToken     *entity.APIToken
	createErr      error
	userLookup     []*entity.APIToken
	userLookupErr  error
	lookupToken    *entity.APIToken
	lookupErr      error
	count          int
	countErr       error
	deletedTokenID int64
	deletedUserID  int
	deleteErr      error
}

func (s *stubAPITokenRepository) Create(ctx context.Context, exec repository.Executor, token *entity.APIToken) error {
	if s.createErr != nil {
		return s.createErr
	}
	copied := *token
	s.savedToken = &copied
	return nil
}

func (s *stubAPITokenRepository) FindByUserID(ctx context.Context, exec repository.Executor, userID int) ([]*entity.APIToken, error) {
	if s.userLookupErr != nil {
		return nil, s.userLookupErr
	}
	tokens := make([]*entity.APIToken, 0, len(s.userLookup))
	for _, token := range s.userLookup {
		if token.UserID == userID {
			tokenCopy := *token
			tokens = append(tokens, &tokenCopy)
		}
	}
	return tokens, nil
}

func (s *stubAPITokenRepository) FindByHashedToken(ctx context.Context, exec repository.Executor, hashedToken string) (*entity.APIToken, error) {
	if s.lookupErr != nil {
		return nil, s.lookupErr
	}
	if s.lookupToken == nil {
		return nil, repository.ErrAPITokenNotFound
	}
	if s.lookupToken.HashedToken != hashedToken {
		return nil, repository.ErrAPITokenNotFound
	}
	tokenCopy := *s.lookupToken
	return &tokenCopy, nil
}

func (s *stubAPITokenRepository) CountByUserID(ctx context.Context, exec repository.Executor, userID int) (int, error) {
	if s.countErr != nil {
		return 0, s.countErr
	}
	return s.count, nil
}

func (s *stubAPITokenRepository) DeleteByID(ctx context.Context, exec repository.Executor, userID int, tokenID int64) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	s.deletedUserID = userID
	s.deletedTokenID = tokenID
	return nil
}

func (s *stubAPITokenRepository) DeleteByUserID(ctx context.Context, exec repository.Executor, userID int) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	s.deletedUserID = userID
	return nil
}

type tokenStubUserRepository struct {
	user    *entity.User
	findErr error
}

func (s *tokenStubUserRepository) FindByID(ctx context.Context, exec repository.Executor, id int) (*entity.User, error) {
	if s.findErr != nil {
		return nil, s.findErr
	}
	if s.user == nil {
		return nil, repository.ErrUserNotFound
	}
	userCopy := *s.user
	return &userCopy, nil
}

func (s *tokenStubUserRepository) FindByIDForUpdate(ctx context.Context, exec repository.Executor, id int) (*entity.User, error) {
	return s.FindByID(ctx, exec, id)
}

func (s *tokenStubUserRepository) FindByUsername(ctx context.Context, exec repository.Executor, username string) (*entity.User, error) {
	return nil, errors.New("not implemented")
}

func (s *tokenStubUserRepository) FindAllWithPlayer(ctx context.Context, exec repository.Executor, limit int, offset int, searchName string) ([]entity.UserWithPlayer, error) {
	return nil, errors.New("not implemented")
}

func (s *tokenStubUserRepository) FindAllWithPlayerForAdmin(ctx context.Context, exec repository.Executor, limit int, offset int, searchName string) ([]entity.UserWithPlayer, error) {
	return nil, errors.New("not implemented")
}

func (s *tokenStubUserRepository) Save(ctx context.Context, exec repository.Executor, user *entity.User) error {
	return errors.New("not implemented")
}

func (s *tokenStubUserRepository) LinkFirebaseUID(ctx context.Context, exec repository.Executor, userID int, currentUID *string, newUID string, updatedAt time.Time) error {
	return errors.New("not implemented")
}

func (s *tokenStubUserRepository) FindByFirebaseUID(_ context.Context, _ repository.Executor, _ string) (*entity.User, error) {
	return nil, errors.New("not implemented")
}

func (s *tokenStubUserRepository) DeleteByID(_ context.Context, _ repository.Executor, _ int) error {
	return errors.New("not implemented")
}

func TestAPITokenService_Generate(t *testing.T) {
	tokenRepo := &stubAPITokenRepository{}
	userRepo := &tokenStubUserRepository{}
	service := NewAPITokenService(nil, tokenRepo, userRepo)

	token, savedToken, err := service.Generate(context.Background(), 1, "メイン")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if token == "" {
		t.Fatalf("expected token to be generated")
	}

	if tokenRepo.savedToken == nil {
		t.Fatalf("expected token to be saved")
	}

	expectedHash := hashToken(token)
	if tokenRepo.savedToken.HashedToken != expectedHash {
		t.Fatalf("expected hashed token %s, got %s", expectedHash, tokenRepo.savedToken.HashedToken)
	}
	if tokenRepo.savedToken.UserID != 1 {
		t.Fatalf("expected user id 1, got %d", tokenRepo.savedToken.UserID)
	}
	if savedToken.Name != "メイン" {
		t.Fatalf("expected token name メイン, got %s", savedToken.Name)
	}
}

func TestAPITokenService_Generate_UsesDefaultNameWhenNameIsBlank(t *testing.T) {
	tokenRepo := &stubAPITokenRepository{}
	userRepo := &tokenStubUserRepository{}
	service := NewAPITokenService(nil, tokenRepo, userRepo)

	_, savedToken, err := service.Generate(context.Background(), 1, " ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if savedToken.Name != "APIキー" {
		t.Fatalf("expected default token name APIキー, got %s", savedToken.Name)
	}
}

func TestAPITokenService_Generate_RejectsTooLongName(t *testing.T) {
	tokenRepo := &stubAPITokenRepository{}
	userRepo := &tokenStubUserRepository{}
	service := NewAPITokenService(nil, tokenRepo, userRepo)

	_, _, err := service.Generate(context.Background(), 1, "1234567890123456")
	if !errors.Is(err, ErrInvalidAPITokenName) {
		t.Fatalf("expected ErrInvalidAPITokenName, got %v", err)
	}
}

func TestAPITokenService_Generate_RejectsWhenUserAlreadyHasTenTokens(t *testing.T) {
	tokenRepo := &stubAPITokenRepository{count: 10}
	userRepo := &tokenStubUserRepository{}
	service := NewAPITokenService(nil, tokenRepo, userRepo)

	_, _, err := service.Generate(context.Background(), 1, "追加用")
	if !errors.Is(err, ErrAPITokenLimitExceeded) {
		t.Fatalf("expected ErrAPITokenLimitExceeded, got %v", err)
	}
}

func TestAPITokenService_List(t *testing.T) {
	createdAt := time.Date(2026, 4, 16, 12, 34, 56, 0, time.UTC)
	tokenRepo := &stubAPITokenRepository{
		userLookup: []*entity.APIToken{{
			ID:        10,
			UserID:    123,
			Name:      "テスト",
			CreatedAt: createdAt,
		}},
	}
	userRepo := &tokenStubUserRepository{}
	service := NewAPITokenService(nil, tokenRepo, userRepo)

	tokens, err := service.List(context.Background(), 123)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tokens) != 1 {
		t.Fatalf("expected 1 token, got %d", len(tokens))
	}
	token := tokens[0]
	if token == nil {
		t.Fatalf("expected token status")
	}
	if token.ID != 10 {
		t.Fatalf("expected token id 10, got %d", token.ID)
	}
	if !token.CreatedAt.Equal(createdAt) {
		t.Fatalf("expected created_at %v, got %v", createdAt, token.CreatedAt)
	}
}

func TestAPITokenService_List_NotFound(t *testing.T) {
	tokenRepo := &stubAPITokenRepository{}
	userRepo := &tokenStubUserRepository{}
	service := NewAPITokenService(nil, tokenRepo, userRepo)

	tokens, err := service.List(context.Background(), 123)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tokens) != 0 {
		t.Fatalf("expected empty tokens, got %#v", tokens)
	}
}

func TestAPITokenService_List_Error(t *testing.T) {
	expectedErr := errors.New("find failed")
	tokenRepo := &stubAPITokenRepository{userLookupErr: expectedErr}
	userRepo := &tokenStubUserRepository{}
	service := NewAPITokenService(nil, tokenRepo, userRepo)

	tokens, err := service.List(context.Background(), 123)
	if err != expectedErr {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
	if tokens != nil {
		t.Fatalf("expected nil tokens, got %#v", tokens)
	}
}

func TestAPITokenService_Validate(t *testing.T) {
	user := &entity.User{ID: 2}
	hashed := hashToken("plain-token")
	tokenRepo := &stubAPITokenRepository{
		lookupToken: &entity.APIToken{ID: 10, UserID: user.ID, HashedToken: hashed},
	}
	userRepo := &tokenStubUserRepository{user: user}
	service := NewAPITokenService(nil, tokenRepo, userRepo)

	gotUser, apiToken, err := service.Validate(context.Background(), "plain-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotUser.ID != user.ID {
		t.Fatalf("expected user id %d, got %d", user.ID, gotUser.ID)
	}
	if apiToken.ID != 10 {
		t.Fatalf("expected token id 10, got %d", apiToken.ID)
	}
}

func TestAPITokenService_Validate_InvalidCases(t *testing.T) {
	user := &entity.User{ID: 1}
	hashed := hashToken("valid")
	cases := map[string]struct {
		tokenRepo *stubAPITokenRepository
		userRepo  *tokenStubUserRepository
		input     string
	}{
		"empty token": {
			tokenRepo: &stubAPITokenRepository{},
			userRepo:  &tokenStubUserRepository{},
			input:     "",
		},
		"token not found": {
			tokenRepo: &stubAPITokenRepository{lookupToken: &entity.APIToken{HashedToken: hashed}},
			userRepo:  &tokenStubUserRepository{},
			input:     "different",
		},
		"user not found": {
			tokenRepo: &stubAPITokenRepository{lookupToken: &entity.APIToken{ID: 1, UserID: user.ID, HashedToken: hashed}},
			userRepo:  &tokenStubUserRepository{user: nil},
			input:     "valid",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			service := NewAPITokenService(nil, tc.tokenRepo, tc.userRepo)
			_, _, err := service.Validate(context.Background(), tc.input)
			if !errors.Is(err, ErrInvalidAPIToken) {
				t.Fatalf("expected ErrInvalidAPIToken, got %v", err)
			}
		})
	}
}

func TestAPITokenService_Delete(t *testing.T) {
	tokenRepo := &stubAPITokenRepository{}
	userRepo := &tokenStubUserRepository{}
	service := NewAPITokenService(nil, tokenRepo, userRepo)

	err := service.Delete(context.Background(), 123, 456)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tokenRepo.deletedUserID != 123 {
		t.Fatalf("expected deletedUserID to be 123, got %d", tokenRepo.deletedUserID)
	}
	if tokenRepo.deletedTokenID != 456 {
		t.Fatalf("expected deletedTokenID to be 456, got %d", tokenRepo.deletedTokenID)
	}
}

func TestAPITokenService_Delete_Error(t *testing.T) {
	expectedErr := errors.New("delete failed")
	tokenRepo := &stubAPITokenRepository{deleteErr: expectedErr}
	userRepo := &tokenStubUserRepository{}
	service := NewAPITokenService(nil, tokenRepo, userRepo)

	err := service.Delete(context.Background(), 123, 456)
	if err != expectedErr {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}
