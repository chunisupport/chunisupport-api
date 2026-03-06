package dto

import (
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo/levelstar"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToLevelStarIntPtr(t *testing.T) {
	t.Run("nilの場合はnilを返す", func(t *testing.T) {
		got := ToLevelStarIntPtr(nil)
		assert.Nil(t, got)
	})

	t.Run("値がある場合はintへ変換する", func(t *testing.T) {
		ls, err := levelstar.NewLevelStar(4)
		require.NoError(t, err)

		got := ToLevelStarIntPtr(&ls)
		require.NotNil(t, got)
		assert.Equal(t, 4, *got)
	})
}
