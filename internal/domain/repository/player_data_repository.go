package repository

import (
	"context"

	"github.com/Qman110101/chunisupport-api/internal/domain/entity"
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

// PlayerDataRepository はプレイヤーデータ登録に関する永続化を扱うリポジトリです。
type PlayerDataRepository interface {
	// LoadMasterData はプレイヤーデータ登録に必要なマスタ情報を取得します。
	LoadMasterData(ctx context.Context, exec Executor, officialIdxList []string) (*PlayerDataMaster, error)

	// SavePlayerData はプレイヤーデータを一括で保存します。
	SavePlayerData(ctx context.Context, exec Executor, input PlayerDataSaveInput) error
}
