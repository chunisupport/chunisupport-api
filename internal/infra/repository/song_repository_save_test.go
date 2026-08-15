package repository

import (
	"context"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSongRepositorySave_既存譜面を集約として保存する(t *testing.T) {
	// Given
	db := setupTestDB(t)
	defer db.Close()
	_, err := db.Exec(`
		INSERT INTO songs (id, display_id, title, artist, genre_id, bpm, official_idx, is_worldsend, is_new, is_deleted)
		VALUES (1, 'DISPLAY001', 'Title', 'Artist', 1, 120, 'IDX001', 0, 0, 0)
	`)
	require.NoError(t, err)
	_, err = db.Exec(`
		INSERT INTO charts (id, song_id, difficulty_id, const, is_const_unknown)
		VALUES (10, 1, 4, 14.5, 1)
	`)
	require.NoError(t, err)
	repo := &songRepository{db: db}
	song, err := repo.FindByOfficialIdx(context.Background(), db, "IDX001")
	require.NoError(t, err)
	updated, err := chartconstant.NewChartConstant(14.7)
	require.NoError(t, err)
	require.NoError(t, song.ChangeChartConstant(4, updated))

	// When
	err = repo.Save(context.Background(), db, song)

	// Then
	require.NoError(t, err)
	var saved struct {
		Const          float64 `db:"const"`
		IsConstUnknown bool    `db:"is_const_unknown"`
	}
	err = db.Get(&saved, `SELECT const, is_const_unknown FROM charts WHERE id = 10`)
	require.NoError(t, err)
	assert.Equal(t, 14.7, saved.Const)
	assert.False(t, saved.IsConstUnknown)
}

func TestSongRepositoryPersistsSongLifecycleState(t *testing.T) {
	tests := []struct {
		name          string
		setupSongID   int
		saveSong      *entity.Song
		expectedErr   error
		assertPersist bool
	}{
		{
			name:        "既存楽曲の集約状態を保存できる",
			setupSongID: 1,
			saveSong: &entity.Song{
				ID:          1,
				DisplayID:   "DISPLAY001-UPDATED",
				Title:       "Updated Title",
				Reading:     stringPtrForSongSaveTest("Updated Reading"),
				Artist:      "Updated Artist",
				GenreID:     intPtrForSongSaveTest(2),
				BPM:         intPtrForSongSaveTest(222),
				OfficialIdx: "IDX001-UPDATED",
				Jacket:      stringPtrForSongSaveTest("updated.png"),
				IsWorldsend: false,
				IsNew:       true,
				IsDeleted:   true,
				ReleasedAt:  nil,
			},
			assertPersist: true,
		},
		{
			name:        "存在しない楽曲はErrSongNotFoundを返す",
			setupSongID: 1,
			saveSong: &entity.Song{
				ID:          999,
				DisplayID:   "NOTFOUND",
				Title:       "Not Found",
				Artist:      "Not Found",
				GenreID:     intPtrForSongSaveTest(1),
				BPM:         intPtrForSongSaveTest(120),
				OfficialIdx: "IDX-NOTFOUND",
				IsWorldsend: false,
				IsDeleted:   true,
			},
			expectedErr: domainrepo.ErrSongNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			db := setupTestDB(t)
			defer db.Close()

			_, err := db.Exec(`
				INSERT INTO songs (id, display_id, title, artist, genre_id, bpm, released_at, official_idx, jacket, is_worldsend, is_new, is_deleted)
				VALUES (?, 'DISPLAY001', 'Original Title', 'Original Artist', 1, 180, ?, 'IDX001', 'original.png', 0, 0, 0)
			`, tt.setupSongID, time.Now().UTC())
			require.NoError(t, err)

			repo := &songRepository{db: db}
			ctx := context.Background()

			// When
			err = repo.Save(ctx, db, tt.saveSong)

			// Then
			if tt.expectedErr != nil {
				require.ErrorIs(t, err, tt.expectedErr)
				return
			}
			require.NoError(t, err)

			if tt.assertPersist {
				var saved struct {
					ID          int     `db:"id"`
					DisplayID   string  `db:"display_id"`
					Title       string  `db:"title"`
					Reading     *string `db:"reading"`
					Artist      string  `db:"artist"`
					GenreID     int     `db:"genre_id"`
					BPM         int     `db:"bpm"`
					OfficialIdx string  `db:"official_idx"`
					Jacket      *string `db:"jacket"`
					IsWorldsend bool    `db:"is_worldsend"`
					IsNew       bool    `db:"is_new"`
					IsDeleted   bool    `db:"is_deleted"`
				}
				err = db.Get(&saved, `
					SELECT id, display_id, title, reading, artist, genre_id, bpm, official_idx, jacket, is_worldsend, is_new, is_deleted
					FROM songs
					WHERE id = ?
				`, tt.saveSong.ID)
				require.NoError(t, err)

				assert.Equal(t, tt.saveSong.ID, saved.ID)
				assert.Equal(t, tt.saveSong.DisplayID, saved.DisplayID)
				assert.Equal(t, tt.saveSong.Title, saved.Title)
				require.NotNil(t, saved.Reading)
				assert.Equal(t, *tt.saveSong.Reading, *saved.Reading)
				assert.Equal(t, tt.saveSong.Artist, saved.Artist)
				assert.Equal(t, *tt.saveSong.GenreID, saved.GenreID)
				assert.Equal(t, *tt.saveSong.BPM, saved.BPM)
				assert.Equal(t, tt.saveSong.OfficialIdx, saved.OfficialIdx)
				require.NotNil(t, saved.Jacket)
				assert.Equal(t, *tt.saveSong.Jacket, *saved.Jacket)
				assert.Equal(t, tt.saveSong.IsWorldsend, saved.IsWorldsend)
				assert.Equal(t, tt.saveSong.IsNew, saved.IsNew)
				assert.Equal(t, tt.saveSong.IsDeleted, saved.IsDeleted)
			}
		})
	}
}

func intPtrForSongSaveTest(v int) *int {
	return &v
}

func stringPtrForSongSaveTest(v string) *string {
	return &v
}
