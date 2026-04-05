package masterdata

import (
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo/master"
)

// Item は単一のマスタ項目を表します。
// sort_order を持たないマスタ（achievement_types など）への汎用型として使用します。
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
	ClassEmblems       map[string]master.ClassEmblem
	ClassEmblemBases   map[string]master.ClassEmblemBase
	ClearLamps         map[string]master.ClearLampType
	ClearLampNamesByID map[int]string
	ComboLamps         map[string]master.ComboLampType
	ComboLampNamesByID map[int]string
	FullChains         map[string]master.FullChainType
	FullChainNamesByID map[int]string
	Slots              map[string]master.Slot
	SlotNamesByID      map[int]string
	HonorTypes         map[string]master.HonorType
	Difficulties       map[string]master.ChartDifficulty
}

// SongMasters は楽曲関連で必要になるマスタ集合です。
type SongMasters struct {
	CommonMasters
	Genres         map[string]master.Genre
	GenreNamesByID map[int]string
	Difficulties   map[string]master.ChartDifficulty
}

// DifficultySortOrderByID は難易度ID→SortOrderのマップを返します。
// レコード一覧をゲームの正規表示順（BASIC < ADVANCED < EXPERT < MASTER < ULTIMA）でソートする際に使用します。
func (m *SongMasters) DifficultySortOrderByID() map[int]int {
	if m == nil || len(m.Difficulties) == 0 {
		return nil
	}
	result := make(map[int]int, len(m.Difficulties))
	for _, d := range m.Difficulties {
		result[d.ID] = d.SortOrder
	}
	return result
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

// MasterDataMasters はマスタデータAPIで必要になるマスタ集合です。
type MasterDataMasters struct {
	Genres           map[string]master.Genre
	Difficulties     map[string]master.ChartDifficulty
	AccountTypes     map[string]master.AccountType
	Versions         map[int]Version
	AchievementTypes map[string]Item
	ClassEmblems     map[string]master.ClassEmblem
	ClassEmblemBases map[string]master.ClassEmblemBase
	ClearLamps       map[string]master.ClearLampType
	ComboLamps       map[string]master.ComboLampType
	FullChains       map[string]master.FullChainType
	Slots            map[string]master.Slot
	HonorTypes       map[string]master.HonorType
}
