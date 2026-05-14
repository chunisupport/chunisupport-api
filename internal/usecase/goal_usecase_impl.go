package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainmasterdata "github.com/chunisupport/chunisupport-api/internal/domain/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/info"
)

var (
	ErrGoalLimitExceeded       = errors.New("goal limit exceeded")
	ErrGoalNotFound            = errors.New("goal not found")
	ErrInvalidGoalInput        = errors.New("invalid goal input")
	ErrInvalidGoalTitle        = errors.New("invalid goal title")
	ErrInvalidAchievementType  = errors.New("invalid achievement type")
	ErrInvalidAchievementParam = errors.New("invalid achievement params")
	ErrInvalidGoalAttributes   = errors.New("invalid goal attributes")
)

type goalUsecase struct {
	db             repository.Executor
	tm             TransactionManager
	goalRepo       repository.GoalRepository
	masterProvider repository.GoalMasterProvider
}

func NewGoalUsecase(db repository.Executor, tm TransactionManager, goalRepo repository.GoalRepository, masterProvider repository.GoalMasterProvider) GoalUsecase {
	return &goalUsecase{db: db, tm: tm, goalRepo: goalRepo, masterProvider: masterProvider}
}

func (u *goalUsecase) List(ctx context.Context, userID int) ([]*GoalOutput, error) {
	goals, err := u.goalRepo.ListByUserID(ctx, u.db, userID)
	if err != nil {
		return nil, err
	}
	return u.toOutputs(goals)
}

func (u *goalUsecase) Create(ctx context.Context, userID int, input *GoalInput) (*GoalOutput, error) {
	validated, err := u.validateInput(ctx, input)
	if err != nil {
		return nil, err
	}

	var created *entity.Goal
	err = u.tm.Transactional(ctx, func(tx repository.Executor) error {
		if err := u.goalRepo.LockUserByID(ctx, tx, userID); err != nil {
			return err
		}
		count, err := u.goalRepo.CountByUserID(ctx, tx, userID)
		if err != nil {
			return err
		}
		if count >= info.GoalMaxPerUser {
			return ErrGoalLimitExceeded
		}
		goal := &entity.Goal{
			UserID:            userID,
			Title:             validated.Title,
			AchievementTypeID: validated.AchievementTypeID,
			AchievementParams: validated.AchievementParams,
			Attributes:        validated.Attributes,
			Invert:            validated.Invert,
		}
		if err := u.goalRepo.Create(ctx, tx, goal); err != nil {
			return err
		}
		created, err = u.goalRepo.FindByIDAndUserID(ctx, tx, goal.ID, userID)
		return err
	})
	if err != nil {
		return nil, err
	}
	outs, err := u.toOutputs([]*entity.Goal{created})
	if err != nil {
		return nil, err
	}
	return outs[0], nil
}

func (u *goalUsecase) Update(ctx context.Context, userID int, id uint32, input *GoalInput) (*GoalOutput, error) {
	validated, err := u.validateInput(ctx, input)
	if err != nil {
		return nil, err
	}
	goal, err := u.goalRepo.FindByIDAndUserID(ctx, u.db, id, userID)
	if err != nil {
		if errors.Is(err, repository.ErrGoalNotFound) {
			return nil, ErrGoalNotFound
		}
		return nil, err
	}
	goal.Title = validated.Title
	goal.AchievementTypeID = validated.AchievementTypeID
	goal.AchievementParams = validated.AchievementParams
	goal.Attributes = validated.Attributes
	goal.Invert = validated.Invert
	if err := u.goalRepo.Update(ctx, u.db, goal); err != nil {
		if errors.Is(err, repository.ErrGoalNotFound) {
			return nil, ErrGoalNotFound
		}
		return nil, err
	}
	outs, err := u.toOutputs([]*entity.Goal{goal})
	if err != nil {
		return nil, err
	}
	return outs[0], nil
}

