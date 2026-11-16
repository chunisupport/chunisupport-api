package usecase

import (
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/chunisupport/chunisupport-api/internal/domain/constants"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/info"
)

// validatePlayerDataPayload はプレイヤーデータの事前検証を行い、明らかに不正なデータを検知します。
// トランザクションに入る前に実行し、改ざんや異常なデータを早期に検出します。
func validatePlayerDataPayload(payload *PlayerDataPayload) error {
	if payload == nil {
		return &PlayerDataValidationError{
			Field:   "payload",
			Message: "payload cannot be nil",
		}
	}
	if payload.Rating == nil {
		return &PlayerDataValidationError{Field: "rating", Message: "rating is required"}
	}
	if *payload.Rating < 0 || *payload.Rating > info.MaxOfficialRating || math.IsNaN(*payload.Rating) || math.IsInf(*payload.Rating, 0) {
		return &PlayerDataValidationError{Field: "rating", Message: "rating is out of range"}
	}
	if !hasOfficialMetricPrecision(*payload.Rating) {
		return &PlayerDataValidationError{Field: "rating", Message: "rating must have at most 2 decimal places"}
	}
	if payload.Overpower.Value == nil {
		return &PlayerDataValidationError{Field: "overpower.value", Message: "overpower.value is required"}
	}
	if *payload.Overpower.Value < 0 || *payload.Overpower.Value > info.MaxOfficialOverpower || math.IsNaN(*payload.Overpower.Value) || math.IsInf(*payload.Overpower.Value, 0) {
		return &PlayerDataValidationError{Field: "overpower.value", Message: "overpower.value is out of range"}
	}
	if !hasOfficialMetricPrecision(*payload.Overpower.Value) {
		return &PlayerDataValidationError{Field: "overpower.value", Message: "overpower.value must have at most 2 decimal places"}
	}
	if payload.Overpower.Percentage == nil {
		return &PlayerDataValidationError{Field: "overpower.percentage", Message: "overpower.percentage is required"}
	}
	if *payload.Overpower.Percentage < 0 || *payload.Overpower.Percentage > info.MaxOfficialOverpowerPercent || math.IsNaN(*payload.Overpower.Percentage) || math.IsInf(*payload.Overpower.Percentage, 0) {
		return &PlayerDataValidationError{Field: "overpower.percentage", Message: "overpower.percentage is out of range"}
	}
	if !hasOfficialMetricPrecision(*payload.Overpower.Percentage) {
		return &PlayerDataValidationError{Field: "overpower.percentage", Message: "overpower.percentage must have at most 2 decimal places"}
	}

	errorCount := 0
	maxErrorsToReport := 10
	errorMessages := make([]string, 0, maxErrorsToReport)

	for i, entry := range payload.Scores.Standard {
		if errorCount >= maxErrorsToReport {
			break
		}
		if err := validateScoreEntry(&entry, "standard", i); err != nil {
			errorCount++
			errorMessages = append(errorMessages, err.Error())
		}
	}

	for i, entry := range payload.Scores.Worldsend {
		if errorCount >= maxErrorsToReport {
			break
		}
		if err := validateScoreEntry(&entry, "worldsend", i); err != nil {
			errorCount++
			errorMessages = append(errorMessages, err.Error())
		}
	}
	for i, entry := range payload.Scores.Course {
		if errorCount >= maxErrorsToReport {
			break
		}
		if err := validateCourseScoreEntry(entry, i); err != nil {
			errorCount++
			errorMessages = append(errorMessages, err.Error())
		}
	}

	if errorCount > 0 {
		msg := fmt.Sprintf("detected %d invalid score entries: %s", errorCount, strings.Join(errorMessages, "; "))
		if errorCount >= maxErrorsToReport {
			msg += " (and more...)"
		}
		return &PlayerDataValidationError{
			Field:   "scores",
			Message: msg,
		}
	}

	// CHUNITHM-NETはプレイ済みでなければ閲覧できないため、通常譜面の記録があるのに本枠が両方空になることはない。
	// 収集中のセッション切れなどで枠ページを取得できなかったpayloadを登録すると、既存の公式枠がすべて解除されてしまうため拒否する。
	if len(payload.Scores.Standard) > 0 && !slices.ContainsFunc(payload.Scores.Standard, isMainSlotEntry) {
		return newPlayerDataSlotValidationError("best or new slot is required when standard scores exist")
	}

	return nil
}

// isMainSlotEntry は通常譜面のスコアがベスト枠・新曲枠のいずれかに属するかを返します。
// 判定は登録時のスロット名解決（resolveSlotID）と同じく前後の空白と大文字小文字を無視します。
func isMainSlotEntry(entry PlayerDataScoreEntry) bool {
	if entry.Slot == nil {
		return false
	}
	slot := strings.TrimSpace(*entry.Slot)
	return strings.EqualFold(slot, "best") || strings.EqualFold(slot, "new")
}

func hasOfficialMetricPrecision(value float64) bool {
	scaled := value * info.OfficialMetricDecimalScale
	return math.Abs(scaled-math.Round(scaled)) <= info.OfficialMetricDecimalTolerance
}

func normalizeOfficialMetric(value float64) float64 {
	return math.Round(value*info.OfficialMetricDecimalScale) / info.OfficialMetricDecimalScale
}

