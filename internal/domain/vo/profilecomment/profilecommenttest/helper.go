package profilecommenttest

import (
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo/profilecomment"
	"github.com/stretchr/testify/require"
)

// New は検証済みのコメントをテスト用に生成します。
func New(tb testing.TB, value string) profilecomment.ProfileComment {
	tb.Helper()
	comment, err := profilecomment.NewProfileComment(value)
	require.NoError(tb, err)
	return comment
}
