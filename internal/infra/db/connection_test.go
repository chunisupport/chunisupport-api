package db

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/config"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConnectWithRetry(t *testing.T) {
	t.Run("キャンセル済みのコンテキストでは接続を試行しない", func(t *testing.T) {
		// Given
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		called := false

		// When
		database, err := connectWithRetry(ctx, config.DbConfig{
			StartupMaxWaitSec:  1,
			StartupIntervalSec: 1,
		}, func(context.Context, config.DbConfig) (*sqlx.DB, error) {
			called = true
			return nil, nil
		})

		// Then
		require.Error(t, err)
		assert.True(t, errors.Is(err, context.Canceled))
		assert.Nil(t, database)
		assert.False(t, called)
	})

	t.Run("最大待機秒数が0の場合は再試行しない", func(t *testing.T) {
		// Given
		attempts := 0

		// When
		database, err := connectWithRetry(context.Background(), config.DbConfig{
			StartupMaxWaitSec:  0,
			StartupIntervalSec: 1,
		}, func(context.Context, config.DbConfig) (*sqlx.DB, error) {
			attempts++
			return nil, fmt.Errorf("connection failed")
		})

		// Then
		require.Error(t, err)
		assert.Nil(t, database)
		assert.Equal(t, 1, attempts)
	})

	t.Run("接続試行には残り待機時間の期限を付ける", func(t *testing.T) {
		// Given
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		deadlineWasSet := false

		// When
		database, err := connectWithRetry(ctx, config.DbConfig{
			StartupMaxWaitSec:  10,
			StartupIntervalSec: 1,
		}, func(ctx context.Context, _ config.DbConfig) (*sqlx.DB, error) {
			deadline, ok := ctx.Deadline()
			deadlineWasSet = ok && time.Until(deadline) > 0
			cancel()
			return nil, fmt.Errorf("connection failed")
		})

		// Then
		require.Error(t, err)
		assert.True(t, errors.Is(err, context.Canceled))
		assert.Nil(t, database)
		assert.True(t, deadlineWasSet)
	})
}
