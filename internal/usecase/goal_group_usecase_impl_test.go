package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubGoalGroupRepo struct {
	groups    []*entity.GoalGroup
	createErr error
	saveErr   error
}

func (s *stubGoalGroupRepo) ListByUserID(ctx context.Context, exec repository.Executor, userID int) ([]*entity.GoalGroup, error) {
	return append([]*entity.GoalGroup(nil), s.groups...), nil
}

func (s *stubGoalGroupRepo) FindByIDAndUserID(ctx context.Context, exec repository.Executor, id uint32, userID int) (*entity.GoalGroup, error) {
	for _, group := range s.groups {
		if group.ID == id && group.UserID == userID {
			return group, nil
		}
	}
	return nil, repository.ErrGoalGroupNotFound
}

func (s *stubGoalGroupRepo) Save(ctx context.Context, exec repository.Executor, group *entity.GoalGroup) error {
	if group.ID == 0 {
		if s.createErr != nil {
			return s.createErr
		}
		group.ID = uint32(len(s.groups) + 1)
		group.CreatedAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		s.groups = append(s.groups, group)
		return nil
	}
	return s.saveErr
}

func (s *stubGoalGroupRepo) DeleteByIDAndUserID(ctx context.Context, exec repository.Executor, id uint32, userID int) error {
	for i, group := range s.groups {
		if group.ID == id && group.UserID == userID {
			s.groups = append(s.groups[:i], s.groups[i+1:]...)
			return nil
		}
	}
	return repository.ErrGoalGroupNotFound
}

func (s *stubGoalGroupRepo) SaveOrder(ctx context.Context, exec repository.Executor, order *entity.GoalGroupOrder) error {
	s.groups = order.Groups()
	return nil
}

func (s *stubGoalGroupRepo) CountByUserID(ctx context.Context, exec repository.Executor, userID int) (int, error) {
	return len(s.groups), nil
}

func newTestGoalGroup(t *testing.T, id uint32, userID int, name string, sortOrder uint16) *entity.GoalGroup {
	t.Helper()
	group, err := entity.NewGoalGroup(userID, name)
	require.NoError(t, err)
	group.ID = id
	group.SortOrder = sortOrder
	return group
}

func TestGoalGroupUsecase_CreateAppendsToEnd(t *testing.T) {
	// Given
	groupRepo := &stubGoalGroupRepo{groups: []*entity.GoalGroup{
		newTestGoalGroup(t, 1, 1, "A", 1),
		newTestGoalGroup(t, 2, 1, "B", 2),
	}}
	u := NewGoalGroupUsecase(nil, &stubTM{}, groupRepo, &stubGoalRepo{})

	// When
	created, err := u.Create(context.Background(), 1, "  C  ")

	// Then
	require.NoError(t, err)
	assert.Equal(t, "C", created.Name)
	assert.Equal(t, uint16(3), created.SortOrder)
}

func TestGoalGroupUsecase_CreateRejectsLimit(t *testing.T) {
	// Given
	groups := make([]*entity.GoalGroup, 0, info.GoalGroupMaxPerUser)
	for i := range info.GoalGroupMaxPerUser {
		groups = append(groups, newTestGoalGroup(t, uint32(i+1), 1, string(rune('A'+i)), uint16(i+1)))
	}
	u := NewGoalGroupUsecase(nil, &stubTM{}, &stubGoalGroupRepo{groups: groups}, &stubGoalRepo{})

	// When
	_, err := u.Create(context.Background(), 1, "上限超過")

	// Then
	assert.ErrorIs(t, err, ErrGoalGroupLimitExceeded)
}

func TestGoalGroupUsecase_DeleteMovesGoalsToUnclassifiedEnd(t *testing.T) {
	// Given
	groupID := uint32(10)
	groupRepo := &stubGoalGroupRepo{groups: []*entity.GoalGroup{
		newTestGoalGroup(t, groupID, 1, "削除対象", 1),
		newTestGoalGroup(t, 20, 1, "残す", 2),
	}}
	goalRepo := &stubGoalRepo{goals: []*entity.Goal{
		{ID: 2, UserID: 1, GroupID: &groupID, SortOrder: 1},
		{ID: 3, UserID: 1, GroupID: &groupID, SortOrder: 2},
		{ID: 1, UserID: 1, SortOrder: 1},
	}}
	u := NewGoalGroupUsecase(nil, &stubTM{}, groupRepo, goalRepo)

	// When
	err := u.Delete(context.Background(), 1, groupID)

	// Then
	require.NoError(t, err)
	assert.Len(t, groupRepo.groups, 1)
	assert.Equal(t, uint16(1), groupRepo.groups[0].SortOrder)
	for _, goal := range goalRepo.goals {
		assert.Nil(t, goal.GroupID)
	}
	assert.Equal(t, []uint32{1, 2, 3}, goalRepo.savedOrder)
	assert.Equal(t, []uint16{1, 2, 3}, goalRepo.savedSorts)
}

func TestGoalUsecase_UpdateMovesGoalToGroupEnd(t *testing.T) {
	// Given
	groupID := uint32(10)
	target := &entity.Goal{ID: 1, UserID: 1, SortOrder: 1}
	goalRepo := &stubGoalRepo{goal: target, goals: []*entity.Goal{
		target,
		{ID: 2, UserID: 1, GroupID: &groupID, SortOrder: 1},
	}}
	groupRepo := &stubGoalGroupRepo{groups: []*entity.GoalGroup{newTestGoalGroup(t, groupID, 1, "移動先", 1)}}
	u := NewGoalUsecase(nil, &stubTM{}, goalRepo, &stubGoalMasterProvider{}, groupRepo)

	// When
	_, err := u.Update(context.Background(), 1, 1, &GoalInput{
		GroupID:           &groupID,
		Title:             "test",
		AchievementType:   "score_count",
		AchievementParams: []byte(`{"score":1000000,"count":1}`),
		Attributes:        []byte(`{}`),
	})

	// Then
	require.NoError(t, err)
	require.NotNil(t, target.GroupID)
	assert.Equal(t, groupID, *target.GroupID)
	assert.Equal(t, uint16(2), target.SortOrder)
}
