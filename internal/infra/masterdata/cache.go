package masterdata

import (
	"context"
	"fmt"
	"maps"
	"strings"

	domainmasterdata "github.com/chunisupport/chunisupport-api/internal/domain/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/master"
	"github.com/jmoiron/sqlx"
)

// Cache は起動時にプリロードされるマスタのセットです。
type Cache struct {
	ClassEmblems         map[string]master.ClassEmblem
	ClassEmblemBases     map[string]master.ClassEmblemBase
	ClearLamps           map[string]master.ClearLampType
	ClearLampNamesByID   map[int]string
	ComboLamps           map[string]master.ComboLampType
	ComboLampNamesByID   map[int]string
	FullChains           map[string]master.FullChainType
	FullChainNamesByID   map[int]string
	Slots                map[string]master.Slot
	SlotNamesByID        map[int]string
	HonorTypes           map[string]master.HonorType
	Difficulties         map[string]master.ChartDifficulty
	DifficultyNamesByID  map[int]string
	Genres               map[string]master.Genre
	GenreNamesByID       map[int]string
	AccountTypes         map[string]master.AccountType
	Versions             map[string]Version
	VersionsByID         map[int]Version
	AchievementTypes     map[string]Item
	AchievementTypesByID map[int]string
}

// simpleRow は sort_order を持たないマスタテーブルの行を表します。
type simpleRow struct {
	ID   int    `db:"id"`
	Name string `db:"name"`
}

// sortedRow は sort_order カラムを持つマスタテーブルの行を表します。
type sortedRow struct {
	ID        int    `db:"id"`
	Name      string `db:"name"`
	SortOrder int    `db:"sort_order"`
}

