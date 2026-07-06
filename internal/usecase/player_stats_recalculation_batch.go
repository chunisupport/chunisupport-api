package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/service"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
	"github.com/chunisupport/chunisupport-api/internal/info"
)

const playerStatsBatchPageSize = 100

type PlayerStatsBatchResult struct {
	StartedAt          time.Time
	OperationalDate    time.Time
	CurrentVersion     string
	UpperBoundPlayerID int
	Processed          int
	Success            int
	CurrentPreserved   int
	LegacyRebuilt      int
	ConflictSkipped    int
	DeletedSkipped     int
	Failed             int
	LastPlayerID       int
}

type PlayerStatsRecalculationBatchUsecase struct {
	repository repository.PlayerStatsBatchRepository
	now        func() time.Time
}

func NewPlayerStatsRecalculationBatchUsecase(batchRepository repository.PlayerStatsBatchRepository) *PlayerStatsRecalculationBatchUsecase {
	return &PlayerStatsRecalculationBatchUsecase{repository: batchRepository, now: time.Now}
}

func (u *PlayerStatsRecalculationBatchUsecase) Execute(ctx context.Context) (PlayerStatsBatchResult, error) {
	startedAt := u.now()
	jst := time.FixedZone("Asia/Tokyo", 9*60*60)
	operationalTime := startedAt.In(jst).Add(-7 * time.Hour)
	operationalDate := time.Date(operationalTime.Year(), operationalTime.Month(), operationalTime.Day(), 0, 0, 0, 0, jst)
	snapshot, err := u.repository.LoadSnapshot(ctx, operationalDate)
	if err != nil {
		return PlayerStatsBatchResult{}, fmt.Errorf("マスタスナップショットの取得に失敗しました: %w", err)
	}
	prepared, err := prepareBatchSnapshot(snapshot, operationalDate, jst)
	if err != nil {
		return PlayerStatsBatchResult{}, err
	}
	result := PlayerStatsBatchResult{
		StartedAt:          startedAt,
		OperationalDate:    operationalDate,
		CurrentVersion:     snapshot.Version.Name,
		UpperBoundPlayerID: snapshot.UpperBound,
	}
	afterID := 0
	for {
		if err := ctx.Err(); err != nil {
			return result, nil
		}
		keys, err := u.repository.ListPlayerKeys(ctx, afterID, snapshot.UpperBound, playerStatsBatchPageSize)
		if err != nil {
			return result, err
		}
		if len(keys) == 0 {
			break
		}
		for _, key := range keys {
			if ctx.Err() != nil {
				return result, nil
			}
			result.Processed++
			result.LastPlayerID = key.ID
			isCurrent := false
			var lastPlayedAt *time.Time
			status, processErr := u.repository.ProcessPlayer(ctx, key, func(data repository.PlayerBatchData) (repository.PlayerBatchUpdate, error) {
				lastPlayedAt = data.LastPlayedAt
				isCurrent = data.LastPlayedAt != nil && !databaseWallClockInJST(*data.LastPlayedAt, prepared.versionStartedAt.Location()).Before(prepared.versionStartedAt)
				return prepared.buildUpdate(data, isCurrent)
			})
			if processErr != nil {
				slog.ErrorContext(ctx, "プレイヤーデータの再計算に失敗しました",
					"player_id", key.ID,
					"rebuild_reason", playerRebuildReason(isCurrent, lastPlayedAt),
					"error", processErr)
				result.Failed++
				continue
			}
			switch status {
			case repository.PlayerBatchDeleted:
				result.DeletedSkipped++
			case repository.PlayerBatchConflict:
				result.ConflictSkipped++
			default:
				result.Success++
				if isCurrent {
					result.CurrentPreserved++
				} else {
					result.LegacyRebuilt++
				}
			}
		}
		afterID = keys[len(keys)-1].ID
	}
	if result.Failed > 0 {
		return result, fmt.Errorf("%d件のプレイヤー再計算に失敗しました", result.Failed)
	}
	return result, nil
}

type preparedBatchSnapshot struct {
	snapshot         repository.PlayerStatsMasterSnapshot
	versionStartedAt time.Time
	songsByID        map[int]repository.BatchSong
	chartsByID       map[int]repository.BatchChart
	officialIndex    map[int]uint64
	operationalDate  time.Time
}

