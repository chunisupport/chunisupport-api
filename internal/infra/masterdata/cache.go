package masterdata

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"strings"
	"sync"
	"time"

	domainmasterdata "github.com/chunisupport/chunisupport-api/internal/domain/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/master"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/jmoiron/sqlx"
)

// Cache は起動時にプリロードされるマスタのセットです。
type Cache struct {
	versionsMu           sync.RWMutex
	versionsReloadMu     sync.Mutex
	db                   *sqlx.DB
	now                  func() time.Time
	allVersions          map[string]Version
	allVersionsByID      map[int]Version
	versionsStale        bool
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
	AchievementTypes     map[string]domainmasterdata.Item
	AchievementTypesByID map[int]string
}

// namedRow は name カラムを持つマスタテーブルの行を表します。
type namedRow struct {
	ID   int    `db:"id"`
	Name string `db:"name"`
}

// achievementTypeRow は achievement_types テーブルの行を表します。
type achievementTypeRow struct {
	ID   int    `db:"id"`
	Code string `db:"code"`
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

	slotRows, err := loadNamedRows(ctx, db, "SELECT id, name FROM slots")
	if err != nil {
		return nil, fmt.Errorf("failed to preload slots: %w", err)
	}
	slots := make(map[string]master.Slot, len(slotRows))
	slotNamesByID := make(map[int]string, len(slotRows))
	for _, row := range slotRows {
		slots[strings.ToLower(row.Name)] = master.Slot{ID: row.ID, Name: row.Name}
		slotNamesByID[row.ID] = row.Name
	}

	honorTypeRows, err := loadNamedRows(ctx, db, "SELECT id, name FROM honor_types")
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

	genreRows, err := loadSortedRows(ctx, db, "SELECT id, name, sort_order FROM genres")
	if err != nil {
		return nil, fmt.Errorf("failed to preload genres: %w", err)
	}
	genres := make(map[string]master.Genre, len(genreRows))
	genreNamesByID := make(map[int]string, len(genreRows))
	for _, row := range genreRows {
		genres[row.Name] = master.Genre{ID: row.ID, Name: row.Name, SortOrder: row.SortOrder}
		genreNamesByID[row.ID] = row.Name
	}

	accountTypeRows, err := loadNamedRows(ctx, db, "SELECT id, name FROM account_types")
	if err != nil {
		return nil, fmt.Errorf("failed to preload account_types: %w", err)
	}
	accountTypes := make(map[string]master.AccountType, len(accountTypeRows))
	for _, row := range accountTypeRows {
		accountTypes[row.Name] = master.AccountType{ID: row.ID, Name: row.Name}
	}

	const versionQuery = `SELECT id, name, released_at FROM versions`
	allVersions, err := loadVersionMasters(ctx, db, versionQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to preload versions: %w", err)
	}
	allVersionsByID := make(map[int]Version, len(allVersions))
	for _, item := range allVersions {
		allVersionsByID[int(item.ID)] = item
	}
	versions, versionsByID := filterReleasedVersions(allVersions, time.Now())

	achievementTypeRows, err := loadAchievementTypeRows(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("failed to preload achievement_types: %w", err)
	}
	achievementTypes := make(map[string]domainmasterdata.Item, len(achievementTypeRows))
	achievementTypesByID := make(map[int]string, len(achievementTypeRows))
	for _, row := range achievementTypeRows {
		achievementTypes[strings.ToLower(row.Code)] = domainmasterdata.Item{ID: row.ID, Name: row.Code}
		achievementTypesByID[row.ID] = row.Code
	}

	return &Cache{
		db:                   db,
		now:                  time.Now,
		allVersions:          allVersions,
		allVersionsByID:      allVersionsByID,
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

// ReloadVersions はDBの全バージョンを読み込み、完成したスナップショットを一括で公開します。
func (c *Cache) ReloadVersions(ctx context.Context) error {
	if c == nil || c.db == nil {
		return errors.New("version cache database is not initialized")
	}
	c.versionsReloadMu.Lock()
	defer c.versionsReloadMu.Unlock()

	const query = `SELECT id, name, released_at FROM versions`
	allVersions, err := loadVersionMasters(ctx, c.db, query)
	if err != nil {
		c.versionsMu.Lock()
		c.versionsStale = true
		c.versionsMu.Unlock()
		return err
	}
	allVersionsByID := make(map[int]Version, len(allVersions))
	for _, item := range allVersions {
		allVersionsByID[int(item.ID)] = item
	}
	versions, versionsByID := filterReleasedVersions(allVersions, c.currentTime())
	c.versionsMu.Lock()
	c.allVersions = allVersions
	c.allVersionsByID = allVersionsByID
	c.Versions = versions
	c.VersionsByID = versionsByID
	c.versionsStale = false
	c.versionsMu.Unlock()
	return nil
}

// AdminVersions は未来版を含む全バージョンのスナップショットを返します。
func (c *Cache) AdminVersions() map[int]Version {
	if c == nil {
		return nil
	}
	c.reloadVersionsIfStale()
	c.versionsMu.RLock()
	defer c.versionsMu.RUnlock()
	if c.allVersionsByID == nil {
		return maps.Clone(c.VersionsByID)
	}
	return maps.Clone(c.allVersionsByID)
}

// PublicVersionsByID はJST当日までにリリースされたバージョンを返します。
func (c *Cache) PublicVersionsByID() map[int]Version {
	if c == nil {
		return nil
	}
	c.reloadVersionsIfStale()
	c.versionsMu.RLock()
	defer c.versionsMu.RUnlock()
	allVersions := maps.Clone(c.allVersions)
	if allVersions == nil {
		return maps.Clone(c.VersionsByID)
	}
	now := c.currentTime()
	_, versionsByID := filterReleasedVersions(allVersions, now)
	return versionsByID
}

func (c *Cache) reloadVersionsIfStale() {
	c.versionsMu.RLock()
	stale := c.versionsStale
	c.versionsMu.RUnlock()
	if !stale {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), info.VersionCacheReloadTimeout)
	defer cancel()
	_ = c.ReloadVersions(ctx)
}

func (c *Cache) currentTime() time.Time {
	if c.now != nil {
		return c.now()
	}
	return time.Now()
}

func filterReleasedVersions(allVersions map[string]Version, now time.Time) (map[string]Version, map[int]Version) {
	japanLoc, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		japanLoc = time.FixedZone("Asia/Tokyo", 9*60*60)
	}
	releaseDate := now.In(japanLoc).Format(time.DateOnly)
	versions := make(map[string]Version, len(allVersions))
	versionsByID := make(map[int]Version, len(allVersions))
	for name, item := range allVersions {
		if item.ReleasedAt.Format(time.DateOnly) > releaseDate {
			continue
		}
		versions[name] = item
		versionsByID[int(item.ID)] = item
	}
	return versions, versionsByID
}

