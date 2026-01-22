package repository

import (
	"context"
	"time"

	"github.com/Qman110101/chunisupport-api/internal/domain/entity"
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

// ExistingPlayerRecord は既存のプレイヤーレコードの状態を表す構造体です。
type ExistingPlayerRecord struct {
	ChartID     int
	Score       int
	ClearLampID int
	ComboLampID int
	FullChainID int
	SlotID      int
	SlotOrder   *int
	UpdatedAt   time.Time
}

// PlayerRecordRepository はプレイヤーレコードに関する永続化を扱うリポジトリです。
type PlayerRecordRepository interface {
	// FindByPlayerID はプレイヤーIDをキーにレコード一覧を取得します。
	FindByPlayerID(ctx context.Context, exec Executor, playerID int) ([]*entity.PlayerRecord, error)

	// FindExistingByPlayerID はプレイヤーIDをキーに既存のレコード状態を取得します。
	FindExistingByPlayerID(ctx context.Context, exec Executor, playerID int) ([]ExistingPlayerRecord, error)
}
