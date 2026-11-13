package entity

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGoalOrder_ReorderAssignsDenseSortOrders(t *testing.T) {
	// Given
	groupID := uint32(5)
	order, err := NewGoalOrder(1, &groupID, []*Goal{
		{ID: 10, UserID: 1, GroupID: &groupID},
		{ID: 20, UserID: 1, GroupID: &groupID},
		{ID: 30, UserID: 1, GroupID: &groupID},
	})
	require.NoError(t, err)

	// When
	err = order.Reorder([]uint32{30, 10, 20})

	// Then
	require.NoError(t, err)
	goals := order.Goals()
	assert.Equal(t, []uint32{30, 10, 20}, []uint32{goals[0].ID, goals[1].ID, goals[2].ID})
	assert.Equal(t, []uint16{1, 2, 3}, []uint16{goals[0].SortOrder, goals[1].SortOrder, goals[2].SortOrder})
}

func TestGoalOrder_ReorderRejectsDifferentGoalSet(t *testing.T) {
	tests := []struct {
		name string
		// When: 並び替え入力
		orderedIDs []uint32
	}{
		{
			name:       "件数不足のID列は拒否する",
			orderedIDs: []uint32{10},
		},
		{
			name:       "同件数でもID重複のID列は拒否する",
			orderedIDs: []uint32{10, 10},
		},
		{
			name:       "未知のIDを含むID列は拒否する",
			orderedIDs: []uint32{10, 99},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			order, err := NewGoalOrder(1, nil, []*Goal{{ID: 10, UserID: 1}, {ID: 20, UserID: 1}})
			require.NoError(t, err)

			// When
			err = order.Reorder(tt.orderedIDs)

			// Then: 拒否時は並び順とSortOrderが変化しないことを固定期待値で保証する
			assert.ErrorIs(t, err, ErrInvalidGoalOrder)
			goals := order.Goals()
			require.Len(t, goals, 2)
			assert.Equal(t, []uint32{10, 20}, []uint32{goals[0].ID, goals[1].ID})
			assert.Equal(t, []uint16{1, 2}, []uint16{goals[0].SortOrder, goals[1].SortOrder})
		})
	}
}

func TestNewGoalOrder_RejectsAnotherUsersGoal(t *testing.T) {
	// When
	order, err := NewGoalOrder(1, nil, []*Goal{{ID: 10, UserID: 1}, {ID: 20, UserID: 2}})

	// Then
	assert.ErrorIs(t, err, ErrInvalidGoalOrder)
	assert.Nil(t, order)
}

func TestNewGoalOrder_RejectsDuplicateGoalID(t *testing.T) {
	// When
	order, err := NewGoalOrder(1, nil, []*Goal{{ID: 10, UserID: 1}, {ID: 10, UserID: 1}})

	// Then
	assert.ErrorIs(t, err, ErrInvalidGoalOrder)
	assert.Nil(t, order)
}

func TestGoalOrder_RemoveRenumbersRemainingGoals(t *testing.T) {
	// Given
	order, err := NewGoalOrder(1, nil, []*Goal{{ID: 10, UserID: 1}, {ID: 20, UserID: 1}, {ID: 30, UserID: 1}})
	require.NoError(t, err)

	// When
	err = order.Remove(20)

	// Then
	require.NoError(t, err)
	goals := order.Goals()
	assert.Equal(t, []uint32{10, 30}, []uint32{goals[0].ID, goals[1].ID})
	assert.Equal(t, []uint16{1, 2}, []uint16{goals[0].SortOrder, goals[1].SortOrder})
}

func TestNewGoalOrder_RejectsGoalFromAnotherGroup(t *testing.T) {
	// Given
	groupID := uint32(5)
	otherGroupID := uint32(6)

	// When
	order, err := NewGoalOrder(1, &groupID, []*Goal{{ID: 10, UserID: 1, GroupID: &otherGroupID}})

	// Then
	assert.ErrorIs(t, err, ErrInvalidGoalOrder)
	assert.Nil(t, order)
}
