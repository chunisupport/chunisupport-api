package entity

import (
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo/playername"
)

// Player はプレイヤーのエンティティを表します。
// 称号情報は player_honors テーブルで管理されるため、このエンティティには含まれません。
type Player struct {
	ID                       int
	UserID                   int
	Name                     playername.PlayerName     // プレイヤー名
	Level                    int                       // プレイヤーレベル
	OfficialRating           float64                   // 公式レーティング (official_player_rating)
	CalculatedRating         *float64                  // 計算レーティング (calculated_player_rating)
	NewAverageRating         *float64                  // 新曲枠平均レーティング (new_average_rating)
	BestAverageRating        *float64                  // ベスト枠平均レーティング (best_average_rating)
	ClassEmblemID            *int                      // クラスエンブレムID
	ClassEmblemBaseID        *int                      // クラスエンブレムのベースID
	LastPlayedAt             *time.Time                // 最終プレイ日時
	OverpowerValue           *float64                  // オーバーパワー値
	OfficialOverpower        float64                   // 公式オーバーパワー値
	OfficialOverpowerPercent *float64                  // 公式オーバーパワー割合
	OverpowerPercent         *float64                  // オーバーパワー割合
	DataCollectedAt          *time.Time                // CHUNITHM-NETからのデータ取得完了日時
	CreatedAt                time.Time                 // 作成日時
	UpdatedAt                time.Time                 // 更新日時
	metricHistoryToAppend    *PlayerMetricHistoryEntry // 今回の集約保存で追記する公式指標履歴
}

// NewPlayer は新規プレイヤーを生成し、永続化に必要な初期状態を設定します。
func NewPlayer(userID int, name playername.PlayerName) *Player {
	now := time.Now().UTC()

	return &Player{
		UserID:    userID,
		Name:      name,
		Level:     DefaultPlayerLevel,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// HasOfficialMetricsChanged は、現在の公式RATING・公式OVER POWER・公式OP%の組から変化するかを判定します。
func (p *Player) HasOfficialMetricsChanged(rating, overpower, overpowerPercent float64) bool {
	return p.OfficialRating != rating ||
		p.OfficialOverpower != overpower ||
		p.OfficialOverpowerPercent == nil ||
		*p.OfficialOverpowerPercent != overpowerPercent
}

// ChangeOfficialMetrics は取得日時の単調性を守りつつ公式指標を更新し、必要なら更新前状態を履歴化します。
func (p *Player) ChangeOfficialMetrics(rating, overpower, overpowerPercent float64, collectedAt time.Time) error {
	if p.DataCollectedAt != nil {
		if collectedAt.Before(*p.DataCollectedAt) {
			return ErrStalePlayerData
		}
		if collectedAt.Equal(*p.DataCollectedAt) && p.HasOfficialMetricsChanged(rating, overpower, overpowerPercent) {
			return ErrConflictingPlayerDataTimestamp
		}
		if p.HasOfficialMetricsChanged(rating, overpower, overpowerPercent) {
			p.metricHistoryToAppend = &PlayerMetricHistoryEntry{
				PlayerID: p.ID, OfficialRating: p.OfficialRating,
				OfficialOverpower:        p.OfficialOverpower,
				OfficialOverpowerPercent: p.OfficialOverpowerPercent,
				DataCollectedAt:          *p.DataCollectedAt,
			}
		}
	}
	p.OfficialRating = rating
	p.OfficialOverpower = overpower
	p.OfficialOverpowerPercent = &overpowerPercent
	p.DataCollectedAt = &collectedAt
	return nil
}

// PendingMetricHistory は次回の集約保存で追記する公式指標履歴を返します。
func (p *Player) PendingMetricHistory() *PlayerMetricHistoryEntry {
	return p.metricHistoryToAppend
}
