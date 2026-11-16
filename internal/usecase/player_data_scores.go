package usecase

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/service"
	api_internal "github.com/chunisupport/chunisupport-api/internal/usecase/playerdataresult"
)

const (
	maxScoreValue = 1010000
	minScoreValue = 0
)

// applyScores はプレイヤーのスコア情報を更新します。
// 通常譜面とWORLD'S END譜面の新規・変更行のみを保存します。
func (us *playerDataUsecase) applyScores(ctx context.Context, tx repository.Executor, playerID int, scores PlayerDataScorePayload, masters *playerDataMaster, updatedAt time.Time, beforeStatistics service.PlayerRecordStatisticsSnapshot) (api_internal.PlayerDataCounts, []api_internal.SkippedRecord, []api_internal.PlayerDataRecordChange, api_internal.PlayerDataStatistics, calculatedOverpowerSummary, error) {
	counts, skipped, fullRecordsToUpsert := applyFullScores(playerID, scores.Standard, masters, updatedAt)
	if err := validateMainSlotRecordsResolved(scores.Standard, fullRecordsToUpsert, masters); err != nil {
		return counts, skipped, nil, api_internal.PlayerDataStatistics{}, calculatedOverpowerSummary{}, err
	}
	worldsendCounts, worldsendSkipped, worldsendRecordsToUpsert := applyWorldsendScores(playerID, scores.Worldsend, masters, updatedAt)
	courseCounts, courseSkipped, courseRecordsToUpsert := applyCourseScores(playerID, scores.Course, masters, updatedAt)
	counts.WorldsendRecordsUpserted = worldsendCounts.WorldsendRecordsUpserted
	counts.WorldsendRecordsSkipped = worldsendCounts.WorldsendRecordsSkipped
	skipped = append(skipped, worldsendSkipped...)
	counts.CourseRecordsUpserted = courseCounts.CourseRecordsUpserted
	counts.CourseRecordsSkipped = courseCounts.CourseRecordsSkipped
	skipped = append(skipped, courseSkipped...)

	fullRecordsToUpsert = normalizeFullRecordsForUpsert(fullRecordsToUpsert)
	worldsendRecordsToUpsert = normalizeWorldsendRecordsForUpsert(worldsendRecordsToUpsert)
	courseRecordsToUpsert = normalizeCourseRecordsForUpsert(courseRecordsToUpsert)
	if err := validateFullRecordSlots(fullRecordsToUpsert, masters); err != nil {
		return counts, skipped, nil, api_internal.PlayerDataStatistics{}, calculatedOverpowerSummary{}, err
	}

	fullBefore, err := us.playerDataRepo.FindPlayerRecordStatesByChartIDs(ctx, tx, playerID, collectFullChartIDs(fullRecordsToUpsert))
	if err != nil {
		return counts, skipped, nil, api_internal.PlayerDataStatistics{}, calculatedOverpowerSummary{}, err
	}
	worldsendBefore, err := us.playerDataRepo.FindWorldsendRecordStatesByChartIDs(ctx, tx, playerID, collectWorldsendChartIDs(worldsendRecordsToUpsert))
	if err != nil {
		return counts, skipped, nil, api_internal.PlayerDataStatistics{}, calculatedOverpowerSummary{}, err
	}
	rankedChartIDs, err := us.playerDataRepo.FindRankedPlayerRecordChartIDs(ctx, tx, playerID)
	if err != nil {
		return counts, skipped, nil, api_internal.PlayerDataStatistics{}, calculatedOverpowerSummary{}, err
	}
	noneSlotID, err := resolveSlotID(nil, masters)
	if err != nil {
		return counts, skipped, nil, api_internal.PlayerDataStatistics{}, calculatedOverpowerSummary{}, err
	}
	courseBefore := make(map[int]repository.CourseRecordState)
	if len(courseRecordsToUpsert) > 0 {
		if us.courseRepo == nil {
			return counts, skipped, nil, api_internal.PlayerDataStatistics{}, calculatedOverpowerSummary{}, errors.New("course repository is not initialized")
		}
		courseBefore, err = us.courseRepo.FindRecordStatesByCourseIDs(ctx, tx, playerID, collectCourseIDs(courseRecordsToUpsert))
		if err != nil {
			return counts, skipped, nil, api_internal.PlayerDataStatistics{}, calculatedOverpowerSummary{}, err
		}
	}

	// Registerが同一ユーザーの登録をユーザー行ロックで直列化するため、ここではロックを追加せず保存前状態から差分を求めます。
	fullRecordChanges := computeFullRecordChanges(ctx, fullBefore, fullRecordsToUpsert, masters)
	worldsendRecordChanges := computeWorldsendRecordChanges(ctx, worldsendBefore, worldsendRecordsToUpsert, masters)
	lampLookup := newLampNameLookup(masters)
	changes := make([]api_internal.PlayerDataRecordChange, 0, len(fullRecordChanges)+len(worldsendRecordChanges))
	changes = append(changes, playerRecordChangesDTO(ctx, fullRecordChanges, lampLookup)...)
	changes = append(changes, worldsendRecordChangesDTO(ctx, worldsendRecordChanges, lampLookup)...)
	courseChanges := computeCourseRecordChanges(ctx, courseBefore, courseRecordsToUpsert, masters, lampLookup)
	changes = append(changes, courseChanges...)
	changes = sortAndLimitRecordChanges(changes)
	counts.FullRecordsActuallyChanged = len(fullRecordChanges)
	counts.WorldsendRecordsActuallyChanged = len(worldsendRecordChanges)
	counts.CourseRecordsActuallyChanged = len(courseChanges)

	standardHistories, standardHistoryChartIDs := buildStandardHistories(playerID, fullBefore, fullRecordsToUpsert, masters)
	worldsendHistories, worldsendHistoryChartIDs := buildWorldsendHistories(playerID, worldsendBefore, worldsendRecordsToUpsert)
	if us.scoreHistoryRepo != nil {
		if err := us.scoreHistoryRepo.BulkInsertStandard(ctx, tx, standardHistories); err != nil {
			return counts, skipped, changes, api_internal.PlayerDataStatistics{}, calculatedOverpowerSummary{}, err
		}
		if err := us.scoreHistoryRepo.BulkInsertWorldsend(ctx, tx, worldsendHistories); err != nil {
			return counts, skipped, changes, api_internal.PlayerDataStatistics{}, calculatedOverpowerSummary{}, err
		}
	}

	writePlan := planPlayerDataWrites(playerID, noneSlotID, fullRecordsToUpsert, fullBefore, rankedChartIDs, worldsendRecordsToUpsert, worldsendBefore)
	if err := us.playerDataRepo.SavePlayerData(ctx, tx, writePlan); err != nil {
		return counts, skipped, changes, api_internal.PlayerDataStatistics{}, calculatedOverpowerSummary{}, err
	}
	if len(courseRecordsToUpsert) > 0 {
		if err := us.courseRepo.UpsertRecords(ctx, tx, courseRecordsToUpsert); err != nil {
			return counts, skipped, changes, api_internal.PlayerDataStatistics{}, calculatedOverpowerSummary{}, err
		}
	}
	if us.scoreHistoryRepo != nil {
		if err := us.scoreHistoryRepo.PruneStandardOverLimit(ctx, tx, playerID, standardHistoryChartIDs); err != nil {
			return counts, skipped, changes, api_internal.PlayerDataStatistics{}, calculatedOverpowerSummary{}, err
		}
		if err := us.scoreHistoryRepo.PruneWorldsendOverLimit(ctx, tx, playerID, worldsendHistoryChartIDs); err != nil {
			return counts, skipped, changes, api_internal.PlayerDataStatistics{}, calculatedOverpowerSummary{}, err
		}
	}

	overpowerTargetStats, err := us.playerDataRepo.GetOverpowerTargetStatsWithExecutor(ctx, tx, repository.OverpowerTargetFilter{
		ExcludeWorldsend: true,
		ExcludeDeleted:   true,
		PlayerID:         &playerID,
	})
	if err != nil {
		return counts, skipped, changes, api_internal.PlayerDataStatistics{}, calculatedOverpowerSummary{}, err
	}

	records, recErr := us.playerRecRepo.FindByPlayerID(ctx, tx, playerID)
	if recErr != nil {
		return counts, skipped, changes, api_internal.PlayerDataStatistics{}, calculatedOverpowerSummary{}, fmt.Errorf("failed to fetch player records for overpower calculation: %w", recErr)
	}
	worldsendRecords, worldsendRecErr := us.worldsendRecRepo.FindByPlayerID(ctx, tx, playerID)
	if worldsendRecErr != nil {
		return counts, skipped, changes, api_internal.PlayerDataStatistics{}, calculatedOverpowerSummary{}, fmt.Errorf("failed to fetch player worldsend records for statistics: %w", worldsendRecErr)
	}
	afterStatistics, statisticsErr := service.CalculatePlayerRecordStatistics(records, worldsendRecords)
	if statisticsErr != nil {
		return counts, skipped, changes, api_internal.PlayerDataStatistics{}, calculatedOverpowerSummary{}, fmt.Errorf("failed to aggregate player records after registration: %w", statisticsErr)
	}
	lockedSongs, lockedErr := us.listLockedSongsForOverpower(ctx, tx, playerID)
	if lockedErr != nil {
		return counts, skipped, changes, api_internal.PlayerDataStatistics{}, calculatedOverpowerSummary{}, fmt.Errorf("failed to fetch locked songs for overpower calculation: %w", lockedErr)
	}
	overpowerSummary, err := calculateOverpowerSummaryFromPlayerRecords(ctx, records, lockedSongs, overpowerTargetStats.MaxOverpowerTotal)
	if err != nil {
		return counts, skipped, changes, api_internal.PlayerDataStatistics{}, calculatedOverpowerSummary{}, fmt.Errorf("failed to aggregate overpower from player records: %w", err)
	}

	return counts, skipped, changes, buildPlayerDataStatisticsDiff(beforeStatistics, afterStatistics), overpowerSummary, nil
}

