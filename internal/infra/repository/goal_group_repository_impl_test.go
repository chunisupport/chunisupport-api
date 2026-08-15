package repository

import (
	"context"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGoalGroupRepository_PreservesEmptyGroupsAndOrder(t *testing.T) {
	// Given
	db := setupGoalRepositorySQLite(t)
	repo := &goalGroupRepository{}
	first, err := entity.NewGoalGroup(1, "first")
	require.NoError(t, err)
	first.SortOrder = 1
	second, err := entity.NewGoalGroup(1, "second")
	require.NoError(t, err)
	second.SortOrder = 2
	require.NoError(t, repo.Save(context.Background(), db, first))
	require.NoError(t, repo.Save(context.Background(), db, second))

	// When
	groups, err := repo.ListByUserID(context.Background(), db, 1)

	// Then
	require.NoError(t, err)
	require.Len(t, groups, 2)
	assert.Equal(t, []string{"first", "second"}, []string{groups[0].Name.String(), groups[1].Name.String()})
	assert.Equal(t, []uint16{1, 2}, []uint16{groups[0].SortOrder, groups[1].SortOrder})
}

func TestGoalGroupRepository_SaveOrder(t *testing.T) {
	// Given
	db := setupGoalRepositorySQLite(t)
	_, err := db.Exec(`INSERT INTO goal_groups (id, user_id, name, sort_order, created_at) VALUES
		(10, 1, 'first', 1, '2026-01-01'),
		(20, 1, 'second', 2, '2026-01-01')`)
	require.NoError(t, err)
	repo := &goalGroupRepository{}
	groups, err := repo.ListByUserID(context.Background(), db, 1)
	require.NoError(t, err)
	order, err := entity.NewGoalGroupOrder(1, groups)
	require.NoError(t, err)
	require.NoError(t, order.Reorder([]uint32{20, 10}))

	// When
	err = repo.SaveOrder(context.Background(), db, order)

	// Then
	require.NoError(t, err)
	groups, err = repo.ListByUserID(context.Background(), db, 1)
	require.NoError(t, err)
	assert.Equal(t, []uint32{20, 10}, []uint32{groups[0].ID, groups[1].ID})
}
