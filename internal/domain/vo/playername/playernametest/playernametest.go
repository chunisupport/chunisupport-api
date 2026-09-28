// Package playernametest はテストでプレイヤー名を簡潔に生成するためのヘルパーを提供します。
// 本番パッケージに panic する Must 系コンストラクタを置かず、生成失敗をテストの失敗として扱うために分離しています。
package playernametest

import (
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo/playername"
	"github.com/stretchr/testify/require"
)

// New は検証済みのプレイヤー名を生成します。生成に失敗した場合はテストを失敗させます。
func New(tb testing.TB, value string) playername.PlayerName {
	tb.Helper()
	v, err := playername.NewPlayerName(value)
	require.NoError(tb, err)
	return v
}
