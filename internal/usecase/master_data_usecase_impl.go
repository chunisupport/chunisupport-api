package usecase

import (
	"cmp"
	"context"
	"slices"

	"github.com/chunisupport/chunisupport-api/internal/domain/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/master"
)

type masterDataUsecase struct {
	masterProvider     repository.MasterDataMasterProvider
	ratingBandProvider repository.ChartStatsMasterProvider
}

// NewMasterDataUsecase は新しい MasterDataUsecase を生成します。
func NewMasterDataUsecase(masterProvider repository.MasterDataMasterProvider, ratingBandProvider repository.ChartStatsMasterProvider) MasterDataUsecase {
	return &masterDataUsecase{
		masterProvider:     masterProvider,
		ratingBandProvider: ratingBandProvider,
	}
}

// GetMasterData はソート済みのマスタデータ一覧を返します。
// 難易度はゲームの正規表示順（SortOrder昇順）でソートされます。
// バージョンはリリース日昇順でソートされます。
// その他のマスタはID昇順でソートされます。
func (u *masterDataUsecase) GetMasterData(_ context.Context) *MasterDataOutput {
	masters := u.masterProvider.MasterDataMasters()
	if masters == nil {
		return &MasterDataOutput{
			Genres:           []masterdata.Item{},
			Difficulties:     []masterdata.Item{},
			AccountTypes:     []masterdata.Item{},
			Versions:         []masterdata.Version{},
			RatingBands:      u.ratingBandProvider.RatingBands(),
			AchievementTypes: []masterdata.Item{},
		}
	}

	return &MasterDataOutput{
		Genres:           sortedByID(masters.Genres, func(g master.Genre) masterdata.Item { return masterdata.Item{ID: g.ID, Name: g.Name} }),
		Difficulties:     sortedDifficultiesBySortOrder(masters.Difficulties),
		AccountTypes:     sortedByID(masters.AccountTypes, func(a master.AccountType) masterdata.Item { return masterdata.Item{ID: a.ID, Name: a.Name} }),
		Versions:         sortedVersionsByReleasedAt(masters.Versions),
		RatingBands:      u.ratingBandProvider.RatingBands(),
		AchievementTypes: sortedByID(masters.AchievementTypes, func(i masterdata.Item) masterdata.Item { return i }),
	}
}

// sortedDifficultiesBySortOrder は難易度をゲームの正規表示順（SortOrder昇順）でソートした Item スライスを返します。
// SortOrder はゲーム内の表示順序（BASIC < ADVANCED < EXPERT < MASTER < ULTIMA）を表します。
func sortedDifficultiesBySortOrder(difficulties map[string]master.ChartDifficulty) []masterdata.Item {
	type entry struct {
		item      masterdata.Item
		sortOrder int
	}
	entries := make([]entry, 0, len(difficulties))
	for _, d := range difficulties {
		entries = append(entries, entry{
			item:      masterdata.Item{ID: d.ID, Name: d.Name},
			sortOrder: d.SortOrder,
		})
	}
	slices.SortFunc(entries, func(a, b entry) int {
		return cmp.Compare(a.sortOrder, b.sortOrder)
	})
	items := make([]masterdata.Item, len(entries))
	for i, e := range entries {
		items[i] = e.item
	}
	return items
}

// sortedByID はマップの値をアイテムに変換し、ID昇順でソートしたスライスを返します。
func sortedByID[V any](m map[string]V, toItem func(V) masterdata.Item) []masterdata.Item {
	items := make([]masterdata.Item, 0, len(m))
	for _, v := range m {
		items = append(items, toItem(v))
	}
	slices.SortFunc(items, func(a, b masterdata.Item) int {
		return cmp.Compare(a.ID, b.ID)
	})
	return items
}

// sortedVersionsByReleasedAt はバージョンをリリース日昇順でソートしたスライスを返します。
// versions テーブルの released_at は一意制約により同一値を持つレコードが存在しないため、
// 不安定ソートで問題ありません。
func sortedVersionsByReleasedAt(versions map[int]masterdata.Version) []masterdata.Version {
	items := make([]masterdata.Version, 0, len(versions))
	for _, v := range versions {
		items = append(items, v)
	}
	slices.SortFunc(items, func(a, b masterdata.Version) int {
		return a.ReleasedAt.Compare(b.ReleasedAt)
	})
	return items
}
