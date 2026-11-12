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

// snapshotGoalPositions は失敗時に配置が変わっていないことを確かめるための写しです。
// ポインタの共有による見かけの一致を避けるため、所属と表示順の値を複写します。
func snapshotGoalPositions(goals []*Goal) map[uint32]struct {
	groupID   *uint32
	sortOrder uint16
} {
	snapshot := make(map[uint32]struct {
		groupID   *uint32
		sortOrder uint16
	}, len(goals))
	for _, goal := range goals {
		snapshot[goal.ID] = struct {
			groupID   *uint32
			sortOrder uint16
		}{groupID: cloneOptionalID(goal.GroupID), sortOrder: goal.SortOrder}
	}
	return snapshot
}

func TestNewGoalArrangement_RejectsInvalidGoals(t *testing.T) {
	tests := []struct {
		name string
		// Given: 生成入力となる目標一覧
		goals []*Goal
	}{
		{
			name: "他のユーザーの目標が混ざっていると拒否する",
			goals: []*Goal{
				{ID: 1, UserID: 1},
				{ID: 2, UserID: 2},
			},
		},
		{
			name: "IDが重複していると拒否する",
			goals: []*Goal{
				{ID: 1, UserID: 1},
				{ID: 1, UserID: 1},
			},
		},
		{
			name: "nilの目標が混ざっていると拒否する",
			goals: []*Goal{
				{ID: 1, UserID: 1},
				nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			arrangement, err := NewGoalArrangement(1, tt.goals)

			// Then
			assert.ErrorIs(t, err, ErrInvalidGoalArrangement)
			assert.Nil(t, arrangement)
		})
	}
}

func TestGoalArrangement_ReorderRearrangesWithinGroup(t *testing.T) {
	// Given
	groupA := uint32(10)
	groupB := uint32(20)
	arrangement, err := NewGoalArrangement(1, []*Goal{
		{ID: 1, UserID: 1, GroupID: &groupA, SortOrder: 1},
		{ID: 2, UserID: 1, GroupID: &groupA, SortOrder: 2},
		{ID: 3, UserID: 1, GroupID: &groupA, SortOrder: 3},
		{ID: 4, UserID: 1, GroupID: &groupB, SortOrder: 1},
		{ID: 5, UserID: 1, GroupID: nil, SortOrder: 1},
	})
	require.NoError(t, err)

	// When
	err = arrangement.Reorder(&groupA, []uint32{3, 1, 2})

	// Then: 対象グループだけが指定順になり、他グループと未分類は変わらない
	require.NoError(t, err)
	assertGoalPositions(t, arrangement.Goals(), map[uint32]struct {
		groupID   *uint32
		sortOrder uint16
	}{
		3: {groupID: &groupA, sortOrder: 1},
		1: {groupID: &groupA, sortOrder: 2},
		2: {groupID: &groupA, sortOrder: 3},
		4: {groupID: &groupB, sortOrder: 1},
		5: {groupID: nil, sortOrder: 1},
	})
}

func TestGoalArrangement_ReorderRejectsUnknownID(t *testing.T) {
	// Given
	groupA := uint32(10)
	arrangement, err := NewGoalArrangement(1, []*Goal{
		{ID: 1, UserID: 1, GroupID: &groupA, SortOrder: 1},
		{ID: 2, UserID: 1, GroupID: &groupA, SortOrder: 2},
	})
	require.NoError(t, err)
	before := snapshotGoalPositions(arrangement.Goals())

	// When: 所属しないIDを含む順序を指定する
	err = arrangement.Reorder(&groupA, []uint32{1, 99})

	// Then: 並び替えは拒否され、配置全体が操作前から変わっていない
	assert.ErrorIs(t, err, ErrInvalidGoalOrder)
	assertGoalPositions(t, arrangement.Goals(), before)
}

func TestGoalArrangement_RemoveRenumbersRemainingGoals(t *testing.T) {
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
	err = arrangement.Remove(2)

	// Then: 同じグループだけが詰められ、他グループは変わらない
	require.NoError(t, err)
	assertGoalPositions(t, arrangement.Goals(), map[uint32]struct {
		groupID   *uint32
		sortOrder uint16
	}{
		1: {groupID: &groupA, sortOrder: 1},
		3: {groupID: &groupA, sortOrder: 2},
		4: {groupID: &groupB, sortOrder: 1},
	})
}

