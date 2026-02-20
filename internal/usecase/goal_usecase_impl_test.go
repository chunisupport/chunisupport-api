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
	count     int
	goal      *entity.Goal
	updateErr error
}

func (s *stubGoalRepo) ListByUserID(ctx context.Context, exec repository.Executor, userID int) ([]*entity.Goal, error) {
	return []*entity.Goal{s.goal}, nil
}
func (s *stubGoalRepo) FindByIDAndUserID(ctx context.Context, exec repository.Executor, id uint32, userID int) (*entity.Goal, error) {
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
	if s.updateErr != nil {
		return s.updateErr
	}
	s.goal = goal
	return nil
}
func (s *stubGoalRepo) DeleteByIDAndUserID(ctx context.Context, exec repository.Executor, id uint32, userID int) error {
	if s.goal == nil || s.goal.ID != id {
		return sql.ErrNoRows
	}
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

type stubNilGoalMasterProvider struct{}

func (s *stubNilGoalMasterProvider) GoalMasters() *domainmasterdata.GoalMasters {
	return nil
}

func (s *stubGoalMasterProvider) GoalMasters() *domainmasterdata.GoalMasters {
	return &domainmasterdata.GoalMasters{
		AchievementTypesByCode: map[string]domainmasterdata.Item{"score_count": {ID: 2, Name: "score_count"}, "overpower_percent": {ID: 8, Name: "overpower_percent"}},
		AchievementTypesByID:   map[int]string{2: "score_count", 8: "overpower_percent"},
		DifficultyNamesByID:    map[int]string{4: "MASTER"},
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

func TestGoalUsecase_Delete(t *testing.T) {
	repo := &stubGoalRepo{goal: &entity.Goal{ID: 1, UserID: 1}}
	u := NewGoalUsecase(nil, &stubTM{}, repo, &stubGoalMasterProvider{})
	err := u.Delete(context.Background(), 1, 1)
	require.NoError(t, err)
	assert.Nil(t, repo.goal)
}

func TestGoalUsecase_DeleteNotFound(t *testing.T) {
	repo := &stubGoalRepo{}
	u := NewGoalUsecase(nil, &stubTM{}, repo, &stubGoalMasterProvider{})
	err := u.Delete(context.Background(), 1, 999)
	assert.True(t, errors.Is(err, ErrGoalNotFound))
}

func TestGoalUsecase_UpdateNotFoundOnSave(t *testing.T) {
	repo := &stubGoalRepo{goal: &entity.Goal{ID: 1, UserID: 1}, updateErr: sql.ErrNoRows}
	u := NewGoalUsecase(nil, &stubTM{}, repo, &stubGoalMasterProvider{})
	_, err := u.Update(context.Background(), 1, 1, &GoalInput{
		Title:             "updated",
		AchievementType:   "score_count",
		AchievementParams: []byte(`{"score":1000000,"count":1}`),
		Attributes:        []byte(`{"diff":4,"genre":1,"ver":20}`),
	})
	assert.True(t, errors.Is(err, ErrGoalNotFound))
}

func TestGoalUsecase_CreateMasterDataUnavailable(t *testing.T) {
	repo := &stubGoalRepo{}
	u := NewGoalUsecase(nil, &stubTM{}, repo, &stubNilGoalMasterProvider{})
	_, err := u.Create(context.Background(), 1, &GoalInput{
		Title:             "test",
		AchievementType:   "score_count",
		AchievementParams: []byte(`{"score":1000000,"count":1}`),
		Attributes:        []byte(`{}`),
	})
	assert.True(t, errors.Is(err, ErrInternalError))
}

func TestGoalUsecase_CreateInvalidDifficultyAttribute(t *testing.T) {
	repo := &stubGoalRepo{}
	u := NewGoalUsecase(nil, &stubTM{}, repo, &stubGoalMasterProvider{})
	_, err := u.Create(context.Background(), 1, &GoalInput{
		Title:             "test",
		AchievementType:   "score_count",
		AchievementParams: []byte(`{"score":1000000,"count":1}`),
		Attributes:        []byte(`{"diff":5,"genre":1,"ver":20}`),
	})
	assert.True(t, errors.Is(err, ErrInvalidGoalAttributes))
}

func TestGoalUsecase_CreateConstAttributeWithOmittedMinUsesDefault(t *testing.T) {
	repo := &stubGoalRepo{}
	u := NewGoalUsecase(nil, &stubTM{}, repo, &stubGoalMasterProvider{})
	out, err := u.Create(context.Background(), 1, &GoalInput{
		Title:             "test",
		AchievementType:   "score_count",
		AchievementParams: []byte(`{"score":1000000,"count":1}`),
		Attributes:        []byte(`{"const":{"max":15.9}}`),
	})
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"const": map[string]any{"min": float64(1), "max": float64(15.9)}}, out.Attributes)
}

func TestGoalUsecase_CreateConstAttributeWithOmittedMaxUsesDefault(t *testing.T) {
	repo := &stubGoalRepo{}
	u := NewGoalUsecase(nil, &stubTM{}, repo, &stubGoalMasterProvider{})
	out, err := u.Create(context.Background(), 1, &GoalInput{
		Title:             "test",
		AchievementType:   "score_count",
		AchievementParams: []byte(`{"score":1000000,"count":1}`),
		Attributes:        []byte(`{"const":{"min":1.2}}`),
	})
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"const": map[string]any{"min": float64(1.2), "max": float64(15.9)}}, out.Attributes)
}

func TestGoalUsecase_CreateOverpowerPercentAcceptsRealValue(t *testing.T) {
	repo := &stubGoalRepo{}
	u := NewGoalUsecase(nil, &stubTM{}, repo, &stubGoalMasterProvider{})
	_, err := u.Create(context.Background(), 1, &GoalInput{
		Title:             "test",
		AchievementType:   "overpower_percent",
		AchievementParams: []byte(`{"total":123.456}`),
		Attributes:        []byte(`{}`),
	})
	require.NoError(t, err)
}

func TestGoalUsecase_CreateOverpowerPercentRejectsMoreThan3Decimals(t *testing.T) {
	repo := &stubGoalRepo{}
	u := NewGoalUsecase(nil, &stubTM{}, repo, &stubGoalMasterProvider{})
	_, err := u.Create(context.Background(), 1, &GoalInput{
		Title:             "test",
		AchievementType:   "overpower_percent",
		AchievementParams: []byte(`{"total":12.3456}`),
		Attributes:        []byte(`{}`),
	})
	assert.True(t, errors.Is(err, ErrInvalidAchievementParam))
}
