package entity

import (
	"fmt"
	"time"

	"github.com/Qman110101/chunisupport-api/internal/domain/vo/score"
)

// PlayerWorldsendRecord はプレイヤーの WORLD'S END 譜面記録エンティティを表します。
// WORLD'S END はレーティング計算の対象外であり、スロット（Best/New等）の概念を持ちません。
type PlayerWorldsendRecord struct {
	PlayerID         int
	WorldsendChartID int
	Score            score.Score
	ClearLampID      int
	ComboLampID      int
	FullChainID      int
	UpdatedAt        time.Time

	// リレーション（取得時に JOIN で設定される）
	WorldsendChart *WorldsendChart
	Song           *Song
	ClearLamp      *ClearLampType
	ComboLamp      *ComboLampType
	FullChain      *FullChainType
}

// Validate は PlayerWorldsendRecord のバリデーションを行います。
func (r *PlayerWorldsendRecord) Validate() error {
	if r.PlayerID <= 0 {
		return fmt.Errorf("player_id: プレイヤーIDは正の整数である必要があります")
	}

	if r.WorldsendChartID <= 0 {
		return fmt.Errorf("worldsend_chart_id: WORLD'S END譜面IDは正の整数である必要があります")
	}

	// Score 値オブジェクトは内部でバリデーション済み
	if r.Score == 0 {
		return fmt.Errorf("score: スコアは必須です")
	}

	if r.ClearLampID <= 0 {
		return fmt.Errorf("clear_lamp_id: クリアランプIDは正の整数である必要があります")
	}

	if r.ComboLampID <= 0 {
		return fmt.Errorf("combo_lamp_id: コンボランプIDは正の整数である必要があります")
	}

	if r.FullChainID <= 0 {
		return fmt.Errorf("full_chain_id: フルチェインIDは正の整数である必要があります")
	}

	return nil
}
