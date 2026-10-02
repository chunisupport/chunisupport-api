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

type cancelOnErrCheckContext struct {
	context.Context
	cancelAt int
	checks   int
	done     chan struct{}
}

func newCancelOnErrCheckContext(cancelAt int) *cancelOnErrCheckContext {
	return &cancelOnErrCheckContext{
		Context:  context.Background(),
		cancelAt: cancelAt,
		done:     make(chan struct{}),
	}
}

func (c *cancelOnErrCheckContext) Done() <-chan struct{} {
	return c.done
}

func (c *cancelOnErrCheckContext) Err() error {
	c.checks++
	if c.checks == c.cancelAt {
		close(c.done)
	}
	if c.checks >= c.cancelAt {
		return context.Canceled
	}
	return nil
}

func TestTemporaryPlayerDataRepository_CreateAndClaim(t *testing.T) {
	repo := NewTemporaryPlayerDataRepository(3, 1024)
	entry := &entity.TemporaryPlayerData{
		Token:     "token",
		IPAddress: "127.0.0.1",
		Payload:   []byte(`{"name":"TEST"}`),
		ExpiresAt: time.Now().UTC().Add(time.Minute),
	}

	err := repo.Create(context.Background(), nil, entry)
	require.NoError(t, err)

	got, err := repo.Claim(context.Background(), nil, "token")
	require.NoError(t, err)
	assert.Equal(t, "token", got.Token)
}

func TestTemporaryPlayerDataRepository_Create後に元Payloadを変更しても保持データは変わらない(t *testing.T) {
	repo := NewTemporaryPlayerDataRepository(3, 1024)
	entry := &entity.TemporaryPlayerData{
		Token:     "token",
		IPAddress: "127.0.0.1",
		Payload:   []byte(`{"name":"TEST"}`),
		ExpiresAt: time.Now().UTC().Add(time.Minute),
	}

	err := repo.Create(context.Background(), nil, entry)
	require.NoError(t, err)

	entry.Payload[0] = 'X'

	got, err := repo.Claim(context.Background(), nil, "token")
	require.NoError(t, err)
	assert.Equal(t, []byte(`{"name":"TEST"}`), got.Payload)
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

	_, err = repo.Claim(context.Background(), nil, "t1")
	require.Error(t, err)
	assert.ErrorIs(t, err, domainrepo.ErrTemporaryPlayerDataNotFound)
}

func TestTemporaryPlayerDataRepository_Claim中は再度Claimできない(t *testing.T) {
	// Given
	repo := NewTemporaryPlayerDataRepository(3, 1024)
	err := repo.Create(context.Background(), nil, &entity.TemporaryPlayerData{
		Token:     "t1",
		IPAddress: "127.0.0.1",
		Payload:   []byte(`{"name":"TEST"}`),
		ExpiresAt: time.Now().UTC().Add(time.Minute),
	})
	require.NoError(t, err)

	// When
	claimed, firstErr := repo.Claim(context.Background(), nil, "t1")
	_, secondErr := repo.Claim(context.Background(), nil, "t1")

	// Then
	require.NoError(t, firstErr)
	assert.Equal(t, "t1", claimed.Token)
	assert.ErrorIs(t, secondErr, domainrepo.ErrTemporaryPlayerDataInUse)
}

func TestTemporaryPlayerDataRepository_Releaseすると再度Claimできる(t *testing.T) {
	// Given
	repo := NewTemporaryPlayerDataRepository(3, 1024)
	err := repo.Create(context.Background(), nil, &entity.TemporaryPlayerData{
		Token:     "t1",
		IPAddress: "127.0.0.1",
		Payload:   []byte(`{"name":"TEST"}`),
		ExpiresAt: time.Now().UTC().Add(time.Minute),
	})
	require.NoError(t, err)
	_, err = repo.Claim(context.Background(), nil, "t1")
	require.NoError(t, err)

	// When
	releaseErr := repo.Release(context.Background(), nil, "t1")
	reclaimed, reclaimErr := repo.Claim(context.Background(), nil, "t1")

	// Then
	require.NoError(t, releaseErr)
	require.NoError(t, reclaimErr)
	assert.Equal(t, []byte(`{"name":"TEST"}`), reclaimed.Payload)
}

func TestTemporaryPlayerDataRepository_Claim後にDeleteすると取得できない(t *testing.T) {
	// Given
	repo := NewTemporaryPlayerDataRepository(3, 1024)
	err := repo.Create(context.Background(), nil, &entity.TemporaryPlayerData{
		Token:     "t1",
		IPAddress: "127.0.0.1",
		Payload:   []byte(`{"name":"TEST"}`),
		ExpiresAt: time.Now().UTC().Add(time.Minute),
	})
	require.NoError(t, err)
	_, err = repo.Claim(context.Background(), nil, "t1")
	require.NoError(t, err)

	// When
	deleteErr := repo.Delete(context.Background(), nil, "t1")
	_, claimErr := repo.Claim(context.Background(), nil, "t1")

	// Then
	require.NoError(t, deleteErr)
	assert.ErrorIs(t, claimErr, domainrepo.ErrTemporaryPlayerDataNotFound)
}

