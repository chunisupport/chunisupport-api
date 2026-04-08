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
	createErr     error
	findErr       error
	consumeErr    error
	deleteErr     error
	found         *entity.TemporaryPlayerData
	consumedToken string
}

func (s *stubTemporaryPlayerDataRepository) Create(_ context.Context, _ domainrepo.Executor, data *entity.TemporaryPlayerData) error {
	if s.createErr != nil {
		return s.createErr
	}
	s.found = data
	return nil
}

func (s *stubTemporaryPlayerDataRepository) FindByToken(_ context.Context, _ domainrepo.Executor, _ string) (*entity.TemporaryPlayerData, error) {
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

func (s *stubTemporaryPlayerDataRepository) ConsumeByToken(ctx context.Context, _ domainrepo.Executor, token string) (*entity.TemporaryPlayerData, error) {
	if s.consumeErr != nil {
		return nil, s.consumeErr
	}
	s.consumedToken = token
	entry, err := s.FindByToken(ctx, nil, token)
	if err != nil {
		return nil, err
	}
	s.found = nil
	return entry, nil
}

func (s *stubTemporaryPlayerDataRepository) Delete(_ context.Context, _ domainrepo.Executor, _ string) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	if s.found == nil {
		return domainrepo.ErrTemporaryPlayerDataNotFound
	}
	s.found = nil
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
	uc := NewTemporaryPlayerDataUsecase(nil, repo, &stubPlayerDataUsecase{}, 5*time.Minute)

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
	uc := NewTemporaryPlayerDataUsecase(nil, repo, &stubPlayerDataUsecase{}, 5*time.Minute)

	_, err := uc.Create(context.Background(), CreateTemporaryPlayerDataInput{IPAddress: "127.0.0.1", Payload: []byte("{}")})

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrTempDataPerIPLimitExceeded)
}

func TestTemporaryPlayerDataUsecase_Create_不正なJSONはバリデーションエラー(t *testing.T) {
	repo := &stubTemporaryPlayerDataRepository{}
	uc := NewTemporaryPlayerDataUsecase(nil, repo, &stubPlayerDataUsecase{}, 5*time.Minute)

	_, err := uc.Create(context.Background(), CreateTemporaryPlayerDataInput{
		IPAddress: "127.0.0.1",
		Payload:   []byte(`{"name":"TEST"`),
	})

	require.Error(t, err)
	var validationErr *PlayerDataValidationError
	require.ErrorAs(t, err, &validationErr)
	assert.Equal(t, "payload", validationErr.Field)
}

func TestTemporaryPlayerDataUsecase_Commit_成功時に削除される(t *testing.T) {
	repo := &stubTemporaryPlayerDataRepository{found: &entity.TemporaryPlayerData{Token: "token-1", Payload: []byte(`{"name":"TEST"}`), BodyHash: "hash"}}
	uc := NewTemporaryPlayerDataUsecase(nil, repo, &stubPlayerDataUsecase{}, 5*time.Minute)

	result, err := uc.Commit(context.Background(), CommitTemporaryPlayerDataInput{
		User:        &entity.User{ID: 10},
		UploadToken: "token-1",
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "token-1", repo.consumedToken)
	assert.Nil(t, repo.found)
}

func TestTemporaryPlayerDataUsecase_Commit_DB失敗時は再試行不可になる(t *testing.T) {
	repo := &stubTemporaryPlayerDataRepository{found: &entity.TemporaryPlayerData{Token: "token-1", Payload: []byte(`{"name":"TEST"}`)}}
	expectedErr := errors.New("db error")
	uc := NewTemporaryPlayerDataUsecase(nil, repo, &stubPlayerDataUsecase{registerFn: func(_ context.Context, _ *entity.User, _ *PlayerDataPayload, _ string) (*api_internal.PlayerDataResult, error) {
		return nil, expectedErr
	}}, 5*time.Minute)

	_, err := uc.Commit(context.Background(), CommitTemporaryPlayerDataInput{User: &entity.User{ID: 10}, UploadToken: "token-1"})

	require.Error(t, err)
	assert.ErrorIs(t, err, expectedErr)
	assert.Equal(t, "token-1", repo.consumedToken)
	assert.Nil(t, repo.found)
}

func TestTemporaryPlayerDataUsecase_Commit_NotFound(t *testing.T) {
	repo := &stubTemporaryPlayerDataRepository{consumeErr: domainrepo.ErrTemporaryPlayerDataNotFound}
	uc := NewTemporaryPlayerDataUsecase(nil, repo, &stubPlayerDataUsecase{}, 5*time.Minute)

	_, err := uc.Commit(context.Background(), CommitTemporaryPlayerDataInput{User: &entity.User{ID: 1}, UploadToken: "x"})

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrTemporaryPlayerDataNotFound)
}
