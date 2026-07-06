package repository

import (
	"context"
	"time"
)

// PlayerBatchKey はキーセットページング時点の競合検知情報です。
type PlayerBatchKey struct {
	ID              int
	DataCollectedAt *time.Time
}

// PlayerStatsMasterSnapshot はrun全体で固定するマスタ情報です。
type PlayerStatsMasterSnapshot struct {
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

type PlayerBatchUpdate struct {
	ResetSlots   bool
	Assignments  []PlayerBatchSlotAssignment
	PlayerRating float64
	BestAverage  float64
	NewAverage   float64
	Overpower    float64
}

type PlayerBatchProcessStatus int

const (
	PlayerBatchUpdated PlayerBatchProcessStatus = iota
	PlayerBatchDeleted
	PlayerBatchConflict
)

// PlayerStatsBatchRepository は再計算バッチ固有の投影と原子的更新を提供します。
type PlayerStatsBatchRepository interface {
	LoadSnapshot(ctx context.Context, operationalDate time.Time) (PlayerStatsMasterSnapshot, error)
	ListPlayerKeys(ctx context.Context, afterID, upperBound, limit int) ([]PlayerBatchKey, error)
	ProcessPlayer(ctx context.Context, key PlayerBatchKey, buildUpdate func(PlayerBatchData) (PlayerBatchUpdate, error)) (PlayerBatchProcessStatus, error)
}

// BatchLock はバッチrun全体の多重起動を防止します。
type BatchLock interface {
	Release(ctx context.Context) error
}

type BatchLockProvider interface {
	TryAcquire(ctx context.Context, name string) (BatchLock, bool, error)
}
