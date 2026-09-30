package repository

import (
	"context"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// insertSongsForUnlockRequiredTest は要解禁フラグ検証用に、要解禁の楽曲を2件登録します。
// WORLD'S END の場合は1曲1譜面の前提を満たすため譜面も登録します。
func insertSongsForUnlockRequiredTest(t *testing.T, db *sqlx.DB, isWorldsend bool) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO songs (id, display_id, title, artist, genre_id, bpm, released_at, official_idx, jacket, is_worldsend, is_new, unlock_required, is_deleted)
		VALUES
			(1, 'DISPLAY001', 'Song 1', 'Artist', 1, 180, NULL, 'IDX001', NULL, ?, 0, 1, 0),
			(2, 'DISPLAY002', 'Song 2', 'Artist', 1, 180, NULL, 'IDX002', NULL, ?, 0, 1, 0)
	`, isWorldsend, isWorldsend)
	require.NoError(t, err)
	if isWorldsend {
		_, err = db.Exec(`
			INSERT INTO worldsend_charts (id, song_id, level_star, attribute, notes, notes_designer)
			VALUES (101, 1, 4, '狂', 1200, NULL), (102, 2, 3, '光', 1000, NULL)
		`)
		require.NoError(t, err)
	}
}

// selectUnlockRequired は楽曲の unlock_required を取得します。
func selectUnlockRequired(t *testing.T, db *sqlx.DB, displayID string) bool {
	t.Helper()
	var unlockRequired bool
	require.NoError(t, db.Get(&unlockRequired, `SELECT unlock_required FROM songs WHERE display_id = ?`, displayID))
	return unlockRequired
}

// newSongForUnlockRequiredTest は更新・作成に渡す楽曲エンティティを生成します。
func newSongForUnlockRequiredTest(displayID string) *entity.Song {
	return &entity.Song{
		DisplayID: displayID,
		Title:     "Song",
		Artist:    "Artist",
		GenreID:   new(1),
		Charts:    []*entity.Chart{},
	}
}

// unlockRequiredBulkUpdateCases は一括更新における unlock_required の書き込みケースです。
// DISPLAY001 には指定値、DISPLAY002 には指定なしの楽曲を同時に渡し、
// CASE 式の引数順序と既存値の維持を同じリクエスト内で検証します。
var unlockRequiredBulkUpdateCases = []struct {
	name string
	// Given: DISPLAY001 の更新内容
	unlockRequired *bool
	// Then: 永続化される値
	expectedSong1 bool
	expectedSong2 bool
}{
	{
		name:           "指定がある楽曲は指定値で更新し、指定がない楽曲は既存値を維持する",
		unlockRequired: new(false),
		expectedSong1:  false,
		expectedSong2:  true,
	},
	{
		name:           "すべての楽曲で指定がない場合は既存値を維持する",
		unlockRequired: nil,
		expectedSong1:  true,
		expectedSong2:  true,
	},
}

func TestSongRepository_UpdateSongs_UnlockRequired(t *testing.T) {
	for _, tt := range unlockRequiredBulkUpdateCases {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			db := setupTestDB(t)
			defer db.Close()
			insertSongsForUnlockRequiredTest(t, db, false)
			repo := &songRepository{db: db}

			// When
			err := repo.UpdateSongs(context.Background(), db, []*domainrepo.SongUpdate{
				{Song: newSongForUnlockRequiredTest("DISPLAY001"), UnlockRequired: tt.unlockRequired},
				{Song: newSongForUnlockRequiredTest("DISPLAY002")},
			})

			// Then
			require.NoError(t, err)
			assert.Equal(t, tt.expectedSong1, selectUnlockRequired(t, db, "DISPLAY001"))
			assert.Equal(t, tt.expectedSong2, selectUnlockRequired(t, db, "DISPLAY002"))
		})
	}
}

func TestWorldsendChartRepository_UpdateSongs_UnlockRequired(t *testing.T) {
	for _, tt := range unlockRequiredBulkUpdateCases {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			db := setupWorldsendUpdateDB(t)
			defer db.Close()
			insertSongsForUnlockRequiredTest(t, db, true)
			repo := &worldsendChartRepository{db: db}

			// When
			err := repo.UpdateSongs(context.Background(), db, []*domainrepo.WorldsendUpdate{
				{Song: newSongForUnlockRequiredTest("DISPLAY001"), UnlockRequired: tt.unlockRequired},
				{Song: newSongForUnlockRequiredTest("DISPLAY002")},
			})

			// Then
			require.NoError(t, err)
			assert.Equal(t, tt.expectedSong1, selectUnlockRequired(t, db, "DISPLAY001"))
			assert.Equal(t, tt.expectedSong2, selectUnlockRequired(t, db, "DISPLAY002"))
		})
	}
}

func TestSongRepository_Create_WritesUnlockRequired(t *testing.T) {
	// Given
	db := setupTestDB(t)
	defer db.Close()
	repo := &songRepository{db: db}
	song := newSongForUnlockRequiredTest("DISPLAY001")
	song.OfficialIdx = "IDX001"
	song.UnlockRequired = true

	// When
	_, err := repo.Create(context.Background(), db, song)

	// Then
	require.NoError(t, err)
	assert.True(t, selectUnlockRequired(t, db, "DISPLAY001"))
}

func TestWorldsendChartRepository_CreateSong_WritesUnlockRequired(t *testing.T) {
	// Given
	db := setupWorldsendUpdateDB(t)
	defer db.Close()
	repo := &worldsendChartRepository{db: db}
	song := newSongForUnlockRequiredTest("DISPLAY001")
	song.OfficialIdx = "IDX001"
	song.IsWorldsend = true
	song.UnlockRequired = true

	// When
	_, err := repo.CreateSong(context.Background(), db, song, nil)

	// Then
	require.NoError(t, err)
	assert.True(t, selectUnlockRequired(t, db, "DISPLAY001"))
}
