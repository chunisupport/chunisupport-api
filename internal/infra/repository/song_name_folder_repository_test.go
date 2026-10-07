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

func TestSongRepositoryPersistsAndRestoresNameFolderCode(t *testing.T) {
	t.Run("Create derives the name folder from reading", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		repo := &songRepository{db: db}
		song := &entity.Song{
			DisplayID:   "CREATE001",
			Title:       "Unrelated title",
			Reading:     stringPtrForNameFolderTest("ココロ"),
			Artist:      "Artist",
			GenreID:     intPtrForNameFolderTest(1),
			OfficialIdx: "IDX-CREATE001",
		}

		created, err := repo.Create(context.Background(), db, song)

		require.NoError(t, err)
		assert.Equal(t, "KA", created.NameFolderCode)
		assert.Equal(t, "KA", song.NameFolderCode)
		assert.Equal(t, "KA", nameFolderCodeForSong(t, db, created.ID))
	})

	t.Run("Save updates the name folder from current reading", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()
		_, err := db.Exec(`
			INSERT INTO songs (id, display_id, title, reading, artist, genre_id, official_idx, is_worldsend)
			VALUES (1, 'SAVE001', 'Title', 'アイウエオ', 'Artist', 1, 'IDX-SAVE001', 0)
		`)
		require.NoError(t, err)

		song := &entity.Song{
			ID:          1,
			DisplayID:   "SAVE001",
			Title:       "Title",
			Reading:     stringPtrForNameFolderTest("ソラ"),
			Artist:      "Artist",
			GenreID:     intPtrForNameFolderTest(1),
			OfficialIdx: "IDX-SAVE001",
		}
		repo := &songRepository{db: db}

		err = repo.Save(context.Background(), db, song)

		require.NoError(t, err)
		assert.Equal(t, "SA", song.NameFolderCode)
		assert.Equal(t, "SA", nameFolderCodeForSong(t, db, song.ID))
		loaded, err := repo.FindByDisplayID(context.Background(), db, song.DisplayID)
		require.NoError(t, err)
		assert.Equal(t, "SA", loaded.NameFolderCode)
	})

	t.Run("UpdateSongs updates multiple name folders in one bulk operation", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()
		_, err := db.Exec(`
			INSERT INTO songs (id, display_id, title, reading, artist, genre_id, official_idx, is_worldsend)
			VALUES (1, 'BULK001', 'Old title', 'アイ', 'Artist', 1, 'IDX-BULK001', 0),
			       (2, 'BULK002', 'Old title', 'アイ', 'Artist', 1, 'IDX-BULK002', 0)
		`)
		require.NoError(t, err)

		first := &entity.Song{DisplayID: "BULK001", Title: "First", Reading: stringPtrForNameFolderTest("ココロ"), Artist: "Artist", GenreID: intPtrForNameFolderTest(1)}
		second := &entity.Song{DisplayID: "BULK002", Title: "Second", Reading: stringPtrForNameFolderTest("ソラ"), Artist: "Artist", GenreID: intPtrForNameFolderTest(1)}
		repo := &songRepository{db: db}

		err = repo.UpdateSongs(context.Background(), db, []*domainrepo.SongUpdate{{Song: first}, {Song: second}})

		require.NoError(t, err)
		assert.Equal(t, "KA", first.NameFolderCode)
		assert.Equal(t, "SA", second.NameFolderCode)
		assert.Equal(t, "KA", nameFolderCodeForSong(t, db, 1))
		assert.Equal(t, "SA", nameFolderCodeForSong(t, db, 2))
	})
}

