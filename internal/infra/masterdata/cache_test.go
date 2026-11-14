package masterdata

import (
	"context"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/gamedate"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/master"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestCache_GetAccountTypeNameByID(t *testing.T) {
	tests := []struct {
		name     string
		cache    *Cache
		id       int
		expected string
	}{
		{
			name: "正常系: 存在するIDの場合は名前を返す",
			cache: &Cache{
				AccountTypes: map[string]master.AccountType{
					"PLAYER": {ID: 1, Name: "PLAYER"},
					"EDITOR": {ID: 2, Name: "EDITOR"},
					"ADMIN":  {ID: 3, Name: "ADMIN"},
				},
			},
			id:       1,
			expected: "PLAYER",
		},
		{
			name: "異常系: 存在しないIDの場合はUNKNOWNを返す",
			cache: &Cache{
				AccountTypes: map[string]master.AccountType{
					"PLAYER": {ID: 1, Name: "PLAYER"},
				},
			},
			id:       999,
			expected: "UNKNOWN",
		},
		{
			name:     "異常系: キャッシュがnilの場合はUNKNOWNを返す",
			cache:    nil,
			id:       1,
			expected: "UNKNOWN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cache.GetAccountTypeNameByID(tt.id)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCache_GetClassEmblemNameByID(t *testing.T) {
	tests := []struct {
		name     string
		cache    *Cache
		id       int
		expected string
	}{
		{
			name: "正常系: 存在するIDの場合は名前を返す",
			cache: &Cache{
				ClassEmblems: map[string]master.ClassEmblem{
					"Bronze": {ID: 1, Name: "Bronze"},
					"Silver": {ID: 2, Name: "Silver"},
				},
			},
			id:       1,
			expected: "Bronze",
		},
		{
			name: "異常系: 存在しないIDの場合は空文字を返す",
			cache: &Cache{
				ClassEmblems: map[string]master.ClassEmblem{
					"Bronze": {ID: 1, Name: "Bronze"},
				},
			},
			id:       999,
			expected: "",
		},
		{
			name:     "異常系: キャッシュがnilの場合は空文字を返す",
			cache:    nil,
			id:       1,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cache.GetClassEmblemNameByID(tt.id)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCache_GetClassEmblemBaseNameByID(t *testing.T) {
	tests := []struct {
		name     string
		cache    *Cache
		id       int
		expected string
	}{
		{
			name: "正常系: 存在するIDの場合は名前を返す",
			cache: &Cache{
				ClassEmblemBases: map[string]master.ClassEmblemBase{
					"I":   {ID: 1, Name: "I"},
					"II":  {ID: 2, Name: "II"},
					"III": {ID: 3, Name: "III"},
				},
			},
			id:       2,
			expected: "II",
		},
		{
			name: "異常系: 存在しないIDの場合は空文字を返す",
			cache: &Cache{
				ClassEmblemBases: map[string]master.ClassEmblemBase{
					"I": {ID: 1, Name: "I"},
				},
			},
			id:       999,
			expected: "",
		},
		{
			name:     "異常系: キャッシュがnilの場合は空文字を返す",
			cache:    nil,
			id:       1,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cache.GetClassEmblemBaseNameByID(tt.id)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPreload_AchievementTypesUsesCodeColumn(t *testing.T) {
	const achievementTypeID = 8
	const achievementCode = "overpower_percent"
	db := setupPreloadSQLite(t)
	insertPreloadMasterRows(t, db, achievementTypeID, achievementCode)

	cache, err := Preload(context.Background(), db)

	require.NoError(t, err)
	require.NotNil(t, cache)
	achievementType, ok := cache.AchievementTypes[achievementCode]
	require.True(t, ok)
	assert.Equal(t, achievementTypeID, achievementType.ID)
	assert.Equal(t, achievementCode, achievementType.Name)
	assert.Equal(t, achievementCode, cache.AchievementTypesByID[achievementTypeID])
	possession, ok := cache.Possessions["normal"]
	require.True(t, ok)
	assert.Equal(t, 1, possession.ID)

	goalMasters := cache.GoalMasters()
	require.NotNil(t, goalMasters)
	assert.Equal(t, achievementCode, goalMasters.AchievementTypesByID[achievementTypeID])
}

func TestPreload_ジャンルとバージョンの超ショート名を読み込む(t *testing.T) {
	// Given
	db := setupPreloadSQLite(t)
	insertPreloadMasterRows(t, db, 1, "score_count")

	// When
	cache, err := Preload(context.Background(), db)

	// Then
	require.NoError(t, err)
	assert.Equal(t, "P&A", cache.Genres["POPS&ANIME"].ShortName)
	masterDataMasters := cache.MasterDataMasters()
	require.NotNil(t, masterDataMasters)
	assert.Equal(t, "P&A", masterDataMasters.Genres["POPS&ANIME"].ShortName)
	assert.Equal(t, "VRS", masterDataMasters.Versions[1].ShortName)
	goalMasters := cache.GoalMasters()
	require.NotNil(t, goalMasters)
	assert.Equal(t, "VRS", goalMasters.VersionsByID[1].ShortName)
}

func setupPreloadSQLite(t *testing.T) *sqlx.DB {
	t.Helper()

	db, err := sqlx.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, db.Close())
	})

	schema := []string{
		`CREATE TABLE name_folders (id INTEGER PRIMARY KEY, code TEXT NOT NULL, name TEXT NOT NULL, sort_order INTEGER NOT NULL)`,
		`CREATE TABLE class_emblems (id INTEGER PRIMARY KEY, name TEXT NOT NULL, sort_order INTEGER NOT NULL)`,
		`CREATE TABLE class_emblem_bases (id INTEGER PRIMARY KEY, name TEXT NOT NULL, sort_order INTEGER NOT NULL)`,
		`CREATE TABLE clear_lamp_types (id INTEGER PRIMARY KEY, name TEXT NOT NULL, sort_order INTEGER NOT NULL)`,
		`CREATE TABLE combo_lamp_types (id INTEGER PRIMARY KEY, name TEXT NOT NULL, sort_order INTEGER NOT NULL)`,
		`CREATE TABLE full_chain_types (id INTEGER PRIMARY KEY, name TEXT NOT NULL, sort_order INTEGER NOT NULL)`,
		`CREATE TABLE slots (id INTEGER PRIMARY KEY, name TEXT NOT NULL)`,
		`CREATE TABLE honor_types (id INTEGER PRIMARY KEY, name TEXT NOT NULL)`,
		`CREATE TABLE possessions (id INTEGER PRIMARY KEY, name TEXT NOT NULL)`,
		`CREATE TABLE difficulties (id INTEGER PRIMARY KEY, name TEXT NOT NULL, sort_order INTEGER NOT NULL)`,
		`CREATE TABLE genres (id INTEGER PRIMARY KEY, name TEXT NOT NULL, short_name TEXT NOT NULL, sort_order INTEGER NOT NULL)`,
		`CREATE TABLE account_types (id INTEGER PRIMARY KEY, name TEXT NOT NULL)`,
		`CREATE TABLE versions (id INTEGER PRIMARY KEY, name TEXT NOT NULL, short_name TEXT NOT NULL, released_at DATE NOT NULL)`,
		`CREATE TABLE achievement_types (id INTEGER PRIMARY KEY, code TEXT NOT NULL)`,
	}

	for _, stmt := range schema {
		_, err := db.Exec(stmt)
		require.NoError(t, err)
	}

	return db
}

func insertPreloadMasterRows(t *testing.T, db *sqlx.DB, achievementTypeID int, achievementCode string) {
	t.Helper()

	releasedAt := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC).Format(time.DateOnly)

	statements := []struct {
		query string
		args  []any
	}{
		{query: `INSERT INTO class_emblems (id, name, sort_order) VALUES (?, ?, ?)`, args: []any{1, "Bronze", 1}},
		{query: `INSERT INTO class_emblem_bases (id, name, sort_order) VALUES (?, ?, ?)`, args: []any{1, "I", 1}},
		{query: `INSERT INTO clear_lamp_types (id, name, sort_order) VALUES (?, ?, ?)`, args: []any{1, "clear", 1}},
		{query: `INSERT INTO combo_lamp_types (id, name, sort_order) VALUES (?, ?, ?)`, args: []any{1, "fullcombo", 1}},
		{query: `INSERT INTO full_chain_types (id, name, sort_order) VALUES (?, ?, ?)`, args: []any{1, "fullchain", 1}},
		{query: `INSERT INTO slots (id, name) VALUES (?, ?)`, args: []any{1, "main"}},
		{query: `INSERT INTO honor_types (id, name) VALUES (?, ?)`, args: []any{1, "normal"}},
		{query: `INSERT INTO possessions (id, name) VALUES (?, ?)`, args: []any{1, "normal"}},
		{query: `INSERT INTO difficulties (id, name, sort_order) VALUES (?, ?, ?)`, args: []any{1, "MASTER", 1}},
		{query: `INSERT INTO genres (id, name, short_name, sort_order) VALUES (?, ?, ?, ?)`, args: []any{1, "POPS&ANIME", "P&A", 1}},
		{query: `INSERT INTO account_types (id, name) VALUES (?, ?)`, args: []any{1, "PLAYER"}},
		{query: `INSERT INTO versions (id, name, short_name, released_at) VALUES (?, ?, ?, ?)`, args: []any{1, "VERSE", "VRS", releasedAt}},
		{query: `INSERT INTO achievement_types (id, code) VALUES (?, ?)`, args: []any{achievementTypeID, achievementCode}},
	}

	for _, stmt := range statements {
		_, err := db.Exec(stmt.query, stmt.args...)
		require.NoError(t, err)
	}
}

func TestPreload_ExcludesFutureVersions(t *testing.T) {
	// Given: 3種類のバージョン（過去版・当日版・未リリース版）を投入
	// 当日はCHUNITHM日付（日本時間7時切り替え）に合わせる。暦日の0時を基準にすると、7時前の実行で当日版が未公開になる。
	today := gamedate.Today(time.Now())
	pastDate := today.AddDate(0, 0, -1)
	futureDate := today.AddDate(0, 0, 1)

	db := setupPreloadSQLite(t)
	insertVersionRows(t, db, pastDate, today, futureDate)

	// When
	cache, err := Preload(context.Background(), db)

	// Then: 未リリース版がキャッシュに含まれていないことを確認
	require.NoError(t, err)
	require.NotNil(t, cache)

	_, pastOK := cache.Versions["PAST_VER"]
	assert.True(t, pastOK, "過去版は Versions に含まれるべき")
	_, todayOK := cache.Versions["TODAY_VER"]
	assert.True(t, todayOK, "当日版は Versions に含まれるべき")
	_, futureOK := cache.Versions["FUTURE_VER"]
	assert.False(t, futureOK, "未リリース版は Versions に含まれるべき")

	_, pastByIDOK := cache.VersionsByID[1]
	assert.True(t, pastByIDOK, "過去版は VersionsByID に含まれるべき")
	_, todayByIDOK := cache.VersionsByID[2]
	assert.True(t, todayByIDOK, "当日版は VersionsByID に含まれるべき")
	_, futureByIDOK := cache.VersionsByID[3]
	assert.False(t, futureByIDOK, "未リリース版は VersionsByID に含まれるべき")

	// MasterDataMasters でも未リリース版が除外されていることを確認
	masterDataMasters := cache.MasterDataMasters()
	require.NotNil(t, masterDataMasters)
	_, masterPastOK := masterDataMasters.Versions[1]
	assert.True(t, masterPastOK, "過去版は MasterDataMasters.Versions に含まれるべき")
	_, masterTodayOK := masterDataMasters.Versions[2]
	assert.True(t, masterTodayOK, "当日版は MasterDataMasters.Versions に含まれるべき")
	_, masterFutureOK := masterDataMasters.Versions[3]
	assert.False(t, masterFutureOK, "未リリース版は MasterDataMasters.Versions に含まれるべき")

	// GoalMasters でも未リリース版が除外されていることを確認
	goalMasters := cache.GoalMasters()
	require.NotNil(t, goalMasters)
	_, goalPastOK := goalMasters.VersionsByID[1]
	assert.True(t, goalPastOK, "過去版は GoalMasters.VersionsByID に含まれるべき")
	_, goalTodayOK := goalMasters.VersionsByID[2]
	assert.True(t, goalTodayOK, "当日版は GoalMasters.VersionsByID に含まれるべき")
	_, goalFutureOK := goalMasters.VersionsByID[3]
	assert.False(t, goalFutureOK, "未リリース版は GoalMasters.VersionsByID に含まれるべき")
}

func TestCache_PublicVersionsByID_稼働日のJST7時で未来版を切り替える(t *testing.T) {
	jst := time.FixedZone("Asia/Tokyo", 9*60*60)
	future := Version{ID: 2, Name: "CHUNITHM FUTURE", ReleasedAt: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)}
	tests := []struct {
		name    string
		now     time.Time
		wantPub bool
	}{
		{
			name:    "稼働日の06時59分59秒は未公開",
			now:     time.Date(2026, 1, 2, 6, 59, 59, 0, jst),
			wantPub: false,
		},
		{
			name:    "稼働日の07時00分00秒から公開",
			now:     time.Date(2026, 1, 2, 7, 0, 0, 0, jst),
			wantPub: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := &Cache{
				now: func() time.Time { return tt.now },
				allVersions: map[string]Version{
					future.Name: future,
				},
				allVersionsByID: map[int]Version{int(future.ID): future},
			}

			_, published := cache.PublicVersionsByID()[2]
			assert.Equal(t, tt.wantPub, published)
			assert.Contains(t, cache.AdminVersions(), 2)
		})
	}
}

func TestCache_PublicVersionsByID_古い状態ならDBから再読込する(t *testing.T) {
	db, err := sqlx.Open("sqlite", ":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	_, err = db.Exec(`CREATE TABLE versions (id INTEGER PRIMARY KEY, name TEXT NOT NULL, short_name TEXT NOT NULL, released_at DATE NOT NULL)`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO versions (id, name, short_name, released_at) VALUES (?, ?, ?, ?)`, 1, "CHUNITHM VERSE", "VRS", time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	cache := &Cache{db: db, now: time.Now, allVersions: map[string]Version{}, allVersionsByID: map[int]Version{}, versionsStale: true}

	versions := cache.PublicVersionsByID()

	require.Contains(t, versions, 1)
	assert.Equal(t, "CHUNITHM VERSE", versions[1].Name)
	assert.Equal(t, "VRS", versions[1].ShortName)
	assert.False(t, cache.versionsStale)
}

// TestCache_PublicVersionsByID_stale再読込失敗時は旧値を保持し回復後に新版が見える は、
// 再読込失敗時に旧スナップショットを維持し、staleを保持したまま次回回復できることを公開メソッドの返却値で保証します。
// 失敗時の空値上書き（旧版が見えなくなる変異）と失敗時のstale解除（回復後に新版が見えなくなる変異）をどちらも検知します。
func TestCache_PublicVersionsByID_stale再読込失敗時は旧値を保持し回復後に新版が見える(t *testing.T) {
	// Given: 公開済みの旧版ID1だけを持つDBと、そのスナップショットを持つキャッシュを用意する。
	// 時刻は固定し、実時計やsleepは使わない。判定は返却値で行う。
	jst := time.FixedZone("Asia/Tokyo", 9*60*60)
	fixedNow := time.Date(2026, 1, 2, 7, 0, 0, 0, jst)
	releasedOld := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	releasedNew := time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)
	db, err := sqlx.Open("sqlite", ":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	_, err = db.Exec(`CREATE TABLE versions (id INTEGER PRIMARY KEY, name TEXT NOT NULL, short_name TEXT NOT NULL, released_at DATE NOT NULL)`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO versions (id, name, short_name, released_at) VALUES (?, ?, ?, ?)`, 1, "CHUNITHM VERSE", "VRS", releasedOld)
	require.NoError(t, err)
	oldVersion := Version{ID: 1, Name: "CHUNITHM VERSE", ShortName: "VRS", ReleasedAt: releasedOld}
	cache := &Cache{
		db:              db,
		now:             func() time.Time { return fixedNow },
		allVersions:     map[string]Version{oldVersion.Name: oldVersion},
		allVersionsByID: map[int]Version{int(oldVersion.ID): oldVersion},
		Versions:        map[string]Version{oldVersion.Name: oldVersion},
		VersionsByID:    map[int]Version{int(oldVersion.ID): oldVersion},
	}
	got := cache.PublicVersionsByID()
	require.Contains(t, got, 1)
	require.Equal(t, "CHUNITHM VERSE", got[1].Name)

	// When: DB障害を起こしてstale化し、公開メソッドから再読込させる。
	// 基点にMarkVersionsStale相当の公開メソッドは無いため、既存成功テストと同様にstale化を直接表す。
	_, err = db.Exec(`DROP TABLE versions`)
	require.NoError(t, err)
	cache.versionsStale = true
	failed := cache.PublicVersionsByID()

	// Then: 再読込に失敗しても旧スナップショットが維持される。空値上書きの変異ではここが空になる。
	require.Contains(t, failed, 1)
	assert.Equal(t, "CHUNITHM VERSE", failed[1].Name)
	assert.Equal(t, "VRS", failed[1].ShortName)
	assert.NotContains(t, failed, 2)
	assert.Len(t, failed, 1)

	// When: DBを復旧して新版ID2を追加し、次回参照で再読込させる。
	_, err = db.Exec(`CREATE TABLE versions (id INTEGER PRIMARY KEY, name TEXT NOT NULL, short_name TEXT NOT NULL, released_at DATE NOT NULL)`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO versions (id, name, short_name, released_at) VALUES (?, ?, ?, ?)`, 1, "CHUNITHM VERSE", "VRS", releasedOld)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO versions (id, name, short_name, released_at) VALUES (?, ?, ?, ?)`, 2, "CHUNITHM NEW", "NEW", releasedNew)
	require.NoError(t, err)
	recovered := cache.PublicVersionsByID()

	// Then: staleが保持されていたため再読込され、新旧両版が見える。失敗時にstale解除する変異ではID2が見えない。
	require.Contains(t, recovered, 1)
	require.Contains(t, recovered, 2)
	assert.Equal(t, "CHUNITHM VERSE", recovered[1].Name)
	assert.Equal(t, "VRS", recovered[1].ShortName)
	assert.Equal(t, "CHUNITHM NEW", recovered[2].Name)
	assert.Equal(t, "NEW", recovered[2].ShortName)
}

func insertVersionRows(t *testing.T, db *sqlx.DB, pastDate, today, futureDate time.Time) {
	t.Helper()

	// 他のマスタテーブルも投入（Preload が通るように）
	minimalMasters := []struct {
		query string
		args  []any
	}{
		{query: `INSERT INTO class_emblems (id, name, sort_order) VALUES (?, ?, ?)`, args: []any{1, "Bronze", 1}},
		{query: `INSERT INTO class_emblem_bases (id, name, sort_order) VALUES (?, ?, ?)`, args: []any{1, "I", 1}},
		{query: `INSERT INTO clear_lamp_types (id, name, sort_order) VALUES (?, ?, ?)`, args: []any{1, "clear", 1}},
		{query: `INSERT INTO combo_lamp_types (id, name, sort_order) VALUES (?, ?, ?)`, args: []any{1, "fullcombo", 1}},
		{query: `INSERT INTO full_chain_types (id, name, sort_order) VALUES (?, ?, ?)`, args: []any{1, "fullchain", 1}},
		{query: `INSERT INTO slots (id, name) VALUES (?, ?)`, args: []any{1, "main"}},
		{query: `INSERT INTO honor_types (id, name) VALUES (?, ?)`, args: []any{1, "normal"}},
		{query: `INSERT INTO possessions (id, name) VALUES (?, ?)`, args: []any{1, "normal"}},
		{query: `INSERT INTO difficulties (id, name, sort_order) VALUES (?, ?, ?)`, args: []any{1, "MASTER", 1}},
		{query: `INSERT INTO genres (id, name, short_name, sort_order) VALUES (?, ?, ?, ?)`, args: []any{1, "POPS&ANIME", "P&A", 1}},
		{query: `INSERT INTO account_types (id, name) VALUES (?, ?)`, args: []any{1, "PLAYER"}},
		{query: `INSERT INTO achievement_types (id, code) VALUES (?, ?)`, args: []any{1, "score_count"}},
	}

	for _, stmt := range minimalMasters {
		_, err := db.Exec(stmt.query, stmt.args...)
		require.NoError(t, err)
	}

	// 過去版・当日版・未リリース版のバージョンを投入
	versionStatements := []struct {
		id         int
		name       string
		releasedAt string
	}{
		{id: 1, name: "PAST_VER", releasedAt: pastDate.Format(time.DateOnly)},
		{id: 2, name: "TODAY_VER", releasedAt: today.Format(time.DateOnly)},
		{id: 3, name: "FUTURE_VER", releasedAt: futureDate.Format(time.DateOnly)},
	}

	for _, v := range versionStatements {
		_, err := db.Exec(`INSERT INTO versions (id, name, short_name, released_at) VALUES (?, ?, ?, ?)`, v.id, v.name, v.name, v.releasedAt)
		require.NoError(t, err)
	}
}