func (u *goalUsecase) Delete(ctx context.Context, userID int, id uint32) error {
	err := u.goalRepo.DeleteByIDAndUserID(ctx, u.db, id, userID)
	if errors.Is(err, repository.ErrGoalNotFound) {
		return ErrGoalNotFound
	}
	return err
}

type validatedGoalInput struct {
	Title             string
	AchievementTypeID int
	AchievementParams []byte
	Attributes        []byte
	Invert            bool
}

type goalAttributeFilter struct {
	DifficultyIDs []int
	ConstMin      *float64
	ConstMax      *float64
	GenreIDs      []int
	VersionRanges []repository.VersionRange
}

type goalAchievementParam struct {
	Score *int
	Count *int
	Total *float64
}

func (u *goalUsecase) validateInput(ctx context.Context, input *GoalInput) (*validatedGoalInput, error) {
	if input == nil {
		return nil, ErrInvalidGoalInput
	}
	title := input.Title
	title = strings.TrimSpace(title)
	if title == "" || len([]rune(title)) > 30 || hasControlCharacter(title) {
		return nil, ErrInvalidGoalTitle
	}
	masters := u.masterProvider.GoalMasters()
	if masters == nil {
		return nil, ErrInternalError
	}
	item, ok := masters.AchievementTypesByCode[input.AchievementType]
	if !ok {
		return nil, ErrInvalidAchievementType
	}
	attrsRaw, attrsFilter, err := validateAttributes(input.Attributes, masters)
	if err != nil {
		return nil, err
	}
	paramsRaw, params, err := validateAchievementParams(input.AchievementType, input.AchievementParams)
	if err != nil {
		return nil, err
	}

	if err := u.validateDynamicUpperBound(ctx, input.AchievementType, attrsFilter, params); err != nil {
		return nil, err
	}

	return &validatedGoalInput{Title: title, AchievementTypeID: item.ID, AchievementParams: paramsRaw, Attributes: attrsRaw, Invert: input.Invert}, nil
}

func validateAttributes(raw []byte, masters *domainmasterdata.GoalMasters) ([]byte, *goalAttributeFilter, error) {
	var attrs map[string]json.RawMessage
	if len(raw) == 0 {
		return []byte("{}"), &goalAttributeFilter{}, nil
	}
	if err := json.Unmarshal(raw, &attrs); err != nil {
		return nil, nil, ErrInvalidGoalAttributes
	}
	allowed := map[string]bool{"diff": true, "const": true, "genre": true, "ver": true}
	for k := range attrs {
		if !allowed[k] {
			return nil, nil, ErrInvalidGoalAttributes
		}
	}

	result := &goalAttributeFilter{}
	if ids, ok, err := validateAndNormalizeAttributeIDs(attrs, "diff", func(id int) bool {
		_, exists := masters.DifficultyNamesByID[id]
		return exists
	}); err != nil {
		return nil, nil, err
	} else if ok {
		result.DifficultyIDs = ids
	}
	if v, ok := attrs["const"]; ok {
		var c struct {
			Min *float64 `json:"min"`
			Max *float64 `json:"max"`
		}
		if err := json.Unmarshal(v, &c); err != nil {
			return nil, nil, ErrInvalidGoalAttributes
		}

		minConst := info.ChartConstMin
		maxConst := info.ChartConstMax
		if c.Min != nil {
			if !isScale(*c.Min, 1) {
				return nil, nil, ErrInvalidGoalAttributes
			}
			minConst = *c.Min
		}
		if c.Max != nil {
			if !isScale(*c.Max, 1) {
				return nil, nil, ErrInvalidGoalAttributes
			}
			maxConst = *c.Max
		}
		if minConst < info.ChartConstMin || maxConst > info.ChartConstMax || minConst > maxConst {
			return nil, nil, ErrInvalidGoalAttributes
		}

		normalizedConst, err := json.Marshal(struct {
			Min float64 `json:"min"`
			Max float64 `json:"max"`
		}{Min: minConst, Max: maxConst})
		if err != nil {
			return nil, nil, ErrInvalidGoalAttributes
		}
		attrs["const"] = normalizedConst
		result.ConstMin = &minConst
		result.ConstMax = &maxConst
	}
	if ids, ok, err := validateAndNormalizeAttributeIDs(attrs, "genre", func(id int) bool {
		_, exists := masters.GenreNamesByID[id]
		return exists
	}); err != nil {
		return nil, nil, err
	} else if ok {
		result.GenreIDs = ids
	}
	if ids, ok, err := validateAndNormalizeAttributeIDs(attrs, "ver", func(id int) bool {
		_, exists := masters.VersionsByID[id]
		return exists
	}); err != nil {
		return nil, nil, err
	} else if ok {
		ranges := make([]repository.VersionRange, 0, len(ids))
		for _, id := range ids {
			version := masters.VersionsByID[id]
			ranges = append(ranges, repository.VersionRange{
				From: version.ReleasedAt,
				To:   findNextVersionReleasedAt(masters, version.ReleasedAt),
			})
		}
		result.VersionRanges = ranges
	}
	canon, err := json.Marshal(attrs)
	if err != nil {
		return nil, nil, ErrInvalidGoalAttributes
	}
	return canon, result, nil
}