func TestWorldsendRepositoryPersistsAndRestoresNameFolderCode(t *testing.T) {
	t.Run("CreateSong derives and returns the name folder", func(t *testing.T) {
		db := setupWorldsendUpdateDB(t)
		defer db.Close()

		repo := &worldsendChartRepository{db: db}
		song := &entity.Song{
			DisplayID:   "WE-CREATE001",
			Title:       "Unrelated title",
			Reading:     stringPtrForNameFolderTest("ホシ"),
			Artist:      "Artist",
			GenreID:     intPtrForNameFolderTest(1),
			OfficialIdx: "WEIDX-CREATE001",
		}

		created, err := repo.CreateSong(context.Background(), db, song, nil)

		require.NoError(t, err)
		assert.Equal(t, "HA", created.Song.NameFolderCode)
		assert.Equal(t, "HA", song.NameFolderCode)
		assert.Equal(t, "HA", nameFolderCodeForSong(t, db, created.Song.ID))
	})

	t.Run("SaveSong updates the name folder from current reading", func(t *testing.T) {
		db := setupWorldsendUpdateDB(t)
		defer db.Close()
		_, err := db.Exec(`
			INSERT INTO songs (id, display_id, title, reading, artist, genre_id, official_idx, is_worldsend)
			VALUES (1, 'WE-SAVE001', 'Title', 'アイ', 'Artist', 1, 'WEIDX-SAVE001', 1)
		`)
		require.NoError(t, err)
		_, err = db.Exec(`INSERT INTO worldsend_charts (song_id) VALUES (1)`)
		require.NoError(t, err)

		song := &entity.Song{
			ID:          1,
			DisplayID:   "WE-SAVE001",
			Title:       "Title",
			Reading:     stringPtrForNameFolderTest("モリ"),
			Artist:      "Artist",
			GenreID:     intPtrForNameFolderTest(1),
			OfficialIdx: "WEIDX-SAVE001",
			IsWorldsend: true,
		}
		repo := &worldsendChartRepository{db: db}

		err = repo.SaveSong(context.Background(), db, song)

		require.NoError(t, err)
		assert.Equal(t, "MA", song.NameFolderCode)
		assert.Equal(t, "MA", nameFolderCodeForSong(t, db, song.ID))
		loaded, err := repo.FindByDisplayID(context.Background(), db, song.DisplayID)
		require.NoError(t, err)
		assert.Equal(t, "MA", loaded.Song.NameFolderCode)
	})

	t.Run("UpdateSongs updates multiple name folders in one bulk operation", func(t *testing.T) {
		db := setupWorldsendUpdateDB(t)
		defer db.Close()
		_, err := db.Exec(`
			INSERT INTO songs (id, display_id, title, reading, artist, genre_id, official_idx, is_worldsend)
			VALUES (1, 'WE-BULK001', 'Old title', 'アイ', 'Artist', 1, 'WEIDX-BULK001', 1),
			       (2, 'WE-BULK002', 'Old title', 'アイ', 'Artist', 1, 'WEIDX-BULK002', 1)
		`)
		require.NoError(t, err)
		_, err = db.Exec(`INSERT INTO worldsend_charts (song_id) VALUES (1), (2)`)
		require.NoError(t, err)

		first := &entity.Song{DisplayID: "WE-BULK001", Title: "First", Reading: stringPtrForNameFolderTest("ココロ"), Artist: "Artist", GenreID: intPtrForNameFolderTest(1), IsWorldsend: true}
		second := &entity.Song{DisplayID: "WE-BULK002", Title: "Second", Reading: stringPtrForNameFolderTest("ソラ"), Artist: "Artist", GenreID: intPtrForNameFolderTest(1), IsWorldsend: true}
		repo := &worldsendChartRepository{db: db}

		err = repo.UpdateSongs(context.Background(), db, []*domainrepo.WorldsendUpdate{{Song: first}, {Song: second}})

		require.NoError(t, err)
		assert.Equal(t, "KA", first.NameFolderCode)
		assert.Equal(t, "SA", second.NameFolderCode)
		assert.Equal(t, "KA", nameFolderCodeForSong(t, db, 1))
		assert.Equal(t, "SA", nameFolderCodeForSong(t, db, 2))
	})
}

func nameFolderCodeForSong(t *testing.T, db *sqlx.DB, songID int) string {
	t.Helper()
	var code string
	err := db.Get(&code, `
		SELECT name_folders.code
		FROM songs
		INNER JOIN name_folders ON name_folders.id = songs.name_folder_id
		WHERE songs.id = ?
	`, songID)
	require.NoError(t, err)
	return code
}

func stringPtrForNameFolderTest(value string) *string {
	return &value
}

func intPtrForNameFolderTest(value int) *int {
	return &value
}
