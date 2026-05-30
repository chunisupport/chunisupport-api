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
	savedToken    *entity.APIToken
	createErr     error
	userLookup    *entity.APIToken
	userLookupErr error
	lookupToken   *entity.APIToken
	lookupErr     error
	deletedUserID int
	deleteErr     error
}

func (s *stubAPITokenRepository) CreateOrReplace(ctx context.Context, exec repository.Executor, token *entity.APIToken) error {
	if s.createErr != nil {
		return s.createErr
	}
	copied := *token
	s.savedToken = &copied
	return nil
}

func (s *stubAPITokenRepository) FindByUserID(ctx context.Context, exec repository.Executor, userID int) (*entity.APIToken, error) {
	if s.userLookupErr != nil {
		return nil, s.userLookupErr
	}
	if s.userLookup == nil || s.userLookup.UserID != userID {
		return nil, repository.ErrAPITokenNotFound
	}
	tokenCopy := *s.userLookup
	return &tokenCopy, nil
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

func TestAPITokenUsecase_Generate(t *testing.T) {
	tokenRepo := &stubAPITokenRepository{}
	userRepo := &tokenStubUserRepository{}
	service := NewAPITokenUsecase(nil, tokenRepo, userRepo)

	token, err := service.Generate(context.Background(), 1)
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
}

func TestAPITokenUsecase_GetStatus(t *testing.T) {
	createdAt := time.Date(2026, 4, 16, 12, 34, 56, 0, time.UTC)
	tokenRepo := &stubAPITokenRepository{
		userLookup: &entity.APIToken{
			ID:        10,
			UserID:    123,
			CreatedAt: createdAt,
		},
	}
	userRepo := &tokenStubUserRepository{}
	service := NewAPITokenUsecase(nil, tokenRepo, userRepo)

	token, err := service.GetStatus(context.Background(), 123)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

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

func TestAPITokenUsecase_GetStatus_NotFound(t *testing.T) {
	tokenRepo := &stubAPITokenRepository{}
	userRepo := &tokenStubUserRepository{}
	service := NewAPITokenUsecase(nil, tokenRepo, userRepo)

	token, err := service.GetStatus(context.Background(), 123)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != nil {
		t.Fatalf("expected nil token, got %#v", token)
	}
}

func TestAPITokenUsecase_GetStatus_Error(t *testing.T) {
	expectedErr := errors.New("find failed")
	tokenRepo := &stubAPITokenRepository{userLookupErr: expectedErr}
	userRepo := &tokenStubUserRepository{}
	service := NewAPITokenUsecase(nil, tokenRepo, userRepo)

	token, err := service.GetStatus(context.Background(), 123)
	if err != expectedErr {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
	if token != nil {
		t.Fatalf("expected nil token, got %#v", token)
	}
}

func TestAPITokenUsecase_Validate(t *testing.T) {
	user := &entity.User{ID: 2}
	hashed := hashToken("plain-token")
	tokenRepo := &stubAPITokenRepository{
		lookupToken: &entity.APIToken{ID: 10, UserID: user.ID, HashedToken: hashed},
	}
	userRepo := &tokenStubUserRepository{user: user}
	service := NewAPITokenUsecase(nil, tokenRepo, userRepo)

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

func TestAPITokenUsecase_Validate_InvalidCases(t *testing.T) {
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
			service := NewAPITokenUsecase(nil, tc.tokenRepo, tc.userRepo)
			_, _, err := service.Validate(context.Background(), tc.input)
			if !errors.Is(err, ErrInvalidAPIToken) {
				t.Fatalf("expected ErrInvalidAPIToken, got %v", err)
			}
		})
	}
}

func TestAPITokenUsecase_Delete(t *testing.T) {
	tokenRepo := &stubAPITokenRepository{}
	userRepo := &tokenStubUserRepository{}
	service := NewAPITokenUsecase(nil, tokenRepo, userRepo)

	err := service.Delete(context.Background(), 123)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tokenRepo.deletedUserID != 123 {
		t.Fatalf("expected deletedUserID to be 123, got %d", tokenRepo.deletedUserID)
	}
}

func TestAPITokenUsecase_Delete_Error(t *testing.T) {
	expectedErr := errors.New("delete failed")
	tokenRepo := &stubAPITokenRepository{deleteErr: expectedErr}
	userRepo := &tokenStubUserRepository{}
	service := NewAPITokenUsecase(nil, tokenRepo, userRepo)

	err := service.Delete(context.Background(), 123)
	if err != expectedErr {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}