func parseIntOrIntSlice(raw json.RawMessage) ([]int, error) {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, err
	}

	parseInt := func(value float64) (int, error) {
		if math.Trunc(value) != value {
			return 0, errors.New("value is not an integer")
		}
		parsed := int(value)
		if float64(parsed) != value {
			return 0, errors.New("integer value out of range")
		}
		return parsed, nil
	}

	var ids []int
	switch value := v.(type) {
	case float64:
		parsed, err := parseInt(value)
		if err != nil {
			return nil, err
		}
		ids = []int{parsed}
	case []any:
		if len(value) == 0 {
			return nil, errors.New("empty int slice")
		}
		ids = make([]int, 0, len(value))
		for _, item := range value {
			floatValue, ok := item.(float64)
			if !ok {
				return nil, errors.New("slice contains non-integer value")
			}
			parsed, err := parseInt(floatValue)
			if err != nil {
				return nil, err
			}
			ids = append(ids, parsed)
		}
	default:
		return nil, errors.New("unsupported type for int or int slice")
	}

	slices.Sort(ids)
	normalized := slices.Compact(ids)
	return normalized, nil
}

func validateAndNormalizeAttributeIDs(attrs map[string]json.RawMessage, key string, isValidID func(int) bool) ([]int, bool, error) {
	v, ok := attrs[key]
	if !ok {
		return nil, false, nil
	}

	ids, err := parseIntOrIntSlice(v)
	if err != nil {
		return nil, false, ErrInvalidGoalAttributes
	}
	for _, id := range ids {
		if !isValidID(id) {
			return nil, false, ErrInvalidGoalAttributes
		}
	}

	normalized, err := json.Marshal(normalizeIntOrSlice(ids))
	if err != nil {
		return nil, false, ErrInvalidGoalAttributes
	}
	attrs[key] = normalized
	return ids, true, nil
}

func normalizeIntOrSlice(ids []int) any {
	if len(ids) == 1 {
		return ids[0]
	}
	return ids
}

func findNextVersionReleasedAt(masters *domainmasterdata.GoalMasters, releasedAt time.Time) *time.Time {
	var next *time.Time
	for _, candidate := range masters.VersionsByID {
		if !candidate.ReleasedAt.After(releasedAt) {
			continue
		}
		if next == nil || candidate.ReleasedAt.Before(*next) {
			t := candidate.ReleasedAt
			next = &t
		}
	}
	return next
}

func hasOnlyKeys(m map[string]json.RawMessage, allowed ...string) bool {
	allow := make(map[string]struct{}, len(allowed))
	for _, k := range allowed {
		allow[k] = struct{}{}
	}
	for k := range m {
		if _, ok := allow[k]; !ok {
			return false
		}
	}
	return true
}

