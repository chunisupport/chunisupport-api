package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/service"
	api_internal "github.com/chunisupport/chunisupport-api/internal/usecase/playerdataresult"
)

const (
	maxPlayerDataChangeDetails          = 100
	playerDataMetricDiffScale           = 10_000
	playerDataOverpowerPercentDiffScale = 100_000
)

func computeCourseRecordChanges(ctx context.Context, before map[int]repository.CourseRecordState, after []repository.CourseRecordForUpsert, masters *playerDataMaster, lookup lampNameLookup) []api_internal.PlayerDataRecordChange {
	byID := make(map[int]*entity.Course, len(masters.courses))
	for _, course := range masters.courses {
		byID[course.ID] = course
	}
	result := make([]api_internal.PlayerDataRecordChange, 0, len(after))
	for _, record := range after {
		old, exists := before[record.CourseID]
		if exists && old.Score == record.State.Score && old.IsClear == record.State.IsClear && old.ComboLampID == record.State.ComboLampID {
			continue
		}
		course := byID[record.CourseID]
		idx := fmt.Sprint(record.CourseID)
		class := ""
		if course != nil {
			idx = course.OfficialIdx
			if course.CourseClass != nil {
				class = course.CourseClass.Name
			}
		}
		clearAfter := record.State.IsClear
		change := api_internal.PlayerDataRecordChange{RecordType: "course", ChangeType: "new", Idx: idx, CourseClass: class, After: api_internal.PlayerDataRecordState{Score: record.State.Score, ComboLamp: lookup.comboLampName(ctx, record.State.ComboLampID), IsClear: &clearAfter}}
		if exists {
			clearBefore := old.IsClear
			change.ChangeType = "updated"
			change.Before = &api_internal.PlayerDataRecordState{Score: old.Score, ComboLamp: lookup.comboLampName(ctx, old.ComboLampID), IsClear: &clearBefore}
		}
		result = append(result, change)
	}
	return result
}

func buildPlayerDataStatisticsDiff(before service.PlayerRecordStatisticsSnapshot, after service.PlayerRecordStatisticsSnapshot) api_internal.PlayerDataStatistics {
	statistics := api_internal.PlayerDataStatistics{
		Overall:      buildPlayerDataStatisticsGroupDiff(before.Overall, after.Overall),
		ByDifficulty: make(map[string]api_internal.PlayerDataStatisticsGroup, len(service.PlayerRecordStatisticsGroupNames())),
	}
	for _, difficulty := range service.PlayerRecordStatisticsGroupNames() {
		statistics.ByDifficulty[difficulty] = buildPlayerDataStatisticsGroupDiff(before.ByDifficulty[difficulty], after.ByDifficulty[difficulty])
	}
	return statistics
}

// buildPlayerDataFloat64Diff は登録前後の値が揃う場合だけ小数差分を計算します。
func buildPlayerDataFloat64Diff(before *float64, after *float64) api_internal.PlayerDataFloat64Diff {
	return buildPlayerDataFloat64DiffWithScale(before, after, playerDataMetricDiffScale)
}

// buildPlayerDataOverpowerPercentDiff は更新前OP値を今回と同じ分母で割合へ変換し、OP%のポイント差を計算します。
func buildPlayerDataOverpowerPercentDiff(beforeValue *float64, afterPercent *float64, maxOverpowerTotal float64) api_internal.PlayerDataFloat64Diff {
	var beforePercent *float64
	if beforeValue != nil {
		beforePercentValue := service.CalcOverpowerPercent(*beforeValue, maxOverpowerTotal)
		beforePercent = &beforePercentValue
	}
	return buildPlayerDataFloat64DiffWithScale(beforePercent, afterPercent, playerDataOverpowerPercentDiffScale)
}

// buildPlayerDataFloat64DiffWithScale は指定精度で登録前後の小数差分を計算します。
func buildPlayerDataFloat64DiffWithScale(before *float64, after *float64, scale float64) api_internal.PlayerDataFloat64Diff {
	diff := api_internal.PlayerDataFloat64Diff{
		Before: cloneFloat64Pointer(before),
		After:  cloneFloat64Pointer(after),
	}
	if before != nil && after != nil {
		delta := math.Round((*after-*before)*scale) / scale
		diff.Delta = &delta
	}
	return diff
}

// cloneFloat64Pointer は呼び出し元の値と共有しないfloat64ポインタを返します。
func cloneFloat64Pointer(value *float64) *float64 {
	if value == nil {
		return nil
	}
	return new(*value)
}

