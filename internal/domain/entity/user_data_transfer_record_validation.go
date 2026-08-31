package entity

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/constants"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/coursescore"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/playername"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/score"
)

func validateUserDataTransferPlayer(player UserDataTransferPlayer, metricHistories []UserDataTransferMetricHistory) error {
	if _, err := playername.NewPlayerName(player.Name.String()); err != nil {
		return invalidUserDataTransfer("player.name is invalid")
	}
	if player.Level < 1 {
		return invalidUserDataTransfer("player.level must be at least 1")
	}
	if !validOfficialMetric(player.OfficialRating, constants.MaxOfficialRating) {
		return invalidUserDataTransfer("player.official_rating is invalid")
	}
	if !validOfficialMetric(player.OfficialOverpower, constants.MaxOfficialOverpower) {
		return invalidUserDataTransfer("player.official_overpower is invalid")
	}
	if player.OfficialOverpowerPercent != nil && !validOfficialMetric(*player.OfficialOverpowerPercent, constants.MaxOfficialOverpowerPercent) {
		return invalidUserDataTransfer("player.official_overpower_percent is invalid")
	}
	if !validOptionalName(player.ClassEmblemName) {
		return invalidUserDataTransfer("player.class_emblem_name is invalid")
	}
	if !validOptionalName(player.ClassEmblemBaseName) {
		return invalidUserDataTransfer("player.class_emblem_base_name is invalid")
	}
	if err := validateOptionalUTCDateTime("player.last_played_at", player.LastPlayedAt); err != nil {
		return err
	}
	if err := validateOptionalUTCDateTime("player.data_collected_at", player.DataCollectedAt); err != nil {
		return err
	}
	if err := validateUTCDateTime("player.created_at", player.CreatedAt); err != nil {
		return err
	}
	if player.DataCollectedAt == nil && len(metricHistories) > 0 {
		return invalidUserDataTransfer("metric_histories require player.data_collected_at")
	}
	return nil
}

func validateUserDataTransferRecords(records []UserDataTransferRecord) error {
	seenKeys := make(map[string]struct{}, len(records))
	seenSlotOrders := make(map[string]map[int]struct{})
	slotCounts := make(map[string]int)
	for i, record := range records {
		path := fmt.Sprintf("records[%d]", i)
		if err := validateSongOfficialIdx(path+".song_official_idx", record.SongOfficialIdx); err != nil {
			return err
		}
		if !validDifficulty(record.Difficulty) {
			return invalidUserDataTransfer(path + ".difficulty is invalid")
		}
		key := record.SongOfficialIdx + "\x00" + record.Difficulty
		if _, exists := seenKeys[key]; exists {
			return invalidUserDataTransfer(path + " duplicates a chart")
		}
		seenKeys[key] = struct{}{}
		if err := validateScoreAndLampNames(path, record.Score, record.ClearLampName, record.ComboLampName, record.FullChainName); err != nil {
			return err
		}
		limit, ranked := userDataTransferSlotLimit(record.SlotName)
		switch {
		case record.SlotName == "none":
			if record.SlotOrder != nil {
				return invalidUserDataTransfer(path + ".slot_order must be null for none slot")
			}
		case !ranked:
			return invalidUserDataTransfer(path + ".slot_name is invalid")
		case record.SlotOrder == nil || *record.SlotOrder < 1 || *record.SlotOrder > limit:
			return invalidUserDataTransfer(path + ".slot_order is out of range")
		default:
			orders := seenSlotOrders[record.SlotName]
			if orders == nil {
				orders = make(map[int]struct{}, limit)
				seenSlotOrders[record.SlotName] = orders
			}
			if _, exists := orders[*record.SlotOrder]; exists {
				return invalidUserDataTransfer(path + ".slot_order is duplicated")
			}
			orders[*record.SlotOrder] = struct{}{}
			slotCounts[record.SlotName]++
			if slotCounts[record.SlotName] > limit {
				return invalidUserDataTransfer(path + ".slot_name exceeds its limit")
			}
		}
		if err := validateUTCDateTime(path+".updated_at", record.UpdatedAt); err != nil {
			return err
		}
	}
	return nil
}

