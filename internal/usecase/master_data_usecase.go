package usecase

import (
	"context"

	"github.com/chunisupport/chunisupport-api/internal/domain/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/ratingband"
)

// MasterDataUsecase はマスタデータAPIのユースケースです。
type MasterDataUsecase interface {
	GetMasterData(ctx context.Context) *MasterDataOutput
}

// MasterDataOutput はマスタデータAPIの出力です。
// 各スライスはユースケース層で決定されたソート順で返されます。
type MasterDataOutput struct {
	// Genres はID昇順のジャンル一覧です。
	Genres []masterdata.Item
	// Difficulties はゲームの正規表示順（SortOrder昇順）の難易度一覧です。
	Difficulties []masterdata.Item
	// AccountTypes はID昇順のアカウントタイプ一覧です。
	AccountTypes []masterdata.Item
	// Versions はID昇順のバージョン一覧です。
	Versions []masterdata.Version
	// RatingBands はプロバイダが返す順序（SortOrder昇順）のレーティング帯一覧です。
	RatingBands []*ratingband.RatingBand
	// AchievementTypes はID昇順の実績タイプ一覧です。
	AchievementTypes []masterdata.Item
}
