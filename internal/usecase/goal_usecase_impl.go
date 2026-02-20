package usecase

import (
	"context"
	"database/sql"
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
	"github.com/chunisupport/chunisupport-api/internal/domain/service"
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
	songRepo       repository.SongRepository
	masterProvider repository.GoalMasterProvider
}

func NewGoalUsecase(db repository.Executor, tm TransactionManager, goalRepo repository.GoalRepository, songRepo repository.SongRepository, masterProvider repository.GoalMasterProvider) GoalUsecase {
	return &goalUsecase{db: db, tm: tm, goalRepo: goalRepo, songRepo: songRepo, masterProvider: masterProvider}
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
		if errors.Is(err, sql.ErrNoRows) {
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
		if errors.Is(err, sql.ErrNoRows) {
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

type validatedGoalAttributes struct {
	Diff     *int
	Genre    *int
	Version  *int
	ConstMin float64
	ConstMax float64
	HasConst bool
}

func (u *goalUsecase) Delete(ctx context.Context, userID int, id uint32) error {
	err := u.goalRepo.DeleteByIDAndUserID(ctx, u.db, id, userID)
	if errors.Is(err, sql.ErrNoRows) {
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

func (u *goalUsecase) validateInput(ctx context.Context, input *GoalInput) (*validatedGoalInput, error) {
	if input == nil {
		return nil, ErrInvalidGoalInput
	}
	title := strings.TrimSpace(input.Title)
	if title == "" || len([]rune(title)) > 30 || hasInvalidGoalTitleRune(title) {
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
	attrsRaw, attrs, err := validateAttributes(input.Attributes, masters)
	if err != nil {
		return nil, err
	}
	paramsRaw, err := validateAchievementParams(input.AchievementType, input.AchievementParams)
	if err != nil {
		return nil, err
	}
	if err := u.validateDynamicUpperBounds(ctx, input.AchievementType, paramsRaw, attrs, masters); err != nil {
		return nil, err
	}
	return &validatedGoalInput{Title: title, AchievementTypeID: item.ID, AchievementParams: paramsRaw, Attributes: attrsRaw, Invert: input.Invert}, nil
}

func validateAttributes(raw []byte, masters *domainmasterdata.GoalMasters) ([]byte, *validatedGoalAttributes, error) {
	var attrs map[string]json.RawMessage
	validated := &validatedGoalAttributes{}
	if len(raw) == 0 {
		return []byte("{}"), validated, nil
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
	if v, ok := attrs["diff"]; ok {
		var diff int
		if err := json.Unmarshal(v, &diff); err != nil {
			return nil, nil, ErrInvalidGoalAttributes
		}
		if _, exists := masters.DifficultyNamesByID[diff]; !exists {
			return nil, nil, ErrInvalidGoalAttributes
		}
		validated.Diff = &diff
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
		validated.ConstMin = minConst
		validated.ConstMax = maxConst
		validated.HasConst = true
	}
	if v, ok := attrs["genre"]; ok {
		var id int
		if err := json.Unmarshal(v, &id); err != nil {
			return nil, nil, ErrInvalidGoalAttributes
		}
		if _, exists := masters.GenreNamesByID[id]; !exists {
			return nil, nil, ErrInvalidGoalAttributes
		}
		validated.Genre = &id
	}
	if v, ok := attrs["ver"]; ok {
		var id int
		if err := json.Unmarshal(v, &id); err != nil {
			return nil, nil, ErrInvalidGoalAttributes
		}
		if _, exists := masters.VersionsByID[id]; !exists {
			return nil, nil, ErrInvalidGoalAttributes
		}
		validated.Version = &id
	}
	canon, err := json.Marshal(attrs)
	if err != nil {
		return nil, nil, ErrInvalidGoalAttributes
	}
	return canon, validated, nil
}

func validateAchievementParams(achievementType string, raw []byte) ([]byte, error) {
	if len(raw) == 0 {
		return nil, ErrInvalidAchievementParam
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, ErrInvalidAchievementParam
	}
	scoreCountTypes := map[string]bool{"rank_count": true, "score_count": true}
	switch {
	case scoreCountTypes[achievementType]:
		if len(m) != 2 {
			return nil, ErrInvalidAchievementParam
		}
		var score, count int
		if err := json.Unmarshal(m["score"], &score); err != nil || score < 0 || score > 1010000 {
			return nil, ErrInvalidAchievementParam
		}
		if err := json.Unmarshal(m["count"], &count); err != nil || count < 1 {
			return nil, ErrInvalidAchievementParam
		}
	case achievementType == "avg_score":
		var score int
		if len(m) != 1 || json.Unmarshal(m["score"], &score) != nil || score < 0 || score > 1010000 {
			return nil, ErrInvalidAchievementParam
		}
	case achievementType == "hardlamp_count" || achievementType == "combolamp_count":
		var lamp string
		var count int
		if len(m) != 2 || json.Unmarshal(m["lamp"], &lamp) != nil || json.Unmarshal(m["count"], &count) != nil || count < 1 {
			return nil, ErrInvalidAchievementParam
		}
		if achievementType == "hardlamp_count" {
			if _, ok := info.HardLampAbbrevToName[lamp]; !ok {
				return nil, ErrInvalidAchievementParam
			}
		} else if _, ok := info.ComboLampAbbrevToName[lamp]; !ok {
			return nil, ErrInvalidAchievementParam
		}
	case achievementType == "total_score":
		var total int64
		if len(m) != 1 || json.Unmarshal(m["total"], &total) != nil || total < 0 {
			return nil, ErrInvalidAchievementParam
		}
	case achievementType == "overpower_value":
		var total float64
		if len(m) != 1 || json.Unmarshal(m["total"], &total) != nil || total < 0 || !isScale(total, 3) {
			return nil, ErrInvalidAchievementParam
		}
	case achievementType == "overpower_percent":
		var total float64
		if len(m) != 1 || json.Unmarshal(m["total"], &total) != nil || total < 0 || total > 100 || !isScale(total, 3) {
			return nil, ErrInvalidAchievementParam
		}
	default:
		return nil, ErrInvalidAchievementType
	}
	b, err := json.Marshal(m)
	if err != nil {
		return nil, ErrInvalidAchievementParam
	}
	return b, nil
}

func (u *goalUsecase) validateDynamicUpperBounds(ctx context.Context, achievementType string, paramsRaw []byte, attrs *validatedGoalAttributes, masters *domainmasterdata.GoalMasters) error {
	if u.songRepo == nil {
		return ErrInternalError
	}
	targets, err := u.collectTargetCharts(ctx, attrs, masters)
	if err != nil {
		return err
	}
	maxCount := len(targets)
	maxTotalScore := int64(maxCount) * 1010000
	maxOPValue := calcMaxOverpowerValue(targets)

	var m map[string]json.RawMessage
	if err := json.Unmarshal(paramsRaw, &m); err != nil {
		return ErrInvalidAchievementParam
	}

	switch achievementType {
	case "score_count", "rank_count", "hardlamp_count", "combolamp_count":
		var count int
		if err := json.Unmarshal(m["count"], &count); err != nil {
			return ErrInvalidAchievementParam
		}
		if count > maxCount {
			return ErrInvalidAchievementParam
		}
	case "total_score":
		var total int64
		if err := json.Unmarshal(m["total"], &total); err != nil {
			return ErrInvalidAchievementParam
		}
		if total > maxTotalScore {
			return ErrInvalidAchievementParam
		}
	case "overpower_value":
		var total float64
		if err := json.Unmarshal(m["total"], &total); err != nil {
			return ErrInvalidAchievementParam
		}
		if total > maxOPValue {
			return ErrInvalidAchievementParam
		}
	}

	return nil
}

func (u *goalUsecase) collectTargetCharts(ctx context.Context, attrs *validatedGoalAttributes, masters *domainmasterdata.GoalMasters) ([]*entity.Chart, error) {
	songs, err := u.songRepo.FindAllExcludingWorldsend(ctx, u.db, false)
	if err != nil {
		return nil, ErrInternalError
	}
	targets := make([]*entity.Chart, 0)
	for _, song := range songs {
		if !matchGoalSong(song, attrs, masters) {
			continue
		}
		if attrs.Diff != nil {
			for _, chart := range song.Charts {
				if chart.DifficultyID == *attrs.Diff {
					targets = append(targets, chart)
				}
			}
			continue
		}
		if attrs.HasConst {
			for _, chart := range song.Charts {
				constValue := float64(chart.Const)
				if constValue >= attrs.ConstMin && constValue <= attrs.ConstMax {
					targets = append(targets, chart)
				}
			}
			continue
		}
		maxChart := maxConstChart(song.Charts)
		if maxChart != nil {
			targets = append(targets, maxChart)
		}
	}
	return targets, nil
}

func matchGoalSong(song *entity.Song, attrs *validatedGoalAttributes, masters *domainmasterdata.GoalMasters) bool {
	if attrs.Genre != nil {
		if song.GenreID == nil || *song.GenreID != *attrs.Genre {
			return false
		}
	}
	if attrs.Version != nil {
		versionID, ok := versionIDByReleasedAt(song.ReleasedAt, masters.VersionsByID)
		if !ok || versionID != *attrs.Version {
			return false
		}
	}
	return true
}

func versionIDByReleasedAt(releasedAt *time.Time, versionsByID map[int]domainmasterdata.Version) (int, bool) {
	if releasedAt == nil {
		return 0, false
	}
	versions := make([]domainmasterdata.Version, 0, len(versionsByID))
	for _, v := range versionsByID {
		versions = append(versions, v)
	}
	slices.SortFunc(versions, func(a, b domainmasterdata.Version) int {
		return a.ReleasedAt.Compare(b.ReleasedAt)
	})
	latestID := 0
	found := false
	for _, v := range versions {
		if !releasedAt.Before(v.ReleasedAt) {
			latestID = int(v.ID)
			found = true
		}
	}
	return latestID, found
}

func maxConstChart(charts []*entity.Chart) *entity.Chart {
	if len(charts) == 0 {
		return nil
	}
	maxChart := charts[0]
	for _, chart := range charts[1:] {
		if float64(chart.Const) > float64(maxChart.Const) {
			maxChart = chart
		}
	}
	return maxChart
}

func calcMaxOverpowerValue(charts []*entity.Chart) float64 {
	total := 0.0
	for _, chart := range charts {
		total += service.CalcSongMaxOP(float64(chart.Const))
	}
	return total
}

func hasInvalidGoalTitleRune(title string) bool {
	for _, r := range title {
		if unicode.IsControl(r) {
			return true
		}
		switch r {
		case '\u200B', '\u200C', '\u200D', '\uFEFF':
			return true
		}
	}
	return false
}

func isScale(v float64, scale int) bool {
	f := math.Pow10(scale)
	return math.Abs(v*f-math.Round(v*f)) < 1e-9
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
