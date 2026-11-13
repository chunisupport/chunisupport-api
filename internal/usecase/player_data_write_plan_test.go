package usecase

import (
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanPlayerDataWrites(t *testing.T) {
	base := repository.PlayerRecordState{Score: 1000000, ClearLampID: 1, ComboLampID: 1, FullChainID: 1, SlotID: 2, SlotOrder: new(1)}
	for _, tt := range []struct {
		name   string
		change func(*repository.PlayerRecordState)
		kind   string
	}{
		{"変更なし", func(s *repository.PlayerRecordState) {}, "none"},
		{"別ポインタの同順位", func(s *repository.PlayerRecordState) { s.SlotOrder = new(1) }, "none"},
		{"更新日時だけ", func(s *repository.PlayerRecordState) { s.UpdatedAt = time.Now() }, "none"},
		{"スコア", func(s *repository.PlayerRecordState) { s.Score++ }, "score"},
		{"クリアランプ", func(s *repository.PlayerRecordState) { s.ClearLampID++ }, "score"},
		{"コンボランプ", func(s *repository.PlayerRecordState) { s.ComboLampID++ }, "score"},
		{"フルチェイン", func(s *repository.PlayerRecordState) { s.FullChainID++ }, "score"},
		{"成績と枠", func(s *repository.PlayerRecordState) { s.Score++; s.SlotID = 3 }, "score"},
		{"枠", func(s *repository.PlayerRecordState) { s.SlotID = 3 }, "slot"},
		{"順位", func(s *repository.PlayerRecordState) { s.SlotOrder = new(2) }, "slot"},
		{"順位解除", func(s *repository.PlayerRecordState) { s.SlotOrder = nil }, "slot"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			after := base
			tt.change(&after)
			record := repository.PlayerRecordForUpsert{PlayerID: 77, ChartID: 101, State: after}
			plan := planPlayerDataWrites(77, 1, []repository.PlayerRecordForUpsert{record}, map[int]repository.PlayerRecordState{101: base}, []int{101}, nil, nil)
			assert.Empty(t, plan.FullRecordsToInsert)
			assert.Empty(t, plan.WorldsendRecordsToInsert)
			assert.Empty(t, plan.WorldsendRecordsToUpdate)
			if tt.kind == "score" {
				assert.Equal(t, []repository.PlayerRecordForUpsert{record}, plan.FullRecordsToUpdate)
			} else {
				assert.Empty(t, plan.FullRecordsToUpdate)
			}
			if tt.kind == "slot" {
				assert.Equal(t, []repository.PlayerRecordSlotUpdate{{PlayerID: 77, ChartID: 101, SlotID: after.SlotID, SlotOrder: after.SlotOrder}}, plan.FullRecordSlotsToUpdate)
			} else {
				assert.Empty(t, plan.FullRecordSlotsToUpdate)
			}
		})
	}
	t.Run("新規と入力外の枠解除", func(t *testing.T) {
		record := repository.PlayerRecordForUpsert{PlayerID: 77, ChartID: 101, State: base}
		plan := planPlayerDataWrites(77, 9, []repository.PlayerRecordForUpsert{record}, nil, []int{102}, nil, nil)
		assert.Equal(t, []repository.PlayerRecordForUpsert{record}, plan.FullRecordsToInsert)
		assert.Equal(t, []repository.PlayerRecordSlotUpdate{{PlayerID: 77, ChartID: 102, SlotID: 9}}, plan.FullRecordSlotsToUpdate)
	})
	t.Run("両方nilの順位と入力外枠なし", func(t *testing.T) {
		base.SlotOrder = nil
		plan := planPlayerDataWrites(77, 1, []repository.PlayerRecordForUpsert{{PlayerID: 77, ChartID: 101, State: base}}, map[int]repository.PlayerRecordState{101: base, 102: base}, nil, nil, nil)
		assert.Empty(t, plan.FullRecordsToUpdate)
		assert.Empty(t, plan.FullRecordSlotsToUpdate)
	})
	t.Run("空入力は全枠解除", func(t *testing.T) {
		plan := planPlayerDataWrites(77, 1, nil, nil, []int{101, 102}, nil, nil)
		require.Len(t, plan.FullRecordSlotsToUpdate, 2)
		for _, r := range plan.FullRecordSlotsToUpdate {
			assert.Equal(t, 1, r.SlotID)
			assert.Nil(t, r.SlotOrder)
		}
	})
}

func TestPlanPlayerDataWrites_Worldsend(t *testing.T) {
	base := repository.WorldsendRecordState{Score: 1000000, ClearLampID: 1, ComboLampID: 1, FullChainID: 1}
	for _, tt := range []struct {
		name   string
		exists bool
		change func(*repository.WorldsendRecordState)
		kind   string
	}{
		{"新規", false, func(s *repository.WorldsendRecordState) {}, "insert"},
		{"変更なし", true, func(s *repository.WorldsendRecordState) {}, "none"},
		{"スコア", true, func(s *repository.WorldsendRecordState) { s.Score++ }, "update"},
		{"クリアランプ", true, func(s *repository.WorldsendRecordState) { s.ClearLampID++ }, "update"},
		{"コンボランプ", true, func(s *repository.WorldsendRecordState) { s.ComboLampID++ }, "update"},
		{"フルチェイン", true, func(s *repository.WorldsendRecordState) { s.FullChainID++ }, "update"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			after := base
			tt.change(&after)
			before := map[int]repository.WorldsendRecordState{}
			if tt.exists {
				before[201] = base
			}
			r := repository.WorldsendRecordForUpsert{PlayerID: 77, ChartID: 201, State: after}
			plan := planPlayerDataWrites(77, 1, nil, nil, nil, []repository.WorldsendRecordForUpsert{r}, before)
			if tt.kind == "insert" {
				assert.Equal(t, []repository.WorldsendRecordForUpsert{r}, plan.WorldsendRecordsToInsert)
			} else {
				assert.Empty(t, plan.WorldsendRecordsToInsert)
			}
			if tt.kind == "update" {
				assert.Equal(t, []repository.WorldsendRecordForUpsert{r}, plan.WorldsendRecordsToUpdate)
			} else {
				assert.Empty(t, plan.WorldsendRecordsToUpdate)
			}
		})
	}
}