func TestGoalArrangement_MissingGoalReturnsNotFound(t *testing.T) {
	tests := []struct {
		name string
		// When: 存在しない目標への操作
		act func(arrangement *GoalArrangement) error
	}{
		{
			name: "存在しない目標の取り除きは見つからない",
			act: func(arrangement *GoalArrangement) error {
				return arrangement.Remove(99)
			},
		},
		{
			name: "存在しない目標の移動は見つからない",
			act: func(arrangement *GoalArrangement) error {
				destination := uint32(20)
				return arrangement.Move(99, &destination)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			groupA := uint32(10)
			groupB := uint32(20)
			arrangement, err := NewGoalArrangement(1, []*Goal{
				{ID: 1, UserID: 1, GroupID: &groupA, SortOrder: 1},
				{ID: 2, UserID: 1, GroupID: &groupB, SortOrder: 1},
			})
			require.NoError(t, err)
			before := snapshotGoalPositions(arrangement.Goals())

			// When
			err = tt.act(arrangement)

			// Then: 見つからないことが返り、配置全体が操作前から変わっていない
			assert.ErrorIs(t, err, ErrGoalArrangementMissing)
			assertGoalPositions(t, arrangement.Goals(), before)
		})
	}
}

func TestGoalArrangement_MoveToSameGroupKeepsPositions(t *testing.T) {
	// Given
	groupA := uint32(10)
	arrangement, err := NewGoalArrangement(1, []*Goal{
		{ID: 1, UserID: 1, GroupID: &groupA, SortOrder: 1},
		{ID: 2, UserID: 1, GroupID: &groupA, SortOrder: 2},
	})
	require.NoError(t, err)

	// When: 同じグループへの移動を指定する
	err = arrangement.Move(1, &groupA)

	// Then: 何も変わらない
	require.NoError(t, err)
	assertGoalPositions(t, arrangement.Goals(), map[uint32]struct {
		groupID   *uint32
		sortOrder uint16
	}{
		1: {groupID: &groupA, sortOrder: 1},
		2: {groupID: &groupA, sortOrder: 2},
	})
}

func TestGoalArrangement_MoveAppendsToNewOrUnclassifiedGroup(t *testing.T) {
	newGroup := uint32(99)
	tests := []struct {
		name string
		// Given: 移動先のグループID（nilは未分類を表す）
		destination *uint32
		// Then: 操作後に期待する配置全体
		expected map[uint32]struct {
			groupID   *uint32
			sortOrder uint16
		}
	}{
		{
			name:        "所属のない新しいグループへの移動は表示順1で末尾になる",
			destination: &newGroup,
			expected: map[uint32]struct {
				groupID   *uint32
				sortOrder uint16
			}{
				2: {groupID: new(uint32(10)), sortOrder: 1},
				3: {groupID: nil, sortOrder: 1},
				1: {groupID: &newGroup, sortOrder: 1},
			},
		},
		{
			name:        "未分類への移動は未分類の末尾になる",
			destination: nil,
			expected: map[uint32]struct {
				groupID   *uint32
				sortOrder uint16
			}{
				2: {groupID: new(uint32(10)), sortOrder: 1},
				3: {groupID: nil, sortOrder: 1},
				1: {groupID: nil, sortOrder: 2},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: ドメインは移動先グループの存在を検証しないため、所有権はユースケースが担う
			groupA := uint32(10)
			arrangement, err := NewGoalArrangement(1, []*Goal{
				{ID: 1, UserID: 1, GroupID: &groupA, SortOrder: 1},
				{ID: 2, UserID: 1, GroupID: &groupA, SortOrder: 2},
				{ID: 3, UserID: 1, GroupID: nil, SortOrder: 1},
			})
			require.NoError(t, err)

			// When
			err = arrangement.Move(1, tt.destination)

			// Then
			require.NoError(t, err)
			assertGoalPositions(t, arrangement.Goals(), tt.expected)
		})
	}
}
