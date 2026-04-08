package repository

import (
	"context"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemporaryPlayerDataRepository_CreateAndFind(t *testing.T) {
	repo := NewTemporaryPlayerDataRepository(3, 1024)
	entry := &entity.TemporaryPlayerData{
		Token:     "token",
		IPAddress: "127.0.0.1",
		Payload:   []byte(`{"name":"TEST"}`),
		ExpiresAt: time.Now().UTC().Add(time.Minute),
	}

	err := repo.Create(context.Background(), nil, entry)
	require.NoError(t, err)

	got, err := repo.FindByToken(context.Background(), nil, "token")
	require.NoError(t, err)
	assert.Equal(t, "token", got.Token)
}

func TestTemporaryPlayerDataRepository_IP上限(t *testing.T) {
	repo := NewTemporaryPlayerDataRepository(1, 1024)
	err := repo.Create(context.Background(), nil, &entity.TemporaryPlayerData{Token: "t1", IPAddress: "127.0.0.1", Payload: []byte("{}"), ExpiresAt: time.Now().UTC().Add(time.Minute)})
	require.NoError(t, err)

	err = repo.Create(context.Background(), nil, &entity.TemporaryPlayerData{Token: "t2", IPAddress: "127.0.0.1", Payload: []byte("{}"), ExpiresAt: time.Now().UTC().Add(time.Minute)})
	require.Error(t, err)
	assert.ErrorIs(t, err, domainrepo.ErrTemporaryPlayerDataPerIPLimitExceeded)
}

func TestTemporaryPlayerDataRepository_総量上限(t *testing.T) {
	repo := NewTemporaryPlayerDataRepository(3, 4)
	err := repo.Create(context.Background(), nil, &entity.TemporaryPlayerData{Token: "t1", IPAddress: "127.0.0.1", Payload: []byte("1234"), ExpiresAt: time.Now().UTC().Add(time.Minute)})
	require.NoError(t, err)

	err = repo.Create(context.Background(), nil, &entity.TemporaryPlayerData{Token: "t2", IPAddress: "127.0.0.2", Payload: []byte("1"), ExpiresAt: time.Now().UTC().Add(time.Minute)})
	require.Error(t, err)
	assert.ErrorIs(t, err, domainrepo.ErrTemporaryPlayerDataTotalSizeLimitExceeded)
}

func TestTemporaryPlayerDataRepository_期限切れは参照不可(t *testing.T) {
	repo := NewTemporaryPlayerDataRepository(3, 1024)
	err := repo.Create(context.Background(), nil, &entity.TemporaryPlayerData{Token: "t1", IPAddress: "127.0.0.1", Payload: []byte("{}"), ExpiresAt: time.Now().UTC().Add(-time.Second)})
	require.NoError(t, err)

	_, err = repo.FindByToken(context.Background(), nil, "t1")
	require.Error(t, err)
	assert.ErrorIs(t, err, domainrepo.ErrTemporaryPlayerDataNotFound)
}

func TestTemporaryPlayerDataRepository_ConsumeByTokenは一度だけ取得できる(t *testing.T) {
	repo := NewTemporaryPlayerDataRepository(3, 1024)
	err := repo.Create(context.Background(), nil, &entity.TemporaryPlayerData{
		Token:     "t1",
		IPAddress: "127.0.0.1",
		Payload:   []byte(`{"name":"TEST"}`),
		ExpiresAt: time.Now().UTC().Add(time.Minute),
	})
	require.NoError(t, err)

	consumed, err := repo.ConsumeByToken(context.Background(), nil, "t1")
	require.NoError(t, err)
	assert.Equal(t, "t1", consumed.Token)

	_, err = repo.ConsumeByToken(context.Background(), nil, "t1")
	require.Error(t, err)
	assert.ErrorIs(t, err, domainrepo.ErrTemporaryPlayerDataNotFound)
}