func validateUserDataTransferRecordHistories(histories []UserDataTransferRecordHistory) error {
	seen := make(map[string]struct{}, len(histories))
	counts := make(map[string]int)
	for i, history := range histories {
		path := fmt.Sprintf("record_histories[%d]", i)
		if err := validateSongOfficialIdx(path+".song_official_idx", history.SongOfficialIdx); err != nil {
			return err
		}
		if !validDifficulty(history.Difficulty) {
			return invalidUserDataTransfer(path + ".difficulty is invalid")
		}
		if err := validateScoreAndLampNames(path, history.Score, history.ClearLampName, history.ComboLampName, history.FullChainName); err != nil {
			return err
		}
		if err := validateUTCDateTime(path+".updated_at", history.UpdatedAt); err != nil {
			return err
		}
		chartKey := history.SongOfficialIdx + "\x00" + history.Difficulty
		counts[chartKey]++
		if counts[chartKey] > constants.MaxScoreHistoryEntriesPerChart {
			return invalidUserDataTransfer(path + " exceeds the per-chart history limit")
		}
		key := chartKey + "\x00" + databaseSecond(history.UpdatedAt).Format(time.RFC3339)
		if _, exists := seen[key]; exists {
			return invalidUserDataTransfer(path + " duplicates a history timestamp")
		}
		seen[key] = struct{}{}
	}
	return nil
}

func validateUserDataTransferWorldsendRecords(records []UserDataTransferWorldsendRecord) error {
	seen := make(map[string]struct{}, len(records))
	for i, record := range records {
		path := fmt.Sprintf("worldsend_records[%d]", i)
		if err := validateSongOfficialIdx(path+".song_official_idx", record.SongOfficialIdx); err != nil {
			return err
		}
		if _, exists := seen[record.SongOfficialIdx]; exists {
			return invalidUserDataTransfer(path + " duplicates a chart")
		}
		seen[record.SongOfficialIdx] = struct{}{}
		if err := validateScoreAndLampNames(path, record.Score, record.ClearLampName, record.ComboLampName, record.FullChainName); err != nil {
			return err
		}
		if err := validateUTCDateTime(path+".updated_at", record.UpdatedAt); err != nil {
			return err
		}
	}
	return nil
}

func validateUserDataTransferWorldsendRecordHistories(histories []UserDataTransferWorldsendRecordHistory) error {
	seen := make(map[string]struct{}, len(histories))
	counts := make(map[string]int)
	for i, history := range histories {
		path := fmt.Sprintf("worldsend_record_histories[%d]", i)
		if err := validateSongOfficialIdx(path+".song_official_idx", history.SongOfficialIdx); err != nil {
			return err
		}
		if err := validateScoreAndLampNames(path, history.Score, history.ClearLampName, history.ComboLampName, history.FullChainName); err != nil {
			return err
		}
		if err := validateUTCDateTime(path+".updated_at", history.UpdatedAt); err != nil {
			return err
		}
		counts[history.SongOfficialIdx]++
		if counts[history.SongOfficialIdx] > constants.MaxScoreHistoryEntriesPerChart {
			return invalidUserDataTransfer(path + " exceeds the per-chart history limit")
		}
		key := history.SongOfficialIdx + "\x00" + databaseSecond(history.UpdatedAt).Format(time.RFC3339)
		if _, exists := seen[key]; exists {
			return invalidUserDataTransfer(path + " duplicates a history timestamp")
		}
		seen[key] = struct{}{}
	}
	return nil
}

