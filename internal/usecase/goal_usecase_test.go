package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainmasterdata "github.com/chunisupport/chunisupport-api/internal/domain/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	dto_internal "github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
)

type mockGoalRepo struct {
	goals []*entity.Goal
}

func (m *mockGoalRepo) FindByUserID(ctx context.Context, exec repository.Executor, userID int) ([]*entity.Goal, error) {
	return m.goals, nil
}
func (m *mockGoalRepo) FindByIDAndUserID(ctx context.Context, exec repository.Executor, id int, userID int) (*entity.Goal, error) {
	for _, goal := range m.goals {
		if goal.ID == id && goal.UserID == userID {
			return goal, nil
		}
	}
	return nil, nil
}
func (m *mockGoalRepo) Create(ctx context.Context, exec repository.Executor, goal *entity.Goal) error {
	goal.ID = len(m.goals) + 1
	goal.CreatedAt = time.Now()
	m.goals = append(m.goals, goal)
	return nil
}
func (m *mockGoalRepo) Update(ctx context.Context, exec repository.Executor, goal *entity.Goal) error {
	return nil
}
func (m *mockGoalRepo) Delete(ctx context.Context, exec repository.Executor, id int, userID int) error {
	return nil
}
func (m *mockGoalRepo) CountByUserID(ctx context.Context, exec repository.Executor, userID int) (int, error) {
	return len(m.goals), nil
}
func (m *mockGoalRepo) LockUser(ctx context.Context, exec repository.Executor, userID int) error {
	return nil
}

type passthroughTM struct{ exec repository.Executor }

func (m *passthroughTM) Transactional(ctx context.Context, f func(tx repository.Executor) error) error {
	return f(m.exec)
}

func newGoalMasters() *domainmasterdata.GoalMasters {
	return &domainmasterdata.GoalMasters{
		AchievementTypesByCode: map[string]domainmasterdata.Item{"score_count": {ID: 2, Name: "score_count"}},
		AchievementTypesByID:   map[int]string{2: "score_count"},
		GenresByID:             map[int]domainmasterdata.Item{1: {ID: 1, Name: "POPS"}},
		VersionsByID:           map[int]domainmasterdata.Version{1: {ID: 1, Name: "LUMINOUS"}},
		ClearLampsByName:       map[string]domainmasterdata.Item{"HARD": {ID: 1, Name: "HARD"}},
		ComboLampsByName:       map[string]domainmasterdata.Item{"FULL COMBO": {ID: 1, Name: "FULL COMBO"}},
	}
}

func TestGoalCreate(t *testing.T) {
	repo := &mockGoalRepo{}
	uc := NewGoalService(nil, &passthroughTM{}, repo, newGoalMasters())
	goal, err := uc.Create(context.Background(), 1, &dto_internal.UpsertGoalRequestDTO{
		Title:             "目標",
		AchievementType:   "score_count",
		AchievementParams: map[string]any{"score": 1000000, "count": 1},
		Attributes:        map[string]any{"diff": 4},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if goal.ID == 0 {
		t.Fatalf("created goal id should not be zero")
	}
}

func TestGoalCreateInvalidAchievementType(t *testing.T) {
	repo := &mockGoalRepo{}
	uc := NewGoalService(nil, &passthroughTM{}, repo, newGoalMasters())
	_, err := uc.Create(context.Background(), 1, &dto_internal.UpsertGoalRequestDTO{
		Title:             "目標",
		AchievementType:   "invalid",
		AchievementParams: map[string]any{"score": 1000000, "count": 1},
	})
	if err == nil {
		t.Fatalf("Create() should return error")
	}
}
