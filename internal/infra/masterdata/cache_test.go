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
		`CREATE TABLE versions (id INTEGER PRIMARY KEY, name TEXT NOT NULL, released_at DATETIME NOT NULL)`,
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

	releasedAt := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)

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
