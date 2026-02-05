package repository

import (
	"context"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// PlayerRecordState はプレイヤーレコードの状態を表す構造体です。
type PlayerRecordState struct {
	Score       int
	ClearLampID int
	ComboLampID int
	FullChainID int
	SlotID      int
	SlotOrder   *int
	UpdatedAt   time.Time
}

// PlayerRecordForUpsert はプレイヤーレコードの一括更新用の構造体です。
type PlayerRecordForUpsert struct {
	PlayerID int
	ChartID  int
	State    PlayerRecordState
}

// PlayerRecordRepository はプレイヤーレコードに関する永続化を扱うリポジトリです。
type PlayerRecordRepository interface {
	// FindByPlayerID はプレイヤーIDをキーにレコード一覧を取得します。
	FindByPlayerID(ctx context.Context, exec Executor, playerID int) ([]*entity.PlayerRecord, error)

	// FindByPlayerIDForRating はレーティング対象のレコードのみを取得します。
	FindByPlayerIDForRating(ctx context.Context, exec Executor, playerID int) ([]*entity.PlayerRecord, error)

	// GetLastScoreUpdate はプレイヤーのスコア最終更新日時を取得します。
	// player_records と player_worldsend_records の両テーブルから最新の updated_at を返します。
	// レコードが存在しない場合は nil を返します。
	GetLastScoreUpdate(ctx context.Context, exec Executor, playerID int) (*time.Time, error)
}