// loadNamedRows は name カラムをそのまま使うマスタに限定して使用します。
func loadNamedRows(ctx context.Context, db *sqlx.DB, query string) ([]namedRow, error) {
	var rows []namedRow
	if err := db.SelectContext(ctx, &rows, query); err != nil {
		return nil, err
	}
	return rows, nil
}

// loadAchievementTypeRows は code カラムを持つ achievement_types 専用ローダです。
func loadAchievementTypeRows(ctx context.Context, db *sqlx.DB) ([]achievementTypeRow, error) {
	const query = "SELECT id, code FROM achievement_types"

	var rows []achievementTypeRow
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

func loadVersionMasters(ctx context.Context, db *sqlx.DB, query string, args ...any) (map[string]Version, error) {
	rows, err := db.QueryxContext(ctx, query, args...)
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
		DifficultyNamesByID: maps.Clone(c.DifficultyNamesByID),
		ClassEmblems:        maps.Clone(c.ClassEmblems),
		ClassEmblemBases:    maps.Clone(c.ClassEmblemBases),
		ClearLamps:          maps.Clone(c.ClearLamps),
		ClearLampNamesByID:  maps.Clone(c.ClearLampNamesByID),
		ComboLamps:          maps.Clone(c.ComboLamps),
		ComboLampNamesByID:  maps.Clone(c.ComboLampNamesByID),
		FullChains:          maps.Clone(c.FullChains),
		FullChainNamesByID:  maps.Clone(c.FullChainNamesByID),
		Slots:               maps.Clone(c.Slots),
		SlotNamesByID:       maps.Clone(c.SlotNamesByID),
		HonorTypes:          maps.Clone(c.HonorTypes),
		Difficulties:        maps.Clone(c.Difficulties),
	}
}

// SongMasters は楽曲更新に必要なマスタ集合を返します。
func (c *Cache) SongMasters() *domainmasterdata.SongMasters {
	if c == nil {
		return nil
	}

	return &domainmasterdata.SongMasters{
		DifficultyNamesByID: maps.Clone(c.DifficultyNamesByID),
		GenreNamesByID:      maps.Clone(c.GenreNamesByID),
		Genres:              maps.Clone(c.Genres),
		Difficulties:        maps.Clone(c.Difficulties),
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

	publicVersions := c.PublicVersionsByID()
	versionsByID := make(map[int]domainmasterdata.Version, len(publicVersions))
	for k, v := range publicVersions {
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

// MasterDataMasters はマスタデータAPIで必要なマスタ集合を返します。
func (c *Cache) MasterDataMasters() *domainmasterdata.MasterDataMasters {
	if c == nil {
		return nil
	}

	publicVersions := c.PublicVersionsByID()
	versionsByID := make(map[int]domainmasterdata.Version, len(publicVersions))
	for k, v := range publicVersions {
		versionsByID[k] = domainmasterdata.Version{
			ID:         v.ID,
			Name:       v.Name,
			ReleasedAt: v.ReleasedAt,
		}
	}

	return &domainmasterdata.MasterDataMasters{
		Genres:           maps.Clone(c.Genres),
		Difficulties:     maps.Clone(c.Difficulties),
		AccountTypes:     maps.Clone(c.AccountTypes),
		Versions:         versionsByID,
		AchievementTypes: maps.Clone(c.AchievementTypes),
		ClassEmblems:     maps.Clone(c.ClassEmblems),
		ClassEmblemBases: maps.Clone(c.ClassEmblemBases),
		ClearLamps:       maps.Clone(c.ClearLamps),
		ComboLamps:       maps.Clone(c.ComboLamps),
		FullChains:       maps.Clone(c.FullChains),
		Slots:            maps.Clone(c.Slots),
		HonorTypes:       maps.Clone(c.HonorTypes),
	}
}
