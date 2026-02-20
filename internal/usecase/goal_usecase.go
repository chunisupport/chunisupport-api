package usecase

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"strings"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainmasterdata "github.com/chunisupport/chunisupport-api/internal/domain/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/info"
)

// GoalUsecase は目標機能のユースケースです。
type GoalUsecase interface {
	List(ctx context.Context, userID int) ([]*api_internal.GoalDTO, error)
	Create(ctx context.Context, userID int, req *api_internal.UpsertGoalRequestDTO) (*api_internal.GoalDTO, error)
	Update(ctx context.Context, userID int, id int, req *api_internal.UpsertGoalRequestDTO) error
	Delete(ctx context.Context, userID int, id int) error
}

type goalService struct {
	db      repository.Executor
	tm      TransactionManager
	repo    repository.GoalRepository
	masters *domainmasterdata.GoalMasters
}

func NewGoalService(db repository.Executor, tm TransactionManager, repo repository.GoalRepository, masters *domainmasterdata.GoalMasters) GoalUsecase {
	return &goalService{db: db, tm: tm, repo: repo, masters: masters}
}

func (s *goalService) List(ctx context.Context, userID int) ([]*api_internal.GoalDTO, error) {
	goals, err := s.repo.FindByUserID(ctx, s.db, userID)
	if err != nil {
		return nil, err
	}
	return api_internal.ToGoalDTOs(goals, s.masters.AchievementTypesByID), nil
}

func (s *goalService) Create(ctx context.Context, userID int, req *api_internal.UpsertGoalRequestDTO) (*api_internal.GoalDTO, error) {
	var created *api_internal.GoalDTO
	err := s.tm.Transactional(ctx, func(tx repository.Executor) error {
		if err := s.repo.LockUser(ctx, tx, userID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrUserNotFound
			}
			return err
		}
		count, err := s.repo.CountByUserID(ctx, tx, userID)
		if err != nil {
			return err
		}
		if count >= info.GoalLimitPerUser {
			return ErrGoalLimitExceeded
		}
		goal, err := s.buildGoal(userID, req)
		if err != nil {
			return err
		}
		if err := s.repo.Create(ctx, tx, goal); err != nil {
			return err
		}
		stored, err := s.repo.FindByIDAndUserID(ctx, tx, goal.ID, userID)
		if err != nil {
			return err
		}
		created = api_internal.ToGoalDTO(stored, s.masters.AchievementTypesByID)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}

func (s *goalService) Update(ctx context.Context, userID int, id int, req *api_internal.UpsertGoalRequestDTO) error {
	goal, err := s.buildGoal(userID, req)
	if err != nil {
		return err
	}
	goal.ID = id
	if err := s.repo.Update(ctx, s.db, goal); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrGoalNotFound
		}
		return err
	}
	return nil
}

func (s *goalService) Delete(ctx context.Context, userID int, id int) error {
	if err := s.repo.Delete(ctx, s.db, id, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrGoalNotFound
		}
		return err
	}
	return nil
}

func (s *goalService) buildGoal(userID int, req *api_internal.UpsertGoalRequestDTO) (*entity.Goal, error) {
	title := strings.TrimSpace(req.Title)
	if title == "" || len([]rune(title)) > 30 {
		return nil, ErrInvalidGoalRequest
	}
	ach, ok := s.masters.AchievementTypesByCode[req.AchievementType]
	if !ok {
		return nil, ErrInvalidAchievementType
	}
	if err := s.validateAttributes(req.Attributes); err != nil {
		return nil, err
	}
	if err := s.validateAchievement(req.AchievementType, req.AchievementParams, req.Attributes); err != nil {
		return nil, err
	}
	return &entity.Goal{
		UserID:            userID,
		Title:             title,
		AchievementTypeID: ach.ID,
		AchievementParams: req.AchievementParams,
		Attributes:        req.Attributes,
		Invert:            req.Invert,
	}, nil
}

