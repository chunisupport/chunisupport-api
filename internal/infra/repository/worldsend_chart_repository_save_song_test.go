package repository

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorldsendRepositoryPersistsWorldsendSongLifecycleState(t *testing.T) {
	releasedAt := time.Date(2024, time.January, 2, 3, 4, 5, 0, time.UTC)

	tests := []struct {
		name          string
		isWorldsend   bool
		saveSong      *entity.Song
		expectedErr   error
		assertPersist bool
	}{
		{
			name:        "既存WORLD'S END楽曲の集約状態を保存できる",
			isWorldsend: true,
			saveSong: &entity.Song{
				ID:          1,
				DisplayID:   "WE001-UPDATED",
				Title:       "Updated WE Title",
				Artist:      "Updated WE Artist",
				GenreID:     intPtrForWorldsendSaveTest(2),
				BPM:         intPtrForWorldsendSaveTest(230),
				OfficialIdx: "WEIDX001-UPDATED",
				Jacket:      stringPtrForWorldsendSaveTest("we-updated.png"),
				IsWorldsend: true,
				IsDeleted:   true,
				ReleasedAt:  timePtrForWorldsendSaveTest(releasedAt),
			},
			assertPersist: true,
		},
		{
			name:        "WORLD'S END以外の楽曲はErrSongNotFoundを返す",
			isWorldsend: false,
			saveSong: &entity.Song{
				ID:          1,
				DisplayID:   "NOT-WE",
				Title:       "Not WE",
				Artist:      "Not WE",
				GenreID:     intPtrForWorldsendSaveTest(1),
				BPM:         intPtrForWorldsendSaveTest(120),
				OfficialIdx: "IDX-NOT-WE",
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

			isWorldsendFlag := 0
			if tt.isWorldsend {
				isWorldsendFlag = 1
			}

			_, err := db.Exec(`
				INSERT INTO songs (id, display_id, title, artist, genre_id, bpm, released_at, official_idx, jacket, is_worldsend, is_deleted)
				VALUES (1, 'WE001', 'WE Title', 'WE Artist', 1, 210, ?, 'WEIDX001', 'we.png', ?, 0)
			`, time.Now().UTC(), isWorldsendFlag)
			require.NoError(t, err)

			repo := &worldsendChartRepository{db: db}
			ctx := context.Background()

			// When
			err = repo.SaveSong(ctx, db, tt.saveSong)

			// Then
			if tt.expectedErr != nil {
				require.ErrorIs(t, err, tt.expectedErr)
				return
			}
			require.NoError(t, err)

			if tt.assertPersist {
				var saved struct {
					ID          int            `db:"id"`
					DisplayID   string         `db:"display_id"`
					Title       string         `db:"title"`
					Artist      string         `db:"artist"`
					GenreID     int            `db:"genre_id"`
					BPM         int            `db:"bpm"`
					ReleasedAt  sql.NullString `db:"released_at"`
					OfficialIdx string         `db:"official_idx"`
					Jacket      *string        `db:"jacket"`
					IsWorldsend bool           `db:"is_worldsend"`
					IsDeleted   bool           `db:"is_deleted"`
				}
				err = db.Get(&saved, `
					SELECT id, display_id, title, artist, genre_id, bpm, released_at, official_idx, jacket, is_worldsend, is_deleted
					FROM songs
					WHERE id = ?
				`, tt.saveSong.ID)
				require.NoError(t, err)

				assert.Equal(t, tt.saveSong.ID, saved.ID)
				assert.Equal(t, tt.saveSong.DisplayID, saved.DisplayID)
				assert.Equal(t, tt.saveSong.Title, saved.Title)
				assert.Equal(t, tt.saveSong.Artist, saved.Artist)
				assert.Equal(t, *tt.saveSong.GenreID, saved.GenreID)
				assert.Equal(t, *tt.saveSong.BPM, saved.BPM)
				require.NotNil(t, tt.saveSong.ReleasedAt)
				require.True(t, saved.ReleasedAt.Valid)
				// DBのカラムはDATE型なので、日付部分のみを比較する
				expectedDate := tt.saveSong.ReleasedAt.UTC().Format(time.DateOnly)
				savedDate, parseErr := parseWorldsendSavedDate(saved.ReleasedAt.String)
				require.NoError(t, parseErr)
				assert.Equal(t, expectedDate, savedDate.Format(time.DateOnly))
				assert.Equal(t, tt.saveSong.OfficialIdx, saved.OfficialIdx)
				require.NotNil(t, saved.Jacket)
				assert.Equal(t, *tt.saveSong.Jacket, *saved.Jacket)
				assert.Equal(t, tt.saveSong.IsWorldsend, saved.IsWorldsend)
				assert.Equal(t, tt.saveSong.IsDeleted, saved.IsDeleted)
			}
		})
	}
}

func intPtrForWorldsendSaveTest(v int) *int {
	return &v
}

func stringPtrForWorldsendSaveTest(v string) *string {
	return &v
}

func timePtrForWorldsendSaveTest(v time.Time) *time.Time {
	return &v
}

func parseWorldsendSavedDate(v string) (time.Time, error) {
	layouts := []string{
		time.DateOnly,
		time.DateTime,
		time.RFC3339,
		"2006-01-02 15:04:05 -0700 MST",
	}

	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, v); err == nil {
			return parsed, nil
		}
	}

	return time.Time{}, fmt.Errorf("released_atの解析に失敗しました: %s", v)
}