func validateAchievementParams(achievementType string, raw []byte) ([]byte, *goalAchievementParam, error) {
	if len(raw) == 0 {
		return nil, nil, ErrInvalidAchievementParam
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, nil, ErrInvalidAchievementParam
	}
	result := &goalAchievementParam{}
	scoreCountTypes := map[string]bool{"rank_count": true, "score_count": true}
	switch {
	case scoreCountTypes[achievementType]:
		if len(m) < 1 || len(m) > 2 || !hasOnlyKeys(m, "score", "count") {
			return nil, nil, ErrInvalidAchievementParam
		}
		score, ok, err := parseOptionalInt(m["score"])
		if err != nil || !ok || score < 0 || score > info.TheoreticalScore {
			return nil, nil, ErrInvalidAchievementParam
		}
		count, ok, err := parseOptionalInt(m["count"])
		if err != nil {
			return nil, nil, ErrInvalidAchievementParam
		}
		if ok && count < 1 {
			return nil, nil, ErrInvalidAchievementParam
		}
		result.Score = &score
		if ok {
			result.Count = &count
		}
	case achievementType == "avg_score":
		var score int
		if len(m) != 1 || !hasOnlyKeys(m, "score") || json.Unmarshal(m["score"], &score) != nil || score < 0 || score > info.TheoreticalScore {
			return nil, nil, ErrInvalidAchievementParam
		}
		result.Score = &score
	case achievementType == "hardlamp_count" || achievementType == "combolamp_count":
		var lamp string
		if len(m) < 1 || len(m) > 2 || !hasOnlyKeys(m, "lamp", "count") || json.Unmarshal(m["lamp"], &lamp) != nil {
			return nil, nil, ErrInvalidAchievementParam
		}
		count, ok, err := parseOptionalInt(m["count"])
		if err != nil {
			return nil, nil, ErrInvalidAchievementParam
		}
		if ok && count < 1 {
			return nil, nil, ErrInvalidAchievementParam
		}
		if achievementType == "hardlamp_count" {
			if _, ok := info.HardLampAbbrevToName[lamp]; !ok {
				return nil, nil, ErrInvalidAchievementParam
			}
		} else if _, ok := info.ComboLampAbbrevToName[lamp]; !ok {
			return nil, nil, ErrInvalidAchievementParam
		}
		if ok {
			result.Count = &count
		}
	case achievementType == "total_score":
		if len(m) > 1 || !hasOnlyKeys(m, "total") {
			return nil, nil, ErrInvalidAchievementParam
		}
		total, ok, err := parseOptionalInt64(m["total"])
		if err != nil {
			return nil, nil, ErrInvalidAchievementParam
		}
		if ok {
			if total < 0 {
				return nil, nil, ErrInvalidAchievementParam
			}
			totalFloat := float64(total)
			result.Total = &totalFloat
		}
	case achievementType == "overpower_value":
		if len(m) > 1 || !hasOnlyKeys(m, "total") {
			return nil, nil, ErrInvalidAchievementParam
		}
		total, ok, err := parseOptionalFloat64(m["total"])
		if err != nil {
			return nil, nil, ErrInvalidAchievementParam
		}
		if ok {
			if total < 0 || !isScale(total, 3) {
				return nil, nil, ErrInvalidAchievementParam
			}
			result.Total = &total
		}
	case achievementType == "overpower_percent":
		var total float64
		if len(m) != 1 || json.Unmarshal(m["total"], &total) != nil || total < 0 || total > 100 || !isScale(total, 3) {
			return nil, nil, ErrInvalidAchievementParam
		}
		result.Total = &total
	default:
		return nil, nil, ErrInvalidAchievementType
	}
	b, err := json.Marshal(m)
	if err != nil {
		return nil, nil, ErrInvalidAchievementParam
	}
	return b, result, nil
}

