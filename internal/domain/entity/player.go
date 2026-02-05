package entity

import (
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo/playername"
)

// Player はプレイヤーのエンティティを表します。
// 称号情報は player_honors テーブルで管理されるため、このエンティティには含まれません。
type Player struct {
	ID                int
	UserID            int
	Name              playername.PlayerName // プレイヤー名
	Level             int                   // プレイヤーレベル
	OfficialRating    *float64              // 公式レーティング (official_player_rating)
	CalculatedRating  *float64              // 計算レーティング (calculated_player_rating)
	NewAverageRating  *float64              // 新曲枠平均レーティング (new_average_rating)
	BestAverageRating *float64              // ベスト枠平均レーティング (best_average_rating)
	ClassEmblemID     *int                  // クラスエンブレムID
	ClassEmblemBaseID *int                  // クラスエンブレムのベースID
	LastPlayedAt      *time.Time            // 最終プレイ日時
	OverpowerValue    *float64              // オーバーパワー値
	OverpowerPercent  *float64              // オーバーパワー割合
	CreatedAt         time.Time             // 作成日時
	UpdatedAt         time.Time             // 更新日時
	Users             *User                 // このプレイヤーに紐づくユーザー
}
