package usecase

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainmasterdata "github.com/chunisupport/chunisupport-api/internal/domain/masterdata"
	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
)

type UserDataTransferGoalValidator interface {
	ValidateTransferredGoals(ctx context.Context, goals entity.UserDataTransferGoals) error
}

type GoalUsecaseWithTransferValidation interface {
	GoalUsecase
	UserDataTransferGoalValidator
}

func (u *goalUsecase) ValidateTransferredGoals(ctx context.Context, goals entity.UserDataTransferGoals) error {
	masters := u.masterProvider.GoalMasters()
	if masters == nil {
		return ErrInternalError
	}
	type dynamicValidation struct {
		achievementType string
		params          *goalAchievementParam
		filter          domainrepo.GoalTargetFilter
	}
	validations := make([]dynamicValidation, 0, len(goals.Ungrouped))
	validate := func(goal entity.UserDataTransferGoal) error {
		attributes, err := internalizeTransferredGoalAttributes(goal.Attributes, masters)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrDataTransferInvalidData, err)
		}
		_, attrs, params, err := u.validateInputStatic(&GoalInput{
			Title:             goal.Title,
			AchievementType:   goal.AchievementType,
			AchievementParams: goal.AchievementParams,
			Attributes:        attributes,
			InvertValue:       goal.InvertValue,
			InvertPercentage:  goal.InvertPercentage,
		})
		if err != nil {
			return fmt.Errorf("%w: transferred goal is invalid: %v", ErrDataTransferInvalidData, err)
		}
		validations = append(validations, dynamicValidation{achievementType: goal.AchievementType, params: params, filter: goalTargetFilter(attrs)})
		return nil
	}
	for _, goal := range goals.Ungrouped {
		if err := validate(goal); err != nil {
			return err
		}
	}
	for _, group := range goals.Groups {
		for _, goal := range group.Goals {
			if err := validate(goal); err != nil {
				return err
			}
		}
	}
	filters := make([]domainrepo.GoalTargetFilter, len(validations))
	for index, validation := range validations {
		filters[index] = validation.filter
	}
	stats, err := u.goalRepo.GetTargetStatsBatch(ctx, u.db, filters)
	if err != nil {
		return err
	}
	if len(stats) != len(validations) {
		return ErrInternalError
	}
	for index, validation := range validations {
		if err := validateDynamicUpperBoundWithStats(validation.achievementType, validation.params, &stats[index]); err != nil {
			return fmt.Errorf("%w: transferred goal is invalid: %v", ErrDataTransferInvalidData, err)
		}
	}
	return nil
}
func internalizeTransferredGoalAttributes(raw json.RawMessage, masters *domainmasterdata.GoalMasters) ([]byte, error) {
	var attrs map[string]json.RawMessage
	if err := json.Unmarshal(raw, &attrs); err != nil {
		return nil, err
	}
	reverse := map[string]map[string]int{
		"diff":  reverseTransferredNames(masters.DifficultyNamesByID),
		"genre": reverseTransferredNames(masters.GenreNamesByID),
		"ver":   reverseTransferredVersions(masters.VersionsByID),
	}
	for _, key := range []string{"diff", "genre", "ver"} {
		value, ok := attrs[key]
		if !ok {
			continue
		}
		var decoded any
		if err := json.Unmarshal(value, &decoded); err != nil {
			return nil, err
		}
		isArray := false
		items := []any{decoded}
		if array, ok := decoded.([]any); ok {
			items = array
			isArray = true
		}
		ids := make([]any, 0, len(items))
		for _, item := range items {
			name, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("%s must contain master names", key)
			}
			id, ok := reverse[key][name]
			if !ok {
				return nil, fmt.Errorf("%s contains an unknown master name", key)
			}
			ids = append(ids, id)
		}
		var normalized any = ids
		if !isArray && len(ids) == 1 {
			normalized = ids[0]
		}
		encoded, err := json.Marshal(normalized)
		if err != nil {
			return nil, err
		}
		attrs[key] = encoded
	}
	return json.Marshal(attrs)
}

func reverseTransferredNames(values map[int]string) map[string]int {
	result := make(map[string]int, len(values))
	for id, name := range values {
		result[name] = id
	}
	return result
}

func reverseTransferredVersions(values map[int]domainmasterdata.Version) map[string]int {
	result := make(map[string]int, len(values))
	for id, version := range values {
		result[version.Name] = id
	}
	return result
}

var _ UserDataTransferGoalValidator = (*goalUsecase)(nil)