func (u *goalUsecase) validateDynamicUpperBound(ctx context.Context, achievementType string, attrs *goalAttributeFilter, params *goalAchievementParam) error {
	filter := repository.GoalTargetFilter{
		DifficultyIDs: attrs.DifficultyIDs,
		GenreIDs:      attrs.GenreIDs,
		VersionRanges: attrs.VersionRanges,
		ConstMin:      attrs.ConstMin,
		ConstMax:      attrs.ConstMax,
	}
	stats, err := u.goalRepo.GetTargetStats(ctx, u.db, filter)
	if err != nil {
		return err
	}

	switch achievementType {
	case "rank_count", "score_count", "hardlamp_count", "combolamp_count":
		if params.Count != nil && *params.Count > stats.ChartCount {
			slog.Info("goal validation failed", "reason", "count_over_dynamic_max", "achievement_type", achievementType, "input", *params.Count, "max", stats.ChartCount)
			return ErrInvalidAchievementParam
		}
	case "total_score":
		if params.Total != nil {
			maxTotal := float64(stats.ChartCount) * float64(info.TheoreticalScore)
			if *params.Total > maxTotal {
				slog.Info("goal validation failed", "reason", "total_score_over_dynamic_max", "input", *params.Total, "max", maxTotal)
				return ErrInvalidAchievementParam
			}
		}
	case "overpower_value":
		if params.Total != nil {
			maxTotal := info.CalcTheoreticalOverpowerTotal(stats.TotalChartConst, stats.ChartCount)
			if *params.Total > maxTotal {
				slog.Info("goal validation failed", "reason", "overpower_value_over_dynamic_max", "input", *params.Total, "max", maxTotal)
				return ErrInvalidAchievementParam
			}
		}
	case "overpower_percent":
		// 割合(0-100)で扱うため動的上限は不要
	}

	return nil
}

func isScale(v float64, scale int) bool {
	f := math.Pow10(scale)
	return math.Abs(v*f-math.Round(v*f)) < 1e-9
}

func parseOptionalInt(raw json.RawMessage) (int, bool, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return 0, false, nil
	}
	var value int
	if err := json.Unmarshal(raw, &value); err != nil {
		return 0, false, err
	}
	return value, true, nil
}

func parseOptionalInt64(raw json.RawMessage) (int64, bool, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return 0, false, nil
	}
	var value int64
	if err := json.Unmarshal(raw, &value); err != nil {
		return 0, false, err
	}
	return value, true, nil
}

func parseOptionalFloat64(raw json.RawMessage) (float64, bool, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return 0, false, nil
	}
	var value float64
	if err := json.Unmarshal(raw, &value); err != nil {
		return 0, false, err
	}
	return value, true, nil
}

func hasControlCharacter(value string) bool {
	for _, r := range value {
		if unicode.IsControl(r) {
			return true
		}
	}
	return false
}

func (u *goalUsecase) toOutputs(goals []*entity.Goal) ([]*GoalOutput, error) {
	masters := u.masterProvider.GoalMasters()
	if masters == nil {
		return nil, ErrInternalError
	}
	outs := make([]*GoalOutput, 0, len(goals))
	for _, g := range goals {
		typeCode := masters.AchievementTypesByID[g.AchievementTypeID]
		if typeCode == "" {
			slog.Error("achievement type code not found in master cache", "goal_id", g.ID, "achievement_type_id", g.AchievementTypeID)
			return nil, ErrInternalError
		}
		var p map[string]any
		if err := json.Unmarshal(g.AchievementParams, &p); err != nil {
			return nil, fmt.Errorf("failed to decode achievement params: %w", err)
		}
		var a map[string]any
		if err := json.Unmarshal(g.Attributes, &a); err != nil {
			return nil, fmt.Errorf("failed to decode attributes: %w", err)
		}
		outs = append(outs, &GoalOutput{ID: g.ID, Title: g.Title, AchievementType: typeCode, AchievementParams: p, Attributes: a, Invert: g.Invert, CreatedAt: g.CreatedAt.Format("2006-01-02T15:04:05Z07:00")})
	}
	return outs, nil
}
