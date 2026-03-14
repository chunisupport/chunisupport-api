package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadMasterData_不正な譜面定数値ならエラーを返す(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "負の譜面定数値なら変換エラーを返す",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			db := setupTestDB(t)
			defer db.Close()

			_, err := db.Exec(`
				CREATE TABLE IF NOT EXISTS worldsend_charts (
					id INTEGER PRIMARY KEY,
					song_id INTEGER NOT NULL
				)
			`)
			require.NoError(t, err)

			_, err = db.Exec(`
				INSERT INTO songs (id, display_id, title, artist, genre_id, bpm, official_idx, is_worldsend, is_deleted)
				VALUES (1, 'c1', 'Song 1', 'Artist 1', 1, 180, 'idx-1', 0, 0)
			`)
			require.NoError(t, err)

			_, err = db.Exec(`
				INSERT INTO charts (id, song_id, difficulty_id, const, is_const_unknown, notes)
				VALUES (1, 1, 3, -1.0, 0, 1000)
			`)
			require.NoError(t, err)

			repo := NewPlayerDataRepository(db)

			// When
			result, err := repo.LoadMasterData(context.Background(), nil, []string{"idx-1"})

			// Then
			require.Error(t, err)
			assert.Nil(t, result)
			assert.ErrorContains(t, err, "failed to convert chart model to entity")
			assert.ErrorContains(t, err, "invalid chart constant")
			assert.ErrorContains(t, err, "chart_id=1")
		})
	}
}

func TestLoadMasterData_正常な譜面定数値ならマスタを読み込める(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "譜面と楽曲をキー付きで返す",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			db := setupTestDB(t)
			defer db.Close()

			_, err := db.Exec(`
				CREATE TABLE IF NOT EXISTS worldsend_charts (
					id INTEGER PRIMARY KEY,
					song_id INTEGER NOT NULL
				)
			`)
			require.NoError(t, err)

			_, err = db.Exec(`
				INSERT INTO songs (id, display_id, title, artist, genre_id, bpm, official_idx, is_worldsend, is_deleted)
				VALUES (1, 'c1', 'Song 1', 'Artist 1', 1, 180, 'idx-1', 0, 0)
			`)
			require.NoError(t, err)

			_, err = db.Exec(`
				INSERT INTO charts (id, song_id, difficulty_id, const, is_const_unknown, notes)
				VALUES (1, 1, 3, 13.5, 0, 1000)
			`)
			require.NoError(t, err)

			repo := NewPlayerDataRepository(db)

			// When
			result, err := repo.LoadMasterData(context.Background(), nil, []string{"idx-1"})

			// Then
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Len(t, result.Songs, 1)
			require.Len(t, result.ChartsByKey, 1)
			require.Len(t, result.ChartsByID, 1)

			song, ok := result.Songs["idx-1"]
			require.True(t, ok)
			assert.Equal(t, 1, song.ID)

			chartByKey, ok := result.ChartsByKey["1:3"]
			require.True(t, ok)
			assert.Equal(t, 1, chartByKey.ID)
			assert.Equal(t, 13.5, float64(chartByKey.Const))

			chartByID, ok := result.ChartsByID[1]
			require.True(t, ok)
			assert.Equal(t, 13.5, float64(chartByID.Const))
		})
	}
}
