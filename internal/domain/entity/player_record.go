package entity

import (
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo/score"
)

// PlayerRecord はプレイヤーの譜面記録エンティティを表します
type PlayerRecord struct {
	PlayerID    int
	ChartID     int
	Score       score.Score
	ClearLampID int
	ComboLampID int
	FullChainID int
	SlotID      int
	SlotOrder   *int
	IsOPTarget  bool
	UpdatedAt   time.Time

	Chart           *Chart
	Song            *Song
	ClearLamp       *ClearLampType
	ComboLamp       *ComboLampType
	FullChain       *FullChainType
	Slot            *Slot
	ChartDifficulty *ChartDifficulty
}

// IsRanked はこのレコードがランキング対象（スロット指定あり）かを判定します。
func (r *PlayerRecord) IsRanked() bool {
	if r.Slot == nil {
		return false
	}
	// "none" スロットはランキング対象外
	return r.Slot.Name != "" && r.Slot.Name != "none"
}

// SlotKey はレコードのスロット種別を示すキーを返します。
// ランキング対象外の場合は空文字列を返します。
func (r *PlayerRecord) SlotKey() string {
	if r.IsRanked() {
		return r.Slot.Name
	}
	return ""
}
