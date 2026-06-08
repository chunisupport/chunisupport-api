package dto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpTargetDifficultyPtr(t *testing.T) {
	t.Run("譜面なしはnil", func(t *testing.T) {
		assert.Nil(t, OpTargetDifficultyPtr(0))
	})

	t.Run("既知の難易度IDは名称を返す", func(t *testing.T) {
		got := OpTargetDifficultyPtr(4)
		require.NotNil(t, got)
		assert.Equal(t, "MASTER", *got)
	})

	t.Run("未知の難易度IDはnil", func(t *testing.T) {
		assert.Nil(t, OpTargetDifficultyPtr(99))
	})
}
