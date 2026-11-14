package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindLatestUpdatedAt_ReturnsMaxAcrossSongRelatedTables(t *testing.T) {
	// songs・charts・worldsend_charts の3テーブルの更新日時のうち最大のものを返すことを、
	// 最新を持つテーブルを入れ替えた3通りで確かめる。
	// 本番の SQL には is_deleted などの絞り込みがないため、削除済み楽曲も対象に含まれる。
	tests := []struct {
		name string
		// Given: 各テーブルの更新日時。songs が最新の場合は削除済み行を最新にする
		songUpdatedAt      time.Time
		songIsDeleted      bool
		chartUpdatedAt     time.Time
		worldsendUpdatedAt time.Time
		// Then: 期待する返り値
		expected time.Time
	}{
		{
			name:               "songsが最新の場合はsongsの日時を返す（削除済み行も対象）",
			songUpdatedAt:      time.Date(2026, 4, 9, 12, 0, 0, 0, time.UTC),
			songIsDeleted:      true,
			chartUpdatedAt:     time.Date(2026, 4, 9, 11, 0, 0, 0, time.UTC),
			worldsendUpdatedAt: time.Date(2026, 4, 9, 10, 0, 0, 0, time.UTC),
			expected:           time.Date(2026, 4, 9, 12, 0, 0, 0, time.UTC),
		},
		{
			name:               "chartsが最新の場合はchartsの日時を返す",
			songUpdatedAt:      time.Date(2026, 4, 9, 10, 0, 0, 0, time.UTC),
			songIsDeleted:      false,
			chartUpdatedAt:     time.Date(2026, 4, 9, 12, 0, 0, 0, time.UTC),
			worldsendUpdatedAt: time.Date(2026, 4, 9, 11, 0, 0, 0, time.UTC),
			expected:           time.Date(2026, 4, 9, 12, 0, 0, 0, time.UTC),
		},
		{
			name:               "worldsend_chartsが最新の場合はworldsend_chartsの日時を返す",
			songUpdatedAt:      time.Date(2026, 4, 9, 10, 0, 0, 0, time.UTC),
			songIsDeleted:      false,
			chartUpdatedAt:     time.Date(2026, 4, 9, 11, 0, 0, 0, time.UTC),
			worldsendUpdatedAt: time.Date(2026, 4, 9, 12, 0, 0, 0, time.UTC),
			expected:           time.Date(2026, 4, 9, 12, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			db := setupTestDB(t)
			defer db.Close()

			_, err := db.Exec(`
				CREATE TABLE IF NOT EXISTS worldsend_charts (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					song_id INTEGER NOT NULL,
					level_star INTEGER,
					attribute TEXT,
					notes INTEGER,
					notes_designer TEXT,
					updated_at TEXT,
					FOREIGN KEY (song_id) REFERENCES songs(id) ON DELETE CASCADE
				)
			`)
			require.NoError(t, err)

			songIsDeleted := 0
			if tt.songIsDeleted {
				songIsDeleted = 1
			}
			_, err = db.Exec(`
				INSERT INTO songs (id, display_id, title, artist, genre_id, bpm, released_at, official_idx, jacket, is_worldsend, is_new, is_deleted, updated_at)
				VALUES
					(1, 'DISPLAY001', 'Song 1', 'Artist 1', 1, 180, NULL, 'IDX001', NULL, 0, 0, ?, ?),
					(2, 'WORLD001', 'World Song', 'Artist W', 1, 200, NULL, 'IDX002', NULL, 1, 0, 0, ?)
			`, songIsDeleted, tt.songUpdatedAt, time.Date(2026, 4, 9, 9, 0, 0, 0, time.UTC))
			require.NoError(t, err)

			_, err = db.Exec(`
				INSERT INTO charts (song_id, difficulty_id, const, is_const_unknown, notes, notes_designer, updated_at)
				VALUES
					(1, 4, 13.8, 0, 1050, '譜面作者', ?)
			`, tt.chartUpdatedAt)
			require.NoError(t, err)

			_, err = db.Exec(`
				INSERT INTO worldsend_charts (song_id, level_star, attribute, notes, notes_designer, updated_at)
				VALUES
					(2, 3, '光', 999, 'WORLD''S END譜面作者', ?)
			`, tt.worldsendUpdatedAt)
			require.NoError(t, err)

			repo := &songRepository{db: db}

			// When
			result, err := repo.FindLatestUpdatedAt(context.Background(), db)

			// Then
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.True(t, tt.expected.Equal(*result))
		})
	}
}

func TestFindLatestUpdatedAt_ReturnsNilWhenNoUpdatedAtExists(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS worldsend_charts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			song_id INTEGER NOT NULL,
			level_star INTEGER,
			attribute TEXT,
			notes INTEGER,
			notes_designer TEXT,
			updated_at TEXT,
			FOREIGN KEY (song_id) REFERENCES songs(id) ON DELETE CASCADE
		)
	`)
	require.NoError(t, err)

	repo := &songRepository{db: db}

	result, err := repo.FindLatestUpdatedAt(context.Background(), db)

	require.NoError(t, err)
	assert.Nil(t, result)
}