func prepareBatchSnapshot(snapshot repository.PlayerStatsMasterSnapshot, operationalDate time.Time, jst *time.Location) (preparedBatchSnapshot, error) {
	requiredSlots := []string{"none", "best", "best_candidate", "new", "new_candidate"}
	for _, name := range requiredSlots {
		if snapshot.SlotIDs[name] == 0 {
			return preparedBatchSnapshot{}, fmt.Errorf("必須スロットがありません: %s", name)
		}
	}
	prepared := preparedBatchSnapshot{
		snapshot:         snapshot,
		versionStartedAt: time.Date(snapshot.Version.ReleasedAt.Year(), snapshot.Version.ReleasedAt.Month(), snapshot.Version.ReleasedAt.Day(), 7, 0, 0, 0, jst),
		songsByID:        make(map[int]repository.BatchSong, len(snapshot.Songs)),
		chartsByID:       make(map[int]repository.BatchChart, len(snapshot.Charts)),
		officialIndex:    make(map[int]uint64, len(snapshot.Songs)),
		operationalDate:  operationalDate,
	}
	usedIndexes := make(map[uint64]int)
	for _, song := range snapshot.Songs {
		prepared.songsByID[song.ID] = song
		if song.IsDeleted || song.IsWorldsend || (song.ReleasedAt != nil && databaseDateInLocation(*song.ReleasedAt, jst).After(operationalDate)) {
			continue
		}
		index, err := strconv.ParseUint(song.OfficialIndex, 10, 64)
		if err != nil {
			return preparedBatchSnapshot{}, fmt.Errorf("official_idxが10進数ではありません: song_id=%d: %w", song.ID, err)
		}
		if duplicateSongID, exists := usedIndexes[index]; exists {
			return preparedBatchSnapshot{}, fmt.Errorf("official_idxが数値として重複しています: song_id=%d, duplicate_song_id=%d", song.ID, duplicateSongID)
		}
		usedIndexes[index] = song.ID
		prepared.officialIndex[song.ID] = index
	}
	for _, chart := range snapshot.Charts {
		if _, err := chartconstant.NewChartConstant(chart.ChartConst); err != nil {
			return preparedBatchSnapshot{}, fmt.Errorf("譜面定数が不正です: chart_id=%d: %w", chart.ID, err)
		}
		prepared.chartsByID[chart.ID] = chart
	}
	return prepared, nil
}

