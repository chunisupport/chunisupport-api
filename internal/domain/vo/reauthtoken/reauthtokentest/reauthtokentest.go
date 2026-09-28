// Package reauthtokentest はテストで再認証トークンを簡潔に生成するためのヘルパーを提供します。
// 本番パッケージに panic する Must 系コンストラクタを置かず、生成失敗をテストの失敗として扱うために分離しています。
package reauthtokentest

import (
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo/reauthtoken"
	"github.com/stretchr/testify/require"
)

// New は検証済みの再認証トークンを生成します。生成に失敗した場合はテストを失敗させます。
func New(tb testing.TB, value string) reauthtoken.ReauthToken {
	tb.Helper()
	v, err := reauthtoken.New(value)
	require.NoError(tb, err)
	return v
}