func buildPlayerDataStatisticsGroupDiff(before service.PlayerRecordStatistics, after service.PlayerRecordStatistics) api_internal.PlayerDataStatisticsGroup {
	intDiff := func(beforeValue int, afterValue int) api_internal.PlayerDataIntDiff {
		return api_internal.PlayerDataIntDiff{Before: beforeValue, After: afterValue, Delta: afterValue - beforeValue}
	}
	return api_internal.PlayerDataStatisticsGroup{
		TotalHighScore: api_internal.PlayerDataInt64Diff{Before: before.TotalHighScore, After: after.TotalHighScore, Delta: after.TotalHighScore - before.TotalHighScore},
		RecordStatistics: api_internal.PlayerDataRecordStatisticsDiff{
			AJ: intDiff(before.Achievements.AJ, after.Achievements.AJ), FC: intDiff(before.Achievements.FC, after.Achievements.FC),
			CLR: intDiff(before.Achievements.CLR, after.Achievements.CLR), FCH: intDiff(before.Achievements.FCH, after.Achievements.FCH),
			MAX: intDiff(before.Achievements.MAX, after.Achievements.MAX), SSSPlus: intDiff(before.Achievements.SSSPlus, after.Achievements.SSSPlus),
			SSS: intDiff(before.Achievements.SSS, after.Achievements.SSS), SSPlus: intDiff(before.Achievements.SSPlus, after.Achievements.SSPlus),
			SS: intDiff(before.Achievements.SS, after.Achievements.SS), SPlus: intDiff(before.Achievements.SPlus, after.Achievements.SPlus),
			S: intDiff(before.Achievements.S, after.Achievements.S),
		},
	}
}

func computeFullRecordChanges(ctx context.Context, before map[int]repository.PlayerRecordState, after []repository.PlayerRecordForUpsert, masters *playerDataMaster) []playerDataRecordChange[repository.PlayerRecordState] {
	return computeRecordChanges(
		ctx,
		before,
		after,
		masters,
		func(record repository.PlayerRecordForUpsert) int { return record.ChartID },
		func(record repository.PlayerRecordForUpsert) repository.PlayerRecordState { return record.State },
		func(ctx context.Context, record repository.PlayerRecordForUpsert, lookup recordDisplayLookup) (string, string) {
			return fullRecordDisplayKeys(ctx, record.ChartID, masters, lookup)
		},
		playerRecordMeaningfullyChanged,
		"standard",
	)
}

func computeWorldsendRecordChanges(ctx context.Context, before map[int]repository.WorldsendRecordState, after []repository.WorldsendRecordForUpsert, masters *playerDataMaster) []playerDataRecordChange[repository.WorldsendRecordState] {
	return computeRecordChanges(
		ctx,
		before,
		after,
		masters,
		func(record repository.WorldsendRecordForUpsert) int { return record.ChartID },
		func(record repository.WorldsendRecordForUpsert) repository.WorldsendRecordState { return record.State },
		func(ctx context.Context, record repository.WorldsendRecordForUpsert, lookup recordDisplayLookup) (string, string) {
			return worldsendRecordDisplayKeys(record.ChartID, lookup)
		},
		worldsendRecordMeaningfullyChanged,
		"worldsend",
	)
}

type playerDataRecordChange[State any] struct {
	RecordType string
	ChangeType string
	Idx        string
	Diff       string
	Before     *State
	After      State
}

// computeRecordChanges は通常譜面とWORLD'S ENDで共通する差分生成の流れを集約します。
// 差分計算そのものをAPI DTOに固定しないため、レスポンス変換は呼び出し側で行います。
func computeRecordChanges[State any, Record any](
	ctx context.Context,
	before map[int]State,
	after []Record,
	masters *playerDataMaster,
	getChartID func(Record) int,
	getState func(Record) State,
	getDisplayKeys func(context.Context, Record, recordDisplayLookup) (string, string),
	meaningfullyChanged func(State, State) bool,
	recordType string,
) []playerDataRecordChange[State] {
	lookup := newRecordDisplayLookup(masters)
	changes := make([]playerDataRecordChange[State], 0, len(after))
	for _, record := range after {
		chartID := getChartID(record)
		afterState := getState(record)
		idx, diff := getDisplayKeys(ctx, record, lookup)
		beforeState, exists := before[chartID]
		if !exists {
			changes = append(changes, playerDataRecordChange[State]{RecordType: recordType, ChangeType: "new", Idx: idx, Diff: diff, After: afterState})
			continue
		}
		if !meaningfullyChanged(beforeState, afterState) {
			continue
		}
		beforeCopy := beforeState
		changes = append(changes, playerDataRecordChange[State]{RecordType: recordType, ChangeType: "updated", Idx: idx, Diff: diff, Before: &beforeCopy, After: afterState})
	}
	return changes
}

