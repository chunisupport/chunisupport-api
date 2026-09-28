// Package usernametest はテストでユーザー名を簡潔に生成するためのヘルパーを提供します。
// 本番パッケージに panic する Must 系コンストラクタを置かず、生成失敗をテストの失敗として扱うために分離しています。
package usernametest

import (
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo/username"
	"github.com/stretchr/testify/require"
)

// New は検証済みのユーザー名を生成します。生成に失敗した場合はテストを失敗させます。
func New(tb testing.TB, value string) username.UserName {
	tb.Helper()
	v, err := username.NewUserName(value)
	require.NoError(tb, err)
	return v
}
