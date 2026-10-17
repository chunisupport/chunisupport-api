package db

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/config"
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMySQLDSN(t *testing.T) {
	tests := []struct {
		name string
		cfg  config.DbConfig
		want string
	}{
		{
			name: "パラメータ付きクエリの往復を避けるためinterpolateParamsを有効にする",
			cfg: config.DbConfig{
				DbUser: "app",
				DbPass: "secret",
				DbHost: "db.internal",
				DbPort: 3306,
				DbName: "chunisupport",
			},
			want: "app:secret@tcp(db.internal:3306)/chunisupport?parseTime=true&loc=UTC&time_zone=%27%2B00%3A00%27&clientFoundRows=true&interpolateParams=true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			cfg := tt.cfg

			// When
			dsn := mysqlDSN(cfg)

			// Then
			assert.Equal(t, tt.want, dsn)

			parsed, err := mysql.ParseDSN(dsn)
			require.NoError(t, err)
			assert.True(t, parsed.InterpolateParams)
			assert.True(t, parsed.ParseTime)
			assert.True(t, parsed.ClientFoundRows)
			assert.Equal(t, time.UTC, parsed.Loc)
			assert.Equal(t, "'+00:00'", parsed.Params["time_zone"])
			// charset / collation を指定しないと、ドライバ既定の utf8mb4_general_ci になる。
			// 埋め込みを拒否する big5 / cp932 / gb2312 / gbk / sjis / gb18030 へ上書きしない。
			assert.Empty(t, parsed.Collation)
			assert.NotContains(t, dsn, "charset=")
			assert.NotContains(t, dsn, "collation=")
		})
	}
}

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