func applyCourseScores(playerID int, entries []PlayerDataCourseEntry, masters *playerDataMaster, updatedAt time.Time) (api_internal.PlayerDataCounts, []api_internal.SkippedRecord, []repository.CourseRecordForUpsert) {
	counts := api_internal.PlayerDataCounts{}
	skipped := make([]api_internal.SkippedRecord, 0)
	records := make([]repository.CourseRecordForUpsert, 0, len(entries))
	for _, entry := range entries {
		counts.CourseRecordsUpserted++
		idx := strings.TrimSpace(entry.Idx)
		course, ok := masters.courses[idx]
		if !ok {
			counts.CourseRecordsSkipped++
			skipped = append(skipped, api_internal.SkippedRecord{RecordType: "course", Reason: "failed to resolve course", Details: "idx=" + idx})
			continue
		}
		combo := entry.ComboLv
		comboID, err := resolveComboLampID(&combo, masters)
		if err != nil {
			counts.CourseRecordsSkipped++
			skipped = append(skipped, api_internal.SkippedRecord{RecordType: "course", Reason: "failed to resolve combo_lamp", Details: fmt.Sprintf("idx=%s, combo_lv=%d, error=%s", idx, combo, err)})
			continue
		}
		records = append(records, repository.CourseRecordForUpsert{PlayerID: playerID, CourseID: course.ID, State: repository.CourseRecordState{Score: entry.Score, IsClear: entry.IsClear, ComboLampID: comboID, UpdatedAt: updatedAt}})
	}
	return counts, skipped, records
}

