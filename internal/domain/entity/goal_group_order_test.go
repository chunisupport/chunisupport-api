package entity

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGoalGroupOrder_ReorderAssignsDenseSortOrders(t *testing.T) {
	// Given
	order, err := NewGoalGroupOrder(1, []*GoalGroup{
		{ID: 10, UserID: 1},
		{ID: 20, UserID: 1},
		{ID: 30, UserID: 1},
	})
	require.NoError(t, err)

	// When
	err = order.Reorder([]uint32{30, 10, 20})

	// Then
	require.NoError(t, err)
	groups := order.Groups()
	assert.Equal(t, []uint32{30, 10, 20}, []uint32{groups[0].ID, groups[1].ID, groups[2].ID})
	assert.Equal(t, []uint16{1, 2, 3}, []uint16{groups[0].SortOrder, groups[1].SortOrder, groups[2].SortOrder})
}

func TestGoalGroupOrder_ReorderRejectsDifferentGroupSet(t *testing.T) {
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
			order, err := NewGoalGroupOrder(1, []*GoalGroup{{ID: 10, UserID: 1}, {ID: 20, UserID: 1}})
			require.NoError(t, err)

			// When
			err = order.Reorder(tt.orderedIDs)

			// Then: 拒否時は並び順とSortOrderが変化しないことを固定期待値で保証する
			assert.ErrorIs(t, err, ErrInvalidGoalGroupOrder)
			groups := order.Groups()
			require.Len(t, groups, 2)
			assert.Equal(t, []uint32{10, 20}, []uint32{groups[0].ID, groups[1].ID})
			assert.Equal(t, []uint16{1, 2}, []uint16{groups[0].SortOrder, groups[1].SortOrder})
		})
	}
}

func TestNewGoalGroupOrder_RejectsAnotherUsersGroup(t *testing.T) {
	// When
	order, err := NewGoalGroupOrder(1, []*GoalGroup{{ID: 10, UserID: 1}, {ID: 20, UserID: 2}})

	// Then
	assert.ErrorIs(t, err, ErrInvalidGoalGroupOrder)
	assert.Nil(t, order)
}

func TestGoalGroupOrder_RemoveRenumbersRemainingGroups(t *testing.T) {
	// Given
	order, err := NewGoalGroupOrder(1, []*GoalGroup{{ID: 10, UserID: 1}, {ID: 20, UserID: 1}, {ID: 30, UserID: 1}})
	require.NoError(t, err)

	// When
	err = order.Remove(20)

	// Then
	require.NoError(t, err)
	groups := order.Groups()
	assert.Equal(t, []uint32{10, 30}, []uint32{groups[0].ID, groups[1].ID})
	assert.Equal(t, []uint16{1, 2}, []uint16{groups[0].SortOrder, groups[1].SortOrder})
}
