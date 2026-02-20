package masterdata

import (
	"context"
	"fmt"
	"maps"
	"strings"

	domainmasterdata "github.com/chunisupport/chunisupport-api/internal/domain/masterdata"
	"github.com/jmoiron/sqlx"
)

// Cache は起動時にプリロードされるマスタのセットです。
type Cache struct {
	ClassEmblems         map[string]Item
	ClassEmblemBases     map[string]Item
	ClearLamps           map[string]Item
	ClearLampNamesByID   map[int]string
	ComboLamps           map[string]Item
	ComboLampNamesByID   map[int]string
	FullChains           map[string]Item
	FullChainNamesByID   map[int]string
	Slots                map[string]Item
	SlotNamesByID        map[int]string
	HonorTypes           map[string]Item
	Difficulties         map[string]Item
	DifficultyNamesByID  map[int]string
	Genres               map[string]Item
	GenreNamesByID       map[int]string
	AccountTypes         map[string]Item
	Versions             map[string]Version
	AchievementTypes     map[string]Item
	AchievementTypesByID map[int]string
	GenresByID           map[int]Item
	VersionsByID         map[int]Version
	ClearLampsByName     map[string]Item
	ComboLampsByName     map[string]Item
}

// Preload は固定値が INSERT されているマスタを読み込み、キャッシュを構築します。
func Preload(ctx context.Context, db *sqlx.DB) (*Cache, error) {
	classEmblems, err := loadSimpleMasters(ctx, db, "SELECT id, name FROM class_emblems", false)
	if err != nil {
		return nil, fmt.Errorf("failed to preload class_emblems: %w", err)
	}

	classEmblemBases, err := loadSimpleMasters(ctx, db, "SELECT id, name FROM class_emblem_bases", false)
	if err != nil {
		return nil, fmt.Errorf("failed to preload class_emblem_bases: %w", err)
	}

	clearLamps, err := loadSimpleMasters(ctx, db, "SELECT id, name FROM clear_lamp_types", true)
	if err != nil {
		return nil, fmt.Errorf("failed to preload clear_lamp_types: %w", err)
	}
	clearLampNamesByID := make(map[int]string, len(clearLamps))
	for _, item := range clearLamps {
		clearLampNamesByID[item.ID] = item.Name
	}

	comboLamps, err := loadSimpleMasters(ctx, db, "SELECT id, name FROM combo_lamp_types", true)
	if err != nil {
		return nil, fmt.Errorf("failed to preload combo_lamp_types: %w", err)
	}
	comboLampNamesByID := make(map[int]string, len(comboLamps))
	for _, item := range comboLamps {
		comboLampNamesByID[item.ID] = item.Name
	}

	fullChains, err := loadSimpleMasters(ctx, db, "SELECT id, name FROM full_chain_types", true)
	if err != nil {
		return nil, fmt.Errorf("failed to preload full_chain_types: %w", err)
	}
	fullChainNamesByID := make(map[int]string, len(fullChains))
	for _, item := range fullChains {
		fullChainNamesByID[item.ID] = item.Name
	}

	slots, err := loadSimpleMasters(ctx, db, "SELECT id, name FROM slots", true)
	if err != nil {
		return nil, fmt.Errorf("failed to preload slots: %w", err)
	}
	slotNamesByID := make(map[int]string, len(slots))
	for _, item := range slots {
		slotNamesByID[item.ID] = item.Name
	}

	honorTypes, err := loadSimpleMasters(ctx, db, "SELECT id, name FROM honor_types", true)
	if err != nil {
		return nil, fmt.Errorf("failed to preload honor_types: %w", err)
	}

	difficulties, err := loadSimpleMasters(ctx, db, "SELECT id, name FROM difficulties", false)
	if err != nil {
		return nil, fmt.Errorf("failed to preload difficulties: %w", err)
	}
	difficultyNamesByID := make(map[int]string, len(difficulties))
	for _, item := range difficulties {
		// 難易度名はデータベースの大文字表記をそのまま使用
		difficultyNamesByID[item.ID] = item.Name
	}

	genres, err := loadSimpleMasters(ctx, db, "SELECT id, name FROM genres", false)
	if err != nil {
		return nil, fmt.Errorf("failed to preload genres: %w", err)
	}
	genreNamesByID := make(map[int]string, len(genres))
	for _, item := range genres {
		genreNamesByID[item.ID] = item.Name
	}

	accountTypes, err := loadSimpleMasters(ctx, db, "SELECT id, name FROM account_types", false)
	if err != nil {
		return nil, fmt.Errorf("failed to preload account_types: %w", err)
	}

	versions, err := loadVersionMasters(ctx, db, "SELECT id, name, released_at FROM versions")
	if err != nil {
		return nil, fmt.Errorf("failed to preload versions: %w", err)
	}

	achievementTypes, err := loadSimpleMasters(ctx, db, "SELECT id, code FROM achievement_types", false)
	if err != nil {
		return nil, fmt.Errorf("failed to preload achievement_types: %w", err)
	}
	achievementTypesByID := make(map[int]string, len(achievementTypes))
	for _, item := range achievementTypes {
		achievementTypesByID[item.ID] = item.Name
	}
	genresByID := make(map[int]Item, len(genres))
	for _, item := range genres {
		genresByID[item.ID] = item
	}
	versionsByID := make(map[int]Version, len(versions))
	for _, version := range versions {
		versionsByID[int(version.ID)] = version
	}
	clearLampsByName := make(map[string]Item, len(clearLamps))
	for _, item := range clearLamps {
		clearLampsByName[strings.ToUpper(item.Name)] = item
	}
	comboLampsByName := make(map[string]Item, len(comboLamps))
	for _, item := range comboLamps {
		comboLampsByName[strings.ToUpper(item.Name)] = item
	}

	return &Cache{
		ClassEmblems:         classEmblems,
		ClassEmblemBases:     classEmblemBases,
		ClearLamps:           clearLamps,
		ClearLampNamesByID:   clearLampNamesByID,
		ComboLamps:           comboLamps,
		ComboLampNamesByID:   comboLampNamesByID,
		FullChains:           fullChains,
		FullChainNamesByID:   fullChainNamesByID,
		Slots:                slots,
		SlotNamesByID:        slotNamesByID,
		HonorTypes:           honorTypes,
		Difficulties:         difficulties,
		DifficultyNamesByID:  difficultyNamesByID,
		Genres:               genres,
		GenreNamesByID:       genreNamesByID,
		AccountTypes:         accountTypes,
		Versions:             versions,
		AchievementTypes:     achievementTypes,
		AchievementTypesByID: achievementTypesByID,
		GenresByID:           genresByID,
		VersionsByID:         versionsByID,
		ClearLampsByName:     clearLampsByName,
		ComboLampsByName:     comboLampsByName,
	}, nil
}