type recordDisplayLookup struct {
	songsByID          map[int]string
	difficultiesByID   map[int]string
	worldsendByChartID map[int]entity.PlayerDataWorldsendChart
}

func newRecordDisplayLookup(masters *playerDataMaster) recordDisplayLookup {
	lookup := recordDisplayLookup{
		songsByID:          make(map[int]string, len(masters.songs)),
		difficultiesByID:   make(map[int]string, len(masters.Difficulties)),
		worldsendByChartID: make(map[int]entity.PlayerDataWorldsendChart, len(masters.worldsendBySongID)),
	}
	for _, song := range masters.songs {
		lookup.songsByID[song.ID] = song.OfficialIdx
	}
	for _, difficulty := range masters.Difficulties {
		lookup.difficultiesByID[difficulty.ID] = difficulty.Name
	}
	for _, chart := range masters.worldsendBySongID {
		lookup.worldsendByChartID[chart.ID] = chart
	}
	return lookup
}

func fullRecordDisplayKeys(ctx context.Context, chartID int, masters *playerDataMaster, lookup recordDisplayLookup) (string, string) {
	chart, ok := masters.chartsByID[chartID]
	if !ok {
		return fmt.Sprintf("%d", chartID), ""
	}
	idx, ok := lookup.songsByID[chart.SongID]
	if !ok {
		idx = fmt.Sprintf("%d", chart.SongID)
	}
	diff, ok := lookup.difficultiesByID[chart.DifficultyID]
	if !ok {
		slog.WarnContext(ctx, "difficulty not found for player data change display", "difficulty_id", chart.DifficultyID, "chart_id", chartID)
		diff = fmt.Sprintf("%d", chart.DifficultyID)
	}
	return idx, diff
}

func worldsendRecordDisplayKeys(chartID int, lookup recordDisplayLookup) (string, string) {
	chart, ok := lookup.worldsendByChartID[chartID]
	if !ok {
		return fmt.Sprintf("%d", chartID), "WE"
	}
	idx, ok := lookup.songsByID[chart.SongID]
	if !ok {
		idx = fmt.Sprintf("%d", chart.SongID)
	}
	return idx, "WE"
}

func playerRecordChangesDTO(ctx context.Context, changes []playerDataRecordChange[repository.PlayerRecordState], lookup lampNameLookup) []api_internal.PlayerDataRecordChange {
	return recordChangesDTO(changes, func(state repository.PlayerRecordState) api_internal.PlayerDataRecordState {
		return playerRecordStateDTO(ctx, state, lookup)
	})
}

func worldsendRecordChangesDTO(ctx context.Context, changes []playerDataRecordChange[repository.WorldsendRecordState], lookup lampNameLookup) []api_internal.PlayerDataRecordChange {
	return recordChangesDTO(changes, func(state repository.WorldsendRecordState) api_internal.PlayerDataRecordState {
		return worldsendRecordStateDTO(ctx, state, lookup)
	})
}

func recordChangesDTO[State any](changes []playerDataRecordChange[State], stateDTO func(State) api_internal.PlayerDataRecordState) []api_internal.PlayerDataRecordChange {
	dtos := make([]api_internal.PlayerDataRecordChange, 0, len(changes))
	for _, change := range changes {
		dto := api_internal.PlayerDataRecordChange{
			RecordType: change.RecordType,
			ChangeType: change.ChangeType,
			Idx:        change.Idx,
			Diff:       change.Diff,
			After:      stateDTO(change.After),
		}
		if change.Before != nil {
			before := stateDTO(*change.Before)
			dto.Before = &before
		}
		dtos = append(dtos, dto)
	}
	return dtos
}

func sortAndLimitRecordChanges(changes []api_internal.PlayerDataRecordChange) []api_internal.PlayerDataRecordChange {
	slices.SortStableFunc(changes, comparePlayerDataRecordChange)
	if len(changes) <= maxPlayerDataChangeDetails {
		return changes
	}
	return changes[:maxPlayerDataChangeDetails]
}