func normalizeCourseRecordsForUpsert(records []repository.CourseRecordForUpsert) []repository.CourseRecordForUpsert {
	last := make(map[int]repository.CourseRecordForUpsert, len(records))
	order := make([]int, 0, len(records))
	for _, record := range records {
		if _, ok := last[record.CourseID]; !ok {
			order = append(order, record.CourseID)
		}
		last[record.CourseID] = record
	}
	result := make([]repository.CourseRecordForUpsert, 0, len(last))
	for _, id := range order {
		result = append(result, last[id])
	}
	return result
}

func collectCourseIDs(records []repository.CourseRecordForUpsert) []int {
	result := make([]int, 0, len(records))
	for _, r := range records {
		result = append(result, r.CourseID)
	}
	return result
}

func buildStandardHistories(
	playerID int,
	before map[int]repository.PlayerRecordState,
	after []repository.PlayerRecordForUpsert,
	masters *playerDataMaster,
) ([]repository.PlayerRecordHistory, []int) {
	rows := make([]repository.PlayerRecordHistory, 0, len(after))
	chartIDs := make([]int, 0, len(after))
	seenChartIDs := make(map[int]struct{}, len(after))
	for _, record := range after {
		beforeState, exists := before[record.ChartID]
		if !exists || !playerRecordMeaningfullyChanged(beforeState, record.State) {
			continue
		}
		chart, exists := masters.chartsByID[record.ChartID]
		if !exists || !entity.SupportsScoreHistory(masters.DifficultyNamesByID[chart.DifficultyID]) {
			continue
		}
		rows = append(rows, repository.PlayerRecordHistory{
			PlayerID: playerID,
			ChartID:  record.ChartID,
			State:    beforeState,
		})
		if _, exists := seenChartIDs[record.ChartID]; !exists {
			seenChartIDs[record.ChartID] = struct{}{}
			chartIDs = append(chartIDs, record.ChartID)
		}
	}
	return rows, chartIDs
}

