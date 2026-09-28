package repository

import (
	"context"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo/masterfingerprint"
)

// PlayerBatchKey はキーセットページング時点の競合検知情報です。
type PlayerBatchKey struct {
	ID              int
	DataCollectedAt *time.Time
}

// PlayerDataMasterSnapshot はrun全体で固定するマスタ情報です。
type PlayerDataMasterSnapshot struct {
	Version    BatchVersion
	Songs      []BatchSong
	Charts     []BatchChart
	SlotIDs    map[string]int
	UpperBound int
}

type BatchVersion struct {
	ID         int
	Name       string
	ReleasedAt time.Time
}

type BatchSong struct {
	ID            int
	ReleasedAt    *time.Time
	IsDeleted     bool
	IsWorldsend   bool
	OfficialIndex string
}

type BatchChart struct {
	ID             int
	SongID         int
	DifficultyID   int
	DifficultyName string
	ChartConst     float64
	IsConstUnknown bool
}

type PlayerBatchData struct {
	ID              int
	LastPlayedAt    *time.Time
	DataCollectedAt *time.Time
	Records         []PlayerBatchRecord
	LockedSongs     []PlayerBatchLockedSong
}

type PlayerBatchRecord struct {
	ChartID     int
	Score       uint32
	ComboLampID int
	SlotName    string
	SlotOrder   *int
}

type PlayerBatchLockedSong struct {
	SongID   int
	IsUltima bool
}

type PlayerBatchSlotAssignment struct {
	ChartID  int
	SlotID   int
	Position int
}

// PlayerBatchUpdate は1プレイヤー分の再計算結果です。
// 枠は現在の状態との差分だけを持ち、変更のない譜面には書き込みません。
type PlayerBatchUpdate struct {
	ClearChartIDs     []int                       // 枠から外す（noneに戻す）譜面
	Assignments       []PlayerBatchSlotAssignment // 枠または順位が変わる譜面
	PlayerRating      float64
	BestAverage       float64
	NewAverage        float64
	Overpower         float64
	MasterFingerprint masterfingerprint.Fingerprint // 再計算に使ったマスタと計算ロジックのフィンガープリント
}

type PlayerBatchProcessStatus int

const (
	PlayerBatchUpdated PlayerBatchProcessStatus = iota
	PlayerBatchDeleted
	PlayerBatchConflict
)

// PlayerDataBatchRepository は再計算バッチ固有の投影と原子的更新を提供します。
type PlayerDataBatchRepository interface {
	LoadSnapshot(ctx context.Context, operationalDate time.Time) (PlayerDataMasterSnapshot, error)
	// ListPlayerKeys は、再計算済みの記録が指定したフィンガープリントと一致しないプレイヤーだけを列挙します。
	ListPlayerKeys(ctx context.Context, afterID, upperBound, limit int, fingerprint masterfingerprint.Fingerprint) ([]PlayerBatchKey, error)
	ProcessPlayer(ctx context.Context, key PlayerBatchKey, buildUpdate func(PlayerBatchData) (PlayerBatchUpdate, error)) (PlayerBatchProcessStatus, error)
}

// BatchLock はバッチrun全体の多重起動を防止します。
type BatchLock interface {
	Release(ctx context.Context) error
}

type BatchLockProvider interface {
	TryAcquire(ctx context.Context, name string) (BatchLock, bool, error)
}
