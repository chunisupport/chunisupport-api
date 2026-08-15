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
	// Given
	order, err := NewGoalGroupOrder(1, []*GoalGroup{{ID: 10, UserID: 1}, {ID: 20, UserID: 1}})
	require.NoError(t, err)

	// When
	err = order.Reorder([]uint32{10, 99})

	// Then
	assert.ErrorIs(t, err, ErrInvalidGoalGroupOrder)
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
