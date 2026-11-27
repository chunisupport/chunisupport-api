package masterdata

import (
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
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
	// ID はDB内部の参照専用で、APIには公開しません。
	ID uint8
	// DisplayID はAPI上の id として公開する識別子です。
	DisplayID  string
	Name       string
	ShortName  string
	ReleasedAt time.Time
}

// Genre はジャンルマスタの1件を表します。
type Genre struct {
	ID        int
	Name      string
	ShortName string
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
	Possessions        map[string]master.Possession
	Difficulties       map[string]master.ChartDifficulty
}

// ProfileMasterNames はプレイヤープロフィールのマスタ参照をAPI公開用の名称で表します。
type ProfileMasterNames struct {
	// ClassEmblem はクラスエンブレム名です。未設定または未知のIDでは nil です。
	ClassEmblem *string
	// ClassEmblemBase はクラスエンブレムベース名です。未設定または未知のIDでは nil です。
	ClassEmblemBase *string
	// Possession はポゼッション名です。未知のIDでは既定値の normal です。
	Possession string
}

// ProfileNames はプロフィールが保持するマスタIDを名称へ解決します。
// DBの数値IDをAPI契約に含めないため、登録直後の結果と保存済み結果の両方で同じ解決規則を使います。
func (m *PlayerDataMasters) ProfileNames(classEmblemID, classEmblemBaseID *int, possessionID int) ProfileMasterNames {
	// ポゼッションは公開契約上必ず名称を持つため、解決できない場合は既定値にします。
	names := ProfileMasterNames{Possession: entity.PossessionNameNormal}
	if m == nil {
		return names
	}
	if classEmblemID != nil {
		for _, item := range m.ClassEmblems {
			if item.ID == *classEmblemID {
				names.ClassEmblem = &item.Name
				break
			}
		}
	}
	if classEmblemBaseID != nil {
		for _, item := range m.ClassEmblemBases {
			if item.ID == *classEmblemBaseID {
				names.ClassEmblemBase = &item.Name
				break
			}
		}
	}
	for _, item := range m.Possessions {
		if item.ID == possessionID {
			names.Possession = item.Name
			break
		}
	}
	return names
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
// 目標の絞り込み属性はDBの数値IDではなく公開識別子で保存するため、各マスタは公開識別子をキーに持ちます。
type GoalMasters struct {
	AchievementTypesByCode map[string]Item
	AchievementTypesByID   map[int]string
	// Difficulties は難易度名をキーにした難易度マスタです。
	Difficulties map[string]master.ChartDifficulty
	// Genres はジャンル名をキーにしたジャンルマスタです。
	Genres map[string]master.Genre
	// VersionsByDisplayID は公開IDをキーにした稼働開始済みのバージョンマスタです。
	VersionsByDisplayID map[string]Version
	NameFoldersByCode   map[string]NameFolder
	ClearLampNamesByID  map[int]string
	ComboLampNamesByID  map[int]string
}

// MasterDataMasters はマスタデータAPIで必要になるマスタ集合です。
type MasterDataMasters struct {
	NameFolders      map[string]NameFolder
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
	Possessions      map[string]master.Possession
}

// NameFolder は楽曲名順フォルダのマスタ項目です。
type NameFolder struct {
	ID        int
	Code      string
	Name      string
	SortOrder int
}