func loadSimpleMasters(ctx context.Context, db *sqlx.DB, query string, normalize bool) (map[string]Item, error) {
	rows, err := db.QueryxContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	masters := make(map[string]Item)
	for rows.Next() {
		var item Item
		if scanErr := rows.Scan(&item.ID, &item.Name); scanErr != nil {
			return nil, scanErr
		}
		key := item.Name
		if normalize {
			key = strings.ToLower(key)
		}
		masters[key] = item
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return masters, nil
}

func loadVersionMasters(ctx context.Context, db *sqlx.DB, query string) (map[string]Version, error) {
	rows, err := db.QueryxContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	versions := make(map[string]Version)
	for rows.Next() {
		var version Version
		if err := rows.StructScan(&version); err != nil {
			return nil, err
		}
		versions[version.Name] = version
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return versions, nil
}

// PlayerDataMasters はプレイヤーデータ登録に必要なマスタ集合を返します。
func (c *Cache) PlayerDataMasters() *domainmasterdata.PlayerDataMasters {
	if c == nil {
		return nil
	}

	return &domainmasterdata.PlayerDataMasters{
		CommonMasters: domainmasterdata.CommonMasters{
			DifficultyNamesByID: maps.Clone(c.DifficultyNamesByID),
		},
		ClassEmblems:       maps.Clone(c.ClassEmblems),
		ClassEmblemBases:   maps.Clone(c.ClassEmblemBases),
		ClearLamps:         maps.Clone(c.ClearLamps),
		ClearLampNamesByID: maps.Clone(c.ClearLampNamesByID),
		ComboLamps:         maps.Clone(c.ComboLamps),
		ComboLampNamesByID: maps.Clone(c.ComboLampNamesByID),
		FullChains:         maps.Clone(c.FullChains),
		FullChainNamesByID: maps.Clone(c.FullChainNamesByID),
		Slots:              maps.Clone(c.Slots),
		SlotNamesByID:      maps.Clone(c.SlotNamesByID),
		HonorTypes:         maps.Clone(c.HonorTypes),
		Difficulties:       maps.Clone(c.Difficulties),
	}
}

// SongMasters は楽曲更新に必要なマスタ集合を返します。
func (c *Cache) SongMasters() *domainmasterdata.SongMasters {
	if c == nil {
		return nil
	}

	return &domainmasterdata.SongMasters{
		CommonMasters: domainmasterdata.CommonMasters{
			DifficultyNamesByID: maps.Clone(c.DifficultyNamesByID),
		},
		GenreNamesByID: maps.Clone(c.GenreNamesByID),
		Genres:         maps.Clone(c.Genres),
		Difficulties:   maps.Clone(c.Difficulties),
	}
}

// GoalMasters は目標機能で必要なマスタ集合を返します。
func (c *Cache) GoalMasters() *domainmasterdata.GoalMasters {
	if c == nil {
		return nil
	}
	versionsByID := make(map[int]domainmasterdata.Version, len(c.VersionsByID))
	for id, version := range c.VersionsByID {
		versionsByID[id] = domainmasterdata.Version{
			ID:         version.ID,
			Name:       version.Name,
			ReleasedAt: version.ReleasedAt,
		}
	}
	return &domainmasterdata.GoalMasters{
		AchievementTypesByCode: maps.Clone(c.AchievementTypes),
		AchievementTypesByID:   maps.Clone(c.AchievementTypesByID),
		GenresByID:             maps.Clone(c.GenresByID),
		VersionsByID:           versionsByID,
		ClearLampsByName:       maps.Clone(c.ClearLampsByName),
		ComboLampsByName:       maps.Clone(c.ComboLampsByName),
	}
}

// GetClassEmblemNameByID はIDからClassEmblem名を取得します。
// 見つからない場合は空文字列を返します。
func (c *Cache) GetClassEmblemNameByID(id int) string {
	if c == nil {
		return ""
	}
	for _, item := range c.ClassEmblems {
		if item.ID == id {
			return item.Name
		}
	}
	return ""
}

// GetClassEmblemBaseNameByID はIDからClassEmblemBase名を取得します。
// 見つからない場合は空文字列を返します。
func (c *Cache) GetClassEmblemBaseNameByID(id int) string {
	if c == nil {
		return ""
	}
	for _, item := range c.ClassEmblemBases {
		if item.ID == id {
			return item.Name
		}
	}
	return ""
}

// GetAccountTypeNameByID はIDからAccountType名を取得します。
// 見つからない場合は"UNKNOWN"を返します。
func (c *Cache) GetAccountTypeNameByID(id int) string {
	if c == nil {
		return "UNKNOWN"
	}
	for _, item := range c.AccountTypes {
		if item.ID == id {
			return item.Name
		}
	}
	return "UNKNOWN"
}
