package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	api_internal "github.com/chunisupport/chunisupport-api/internal/usecase/playerdataresult"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubTemporaryPlayerDataRepository struct {
	createErr    error
	claimErr     error
	deleteErr    error
	found        *entity.TemporaryPlayerData
	claimed      bool
	claimedToken string
}

func (s *stubTemporaryPlayerDataRepository) Create(_ context.Context, _ domainrepo.Executor, data *entity.TemporaryPlayerData) error {
	if s.createErr != nil {
		return s.createErr
	}
	s.found = data
	return nil
}

func (s *stubTemporaryPlayerDataRepository) Claim(_ context.Context, _ domainrepo.Executor, token string) (*entity.TemporaryPlayerData, error) {
	if s.claimErr != nil {
		return nil, s.claimErr
	}
	if s.found == nil {
		return nil, domainrepo.ErrTemporaryPlayerDataNotFound
	}
	if s.claimed {
		return nil, domainrepo.ErrTemporaryPlayerDataInUse
	}
	s.claimed = true
	s.claimedToken = token
	copyData := *s.found
	copyData.Payload = append([]byte(nil), s.found.Payload...)
	return &copyData, nil
}

// Release は実リポジトリと同様にキャンセル済みcontextでは解放しません。
func (s *stubTemporaryPlayerDataRepository) Release(ctx context.Context, _ domainrepo.Executor, _ string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s.found == nil {
		return domainrepo.ErrTemporaryPlayerDataNotFound
	}
	s.claimed = false
	return nil
}

func (s *stubTemporaryPlayerDataRepository) Delete(_ context.Context, _ domainrepo.Executor, _ string) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	if s.found == nil {
		return domainrepo.ErrTemporaryPlayerDataNotFound
	}
	s.found = nil
	s.claimed = false
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

func (s *stubPlayerDataUsecase) GetLatestUpdate(_ context.Context, _ *entity.User) (json.RawMessage, error) {
	return nil, ErrPlayerLatestUpdateNotFound
}

func (s *stubPlayerDataUsecase) GetRecentUpdates(_ context.Context, _ *entity.User) ([]json.RawMessage, error) {
	return nil, nil
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

func TestTemporaryPlayerDataUsecase_Create_壊れたJSONでも一時保存できる(t *testing.T) {
	repo := &stubTemporaryPlayerDataRepository{}
	uc := NewTemporaryPlayerDataUsecase(nil, repo, &stubPlayerDataUsecase{}, 5*time.Minute)

	result, err := uc.Create(context.Background(), CreateTemporaryPlayerDataInput{
		IPAddress: "127.0.0.1",
		Payload:   []byte(`{"name":"TEST"`),
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, repo.found)
	assert.Equal(t, []byte(`{"name":"TEST"`), repo.found.Payload)
}

func TestTemporaryPlayerDataUsecase_Commit_登録後に消費される(t *testing.T) {
	repo := &stubTemporaryPlayerDataRepository{found: &entity.TemporaryPlayerData{Token: "token-1", Payload: []byte(`{"name":"TEST"}`), BodyHash: "hash"}}
	uc := NewTemporaryPlayerDataUsecase(nil, repo, &stubPlayerDataUsecase{}, 5*time.Minute)

	result, err := uc.Commit(context.Background(), CommitTemporaryPlayerDataInput{
		User:        &entity.User{ID: 10},
		UploadToken: "token-1",
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "token-1", repo.claimedToken)
	assert.Nil(t, repo.found)
}

func TestTemporaryPlayerDataUsecase_Commit_DB失敗時は同じトークンで再試行できる(t *testing.T) {
	// Given
	repo := &stubTemporaryPlayerDataRepository{found: &entity.TemporaryPlayerData{Token: "token-1", Payload: []byte(`{"name":"TEST"}`)}}
	expectedErr := errors.New("db error")
	registerCalls := 0
	uc := NewTemporaryPlayerDataUsecase(nil, repo, &stubPlayerDataUsecase{registerFn: func(_ context.Context, _ *entity.User, _ *PlayerDataPayload, _ string) (*api_internal.PlayerDataResult, error) {
		registerCalls++
		if registerCalls == 1 {
			return nil, expectedErr
		}
		return &api_internal.PlayerDataResult{PlayerID: 10}, nil
	}}, 5*time.Minute)
	input := CommitTemporaryPlayerDataInput{User: &entity.User{ID: 10}, UploadToken: "token-1"}

	// When
	_, firstErr := uc.Commit(context.Background(), input)
	result, retryErr := uc.Commit(context.Background(), input)

	// Then
	require.Error(t, firstErr)
	assert.ErrorIs(t, firstErr, expectedErr)
	require.NoError(t, retryErr)
	require.NotNil(t, result)
	assert.Equal(t, 2, registerCalls)
	assert.Nil(t, repo.found)
}

func TestTemporaryPlayerDataUsecase_Commit_クライアント切断時もトークンを解放する(t *testing.T) {
	// Given
	repo := &stubTemporaryPlayerDataRepository{found: &entity.TemporaryPlayerData{Token: "token-1", Payload: []byte(`{"name":"TEST"}`)}}
	ctx, cancel := context.WithCancel(context.Background())
	uc := NewTemporaryPlayerDataUsecase(nil, repo, &stubPlayerDataUsecase{registerFn: func(ctx context.Context, _ *entity.User, _ *PlayerDataPayload, _ string) (*api_internal.PlayerDataResult, error) {
		cancel()
		return nil, ctx.Err()
	}}, 5*time.Minute)

	// When
	_, err := uc.Commit(ctx, CommitTemporaryPlayerDataInput{User: &entity.User{ID: 10}, UploadToken: "token-1"})

	// Then
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
	assert.False(t, repo.claimed)
	assert.NotNil(t, repo.found)
}

func TestTemporaryPlayerDataUsecase_Commit_処理中のトークンは競合エラーになる(t *testing.T) {
	// Given
	repo := &stubTemporaryPlayerDataRepository{claimErr: domainrepo.ErrTemporaryPlayerDataInUse}
	uc := NewTemporaryPlayerDataUsecase(nil, repo, &stubPlayerDataUsecase{}, 5*time.Minute)

	// When
	_, err := uc.Commit(context.Background(), CommitTemporaryPlayerDataInput{User: &entity.User{ID: 1}, UploadToken: "token-1"})

	// Then
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrTemporaryPlayerDataInProgress)
}

func TestTemporaryPlayerDataUsecase_Commit_NotFound(t *testing.T) {
	repo := &stubTemporaryPlayerDataRepository{claimErr: domainrepo.ErrTemporaryPlayerDataNotFound}
	uc := NewTemporaryPlayerDataUsecase(nil, repo, &stubPlayerDataUsecase{}, 5*time.Minute)

	_, err := uc.Commit(context.Background(), CommitTemporaryPlayerDataInput{User: &entity.User{ID: 1}, UploadToken: "x"})

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrTemporaryPlayerDataNotFound)
}

func TestTemporaryPlayerDataUsecase_Commit_壊れたJSONはBadRequest相当エラーになる(t *testing.T) {
	repo := &stubTemporaryPlayerDataRepository{found: &entity.TemporaryPlayerData{Token: "token-1", Payload: []byte(`{"name":"TEST"`)}}
	uc := NewTemporaryPlayerDataUsecase(nil, repo, &stubPlayerDataUsecase{}, 5*time.Minute)

	_, err := uc.Commit(context.Background(), CommitTemporaryPlayerDataInput{User: &entity.User{ID: 1}, UploadToken: "token-1"})

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrTempDataPayloadInvalidJSON)
}
