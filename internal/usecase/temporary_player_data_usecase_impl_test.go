package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubTemporaryPlayerDataRepository struct {
	createErr    error
	findErr      error
	found        *entity.TemporaryPlayerData
	deletedToken string
}

func (s *stubTemporaryPlayerDataRepository) Create(_ context.Context, data *entity.TemporaryPlayerData) error {
	if s.createErr != nil {
		return s.createErr
	}
	s.found = data
	return nil
}

func (s *stubTemporaryPlayerDataRepository) FindByToken(_ context.Context, _ string) (*entity.TemporaryPlayerData, error) {
	if s.findErr != nil {
		return nil, s.findErr
	}
	if s.found == nil {
		return nil, domainrepo.ErrTemporaryPlayerDataNotFound
	}
	copyData := *s.found
	copyData.Payload = append([]byte(nil), s.found.Payload...)
	return &copyData, nil
}

func (s *stubTemporaryPlayerDataRepository) ConsumeByToken(ctx context.Context, token string) (*entity.TemporaryPlayerData, error) {
	return s.FindByToken(ctx, token)
}

func (s *stubTemporaryPlayerDataRepository) Delete(_ context.Context, token string) error {
	s.deletedToken = token
	return nil
}

type stubPlayerDataUsecase struct {
	registerFn func(ctx context.Context, user *entity.User, payload *PlayerDataPayload, hash string) (*api_internal.PlayerDataResult, error)
}

func (s *stubPlayerDataUsecase) Register(ctx context.Context, user *entity.User, payload *PlayerDataPayload, hash string) (*api_internal.PlayerDataResult, error) {
	if s.registerFn != nil {
		return s.registerFn(ctx, user, payload, hash)
	}
	return &api_internal.PlayerDataResult{PlayerID: 1}, nil
}

func (s *stubPlayerDataUsecase) Delete(_ context.Context, _ *entity.User) error { return nil }

func TestTemporaryPlayerDataUsecase_Create(t *testing.T) {
	repo := &stubTemporaryPlayerDataRepository{}
	uc := NewTemporaryPlayerDataUsecase(repo, &stubPlayerDataUsecase{}, 5*time.Minute)

	result, err := uc.Create(context.Background(), CreateTemporaryPlayerDataInput{
		IPAddress: "127.0.0.1",
		Payload:   []byte(`{"name":"TEST"}`),
		BodyHash:  "hash",
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotEmpty(t, result.UploadToken)
	assert.WithinDuration(t, time.Now().UTC().Add(5*time.Minute), result.ExpiresAt, 3*time.Second)
}

func TestTemporaryPlayerDataUsecase_Create_PerIP上限超過(t *testing.T) {
	repo := &stubTemporaryPlayerDataRepository{createErr: domainrepo.ErrTemporaryPlayerDataPerIPLimitExceeded}
	uc := NewTemporaryPlayerDataUsecase(repo, &stubPlayerDataUsecase{}, 5*time.Minute)

	_, err := uc.Create(context.Background(), CreateTemporaryPlayerDataInput{IPAddress: "127.0.0.1", Payload: []byte("{}")})

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrTempDataPerIPLimitExceeded)
}

func TestTemporaryPlayerDataUsecase_Commit_成功時に削除される(t *testing.T) {
	repo := &stubTemporaryPlayerDataRepository{found: &entity.TemporaryPlayerData{Token: "token-1", Payload: []byte(`{"name":"TEST"}`), BodyHash: "hash"}}
	uc := NewTemporaryPlayerDataUsecase(repo, &stubPlayerDataUsecase{}, 5*time.Minute)

	result, err := uc.Commit(context.Background(), CommitTemporaryPlayerDataInput{
		User:        &entity.User{ID: 10},
		UploadToken: "token-1",
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "token-1", repo.deletedToken)
}

func TestTemporaryPlayerDataUsecase_Commit_DB失敗時は保持(t *testing.T) {
	repo := &stubTemporaryPlayerDataRepository{found: &entity.TemporaryPlayerData{Token: "token-1", Payload: []byte(`{"name":"TEST"}`)}}
	expectedErr := errors.New("db error")
	uc := NewTemporaryPlayerDataUsecase(repo, &stubPlayerDataUsecase{registerFn: func(_ context.Context, _ *entity.User, _ *PlayerDataPayload, _ string) (*api_internal.PlayerDataResult, error) {
		return nil, expectedErr
	}}, 5*time.Minute)

	_, err := uc.Commit(context.Background(), CommitTemporaryPlayerDataInput{User: &entity.User{ID: 10}, UploadToken: "token-1"})

	require.Error(t, err)
	assert.ErrorIs(t, err, expectedErr)
	require.NotNil(t, repo.found)
	assert.Equal(t, "token-1", repo.found.Token)
}

func TestTemporaryPlayerDataUsecase_Commit_NotFound(t *testing.T) {
	repo := &stubTemporaryPlayerDataRepository{findErr: domainrepo.ErrTemporaryPlayerDataNotFound}
	uc := NewTemporaryPlayerDataUsecase(repo, &stubPlayerDataUsecase{}, 5*time.Minute)

	_, err := uc.Commit(context.Background(), CommitTemporaryPlayerDataInput{User: &entity.User{ID: 1}, UploadToken: "x"})

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrTemporaryPlayerDataNotFound)
}