func buildWorldsendHistories(
	playerID int,
	before map[int]repository.WorldsendRecordState,
	after []repository.WorldsendRecordForUpsert,
) ([]repository.PlayerWorldsendRecordHistory, []int) {
	rows := make([]repository.PlayerWorldsendRecordHistory, 0, len(after))
	chartIDs := make([]int, 0, len(after))
	seenChartIDs := make(map[int]struct{}, len(after))
	for _, record := range after {
		beforeState, exists := before[record.ChartID]
		if !exists || !worldsendRecordMeaningfullyChanged(beforeState, record.State) {
			continue
		}
		rows = append(rows, repository.PlayerWorldsendRecordHistory{
			PlayerID:         playerID,
			WorldsendChartID: record.ChartID,
			State:            beforeState,
		})
		if _, exists := seenChartIDs[record.ChartID]; !exists {
			seenChartIDs[record.ChartID] = struct{}{}
			chartIDs = append(chartIDs, record.ChartID)
		}
	}
	return rows, chartIDs
}

type resolvedLampIDs struct {
	clearLampID int
	comboLampID int
	fullChainID int
}

func applyFullScores(playerID int, entries []PlayerDataScoreEntry, masters *playerDataMaster, updatedAt time.Time) (api_internal.PlayerDataCounts, []api_internal.SkippedRecord, []repository.PlayerRecordForUpsert) {
	counts := api_internal.PlayerDataCounts{}
	skipped := make([]api_internal.SkippedRecord, 0, len(entries))
	fullRecordsToUpsert := make([]repository.PlayerRecordForUpsert, 0, len(entries))

	for _, entry := range entries {
		counts.FullRecordsUpserted++

		chart, song, _, err := resolveChart(entry, masters)
		if err != nil {
			counts.FullRecordsSkipped++
			skipped = append(skipped, api_internal.SkippedRecord{
				RecordType: "standard",
				Reason:     "failed to resolve chart",
				Details:    fmt.Sprintf("idx=%s, diff=%s, error=%s", entry.Idx, entry.Diff, err.Error()),
			})
			continue
		}

		if skippedRecord, ok := validateScoreRange("standard", entry, song); ok {
			counts.FullRecordsSkipped++
			skipped = append(skipped, skippedRecord)
			continue
		}

		lampIDs, skippedRecord := resolveCommonLampIDs("standard", entry, song, masters)
		if skippedRecord != nil {
			counts.FullRecordsSkipped++
			skipped = append(skipped, *skippedRecord)
			continue
		}

		slotID, err := resolveSlotID(entry.Slot, masters)
		if err != nil {
			counts.FullRecordsSkipped++
			skipped = append(skipped, newResolveSkippedRecord("standard", "slot", "slot", entry, song, optionalStringValue(entry.Slot), err))
			continue
		}

		fullRecordsToUpsert = append(fullRecordsToUpsert, repository.PlayerRecordForUpsert{
			PlayerID: playerID,
			ChartID:  chart.ID,
			State: repository.PlayerRecordState{
				Score:       entry.Score,
				ClearLampID: lampIDs.clearLampID,
				ComboLampID: lampIDs.comboLampID,
				FullChainID: lampIDs.fullChainID,
				SlotID:      slotID,
				SlotOrder:   entry.Order,
				UpdatedAt:   updatedAt,
			},
		})
	}

	return counts, skipped, fullRecordsToUpsert
}

func applyWorldsendScores(playerID int, entries []PlayerDataScoreEntry, masters *playerDataMaster, updatedAt time.Time) (api_internal.PlayerDataCounts, []api_internal.SkippedRecord, []repository.WorldsendRecordForUpsert) {
	counts := api_internal.PlayerDataCounts{}
	skipped := make([]api_internal.SkippedRecord, 0, len(entries))
	worldsendRecordsToUpsert := make([]repository.WorldsendRecordForUpsert, 0, len(entries))

	for _, entry := range entries {
		counts.WorldsendRecordsUpserted++

		chart, song, err := resolveWorldsendChart(entry, masters)
		if err != nil {
			counts.WorldsendRecordsSkipped++
			skipped = append(skipped, api_internal.SkippedRecord{
				RecordType: "worldsend",
				Reason:     "failed to resolve worldsend chart",
				Details:    fmt.Sprintf("idx=%s, error=%s", entry.Idx, err.Error()),
			})
			continue
		}

		if skippedRecord, ok := validateScoreRange("worldsend", entry, song); ok {
			counts.WorldsendRecordsSkipped++
			skipped = append(skipped, skippedRecord)
			continue
		}

		lampIDs, skippedRecord := resolveCommonLampIDs("worldsend", entry, song, masters)
		if skippedRecord != nil {
			counts.WorldsendRecordsSkipped++
			skipped = append(skipped, *skippedRecord)
			continue
		}

		worldsendRecordsToUpsert = append(worldsendRecordsToUpsert, repository.WorldsendRecordForUpsert{
			PlayerID: playerID,
			ChartID:  chart.ID,
			State: repository.WorldsendRecordState{
				Score:       entry.Score,
				ClearLampID: lampIDs.clearLampID,
				ComboLampID: lampIDs.comboLampID,
				FullChainID: lampIDs.fullChainID,
				UpdatedAt:   updatedAt,
			},
		})
	}

	return counts, skipped, worldsendRecordsToUpsert
}

