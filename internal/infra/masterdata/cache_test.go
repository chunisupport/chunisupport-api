package masterdata

import (
	"context"
	"testing"
	"time"

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
			name: "正常系: ADMIN IDの場合",
			cache: &Cache{
				AccountTypes: map[string]master.AccountType{
					"PLAYER": {ID: 1, Name: "PLAYER"},
					"EDITOR": {ID: 2, Name: "EDITOR"},
					"ADMIN":  {ID: 3, Name: "ADMIN"},
				},
			},
			id:       3,
			expected: "ADMIN",
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
		{
			name: "異常系: AccountTypesが空の場合はUNKNOWNを返す",
			cache: &Cache{
				AccountTypes: map[string]master.AccountType{},
			},
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
	tests := []struct {
		name              string
		achievementTypeID int
		achievementCode   string
	}{
		{
			name:              "achievement_types の code 列を成果種別コードとしてキャッシュできる",
			achievementTypeID: 8,
			achievementCode:   "overpower_percent",
		},
		{
			name:              "別の achievement_types.code でも同様にキャッシュできる",
			achievementTypeID: 2,
			achievementCode:   "score_count",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			db := setupPreloadSQLite(t)
			insertPreloadMasterRows(t, db, tt.achievementTypeID, tt.achievementCode)

			// When
			cache, err := Preload(context.Background(), db)

			// Then
			require.NoError(t, err)
			require.NotNil(t, cache)
			achievementType, ok := cache.AchievementTypes[tt.achievementCode]
			require.True(t, ok)
			assert.Equal(t, tt.achievementTypeID, achievementType.ID)
			assert.Equal(t, tt.achievementCode, achievementType.Name)
			assert.Equal(t, tt.achievementCode, cache.AchievementTypesByID[tt.achievementTypeID])

			goalMasters := cache.GoalMasters()
			require.NotNil(t, goalMasters)
			assert.Equal(t, tt.achievementCode, goalMasters.AchievementTypesByID[tt.achievementTypeID])
		})
	}
}

func setupPreloadSQLite(t *testing.T) *sqlx.DB {
	t.Helper()

	db, err := sqlx.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, db.Close())
	})

	schema := []string{
		`CREATE TABLE class_emblems (id INTEGER PRIMARY KEY, name TEXT NOT NULL, sort_order INTEGER NOT NULL)`,
		`CREATE TABLE class_emblem_bases (id INTEGER PRIMARY KEY, name TEXT NOT NULL, sort_order INTEGER NOT NULL)`,
		`CREATE TABLE clear_lamp_types (id INTEGER PRIMARY KEY, name TEXT NOT NULL, sort_order INTEGER NOT NULL)`,
		`CREATE TABLE combo_lamp_types (id INTEGER PRIMARY KEY, name TEXT NOT NULL, sort_order INTEGER NOT NULL)`,
		`CREATE TABLE full_chain_types (id INTEGER PRIMARY KEY, name TEXT NOT NULL, sort_order INTEGER NOT NULL)`,
		`CREATE TABLE slots (id INTEGER PRIMARY KEY, name TEXT NOT NULL)`,
		`CREATE TABLE honor_types (id INTEGER PRIMARY KEY, name TEXT NOT NULL)`,
		`CREATE TABLE difficulties (id INTEGER PRIMARY KEY, name TEXT NOT NULL, sort_order INTEGER NOT NULL)`,
		`CREATE TABLE genres (id INTEGER PRIMARY KEY, name TEXT NOT NULL, sort_order INTEGER NOT NULL)`,
		`CREATE TABLE account_types (id INTEGER PRIMARY KEY, name TEXT NOT NULL)`,
		`CREATE TABLE versions (id INTEGER PRIMARY KEY, name TEXT NOT NULL, released_at DATE NOT NULL)`,
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
		{query: `INSERT INTO difficulties (id, name, sort_order) VALUES (?, ?, ?)`, args: []any{1, "MASTER", 1}},
		{query: `INSERT INTO genres (id, name, sort_order) VALUES (?, ?, ?)`, args: []any{1, "POPS&ANIME", 1}},
		{query: `INSERT INTO account_types (id, name) VALUES (?, ?)`, args: []any{1, "PLAYER"}},
		{query: `INSERT INTO versions (id, name, released_at) VALUES (?, ?, ?)`, args: []any{1, "VERSE", releasedAt}},
		{query: `INSERT INTO achievement_types (id, code) VALUES (?, ?)`, args: []any{achievementTypeID, achievementCode}},
	}

	for _, stmt := range statements {
		_, err := db.Exec(stmt.query, stmt.args...)
		require.NoError(t, err)
	}
}

func TestLoadVersionMasters_FixedBaseDate(t *testing.T) {
	// Given: 基準日を 2026-06-22 に固定し、過去版・当日版・未リリース版の3行を投入
	baseDate := time.Date(2026, 6, 22, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name         string
		releasedAt   time.Time
		expectLoaded bool
	}{
		{
			name:         "過去版: 基準日の前日は読み込まれる",
			releasedAt:   baseDate.AddDate(0, 0, -1),
			expectLoaded: true,
		},
		{
			name:         "当日版: 基準日当日は読み込まれる",
			releasedAt:   baseDate,
			expectLoaded: true,
		},
		{
			name:         "未リリース版: 基準日の翌日は読み込まれない",
			releasedAt:   baseDate.AddDate(0, 0, 1),
			expectLoaded: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: SQLite にバージョン行を投入
			db := setupPreloadSQLite(t)
			_, err := db.Exec(`INSERT INTO versions (id, name, released_at) VALUES (?, ?, ?)`, 1, "TEST_VER", tt.releasedAt.Format(time.DateOnly))
			require.NoError(t, err)

			// When: 基準日を引数として loadVersionMasters を呼び出す
			query := `SELECT id, name, released_at FROM versions WHERE released_at <= ?`
			releaseDate := baseDate.Format(time.DateOnly)
			versions, err := loadVersionMasters(context.Background(), db, query, releaseDate)

			// Then
			require.NoError(t, err)
			if tt.expectLoaded {
				assert.Len(t, versions, 1)
				_, ok := versions["TEST_VER"]
				assert.True(t, ok)
			} else {
				assert.Empty(t, versions)
			}
		})
	}
}

func TestPreload_ExcludesFutureVersions(t *testing.T) {
	// Given: 3種類のバージョン（過去版・当日版・未リリース版）を投入
	// Preload と同じ Asia/Tokyo の当日を基準日に使う
	japanLoc, err := time.LoadLocation("Asia/Tokyo")
	require.NoError(t, err)
	today := time.Now().In(japanLoc)
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
		{query: `INSERT INTO difficulties (id, name, sort_order) VALUES (?, ?, ?)`, args: []any{1, "MASTER", 1}},
		{query: `INSERT INTO genres (id, name, sort_order) VALUES (?, ?, ?)`, args: []any{1, "POPS&ANIME", 1}},
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
		_, err := db.Exec(`INSERT INTO versions (id, name, released_at) VALUES (?, ?, ?)`, v.id, v.name, v.releasedAt)
		require.NoError(t, err)
	}
}
