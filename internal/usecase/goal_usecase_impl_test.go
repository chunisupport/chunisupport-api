package usecase

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainmasterdata "github.com/chunisupport/chunisupport-api/internal/domain/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubGoalRepo struct {
	count int
	goal  *entity.Goal
}

func (s *stubGoalRepo) ListByUserID(ctx context.Context, exec repository.Executor, userID int) ([]*entity.Goal, error) {
	return []*entity.Goal{s.goal}, nil
}
func (s *stubGoalRepo) FindByIDAndUserID(ctx context.Context, exec repository.Executor, id int64, userID int) (*entity.Goal, error) {
	if s.goal == nil || s.goal.ID != id {
		return nil, sql.ErrNoRows
	}
	return s.goal, nil
}
func (s *stubGoalRepo) Create(ctx context.Context, exec repository.Executor, goal *entity.Goal) error {
	goal.ID = 1
	goal.CreatedAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	s.goal = goal
	return nil
}
func (s *stubGoalRepo) Update(ctx context.Context, exec repository.Executor, goal *entity.Goal) error {
	s.goal = goal
	return nil
}
func (s *stubGoalRepo) DeleteByIDAndUserID(ctx context.Context, exec repository.Executor, id int64, userID int) error {
	s.goal = nil
	return nil
}
func (s *stubGoalRepo) CountByUserID(ctx context.Context, exec repository.Executor, userID int) (int, error) {
	return s.count, nil
}
func (s *stubGoalRepo) LockUserByID(ctx context.Context, exec repository.Executor, userID int) error {
	return nil
}

type stubTM struct{}

func (s *stubTM) Transactional(ctx context.Context, f func(tx repository.Executor) error) error {
	return f(nil)
}

type stubGoalMasterProvider struct{}

func (s *stubGoalMasterProvider) GoalMasters() *domainmasterdata.GoalMasters {
	return &domainmasterdata.GoalMasters{
		AchievementTypesByCode: map[string]domainmasterdata.Item{"score_count": {ID: 2, Name: "score_count"}},
		AchievementTypesByID:   map[int]string{2: "score_count"},
		GenreNamesByID:         map[int]string{1: "POPS & ANIME"},
		VersionsByID:           map[int]domainmasterdata.Version{20: {ID: 20, Name: "VERSE"}},
	}
}

func TestGoalUsecase_Create(t *testing.T) {
	repo := &stubGoalRepo{}
	u := NewGoalUsecase(nil, &stubTM{}, repo, &stubGoalMasterProvider{})
	in := &GoalInput{Title: "  test  ", AchievementType: "score_count", AchievementParams: []byte(`{"score":1000000,"count":1}`), Attributes: []byte(`{"diff":4,"genre":1,"ver":20}`)}
	out, err := u.Create(context.Background(), 1, in)
	require.NoError(t, err)
	assert.Equal(t, "test", out.Title)
	assert.Equal(t, "score_count", out.AchievementType)
}

func TestGoalUsecase_CreateLimitExceeded(t *testing.T) {
	repo := &stubGoalRepo{count: 100}
	u := NewGoalUsecase(nil, &stubTM{}, repo, &stubGoalMasterProvider{})
	_, err := u.Create(context.Background(), 1, &GoalInput{Title: "test", AchievementType: "score_count", AchievementParams: []byte(`{"score":1000000,"count":1}`), Attributes: []byte(`{}`)})
	assert.True(t, errors.Is(err, ErrGoalLimitExceeded))
}