// Preload は固定値が INSERT されているマスタを読み込み、キャッシュを構築します。
func Preload(ctx context.Context, db *sqlx.DB) (*Cache, error) {
	classEmblemRows, err := loadSortedRows(ctx, db, "SELECT id, name, sort_order FROM class_emblems")
	if err != nil {
		return nil, fmt.Errorf("failed to preload class_emblems: %w", err)
	}
	classEmblems := make(map[string]master.ClassEmblem, len(classEmblemRows))
	for _, row := range classEmblemRows {
		classEmblems[row.Name] = master.ClassEmblem{ID: row.ID, Name: row.Name, SortOrder: row.SortOrder}
	}

	classEmblemBaseRows, err := loadSortedRows(ctx, db, "SELECT id, name, sort_order FROM class_emblem_bases")
	if err != nil {
		return nil, fmt.Errorf("failed to preload class_emblem_bases: %w", err)
	}
	classEmblemBases := make(map[string]master.ClassEmblemBase, len(classEmblemBaseRows))
	for _, row := range classEmblemBaseRows {
		classEmblemBases[row.Name] = master.ClassEmblemBase{ID: row.ID, Name: row.Name, SortOrder: row.SortOrder}
	}

	clearLampRows, err := loadSortedRows(ctx, db, "SELECT id, name, sort_order FROM clear_lamp_types")
	if err != nil {
		return nil, fmt.Errorf("failed to preload clear_lamp_types: %w", err)
	}
	clearLamps := make(map[string]master.ClearLampType, len(clearLampRows))
	clearLampNamesByID := make(map[int]string, len(clearLampRows))
	for _, row := range clearLampRows {
		clearLamps[strings.ToLower(row.Name)] = master.ClearLampType{ID: row.ID, Name: row.Name, SortOrder: row.SortOrder}
		clearLampNamesByID[row.ID] = row.Name
	}

	comboLampRows, err := loadSortedRows(ctx, db, "SELECT id, name, sort_order FROM combo_lamp_types")
	if err != nil {
		return nil, fmt.Errorf("failed to preload combo_lamp_types: %w", err)
	}
	comboLamps := make(map[string]master.ComboLampType, len(comboLampRows))
	comboLampNamesByID := make(map[int]string, len(comboLampRows))
	for _, row := range comboLampRows {
		comboLamps[strings.ToLower(row.Name)] = master.ComboLampType{ID: row.ID, Name: row.Name, SortOrder: row.SortOrder}
		comboLampNamesByID[row.ID] = row.Name
	}

	fullChainRows, err := loadSortedRows(ctx, db, "SELECT id, name, sort_order FROM full_chain_types")
	if err != nil {
		return nil, fmt.Errorf("failed to preload full_chain_types: %w", err)
	}
	fullChains := make(map[string]master.FullChainType, len(fullChainRows))
	fullChainNamesByID := make(map[int]string, len(fullChainRows))
	for _, row := range fullChainRows {
		fullChains[strings.ToLower(row.Name)] = master.FullChainType{ID: row.ID, Name: row.Name, SortOrder: row.SortOrder}
		fullChainNamesByID[row.ID] = row.Name
	}

	slotRows, err := loadSimpleRows(ctx, db, "SELECT id, name FROM slots")
	if err != nil {
		return nil, fmt.Errorf("failed to preload slots: %w", err)
	}
	slots := make(map[string]master.Slot, len(slotRows))
	slotNamesByID := make(map[int]string, len(slotRows))
	for _, row := range slotRows {
		slots[strings.ToLower(row.Name)] = master.Slot{ID: row.ID, Name: row.Name}
		slotNamesByID[row.ID] = row.Name
	}

	honorTypeRows, err := loadSimpleRows(ctx, db, "SELECT id, name FROM honor_types")
	if err != nil {
		return nil, fmt.Errorf("failed to preload honor_types: %w", err)
	}
	honorTypes := make(map[string]master.HonorType, len(honorTypeRows))
	for _, row := range honorTypeRows {
		honorTypes[strings.ToLower(row.Name)] = master.HonorType{ID: row.ID, Name: row.Name}
	}

	difficultyRows, err := loadSortedRows(ctx, db, "SELECT id, name, sort_order FROM difficulties")
	if err != nil {
		return nil, fmt.Errorf("failed to preload difficulties: %w", err)
	}
	difficulties := make(map[string]master.ChartDifficulty, len(difficultyRows))
	difficultyNamesByID := make(map[int]string, len(difficultyRows))
	for _, row := range difficultyRows {
		// 難易度名はデータベースの大文字表記をそのまま使用
		difficulties[row.Name] = master.ChartDifficulty{ID: row.ID, Name: row.Name, SortOrder: row.SortOrder}
		difficultyNamesByID[row.ID] = row.Name
	}

	genreRows, err := loadSimpleRows(ctx, db, "SELECT id, name FROM genres")
	if err != nil {
		return nil, fmt.Errorf("failed to preload genres: %w", err)
	}
	genres := make(map[string]master.Genre, len(genreRows))
	genreNamesByID := make(map[int]string, len(genreRows))
	for _, row := range genreRows {
		genres[row.Name] = master.Genre{ID: row.ID, Name: row.Name}
		genreNamesByID[row.ID] = row.Name
	}

	accountTypeRows, err := loadSimpleRows(ctx, db, "SELECT id, name FROM account_types")
	if err != nil {
		return nil, fmt.Errorf("failed to preload account_types: %w", err)
	}
	accountTypes := make(map[string]master.AccountType, len(accountTypeRows))
	for _, row := range accountTypeRows {
		accountTypes[row.Name] = master.AccountType{ID: row.ID, Name: row.Name}
	}

	versions, err := loadVersionMasters(ctx, db, "SELECT id, name, released_at FROM versions")
	if err != nil {
		return nil, fmt.Errorf("failed to preload versions: %w", err)
	}
	versionsByID := make(map[int]Version, len(versions))
	for _, item := range versions {
		versionsByID[int(item.ID)] = item
	}

	achievementTypeRows, err := loadSimpleRows(ctx, db, "SELECT id, code FROM achievement_types")
	if err != nil {
		return nil, fmt.Errorf("failed to preload achievement_types: %w", err)
	}
	achievementTypes := make(map[string]Item, len(achievementTypeRows))
	achievementTypesByID := make(map[int]string, len(achievementTypeRows))
	for _, row := range achievementTypeRows {
		achievementTypes[row.Name] = Item{ID: row.ID, Name: row.Name}
		achievementTypesByID[row.ID] = row.Name
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
		VersionsByID:         versionsByID,
		AchievementTypes:     achievementTypes,
		AchievementTypesByID: achievementTypesByID,
	}, nil
}

func loadSimpleRows(ctx context.Context, db *sqlx.DB, query string) ([]simpleRow, error) {
	var rows []simpleRow
	if err := db.SelectContext(ctx, &rows, query); err != nil {
		return nil, err
	}
	return rows, nil
}

func loadSortedRows(ctx context.Context, db *sqlx.DB, query string) ([]sortedRow, error) {
	var rows []sortedRow
	if err := db.SelectContext(ctx, &rows, query); err != nil {
		return nil, err
	}
	return rows, nil
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

// GoalMasters は目標機能で必要なマスタ集合を返します。
func (c *Cache) GoalMasters() *domainmasterdata.GoalMasters {
	if c == nil {
		return nil
	}

	versionsByID := make(map[int]domainmasterdata.Version, len(c.VersionsByID))
	for k, v := range c.VersionsByID {
		versionsByID[k] = domainmasterdata.Version{
			ID:         v.ID,
			Name:       v.Name,
			ReleasedAt: v.ReleasedAt,
		}
	}

	return &domainmasterdata.GoalMasters{
		AchievementTypesByCode: maps.Clone(c.AchievementTypes),
		AchievementTypesByID:   maps.Clone(c.AchievementTypesByID),
		DifficultyNamesByID:    maps.Clone(c.DifficultyNamesByID),
		GenreNamesByID:         maps.Clone(c.GenreNamesByID),
		VersionsByID:           versionsByID,
		ClearLampNamesByID:     maps.Clone(c.ClearLampNamesByID),
		ComboLampNamesByID:     maps.Clone(c.ComboLampNamesByID),
	}
}
