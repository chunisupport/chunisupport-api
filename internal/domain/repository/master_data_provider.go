package repository

import (
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/masterdata"
)

// PlayerDataMasterProvider は、プレイヤーデータ登録時に必要なマスタデータを提供します。
// Interface Segregation Principleに従い、PlayerDataUsecaseが必要とするメソッドのみを定義します。
type PlayerDataMasterProvider interface {
	PlayerDataMasters() *masterdata.PlayerDataMasters
}

// SongMasterProvider は、楽曲検索時に必要なマスタデータを提供します。
// Interface Segregation Principleに従い、SongUsecaseが必要とするメソッドのみを定義します。
type SongMasterProvider interface {
	SongMasters() *masterdata.SongMasters
}

// AccountTypeMasterProvider は、アカウントタイプ情報を提供します。
// Interface Segregation Principleに従い、AuthUsecaseが必要とするメソッドのみを定義します。
type AccountTypeMasterProvider interface {
	GetAccountTypeNameByID(id int) string
}

// ChartStatsMasterProvider は譜面統計取得で必要なマスタデータを提供します。
// Interface Segregation Principleに従い、ChartStatsUsecaseが必要とするメソッドのみを定義します。
type ChartStatsMasterProvider interface {
	RatingBands() []*entity.RatingBand
}

// GoalMasterProvider は目標機能で必要なマスタデータを提供します。
type GoalMasterProvider interface {
	GoalMasters() *masterdata.GoalMasters
}
