package entity

import (
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo/goalgroupname"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGoalGroup(t *testing.T) {
	group, err := NewGoalGroup(42, "レーティング目標")

	require.NoError(t, err)
	assert.Equal(t, 42, group.UserID)
	assert.Equal(t, "レーティング目標", group.Name.String())
}

func TestNewGoalGroup_不正な名前(t *testing.T) {
	group, err := NewGoalGroup(42, "")

	assert.Nil(t, group)
	assert.ErrorIs(t, err, goalgroupname.ErrInvalidGoalGroupName)
}

func TestGoalGroup_Rename(t *testing.T) {
	group, err := NewGoalGroup(42, "変更前")
	require.NoError(t, err)

	require.NoError(t, group.Rename("変更後"))

	assert.Equal(t, "変更後", group.Name.String())
}

func TestGoalGroup_Rename_不正な名前では変更しない(t *testing.T) {
	group, err := NewGoalGroup(42, "変更前")
	require.NoError(t, err)

	err = group.Rename("")

	assert.ErrorIs(t, err, goalgroupname.ErrInvalidGoalGroupName)
	assert.Equal(t, "変更前", group.Name.String())
}