func TestTemporaryPlayerDataRepository_Claim中に期限切れになると予約も破棄される(t *testing.T) {
	// Given
	repo := NewTemporaryPlayerDataRepository(3, 1024)
	err := repo.Create(context.Background(), nil, &entity.TemporaryPlayerData{
		Token:     "t1",
		IPAddress: "127.0.0.1",
		Payload:   []byte(`{"name":"TEST"}`),
		ExpiresAt: time.Now().UTC().Add(50 * time.Millisecond),
	})
	require.NoError(t, err)
	_, err = repo.Claim(context.Background(), nil, "t1")
	require.NoError(t, err)

	// When
	time.Sleep(100 * time.Millisecond)
	_, claimErr := repo.Claim(context.Background(), nil, "t1")
	releaseErr := repo.Release(context.Background(), nil, "t1")

	// Then
	assert.ErrorIs(t, claimErr, domainrepo.ErrTemporaryPlayerDataNotFound)
	assert.ErrorIs(t, releaseErr, domainrepo.ErrTemporaryPlayerDataNotFound)
	assert.Empty(t, repo.(*temporaryPlayerDataRepository).claimedTokens)
}

func TestTemporaryPlayerDataRepository_存在しないトークンはReleaseできない(t *testing.T) {
	// Given
	repo := NewTemporaryPlayerDataRepository(3, 1024)

	// When
	err := repo.Release(context.Background(), nil, "missing")

	// Then
	assert.ErrorIs(t, err, domainrepo.ErrTemporaryPlayerDataNotFound)
}

func TestTemporaryPlayerDataRepository_キャンセル済みContextでは処理しない(t *testing.T) {
	tests := []struct {
		name string
		call func(context.Context, domainrepo.TemporaryPlayerDataRepository) error
	}{
		{
			name: "Create",
			call: func(ctx context.Context, repo domainrepo.TemporaryPlayerDataRepository) error {
				return repo.Create(ctx, nil, &entity.TemporaryPlayerData{
					Token:     "new-token",
					IPAddress: "127.0.0.2",
					Payload:   []byte("{}"),
					ExpiresAt: time.Now().UTC().Add(time.Minute),
				})
			},
		},
		{
			name: "Claim",
			call: func(ctx context.Context, repo domainrepo.TemporaryPlayerDataRepository) error {
				_, err := repo.Claim(ctx, nil, "stored-token")
				return err
			},
		},
		{
			name: "Release",
			call: func(ctx context.Context, repo domainrepo.TemporaryPlayerDataRepository) error {
				return repo.Release(ctx, nil, "stored-token")
			},
		},
		{
			name: "Delete",
			call: func(ctx context.Context, repo domainrepo.TemporaryPlayerDataRepository) error {
				return repo.Delete(ctx, nil, "stored-token")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			repo := NewTemporaryPlayerDataRepository(3, 1024)
			err := repo.Create(context.Background(), nil, &entity.TemporaryPlayerData{
				Token:     "stored-token",
				IPAddress: "127.0.0.1",
				Payload:   []byte("{}"),
				ExpiresAt: time.Now().UTC().Add(time.Minute),
			})
			require.NoError(t, err)

			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			// When
			err = tt.call(ctx, repo)

			// Then
			require.Error(t, err)
			assert.ErrorIs(t, err, context.Canceled)

			stored, claimErr := repo.Claim(context.Background(), nil, "stored-token")
			require.NoError(t, claimErr)
			assert.Equal(t, "stored-token", stored.Token)

			_, claimErr = repo.Claim(context.Background(), nil, "new-token")
			assert.ErrorIs(t, claimErr, domainrepo.ErrTemporaryPlayerDataNotFound)
		})
	}
}

func TestTemporaryPlayerDataRepository_lockWithContext_Mutex取得までにキャンセルされた場合は解放する(t *testing.T) {
	// Given
	repo := &temporaryPlayerDataRepository{}
	ctx := newCancelOnErrCheckContext(2)

	// When
	err := repo.lockWithContext(ctx)

	// Then
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)

	unlocked := repo.mu.TryLock()
	require.True(t, unlocked)
	repo.mu.Unlock()
}

func TestTemporaryPlayerDataRepository_Payloadコピー中にキャンセルされた場合は状態を変更しない(t *testing.T) {
	tests := []struct {
		name string
		call func(context.Context, domainrepo.TemporaryPlayerDataRepository) error
	}{
		{
			name: "Create",
			call: func(ctx context.Context, repo domainrepo.TemporaryPlayerDataRepository) error {
				return repo.Create(ctx, nil, &entity.TemporaryPlayerData{
					Token:     "new-token",
					IPAddress: "127.0.0.2",
					Payload:   []byte("{}"),
					ExpiresAt: time.Now().UTC().Add(time.Minute),
				})
			},
		},
		{
			name: "Claim",
			call: func(ctx context.Context, repo domainrepo.TemporaryPlayerDataRepository) error {
				_, err := repo.Claim(ctx, nil, "stored-token")
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			repo := NewTemporaryPlayerDataRepository(3, 1024)
			err := repo.Create(context.Background(), nil, &entity.TemporaryPlayerData{
				Token:     "stored-token",
				IPAddress: "127.0.0.1",
				Payload:   []byte("{}"),
				ExpiresAt: time.Now().UTC().Add(time.Minute),
			})
			require.NoError(t, err)

			ctx := newCancelOnErrCheckContext(4)

			// When
			err = tt.call(ctx, repo)

			// Then
			require.Error(t, err)
			assert.ErrorIs(t, err, context.Canceled)

			stored, claimErr := repo.Claim(context.Background(), nil, "stored-token")
			require.NoError(t, claimErr)
			assert.Equal(t, "stored-token", stored.Token)

			_, claimErr = repo.Claim(context.Background(), nil, "new-token")
			assert.ErrorIs(t, claimErr, domainrepo.ErrTemporaryPlayerDataNotFound)
		})
	}
}