func validateUserDataTransferMetricHistories(histories []UserDataTransferMetricHistory, currentCollectedAt *time.Time) error {
	if len(histories) > constants.MaxMetricHistoryEntriesPerPlayer {
		return invalidUserDataTransfer("metric_histories exceeds the per-player history limit")
	}
	var previous time.Time
	seen := make(map[time.Time]struct{}, len(histories))
	for i, history := range histories {
		path := fmt.Sprintf("metric_histories[%d]", i)
		if !validOfficialMetric(history.OfficialRating, constants.MaxOfficialRating) {
			return invalidUserDataTransfer(path + ".official_rating is invalid")
		}
		if !validOfficialMetric(history.OfficialOverpower, constants.MaxOfficialOverpower) {
			return invalidUserDataTransfer(path + ".official_overpower is invalid")
		}
		if history.OfficialOverpowerPercent != nil && !validOfficialMetric(*history.OfficialOverpowerPercent, constants.MaxOfficialOverpowerPercent) {
			return invalidUserDataTransfer(path + ".official_overpower_percent is invalid")
		}
		if err := validateUTCDateTime(path+".data_collected_at", history.DataCollectedAt); err != nil {
			return err
		}
		normalized := databaseSecond(history.DataCollectedAt)
		if _, exists := seen[normalized]; exists {
			return invalidUserDataTransfer(path + ".data_collected_at is duplicated")
		}
		seen[normalized] = struct{}{}
		if !previous.IsZero() && !normalized.After(previous) {
			return invalidUserDataTransfer(path + ".data_collected_at is not in ascending order")
		}
		previous = normalized
		if currentCollectedAt != nil && !normalized.Before(databaseSecond(*currentCollectedAt)) {
			return invalidUserDataTransfer(path + ".data_collected_at must precede the current player metrics")
		}
	}
	return nil
}

func validateUserDataTransferCourseRecords(records []UserDataTransferCourseRecord) error {
	seen := make(map[string]struct{}, len(records))
	for i, record := range records {
		path := fmt.Sprintf("course_records[%d]", i)
		if strings.TrimSpace(record.CourseOfficialIdx) == "" || len(record.CourseOfficialIdx) > 32 {
			return invalidUserDataTransfer(path + ".course_official_idx is invalid")
		}
		if _, exists := seen[record.CourseOfficialIdx]; exists {
			return invalidUserDataTransfer(path + " duplicates a course")
		}
		seen[record.CourseOfficialIdx] = struct{}{}
		if _, err := coursescore.New(record.Score.Uint32()); err != nil {
			return invalidUserDataTransfer(path + ".score is invalid")
		}
		if strings.TrimSpace(record.ComboLampName) == "" {
			return invalidUserDataTransfer(path + ".combo_lamp_name is required")
		}
		if record.IsClear && record.Score == 0 {
			return invalidUserDataTransfer(path + ".score cannot be zero when cleared")
		}
		if record.Score.Uint32() == coursescore.Max && record.ComboLampName != "ALL JUSTICE" {
			return invalidUserDataTransfer(path + ".combo_lamp_name must be ALL JUSTICE at the theoretical score")
		}
		if record.ComboLampName == "ALL JUSTICE" && record.Score < 3_000_000 {
			return invalidUserDataTransfer(path + ".score is too low for ALL JUSTICE")
		}
		if err := validateUTCDateTime(path+".updated_at", record.UpdatedAt); err != nil {
			return err
		}
	}
	return nil
}

func validOfficialMetric(value, maximum float64) bool {
	if value < 0 || value > maximum || math.IsNaN(value) || math.IsInf(value, 0) {
		return false
	}
	scaled := value * constants.OfficialMetricDecimalScale
	return math.Abs(scaled-math.Round(scaled)) <= constants.OfficialMetricDecimalTolerance
}

func validateScoreAndLampNames(path string, value score.Score, clearLamp, comboLamp, fullChain string) error {
	if _, err := score.NewScore(uint32(value)); err != nil {
		return invalidUserDataTransfer(path + ".score is invalid")
	}
	if strings.TrimSpace(clearLamp) == "" || strings.TrimSpace(comboLamp) == "" || strings.TrimSpace(fullChain) == "" {
		return invalidUserDataTransfer(path + " has an empty lamp name")
	}
	if comboLamp == "ALL JUSTICE" && value < 1_000_000 {
		return invalidUserDataTransfer(path + ".score is too low for ALL JUSTICE")
	}
	if value == score.Score(constants.TheoreticalScore) && comboLamp != "ALL JUSTICE" {
		return invalidUserDataTransfer(path + ".combo_lamp_name must be ALL JUSTICE at the theoretical score")
	}
	if fullChain != "NONE" && comboLamp != "FULL COMBO" && comboLamp != "ALL JUSTICE" {
		return invalidUserDataTransfer(path + ".full_chain_name requires a combo lamp")
	}
	return nil
}
