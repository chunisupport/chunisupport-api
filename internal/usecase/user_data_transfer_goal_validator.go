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
			return fmt.Errorf("%w: %w", ErrDataTransferInvalidData, err)
		}
		_, attrs, params, err := u.validateInputStatic(&GoalInput{
			Title:                       goal.Title,
			AchievementType:             goal.AchievementType,
			AchievementParams:           goal.AchievementParams,
			Attributes:                  attributes,
			InvertValue:                 goal.InvertValue,
			InvertPercentage:            goal.InvertPercentage,
			GaugeStart:                  goal.GaugeStart,
			ApplyGaugeStartToPercentage: goal.ApplyGaugeStartToPercentage,
		})
		if err != nil {
			return fmt.Errorf("%w: transferred goal is invalid: %w", ErrDataTransferInvalidData, err)
		}
		validations = append(validations, dynamicValidation{achievementType: goal.AchievementType, params: params, filter: goalTargetFilter(attrs, goal.AchievementType, params)})
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
		if err := validateDynamicUpperBoundWithStats(ctx, validation.achievementType, validation.params, &stats[index]); err != nil {
			return fmt.Errorf("%w: transferred goal is invalid: %w", ErrDataTransferInvalidData, err)
		}
	}
	return nil
}

// internalizeTransferredGoalAttributes は移行ファイルの絞り込み属性（マスタ名）を保存形式へ変換します。
// 難易度・ジャンルはマスタ名のまま保存し、バージョンは改名で参照が切れないよう公開IDへ変換します。
func internalizeTransferredGoalAttributes(raw json.RawMessage, masters *domainmasterdata.GoalMasters) ([]byte, error) {
	var attrs map[string]json.RawMessage
	if err := json.Unmarshal(raw, &attrs); err != nil {
		return nil, err
	}
	stored := map[string]map[string]string{
		"diff":  make(map[string]string, len(masters.Difficulties)),
		"genre": make(map[string]string, len(masters.Genres)),
		"ver":   make(map[string]string, len(masters.VersionsByDisplayID)),
	}
	for name := range masters.Difficulties {
		stored["diff"][name] = name
	}
	for name := range masters.Genres {
		stored["genre"][name] = name
	}
	for displayID, version := range masters.VersionsByDisplayID {
		stored["ver"][version.Name] = displayID
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
		values := make([]any, 0, len(items))
		for _, item := range items {
			name, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("%s must contain master names", key)
			}
			storedValue, ok := stored[key][name]
			if !ok {
				return nil, fmt.Errorf("%s contains an unknown master name", key)
			}
			values = append(values, storedValue)
		}
		var normalized any = values
		if !isArray && len(values) == 1 {
			normalized = values[0]
		}
		encoded, err := json.Marshal(normalized)
		if err != nil {
			return nil, err
		}
		attrs[key] = encoded
	}
	return json.Marshal(attrs)
}

var _ UserDataTransferGoalValidator = (*goalUsecase)(nil)