func normalizeFullRecordsForUpsert(records []repository.PlayerRecordForUpsert) []repository.PlayerRecordForUpsert {
	positions := make(map[int]int, len(records))
	normalized := make([]repository.PlayerRecordForUpsert, 0, len(records))
	for _, record := range records {
		if pos, ok := positions[record.ChartID]; ok {
			normalized[pos] = record
			continue
		}
		positions[record.ChartID] = len(normalized)
		normalized = append(normalized, record)
	}
	return normalized
}

func normalizeWorldsendRecordsForUpsert(records []repository.WorldsendRecordForUpsert) []repository.WorldsendRecordForUpsert {
	positions := make(map[int]int, len(records))
	normalized := make([]repository.WorldsendRecordForUpsert, 0, len(records))
	for _, record := range records {
		if pos, ok := positions[record.ChartID]; ok {
			normalized[pos] = record
			continue
		}
		positions[record.ChartID] = len(normalized)
		normalized = append(normalized, record)
	}
	return normalized
}

func collectFullChartIDs(records []repository.PlayerRecordForUpsert) []int {
	ids := make([]int, 0, len(records))
	for _, record := range records {
		ids = append(ids, record.ChartID)
	}
	slices.Sort(ids)
	return ids
}

func collectWorldsendChartIDs(records []repository.WorldsendRecordForUpsert) []int {
	ids := make([]int, 0, len(records))
	for _, record := range records {
		ids = append(ids, record.ChartID)
	}
	slices.Sort(ids)
	return ids
}

func validateScoreRange(recordType string, entry PlayerDataScoreEntry, song entity.PlayerDataSong) (api_internal.SkippedRecord, bool) {
	if entry.Score >= minScoreValue && entry.Score <= maxScoreValue {
		return api_internal.SkippedRecord{}, false
	}

	return api_internal.SkippedRecord{
		RecordType: recordType,
		Reason:     fmt.Sprintf("score out of range: %d", entry.Score),
		Details:    fmt.Sprintf("idx=%s (%s), score=%d", entry.Idx, song.Title, entry.Score),
	}, true
}

func resolveCommonLampIDs(recordType string, entry PlayerDataScoreEntry, song entity.PlayerDataSong, masters *playerDataMaster) (resolvedLampIDs, *api_internal.SkippedRecord) {
	clearLampID, err := resolveClearLampID(entry.ClearLamp, masters)
	if err != nil {
		skipped := newResolveSkippedRecord(recordType, "clear_lamp", "clear_lamp", entry, song, optionalStringValue(entry.ClearLamp), err)
		return resolvedLampIDs{}, &skipped
	}

	comboLampID, err := resolveComboLampID(entry.ComboLv, masters)
	if err != nil {
		skipped := newResolveSkippedRecord(recordType, "combo_lamp", "combo_lv", entry, song, optionalIntValue(entry.ComboLv), err)
		return resolvedLampIDs{}, &skipped
	}

	fullChainID, err := resolveFullChainID(entry.FullChain, masters)
	if err != nil {
		skipped := newResolveSkippedRecord(recordType, "full_chain", "full_chain", entry, song, optionalIntValue(entry.FullChain), err)
		return resolvedLampIDs{}, &skipped
	}

	return resolvedLampIDs{
		clearLampID: clearLampID,
		comboLampID: comboLampID,
		fullChainID: fullChainID,
	}, nil
}

func newResolveSkippedRecord(recordType, reasonField, detailField string, entry PlayerDataScoreEntry, song entity.PlayerDataSong, value string, err error) api_internal.SkippedRecord {
	return api_internal.SkippedRecord{
		RecordType: recordType,
		Reason:     fmt.Sprintf("failed to resolve %s", reasonField),
		Details:    fmt.Sprintf("idx=%s (%s), %s=%s, error=%s", entry.Idx, song.Title, detailField, value, err.Error()),
	}
}

func optionalStringValue(value *string) string {
	if value == nil {
		return "nil"
	}
	return *value
}

func optionalIntValue(value *int) string {
	if value == nil {
		return "nil"
	}
	return fmt.Sprintf("%d", *value)
}
