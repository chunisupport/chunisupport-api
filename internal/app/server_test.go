package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServerAddress(t *testing.T) {
	// Given: アプリケーションの待受ポート
	const port = 3000

	// When
	actual := serverAddress(port)

	// Then: ループバックIPv4のみで待ち受ける
	assert.Equal(t, "127.0.0.1:3000", actual)
}
