package usecase

import (
	"cmp"
	"context"
	"maps"
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
// 難易度とジャンルはゲームの正規表示順（SortOrder昇順）でソートされます。
// バージョンはリリース日昇順でソートされます。
// 名前順フォルダはSortOrder昇順でソートされます。
// その他のマスタはID昇順でソートされます。
func (u *masterDataUsecase) GetMasterData(_ context.Context) *MasterDataOutput {
	masters := u.masterProvider.MasterDataMasters()
	if masters == nil {
		return &MasterDataOutput{
			NameFolders:      []masterdata.NameFolder{},
			Genres:           []masterdata.Genre{},
			Difficulties:     []masterdata.Item{},
			AccountTypes:     []masterdata.Item{},
			Versions:         []masterdata.Version{},
			RatingBands:      u.ratingBandProvider.RatingBands(),
			AchievementTypes: []masterdata.Item{},
			ClassEmblems:     []masterdata.Item{},
			ClassEmblemBases: []masterdata.Item{},
			ClearLamps:       []masterdata.Item{},
			ComboLamps:       []masterdata.Item{},
			FullChains:       []masterdata.Item{},
			Slots:            []masterdata.Item{},
			HonorTypes:       []masterdata.Item{},
			Possessions:      []masterdata.Item{},
		}
	}

	return &MasterDataOutput{
		NameFolders:      sortedNameFolders(masters.NameFolders),
		Genres:           sortedGenresBySortOrder(masters.Genres),
		Difficulties:     sortedDifficultiesBySortOrder(masters.Difficulties),
		AccountTypes:     sortedByID(masters.AccountTypes, func(a master.AccountType) masterdata.Item { return masterdata.Item{ID: a.ID, Name: a.Name} }),
		Versions:         sortedVersionsByReleasedAt(masters.Versions),
		RatingBands:      u.ratingBandProvider.RatingBands(),
		AchievementTypes: sortedByID(masters.AchievementTypes, func(i masterdata.Item) masterdata.Item { return i }),
		ClassEmblems: sortedBySortOrder(masters.ClassEmblems, func(v master.ClassEmblem) (masterdata.Item, int) {
			return masterdata.Item{ID: v.ID, Name: v.Name}, v.SortOrder
		}),
		ClassEmblemBases: sortedBySortOrder(masters.ClassEmblemBases, func(v master.ClassEmblemBase) (masterdata.Item, int) {
			return masterdata.Item{ID: v.ID, Name: v.Name}, v.SortOrder
		}),
		ClearLamps: sortedBySortOrder(masters.ClearLamps, func(v master.ClearLampType) (masterdata.Item, int) {
			return masterdata.Item{ID: v.ID, Name: v.Name}, v.SortOrder
		}),
		ComboLamps: sortedBySortOrder(masters.ComboLamps, func(v master.ComboLampType) (masterdata.Item, int) {
			return masterdata.Item{ID: v.ID, Name: v.Name}, v.SortOrder
		}),
		FullChains: sortedBySortOrder(masters.FullChains, func(v master.FullChainType) (masterdata.Item, int) {
			return masterdata.Item{ID: v.ID, Name: v.Name}, v.SortOrder
		}),
		Slots:      sortedByID(masters.Slots, func(v master.Slot) masterdata.Item { return masterdata.Item{ID: v.ID, Name: v.Name} }),
		HonorTypes: sortedByID(masters.HonorTypes, func(v master.HonorType) masterdata.Item { return masterdata.Item{ID: v.ID, Name: v.Name} }),
		Possessions: sortedByID(masters.Possessions, func(v master.Possession) masterdata.Item {
			return masterdata.Item{ID: v.ID, Name: v.Name}
		}),
	}
}

// GetVersions はリリース日昇順のバージョン一覧を返します。
func (u *masterDataUsecase) GetVersions(_ context.Context) []masterdata.Version {
	masters := u.masterProvider.MasterDataMasters()
	if masters == nil {
		return []masterdata.Version{}
	}

	return sortedVersionsByReleasedAt(masters.Versions)
}

// GetGenres はゲームの正規表示順（SortOrder昇順）のジャンル一覧を返します。
func (u *masterDataUsecase) GetGenres(_ context.Context) []masterdata.Genre {
	masters := u.masterProvider.MasterDataMasters()
	if masters == nil {
		return []masterdata.Genre{}
	}

	return sortedGenresBySortOrder(masters.Genres)
}

// GetHonorTypes はID昇順の称号タイプ一覧を返します。
func (u *masterDataUsecase) GetHonorTypes(_ context.Context) []masterdata.Item {
	masters := u.masterProvider.MasterDataMasters()
	if masters == nil {
		return []masterdata.Item{}
	}

	return sortedByID(masters.HonorTypes, func(v master.HonorType) masterdata.Item {
		return masterdata.Item{ID: v.ID, Name: v.Name}
	})
}

// sortedGenresBySortOrder はジャンルをゲームの正規表示順（SortOrder昇順）でソートしたスライスを返します。
// 超ショート名も返すため、Item へ変換する sortedBySortOrder は使いません。
func sortedGenresBySortOrder(genres map[string]master.Genre) []masterdata.Genre {
	sorted := slices.SortedFunc(maps.Values(genres), func(a, b master.Genre) int {
		return cmp.Compare(a.SortOrder, b.SortOrder)
	})
	items := make([]masterdata.Genre, len(sorted))
	for i, g := range sorted {
		items[i] = masterdata.Genre{ID: g.ID, Name: g.Name, ShortName: g.ShortName}
	}
	return items
}

// sortedDifficultiesBySortOrder は難易度をゲームの正規表示順（SortOrder昇順）でソートした Item スライスを返します。
// SortOrder はゲーム内の表示順序（BASIC < ADVANCED < EXPERT < MASTER < ULTIMA）を表します。
func sortedDifficultiesBySortOrder(difficulties map[string]master.ChartDifficulty) []masterdata.Item {
	return sortedBySortOrder(difficulties, func(d master.ChartDifficulty) (masterdata.Item, int) {
		return masterdata.Item{ID: d.ID, Name: d.Name}, d.SortOrder
	})
}

// sortedBySortOrder はマップの値を Item に変換し、SortOrder 昇順でソートしたスライスを返します。
func sortedBySortOrder[V any](m map[string]V, toSortedItem func(V) (masterdata.Item, int)) []masterdata.Item {
	type entry struct {
		item      masterdata.Item
		sortOrder int
	}
	entries := make([]entry, 0, len(m))
	for _, v := range m {
		item, order := toSortedItem(v)
		entries = append(entries, entry{item: item, sortOrder: order})
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

// GetPermissions は権限変更の入力候補として、内部IDを公開せずマスタ順の権限名を返します。
func (u *masterDataUsecase) GetPermissions(_ context.Context) []string {
	masters := u.masterProvider.MasterDataMasters()
	if masters == nil {
		return []string{}
	}
	items := sortedByID(masters.AccountTypes, func(a master.AccountType) masterdata.Item {
		return masterdata.Item{ID: a.ID, Name: a.Name}
	})
	names := make([]string, len(items))
	for i, item := range items {
		names[i] = item.Name
	}
	return names
}

// GetNameFolders は名前順フォルダを表示順で返します。
func (u *masterDataUsecase) GetNameFolders(_ context.Context) []masterdata.NameFolder {
	masters := u.masterProvider.MasterDataMasters()
	if masters == nil {
		return []masterdata.NameFolder{}
	}
	return sortedNameFolders(masters.NameFolders)
}

// sortedNameFolders はマスタの表示順を保持した一覧を返します。
func sortedNameFolders(folders map[string]masterdata.NameFolder) []masterdata.NameFolder {
	items := make([]masterdata.NameFolder, 0, len(folders))
	for _, folder := range folders {
		items = append(items, folder)
	}
	slices.SortFunc(items, func(a, b masterdata.NameFolder) int {
		return cmp.Or(cmp.Compare(a.SortOrder, b.SortOrder), cmp.Compare(a.Code, b.Code))
	})
	return items
}