func validateCourseScoreEntry(entry PlayerDataCourseEntry, index int) error {
	if strings.TrimSpace(entry.Idx) == "" {
		return fmt.Errorf("course[%d]: idx is required", index)
	}
	if entry.Score < 0 || entry.Score > 3030000 {
		return fmt.Errorf("course[%d]: score must be between 0 and 3030000 (idx=%s)", index, entry.Idx)
	}
	if entry.ComboLv < 1 || entry.ComboLv > 3 {
		return fmt.Errorf("course[%d]: unknown combo level: %d (idx=%s)", index, entry.ComboLv, entry.Idx)
	}
	if entry.Score == 3030000 && entry.ComboLv != 3 {
		return fmt.Errorf("course[%d]: score=3030000 without AJ (cmb_lv=3, idx=%s)", index, entry.Idx)
	}
	if entry.ComboLv == 3 && entry.Score < 3000000 {
		return fmt.Errorf("course[%d]: AJ requires score>=3000000 (idx=%s)", index, entry.Idx)
	}
	if entry.IsClear && entry.Score == 0 {
		return fmt.Errorf("course[%d]: cleared course cannot have score=0 (idx=%s)", index, entry.Idx)
	}
	return nil
}

// validateScoreEntry は個別のスコアエントリーを検証します。
// AJ（All Justice: cmb_lv=3）である場合、必ず1,000,000点以上でなければならないという整合性をチェックします。
func validateScoreEntry(entry *PlayerDataScoreEntry, recordType string, index int) error {
	// AJかつ100万点未満は矛盾している
	if entry.ComboLv != nil && *entry.ComboLv == 3 {
		if entry.Score < 1000000 {
			return fmt.Errorf("%s[%d]: inconsistent data - AJ (cmb_lv=3) with score=%d (must be >= 1,000,000, idx=%s)",
				recordType, index, entry.Score, entry.Idx)
		}
	}

	// 1010000点かつAJでないのは矛盾している
	if entry.Score == 1010000 && (entry.ComboLv == nil || *entry.ComboLv != 3) {
		return fmt.Errorf("%s[%d]: inconsistent data - score=1,010,000 without AJ (cmb_lv=3), idx=%s",
			recordType, index, entry.Idx)
	}

	// FULL CHAINは複数人でAJまたはFCを達成したときのランプなので、個人のAJ/FCなしでは成立しない
	if entry.FullChain != nil && (*entry.FullChain == 2 || *entry.FullChain == 3) &&
		(entry.ComboLv == nil || (*entry.ComboLv != 2 && *entry.ComboLv != 3)) {
		return fmt.Errorf("%s[%d]: inconsistent data - FULL CHAIN (fch_lv=%d) without AJ/FC (cmb_lv=2 or 3), idx=%s",
			recordType, index, *entry.FullChain, entry.Idx)
	}

	return nil
}

// validateMainSlotRecordsResolved は本枠のスキップによる枠の消失を防ぎます。
// 重複の正規化前に件数を比較し、末尾の順位が保存できない場合も拒否します。
func validateMainSlotRecordsResolved(entries []PlayerDataScoreEntry, records []repository.PlayerRecordForUpsert, masters *playerDataMaster) error {
	missing := 0
	for _, entry := range entries {
		if isMainSlotEntry(entry) {
			missing++
		}
	}
	for _, record := range records {
		slot := masters.SlotNamesByID[record.State.SlotID]
		if slot == "best" || slot == "new" {
			missing--
		}
	}
	if missing > 0 {
		return newPlayerDataSlotValidationError("best or new slot records could not be saved")
	}
	return nil
}

func validateFullRecordSlots(records []repository.PlayerRecordForUpsert, masters *playerDataMaster) error {
	seenOrders := make(map[string]map[int]struct{})
	counts := make(map[string]int)
	for _, record := range records {
		slotName, ok := masters.SlotNamesByID[record.State.SlotID]
		if !ok {
			return fmt.Errorf("failed to resolve slot name by id: %d", record.State.SlotID)
		}
		if slotName == defaultSlotName {
			if record.State.SlotOrder != nil {
				return newPlayerDataSlotValidationError("slot_order must be null for none slot")
			}
			continue
		}

		limit, ok := playerDataSlotLimit(slotName)
		if !ok {
			return fmt.Errorf("unsupported resolved slot name: %s", slotName)
		}
		counts[slotName]++
		if counts[slotName] > limit {
			return newPlayerDataSlotValidationError(fmt.Sprintf("%s slot exceeds its limit: max %d", slotName, limit))
		}
		if record.State.SlotOrder == nil || *record.State.SlotOrder < 1 || *record.State.SlotOrder > limit {
			return newPlayerDataSlotValidationError(fmt.Sprintf("%s slot_order is out of range: must be between 1 and %d", slotName, limit))
		}
		orders := seenOrders[slotName]
		if orders == nil {
			orders = make(map[int]struct{}, limit)
			seenOrders[slotName] = orders
		}
		if _, exists := orders[*record.State.SlotOrder]; exists {
			return newPlayerDataSlotValidationError(fmt.Sprintf("%s slot_order is duplicated: %d", slotName, *record.State.SlotOrder))
		}
		orders[*record.State.SlotOrder] = struct{}{}
	}
	// 上限未満の正常な枠は許可し、難易度ごとの収集漏れなどによる欠番を拒否します。
	for _, slotName := range []string{"best", "new"} {
		for order := 1; order <= counts[slotName]; order++ {
			if _, exists := seenOrders[slotName][order]; !exists {
				return newPlayerDataSlotValidationError(fmt.Sprintf("%s slot_order must be consecutive from 1", slotName))
			}
		}
	}
	return nil
}

func playerDataSlotLimit(slotName string) (int, bool) {
	switch slotName {
	case "best":
		return constants.BestSlotMaxCount, true
	case "new":
		return constants.NewSlotMaxCount, true
	case "best_candidate", "new_candidate":
		return constants.CandidateSlotMaxCount, true
	default:
		return 0, false
	}
}

func newPlayerDataSlotValidationError(message string) error {
	return &PlayerDataValidationError{Field: "scores.standard", Message: message}
}
