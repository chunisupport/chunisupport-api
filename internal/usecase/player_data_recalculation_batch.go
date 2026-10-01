package usecase

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"strconv"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/constants"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/service"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/masterfingerprint"
	"github.com/chunisupport/chunisupport-api/internal/info"
)

const playerDataBatchPageSize = 100

type PlayerDataBatchResult struct {
	StartedAt            time.Time
	OperationalDate      time.Time
	CurrentVersion       string
	MasterFingerprint    masterfingerprint.Fingerprint
	UpperBoundPlayerID   int
	Processed            int
	Success              int
	CurrentPreserved     int
	CurrentBrokenRebuilt int
	LegacyRebuilt        int
	SlotsUnchanged       int
	ConflictSkipped      int
	DeletedSkipped       int
	Failed               int
	LastPlayerID         int
}

type PlayerDataRecalculationBatchUsecase struct {
	repository repository.PlayerDataBatchRepository
	now        func() time.Time
}

func NewPlayerDataRecalculationBatchUsecase(batchRepository repository.PlayerDataBatchRepository) *PlayerDataRecalculationBatchUsecase {
	return &PlayerDataRecalculationBatchUsecase{repository: batchRepository, now: time.Now}
}

func (u *PlayerDataRecalculationBatchUsecase) Execute(ctx context.Context) (PlayerDataBatchResult, error) {
	startedAt := u.now()
	jst := time.FixedZone("Asia/Tokyo", 9*60*60)
	operationalTime := startedAt.In(jst).Add(-7 * time.Hour)
	operationalDate := time.Date(operationalTime.Year(), operationalTime.Month(), operationalTime.Day(), 0, 0, 0, 0, jst)
	snapshot, err := u.repository.LoadSnapshot(ctx, operationalDate)
	if err != nil {
		return PlayerDataBatchResult{}, fmt.Errorf("マスタスナップショットの取得に失敗しました: %w", err)
	}
	prepared, err := prepareBatchSnapshot(snapshot, operationalDate, jst)
	if err != nil {
		return PlayerDataBatchResult{}, err
	}
	result := PlayerDataBatchResult{
		StartedAt:          startedAt,
		OperationalDate:    operationalDate,
		CurrentVersion:     snapshot.Version.Name,
		MasterFingerprint:  prepared.fingerprint,
		UpperBoundPlayerID: snapshot.UpperBound,
	}
	afterID := 0
	for {
		if err := ctx.Err(); err != nil {
			return result, nil
		}
		keys, err := u.repository.ListPlayerKeys(ctx, afterID, snapshot.UpperBound, playerDataBatchPageSize, prepared.fingerprint)
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
			currentBroken := false
			slotsUnchanged := false
			var lastPlayedAt *time.Time
			status, processErr := u.repository.ProcessPlayer(ctx, key, func(data repository.PlayerBatchData) (repository.PlayerBatchUpdate, error) {
				lastPlayedAt = data.LastPlayedAt
				isCurrent = data.LastPlayedAt != nil && !databaseWallClockInJST(*data.LastPlayedAt, prepared.versionStartedAt.Location()).Before(prepared.versionStartedAt)
				update, broken, err := prepared.buildUpdate(data, isCurrent)
				currentBroken = broken
				slotsUnchanged = (!isCurrent || broken) && len(update.ClearChartIDs) == 0 && len(update.Assignments) == 0
				return update, err
			})
			if processErr != nil {
				slog.ErrorContext(ctx, "プレイヤーデータの再計算に失敗しました",
					"player_id", key.ID,
					"rebuild_reason", playerRebuildReason(isCurrent, currentBroken, lastPlayedAt),
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
				if currentBroken {
					result.CurrentBrokenRebuilt++
				} else if isCurrent {
					result.CurrentPreserved++
				} else {
					result.LegacyRebuilt++
				}
				if slotsUnchanged {
					result.SlotsUnchanged++
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
	snapshot         repository.PlayerDataMasterSnapshot
	versionStartedAt time.Time
	songsByID        map[int]repository.BatchSong
	chartsByID       map[int]repository.BatchChart
	officialIndex    map[int]uint64
	operationalDate  time.Time
	fingerprint      masterfingerprint.Fingerprint
}

func prepareBatchSnapshot(snapshot repository.PlayerDataMasterSnapshot, operationalDate time.Time, jst *time.Location) (preparedBatchSnapshot, error) {
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
	fingerprint, err := computeMasterFingerprint(snapshot)
	if err != nil {
		return preparedBatchSnapshot{}, err
	}
	prepared.fingerprint = fingerprint
	return prepared, nil
}

// fingerprintSource はフィンガープリントの算出対象を正規化した構造です。
// buildUpdate と prepareBatchSnapshot が計算に使う項目だけを持ちます。
// 表示用の項目や計算に使わない項目を含めると、計算結果が変わらない変更でも全プレイヤーが再計算になるため含めません。
// 配信前の楽曲はidxが確定しないためマスタに存在せず、運用日による配信済み判定は常に同じ結果になるので含めません。
type fingerprintSource struct {
	LogicVersion      int
	VersionID         int
	VersionReleasedAt time.Time
	Songs             []fingerprintSong
	Charts            []fingerprintChart
	Slots             []fingerprintSlot
}

type fingerprintSong struct {
	ID            int
	ReleasedAt    *time.Time
	IsDeleted     bool
	IsWorldsend   bool
	OfficialIndex string
}

type fingerprintChart struct {
	ID             int
	SongID         int
	DifficultyName string
	ChartConst     float64
}

type fingerprintSlot struct {
	Name string
	ID   int
}

// computeMasterFingerprint は計算結果に影響するマスタ項目と計算ロジックのバージョンからフィンガープリントを求めます。
// 取得順に依存しないよう、楽曲と譜面はID順、枠は名前順に並べてからシリアライズします。
func computeMasterFingerprint(snapshot repository.PlayerDataMasterSnapshot) (masterfingerprint.Fingerprint, error) {
	source := fingerprintSource{
		LogicVersion:      info.PlayerRecalculationLogicVersion,
		VersionID:         snapshot.Version.ID,
		VersionReleasedAt: snapshot.Version.ReleasedAt,
		Songs:             make([]fingerprintSong, 0, len(snapshot.Songs)),
		Charts:            make([]fingerprintChart, 0, len(snapshot.Charts)),
		Slots:             make([]fingerprintSlot, 0, len(snapshot.SlotIDs)),
	}
	for _, song := range snapshot.Songs {
		source.Songs = append(source.Songs, fingerprintSong{ID: song.ID, ReleasedAt: song.ReleasedAt, IsDeleted: song.IsDeleted, IsWorldsend: song.IsWorldsend, OfficialIndex: song.OfficialIndex})
	}
	slices.SortFunc(source.Songs, func(a, b fingerprintSong) int { return cmp.Compare(a.ID, b.ID) })
	for _, chart := range snapshot.Charts {
		source.Charts = append(source.Charts, fingerprintChart{ID: chart.ID, SongID: chart.SongID, DifficultyName: chart.DifficultyName, ChartConst: chart.ChartConst})
	}
	slices.SortFunc(source.Charts, func(a, b fingerprintChart) int { return cmp.Compare(a.ID, b.ID) })
	for _, name := range slices.Sorted(maps.Keys(snapshot.SlotIDs)) {
		source.Slots = append(source.Slots, fingerprintSlot{Name: name, ID: snapshot.SlotIDs[name]})
	}
	data, err := json.Marshal(source)
	if err != nil {
		return masterfingerprint.Fingerprint{}, fmt.Errorf("マスタのフィンガープリントの算出に失敗しました: %w", err)
	}
	return masterfingerprint.Compute(data), nil
}

func (p preparedBatchSnapshot) buildUpdate(data repository.PlayerBatchData, current bool) (repository.PlayerBatchUpdate, bool, error) {
	currentBroken := false
	if current {
		slotName, err := validateOfficialMainSlots(data.Records)
		if err != nil {
			currentBroken = true
			slog.Warn("現行プレイヤーの本枠が不正なため再構築します",
				"player_id", data.ID,
				"slot_name", slotName,
				"rebuild_reason", "current_broken_slots",
				"error", err)
		}
	}
	rebuild := !current || currentBroken
	mainSlotsMissing := current && !slices.ContainsFunc(data.Records, func(record repository.PlayerBatchRecord) bool {
		return record.SlotName == "best" || record.SlotName == "new"
	})
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
			return repository.PlayerBatchUpdate{}, currentBroken, fmt.Errorf("譜面マスタを解決できません: chart_id=%d", record.ChartID)
		}
		song, ok := p.songsByID[chart.SongID]
		if !ok {
			return repository.PlayerBatchUpdate{}, currentBroken, fmt.Errorf("楽曲マスタを解決できません: song_id=%d", chart.SongID)
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
		if mainSlotsMissing && !rebuild {
			currentBroken = true
			rebuild = true
			slog.Warn("対象記録がある現行プレイヤーの本枠が両方空のため再構築します",
				"player_id", data.ID,
				"rebuild_reason", "current_broken_slots")
		}
		ratingRecord := service.RatingSlotRecord{ChartID: record.ChartID, Score: record.Score, ChartConst: chart.ChartConst, OfficialIndex: p.officialIndex[song.ID]}
		if !rebuild {
			switch record.SlotName {
			case "best":
				best = append(best, ratingRecord)
			case "new":
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
	if !rebuild {
		for _, candidateSlot := range []string{"best_candidate", "new_candidate"} {
			if err := validateOfficialSlotSet(data.Records, candidateSlot, constants.CandidateSlotMaxCount); err != nil {
				slog.Warn("候補枠の公式順が不正です",
					"player_id", data.ID, "slot_name", candidateSlot, "error", err)
			}
		}
		stats := service.AggregateOfficialRating(best, newRecords)
		op, _ := service.CalcOverpowerSummary(opRecords, 0)
		return repository.PlayerBatchUpdate{PlayerRating: stats.PlayerRating, BestAverage: stats.BestAverage, NewAverage: stats.NewAverage, Overpower: op, MasterFingerprint: p.fingerprint}, currentBroken, nil
	}
	bestSlots := service.BuildRatingSlots(best, constants.BestSlotMaxCount, constants.CandidateSlotMaxCount)
	newSlots := service.BuildRatingSlots(newRecords, constants.NewSlotMaxCount, constants.CandidateSlotMaxCount)
	clearChartIDs, assignments := p.diffSlots(data.Records, []slotGroup{
		{name: "best", records: bestSlots.Main},
		{name: "best_candidate", records: bestSlots.Candidates},
		{name: "new", records: newSlots.Main},
		{name: "new_candidate", records: newSlots.Candidates},
	})
	stats := service.AggregateOfficialRating(bestSlots.Main, newSlots.Main)
	op, _ := service.CalcOverpowerSummary(opRecords, 0)
	return repository.PlayerBatchUpdate{ClearChartIDs: clearChartIDs, Assignments: assignments, PlayerRating: stats.PlayerRating, BestAverage: stats.BestAverage, NewAverage: stats.NewAverage, Overpower: op, MasterFingerprint: p.fingerprint}, currentBroken, nil
}

// slotGroup は再構築後の1つの枠と、その枠に入る順に並んだ譜面です。
type slotGroup struct {
	name    string
	records []service.RatingSlotRecord
}

// diffSlots は現在の枠と再構築後の枠を比べ、外す譜面と付け替える譜面だけを返します。
// 結果が前回と同じプレイヤーで player_records を書き換えないよう、全件の付け替えではなく差分で更新します。
// noneなのに順位が残っている不整合な行も外す対象に含めます。
func (p preparedBatchSnapshot) diffSlots(records []repository.PlayerBatchRecord, groups []slotGroup) ([]int, []repository.PlayerBatchSlotAssignment) {
	current := make(map[int]repository.PlayerBatchRecord)
	for _, record := range records {
		if record.SlotName != "none" || record.SlotOrder != nil {
			current[record.ChartID] = record
		}
	}
	assigned := make(map[int]struct{})
	assignments := make([]repository.PlayerBatchSlotAssignment, 0)
	for _, group := range groups {
		for i, record := range group.records {
			position := i + 1
			assigned[record.ChartID] = struct{}{}
			if before, ok := current[record.ChartID]; ok && before.SlotName == group.name && before.SlotOrder != nil && *before.SlotOrder == position {
				continue
			}
			assignments = append(assignments, repository.PlayerBatchSlotAssignment{ChartID: record.ChartID, SlotID: p.snapshot.SlotIDs[group.name], Position: position})
		}
	}
	clearChartIDs := make([]int, 0)
	for _, record := range records {
		if _, slotted := current[record.ChartID]; !slotted {
			continue
		}
		if _, ok := assigned[record.ChartID]; !ok {
			clearChartIDs = append(clearChartIDs, record.ChartID)
		}
	}
	return clearChartIDs, assignments
}

func databaseDateInLocation(value time.Time, location *time.Location) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, location)
}

func databaseWallClockInJST(value time.Time, jst *time.Location) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), value.Hour(), value.Minute(), value.Second(), value.Nanosecond(), jst)
}

func playerRebuildReason(current, currentBroken bool, lastPlayedAt *time.Time) string {
	if currentBroken {
		return "current_broken_slots"
	}
	if current {
		return "current_preserved"
	}
	if lastPlayedAt == nil {
		return "legacy_null_last_played"
	}
	return "legacy_old_last_played"
}

func validateOfficialMainSlots(records []repository.PlayerBatchRecord) (string, error) {
	mainSlots := []struct {
		name  string
		limit int
	}{
		{name: "best", limit: constants.BestSlotMaxCount},
		{name: "new", limit: constants.NewSlotMaxCount},
	}
	for _, slot := range mainSlots {
		if err := validateOfficialSlotSet(records, slot.name, slot.limit); err != nil {
			return slot.name, err
		}
	}
	return "", nil
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
		if count > limit {
			return fmt.Errorf("%s枠が%d件を超えています", name, limit)
		}
		if err := validateOfficialSlot(record.SlotOrder, limit); err != nil {
			return err
		}
		if _, exists := seen[*record.SlotOrder]; exists {
			return fmt.Errorf("%s枠のslot_orderが重複しています: %d", name, *record.SlotOrder)
		}
		seen[*record.SlotOrder] = struct{}{}
	}
	return nil
}