func (s *goalService) validateAttributes(attrs map[string]any) error {
	if attrs == nil {
		return nil
	}
	if diffValue, ok := attrs["diff"]; ok {
		diff, ok := toInt(diffValue)
		if !ok || diff < 1 || diff > 5 {
			return ErrInvalidGoalRequest
		}
	}
	if genreValue, ok := attrs["genre"]; ok {
		genreID, ok := toInt(genreValue)
		if !ok {
			return ErrInvalidGoalRequest
		}
		if _, exists := s.masters.GenresByID[genreID]; !exists {
			return ErrInvalidGoalRequest
		}
	}
	if versionValue, ok := attrs["ver"]; ok {
		versionID, ok := toInt(versionValue)
		if !ok {
			return ErrInvalidGoalRequest
		}
		if _, exists := s.masters.VersionsByID[versionID]; !exists {
			return ErrInvalidGoalRequest
		}
	}
	if constValue, ok := attrs["const"]; ok {
		obj, ok := constValue.(map[string]any)
		if !ok {
			return ErrInvalidGoalRequest
		}
		minValue, ok := toFloat(obj["min"])
		if !ok {
			return ErrInvalidGoalRequest
		}
		maxValue, ok := toFloat(obj["max"])
		if !ok {
			return ErrInvalidGoalRequest
		}
		if minValue < info.ChartConstMin || maxValue > info.ChartConstMax || minValue > maxValue {
			return ErrInvalidGoalRequest
		}
	}
	return nil
}

func (s *goalService) validateAchievement(kind string, params map[string]any, attrs map[string]any) error {
	if params == nil {
		return ErrInvalidGoalRequest
	}
	switch kind {
	case "rank_count", "score_count":
		score, ok := toInt(params["score"])
		if !ok || score < 0 || score > 1010000 {
			return ErrInvalidGoalRequest
		}
		count, ok := toInt(params["count"])
		if !ok || count < 1 {
			return ErrInvalidGoalRequest
		}
	case "avg_score":
		score, ok := toInt(params["score"])
		if !ok || score < 0 || score > 1010000 {
			return ErrInvalidGoalRequest
		}
	case "hardlamp_count":
		lamp, ok := params["lamp"].(string)
		if !ok {
			return ErrInvalidGoalRequest
		}
		name, ok := info.HardLampAbbrevToName[lamp]
		if !ok {
			return ErrInvalidGoalRequest
		}
		if _, ok := s.masters.ClearLampsByName[name]; !ok {
			return ErrInvalidGoalRequest
		}
		count, ok := toInt(params["count"])
		if !ok || count < 1 {
			return ErrInvalidGoalRequest
		}
	case "combolamp_count":
		lamp, ok := params["lamp"].(string)
		if !ok {
			return ErrInvalidGoalRequest
		}
		name, ok := info.ComboLampAbbrevToName[lamp]
		if !ok {
			return ErrInvalidGoalRequest
		}
		if _, ok := s.masters.ComboLampsByName[name]; !ok {
			return ErrInvalidGoalRequest
		}
		count, ok := toInt(params["count"])
		if !ok || count < 1 {
			return ErrInvalidGoalRequest
		}
	case "total_score":
		total, ok := toInt(params["total"])
		if !ok || total < 0 {
			return ErrInvalidGoalRequest
		}
	case "overpower_value":
		total, ok := toFloat(params["total"])
		if !ok || total < 0 || roundDigits(total, 3) != total {
			return ErrInvalidGoalRequest
		}
	case "overpower_percent":
		total, ok := toFloat(params["total"])
		if !ok || total < 0 || total > 100 || roundDigits(total, 2) != total {
			return ErrInvalidGoalRequest
		}
	default:
		return ErrInvalidAchievementType
	}
	_ = attrs
	return nil
}

func toInt(v any) (int, bool) {
	switch t := v.(type) {
	case int:
		return t, true
	case float64:
		if math.Trunc(t) != t {
			return 0, false
		}
		return int(t), true
	default:
		return 0, false
	}
}

func toFloat(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case int:
		return float64(t), true
	default:
		return 0, false
	}
}

func roundDigits(v float64, digits int) float64 {
	pow := math.Pow10(digits)
	return math.Round(v*pow) / pow
}
