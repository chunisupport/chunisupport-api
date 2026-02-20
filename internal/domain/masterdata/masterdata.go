package masterdata

import "time"

// Item は単一のマスタ項目を表します。
type Item struct {
	ID   int
	Name string
}

// Version はバージョンマスタの1件を表します。
type Version struct {
	ID         uint8
	Name       string
	ReleasedAt time.Time
}

// CommonMasters は複数の集約で共有されるマスタ集合です。
type CommonMasters struct {
	DifficultyNamesByID map[int]string
}

// PlayerDataMasters はプレイヤーデータ登録で必要になるマスタ集合です。
type PlayerDataMasters struct {
	CommonMasters
	ClassEmblems       map[string]Item
	ClassEmblemBases   map[string]Item
	ClearLamps         map[string]Item
	ClearLampNamesByID map[int]string
	ComboLamps         map[string]Item
	ComboLampNamesByID map[int]string
	FullChains         map[string]Item
	FullChainNamesByID map[int]string
	Slots              map[string]Item
	SlotNamesByID      map[int]string
	HonorTypes         map[string]Item
	Difficulties       map[string]Item
}

// SongMasters は楽曲関連で必要になるマスタ集合です。
type SongMasters struct {
	CommonMasters
	Genres         map[string]Item
	GenreNamesByID map[int]string
	Difficulties   map[string]Item
}

// GoalMasters は目標機能で必要になるマスタ集合です。
type GoalMasters struct {
	AchievementTypesByCode map[string]Item
	AchievementTypesByID   map[int]string
	DifficultyNamesByID    map[int]string
	GenreNamesByID         map[int]string
	VersionsByID           map[int]Version
	ClearLampNamesByID     map[int]string
	ComboLampNamesByID     map[int]string
}