func (p preparedBatchSnapshot) buildUpdate(data repository.PlayerBatchData, current bool) (repository.PlayerBatchUpdate, error) {
	best := make([]service.RatingSlotRecord, 0)
	newRecords := make([]service.RatingSlotRecord, 0)
	opRecords := make([]service.OverpowerRecord, 0)
	locked := make(map[string]struct{}, len(data.LockedSongs))
	for _, item := range data.LockedSongs {
		locked[fmt.Sprintf("%d:%t", item.SongID, item.IsUltima)] = struct{}{}
	}
	for _, record := range data.Records {
		chart, ok := p.chartsByID[record.ChartID]
		if !ok {
			return repository.PlayerBatchUpdate{}, fmt.Errorf("譜面マスタを解決できません: chart_id=%d", record.ChartID)
		}
		song, ok := p.songsByID[chart.SongID]
		if !ok {
			return repository.PlayerBatchUpdate{}, fmt.Errorf("楽曲マスタを解決できません: song_id=%d", chart.SongID)
		}
		if !song.IsDeleted && !song.IsWorldsend {
			_, songLocked := locked[fmt.Sprintf("%d:false", song.ID)]
			_, ultimaLocked := locked[fmt.Sprintf("%d:true", song.ID)]
			if !songLocked && !(chart.DifficultyName == info.DifficultyNameUltima && ultimaLocked) {
				opRecords = append(opRecords, service.OverpowerRecord{SongID: song.ID, Score: record.Score, ChartConst: chart.ChartConst, ComboLampID: record.ComboLampID})
			}
		}
		if song.IsDeleted || song.IsWorldsend || (song.ReleasedAt != nil && databaseDateInLocation(*song.ReleasedAt, p.operationalDate.Location()).After(p.operationalDate)) {
			continue
		}
		ratingRecord := service.RatingSlotRecord{ChartID: record.ChartID, Score: record.Score, ChartConst: chart.ChartConst, OfficialIndex: p.officialIndex[song.ID]}
		if current {
			switch record.SlotName {
			case "best":
				if err := validateOfficialSlot(record.SlotOrder, 30); err != nil {
					return repository.PlayerBatchUpdate{}, fmt.Errorf("best枠の公式順が不正です: chart_id=%d: %w", record.ChartID, err)
				}
				best = append(best, ratingRecord)
			case "new":
				if err := validateOfficialSlot(record.SlotOrder, 20); err != nil {
					return repository.PlayerBatchUpdate{}, fmt.Errorf("new枠の公式順が不正です: chart_id=%d: %w", record.ChartID, err)
				}
				newRecords = append(newRecords, ratingRecord)
			}
			continue
		}
		if song.ReleasedAt == nil || !song.ReleasedAt.Before(p.snapshot.Version.ReleasedAt) {
			newRecords = append(newRecords, ratingRecord)
		} else {
			best = append(best, ratingRecord)
		}
	}
	if current {
		for _, candidateSlot := range []string{"best_candidate", "new_candidate"} {
			if err := validateOfficialSlotSet(data.Records, candidateSlot, 10); err != nil {
				slog.Warn("候補枠の公式順が不正です",
					"player_id", data.ID, "slot_name", candidateSlot, "error", err)
			}
		}
		if err := validateOfficialSlotSet(data.Records, "best", 30); err != nil {
			return repository.PlayerBatchUpdate{}, err
		}
		if err := validateOfficialSlotSet(data.Records, "new", 20); err != nil {
			return repository.PlayerBatchUpdate{}, err
		}
		stats := service.AggregateOfficialRating(best, newRecords)
		op, _ := service.CalcOverpowerSummary(opRecords, 0)
		return repository.PlayerBatchUpdate{PlayerRating: stats.PlayerRating, BestAverage: stats.BestAverage, NewAverage: stats.NewAverage, Overpower: op}, nil
	}
	bestSlots := service.BuildRatingSlots(best, 30, 10)
	newSlots := service.BuildRatingSlots(newRecords, 20, 10)
	assignments := make([]repository.PlayerBatchSlotAssignment, 0, len(bestSlots.Main)+len(bestSlots.Candidates)+len(newSlots.Main)+len(newSlots.Candidates))
	assignments = appendAssignments(assignments, bestSlots.Main, p.snapshot.SlotIDs["best"])
	assignments = appendAssignments(assignments, bestSlots.Candidates, p.snapshot.SlotIDs["best_candidate"])
	assignments = appendAssignments(assignments, newSlots.Main, p.snapshot.SlotIDs["new"])
	assignments = appendAssignments(assignments, newSlots.Candidates, p.snapshot.SlotIDs["new_candidate"])
	stats := service.AggregateOfficialRating(bestSlots.Main, newSlots.Main)
	op, _ := service.CalcOverpowerSummary(opRecords, 0)
	return repository.PlayerBatchUpdate{ResetSlots: true, Assignments: assignments, PlayerRating: stats.PlayerRating, BestAverage: stats.BestAverage, NewAverage: stats.NewAverage, Overpower: op}, nil
}

func databaseDateInLocation(value time.Time, location *time.Location) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, location)
}

func databaseWallClockInJST(value time.Time, jst *time.Location) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), value.Hour(), value.Minute(), value.Second(), value.Nanosecond(), jst)
}

func playerRebuildReason(current bool, lastPlayedAt *time.Time) string {
	if current {
		return "current_preserved"
	}
	if lastPlayedAt == nil {
		return "legacy_null_last_played"
	}
	return "legacy_old_last_played"
}

func appendAssignments(target []repository.PlayerBatchSlotAssignment, records []service.RatingSlotRecord, slotID int) []repository.PlayerBatchSlotAssignment {
	for i, record := range records {
		target = append(target, repository.PlayerBatchSlotAssignment{ChartID: record.ChartID, SlotID: slotID, Position: i + 1})
	}
	return target
}

func validateOfficialSlot(order *int, limit int) error {
	if order == nil || *order < 1 || *order > limit {
		return fmt.Errorf("slot_orderは1以上%d以下の必須値です", limit)
	}
	return nil
}

func validateOfficialSlotSet(records []repository.PlayerBatchRecord, name string, limit int) error {
	seen := make(map[int]struct{}, limit)
	count := 0
	for _, record := range records {
		if record.SlotName != name {
			continue
		}
		count++
		if err := validateOfficialSlot(record.SlotOrder, limit); err != nil {
			return err
		}
		if _, exists := seen[*record.SlotOrder]; exists {
			return fmt.Errorf("%s枠のslot_orderが重複しています: %d", name, *record.SlotOrder)
		}
		seen[*record.SlotOrder] = struct{}{}
	}
	if count > limit {
		return fmt.Errorf("%s枠が%d件を超えています", name, limit)
	}
	return nil
}
