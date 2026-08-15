package entity

import (
	"errors"
	"time"
)

var (
	// ErrStalePlayerData は現在より古い取得日時のデータで更新しようとした場合に返されます。
	ErrStalePlayerData = errors.New("stale player data")
	// ErrConflictingPlayerDataTimestamp は同一取得日時に異なる公式指標がある場合に返されます。
	ErrConflictingPlayerDataTimestamp = errors.New("conflicting player data timestamp")
)

// PlayerMetricHistoryEntry は、ある取得時点における公式RATINGと公式OVER POWERの組を表します。
type PlayerMetricHistoryEntry struct {
	PlayerID          int
	OfficialRating    float64
	OfficialOverpower float64
	DataCollectedAt   time.Time
}
