package repository

import (
	"context"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// PlayerDataMaster はプレイヤーデータ登録に必要なマスタ情報を保持します。
type PlayerDataMaster struct {
	Songs             map[string]entity.PlayerDataSong
	ChartsByKey       map[string]entity.PlayerDataChart
	ChartsByID        map[int]entity.PlayerDataChart
	WorldsendBySongID map[int]entity.PlayerDataWorldsendChart
}

// PlayerDataSaveInput はプレイヤーデータの永続化に必要な入力データです。
type PlayerDataSaveInput struct {
	FullRecords      []PlayerRecordForUpsert
	WorldsendRecords []WorldsendRecordForUpsert
}

// OverpowerTargetFilter はOVER POWER集計対象楽曲の絞り込み条件です。
type OverpowerTargetFilter struct {
	ExcludeWorldsend bool
	ExcludeDeleted   bool
}

// OverpowerTargetStats はOVER POWER割合計算で使う全体集計値です。
type OverpowerTargetStats struct {
	SongCount         int
	MaxOverpowerTotal float64
}

// PlayerDataRepository はプレイヤーデータ登録に関する永続化を扱うリポジトリです。
type PlayerDataRepository interface {
	// LoadMasterData はプレイヤーデータ登録に必要なマスタ情報を取得します。
	// songs/charts/worldsend_chartsの読み取りのみのためトランザクション外で呼び出せます。
	LoadMasterData(ctx context.Context, officialIdxList []string) (*PlayerDataMaster, error)

	// SavePlayerData はプレイヤーデータを一括で保存します。
	// 書き込み操作のため必ずトランザクション内で呼び出してください。exec が nil の場合はエラーを返します。
	SavePlayerData(ctx context.Context, exec Executor, input PlayerDataSaveInput) error

	// GetOverpowerTargetStats はOVER POWER割合計算の分母となる対象楽曲の最大OP合計を取得します。
	// songs/chartsの読み取りのみのためトランザクション外で呼び出せます。
	GetOverpowerTargetStats(ctx context.Context, filter OverpowerTargetFilter) (*OverpowerTargetStats, error)
}
