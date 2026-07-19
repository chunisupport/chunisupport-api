package entity

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGoalArrangement_MoveAppendsToDestinationAndRenumbersSource(t *testing.T) {
	// Given
	groupA := uint32(10)
	groupB := uint32(20)
	arrangement, err := NewGoalArrangement(1, []*Goal{
		{ID: 1, UserID: 1, GroupID: &groupA, SortOrder: 1},
		{ID: 2, UserID: 1, GroupID: &groupA, SortOrder: 2},
		{ID: 3, UserID: 1, GroupID: &groupA, SortOrder: 3},
		{ID: 4, UserID: 1, GroupID: &groupB, SortOrder: 1},
	})
	require.NoError(t, err)

	// When
	err = arrangement.Move(2, &groupB)

	// Then
	require.NoError(t, err)
	assertGoalPositions(t, arrangement.Goals(), map[uint32]struct {
		groupID   *uint32
		sortOrder uint16
	}{
		1: {groupID: &groupA, sortOrder: 1},
		3: {groupID: &groupA, sortOrder: 2},
		4: {groupID: &groupB, sortOrder: 1},
		2: {groupID: &groupB, sortOrder: 2},
	})
}

func TestGoalArrangement_RemoveGroupMovesGoalsToUnclassifiedEnd(t *testing.T) {
	// Given
	groupID := uint32(10)
	arrangement, err := NewGoalArrangement(1, []*Goal{
		{ID: 2, UserID: 1, GroupID: &groupID, SortOrder: 1},
		{ID: 3, UserID: 1, GroupID: &groupID, SortOrder: 2},
		{ID: 1, UserID: 1, SortOrder: 1},
	})
	require.NoError(t, err)

	// When
	arrangement.RemoveGroup(groupID)

	// Then
	goals := arrangement.GoalsInGroup(nil)
	require.Len(t, goals, 3)
	assert.Equal(t, []uint32{1, 2, 3}, []uint32{goals[0].ID, goals[1].ID, goals[2].ID})
	assert.Equal(t, []uint16{1, 2, 3}, []uint16{goals[0].SortOrder, goals[1].SortOrder, goals[2].SortOrder})
}

func assertGoalPositions(t *testing.T, goals []*Goal, expected map[uint32]struct {
	groupID   *uint32
	sortOrder uint16
}) {
	t.Helper()
	for _, goal := range goals {
		want := expected[goal.ID]
		assert.True(t, sameOptionalID(want.groupID, goal.GroupID), "goal %d group", goal.ID)
		assert.Equal(t, want.sortOrder, goal.SortOrder, "goal %d sort_order", goal.ID)
	}
}