func comparePlayerDataRecordChange(a, b api_internal.PlayerDataRecordChange) int {
	aIdx, aOK := parseChangeIdx(a.Idx)
	bIdx, bOK := parseChangeIdx(b.Idx)
	if aOK != bOK {
		if aOK {
			return -1
		}
		return 1
	}
	if aOK && aIdx != bIdx {
		return aIdx - bIdx
	}
	if a.Idx != b.Idx {
		return strings.Compare(a.Idx, b.Idx)
	}
	if a.RecordType != b.RecordType {
		return strings.Compare(a.RecordType, b.RecordType)
	}
	if a.Diff != b.Diff {
		return strings.Compare(a.Diff, b.Diff)
	}
	return strings.Compare(a.ChangeType, b.ChangeType)
}

func parseChangeIdx(idx string) (int, bool) {
	value, err := strconv.Atoi(idx)
	return value, err == nil
}

type lampNameLookup struct {
	clearLamps map[int]string
	comboLamps map[int]string
	fullChains map[int]string
}

func newLampNameLookup(masters *playerDataMaster) lampNameLookup {
	lookup := lampNameLookup{
		clearLamps: make(map[int]string, len(masters.ClearLamps)),
		comboLamps: make(map[int]string, len(masters.ComboLamps)),
		fullChains: make(map[int]string, len(masters.FullChains)),
	}
	for _, item := range masters.ClearLamps {
		lookup.clearLamps[item.ID] = item.Name
	}
	for _, item := range masters.ComboLamps {
		lookup.comboLamps[item.ID] = item.Name
	}
	for _, item := range masters.FullChains {
		lookup.fullChains[item.ID] = item.Name
	}
	return lookup
}

func (l lampNameLookup) clearLampName(ctx context.Context, id int) *string {
	return playerDataLampNamePtr(ctx, l.clearLamps[id], id, "clear_lamp")
}

func (l lampNameLookup) comboLampName(ctx context.Context, id int) *string {
	return playerDataLampNamePtr(ctx, l.comboLamps[id], id, "combo_lamp")
}

func (l lampNameLookup) fullChainName(ctx context.Context, id int) *string {
	return playerDataLampNamePtr(ctx, l.fullChains[id], id, "full_chain")
}

// playerDataLampNamePtr は none 相当およびマスタ未解決を null として返します。
func playerDataLampNamePtr(ctx context.Context, name string, id int, resource string) *string {
	if name == "" {
		if id != 0 {
			slog.WarnContext(ctx, "lamp name not found for player data change display", "resource", resource, "id", id)
		}
		return nil
	}
	if strings.EqualFold(name, "none") {
		return nil
	}
	return &name
}

func playerRecordStateDTO(ctx context.Context, state repository.PlayerRecordState, lookup lampNameLookup) api_internal.PlayerDataRecordState {
	return api_internal.PlayerDataRecordState{
		Score:     state.Score,
		ClearLamp: lookup.clearLampName(ctx, state.ClearLampID),
		ComboLamp: lookup.comboLampName(ctx, state.ComboLampID),
		FullChain: lookup.fullChainName(ctx, state.FullChainID),
	}
}

func worldsendRecordStateDTO(ctx context.Context, state repository.WorldsendRecordState, lookup lampNameLookup) api_internal.PlayerDataRecordState {
	return api_internal.PlayerDataRecordState{
		Score:     state.Score,
		ClearLamp: lookup.clearLampName(ctx, state.ClearLampID),
		ComboLamp: lookup.comboLampName(ctx, state.ComboLampID),
		FullChain: lookup.fullChainName(ctx, state.FullChainID),
	}
}

// playerRecordMeaningfullyChanged は履歴と更新日時の対象になる成績4項目だけを比較します。
func playerRecordMeaningfullyChanged(before, after repository.PlayerRecordState) bool {
	return before.Score != after.Score ||
		before.ClearLampID != after.ClearLampID ||
		before.ComboLampID != after.ComboLampID ||
		before.FullChainID != after.FullChainID
}

// worldsendRecordMeaningfullyChanged は履歴と更新日時の対象になる成績4項目だけを比較します。
func worldsendRecordMeaningfullyChanged(before, after repository.WorldsendRecordState) bool {
	return before.Score != after.Score ||
		before.ClearLampID != after.ClearLampID ||
		before.ComboLampID != after.ComboLampID ||
		before.FullChainID != after.FullChainID
}
