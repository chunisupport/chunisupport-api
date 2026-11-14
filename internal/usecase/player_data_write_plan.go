package usecase

import "github.com/chunisupport/chunisupport-api/internal/domain/repository"

// planPlayerDataWrites は保存前と最終状態を比較し、変化のある行だけを永続化対象にします。
// 入力外の枠付き行は成績を保持して枠だけ解除し、一律解除による不要な書き戻しを避けます。
func planPlayerDataWrites(playerID, noneSlotID int, full []repository.PlayerRecordForUpsert, fullBefore map[int]repository.PlayerRecordState, rankedChartIDs []int, worldsend []repository.WorldsendRecordForUpsert, worldsendBefore map[int]repository.WorldsendRecordState) repository.PlayerDataSaveInput {
	plan := repository.PlayerDataSaveInput{}
	included := make(map[int]struct{}, len(full))
	for _, record := range full {
		included[record.ChartID] = struct{}{}
		before, exists := fullBefore[record.ChartID]
		switch {
		case !exists:
			plan.FullRecordsToInsert = append(plan.FullRecordsToInsert, record)
		case playerRecordMeaningfullyChanged(before, record.State):
			plan.FullRecordsToUpdate = append(plan.FullRecordsToUpdate, record)
		case before.SlotID != record.State.SlotID || !sameSlotOrder(before.SlotOrder, record.State.SlotOrder):
			plan.FullRecordSlotsToUpdate = append(plan.FullRecordSlotsToUpdate, repository.PlayerRecordSlotUpdate{PlayerID: record.PlayerID, ChartID: record.ChartID, SlotID: record.State.SlotID, SlotOrder: record.State.SlotOrder})
		}
	}
	for _, chartID := range rankedChartIDs {
		if _, exists := included[chartID]; !exists {
			plan.FullRecordSlotsToUpdate = append(plan.FullRecordSlotsToUpdate, repository.PlayerRecordSlotUpdate{PlayerID: playerID, ChartID: chartID, SlotID: noneSlotID})
		}
	}
	for _, record := range worldsend {
		before, exists := worldsendBefore[record.ChartID]
		if !exists {
			plan.WorldsendRecordsToInsert = append(plan.WorldsendRecordsToInsert, record)
		} else if worldsendRecordMeaningfullyChanged(before, record.State) {
			plan.WorldsendRecordsToUpdate = append(plan.WorldsendRecordsToUpdate, record)
		}
	}
	return plan
}

func sameSlotOrder(a, b *int) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}
