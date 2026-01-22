package repository

import (
	"context"
	"time"

	"github.com/Qman110101/chunisupport-api/internal/domain/entity"
)

// WorldsendRecordState はWorldsendレコードの状態を表す構造体です。
type WorldsendRecordState struct {
	Score       int
	ClearLampID int
	ComboLampID int
	FullChainID int
	UpdatedAt   time.Time
}

// WorldsendRecordForUpsert はWorldsendレコードの一括更新用の構造体です。
type WorldsendRecordForUpsert struct {
	PlayerID int
	ChartID  int // worldsend_chart_id
	State    WorldsendRecordState
}

// ExistingWorldsendRecord は既存のWorldsendレコードの状態を表す構造体です。
type ExistingWorldsendRecord struct {
	WorldsendChartID int
	Score            int
	ClearLampID      int
	ComboLampID      int
	FullChainID      int
	UpdatedAt        time.Time
}

// WorldsendRecordRepository はWorldsendレコードに関する永続化を扱うリポジトリです。
type WorldsendRecordRepository interface {
	// FindByPlayerID はプレイヤーIDをキーに WORLD'S END レコードを詳細情報付きで取得します。
	// 楽曲、譜面、マスターデータを含む完全な PlayerWorldsendRecord エンティティを返します。
	FindByPlayerID(ctx context.Context, exec Executor, playerID int) ([]*entity.PlayerWorldsendRecord, error)

	// FindExistingByPlayerID はプレイヤーIDをキーに既存のWorldsendレコード状態を取得します。
	FindExistingByPlayerID(ctx context.Context, exec Executor, playerID int) ([]ExistingWorldsendRecord, error)
}
