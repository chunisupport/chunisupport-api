package app

import (
	"net/http"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// HTTPタイムアウトがEchoの既定値変更に左右されないことを保証します。
func TestApplyHTTPServerTimeouts_4種のタイムアウトを明示する(t *testing.T) {
	// Given: ゼロ値のHTTPサーバー（Echoの既定値変更前と同等の状態）
	server := &http.Server{}

	// When: アプリケーション側のタイムアウト方針を適用する
	applyHTTPServerTimeouts(server)

	// Then: 4種類のタイムアウトがすべて明示されている
	assert.Equal(t, info.HTTPReadHeaderTimeout, server.ReadHeaderTimeout)
	assert.Equal(t, info.HTTPReadTimeout, server.ReadTimeout)
	assert.Equal(t, info.HTTPWriteTimeout, server.WriteTimeout)
	assert.Equal(t, info.HTTPIdleTimeout, server.IdleTimeout)
	assert.Positive(t, int64(server.ReadHeaderTimeout))
	assert.Positive(t, int64(server.ReadTimeout))
	assert.Positive(t, int64(server.WriteTimeout))
	assert.Positive(t, int64(server.IdleTimeout))
}

func TestNewStartConfig_BeforeServeFuncでタイムアウトを保証する(t *testing.T) {
	// Given: 起動設定の元になるアドレスとgracefulタイムアウト
	const address = "127.0.0.1:3000"
	const gracefulTimeout = 20 * time.Second

	// When: Echoの起動設定を組み立てる
	startConfig := newStartConfig(address, gracefulTimeout)

	// Then: アドレスとgracefulタイムアウトが維持される
	assert.Equal(t, address, startConfig.Address)
	assert.Equal(t, gracefulTimeout, startConfig.GracefulTimeout)

	// Then: BeforeServeFuncが設定され、Echo既定値の上から4種を明示する
	require.NotNil(t, startConfig.BeforeServeFunc)
	server := &http.Server{}
	require.NoError(t, startConfig.BeforeServeFunc(server))
	assert.Equal(t, info.HTTPReadHeaderTimeout, server.ReadHeaderTimeout)
	assert.Equal(t, info.HTTPReadTimeout, server.ReadTimeout)
	assert.Equal(t, info.HTTPWriteTimeout, server.WriteTimeout)
	assert.Equal(t, info.HTTPIdleTimeout, server.IdleTimeout)
}
