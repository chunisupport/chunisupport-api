package repository

import (
	"context"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindByDisplayID_ScansLevelStarValueObject(t *testing.T) {
	db := setupWorldsendUpdateDB(t)
	defer db.Close()

	_, err := db.Exec(`
		INSERT INTO songs (id, display_id, title, artist, genre_id, bpm, released_at, official_idx, jacket, is_worldsend, is_new, is_deleted)
		VALUES (1, 'WE001', 'title', 'artist', 1, 180, NULL, 'WEIDX001', 'we.png', 1, 0, 0)
	`)
	require.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO worldsend_charts (id, song_id, level_star, attribute, notes, notes_designer)
		VALUES (101, 1, 4, '狂', 1200, '譜面作者A')
	`)
	require.NoError(t, err)

	repo := &worldsendChartRepository{db: db}

	got, err := repo.FindByDisplayID(context.Background(), db, "WE001")
	require.NoError(t, err)
	require.NotNil(t, got)
	require.NotNil(t, got.Chart)
	require.NotNil(t, got.Chart.LevelStar)
	assert.Equal(t, 4, got.Chart.LevelStar.Int())
	require.NotNil(t, got.Chart.NotesDesigner)
	assert.Equal(t, "譜面作者A", *got.Chart.NotesDesigner)
}

func TestFindByDisplayID_ScansNilLevelStarAsNil(t *testing.T) {
	db := setupWorldsendUpdateDB(t)
	defer db.Close()

	_, err := db.Exec(`
		INSERT INTO songs (id, display_id, title, artist, genre_id, bpm, released_at, official_idx, jacket, is_worldsend, is_new, is_deleted)
		VALUES (1, 'WE001', 'title', 'artist', 1, 180, NULL, 'WEIDX001', 'we.png', 1, 0, 0)
	`)
	require.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO worldsend_charts (id, song_id, level_star, attribute, notes, notes_designer)
		VALUES (101, 1, NULL, '狂', 1200, NULL)
	`)
	require.NoError(t, err)

	repo := &worldsendChartRepository{db: db}

	got, err := repo.FindByDisplayID(context.Background(), db, "WE001")
	require.NoError(t, err)
	require.NotNil(t, got)
	require.NotNil(t, got.Chart)
	assert.Nil(t, got.Chart.LevelStar)
}

func TestWorldsendChartRepository_LoadsWikiPageTitle(t *testing.T) {
	wikiPageTitle := "title(WORLD'S END)"
	tests := []struct {
		name string
		// When: 楽曲の取得方法
		find func(ctx context.Context, repo *worldsendChartRepository, exec repository.Executor) (map[string]*entity.Song, error)
	}{
		{
			name: "FindAllでWikiページタイトルを取得できる",
			find: func(ctx context.Context, repo *worldsendChartRepository, exec repository.Executor) (map[string]*entity.Song, error) {
				songsWithCharts, err := repo.FindAll(ctx, exec, false)
				songs := make([]*entity.Song, 0, len(songsWithCharts))
				for _, swc := range songsWithCharts {
					songs = append(songs, swc.Song)
				}
				return songsByDisplayID(songs), err
			},
		},
		{
			name: "FindByDisplayIDでWikiページタイトルを取得できる",
			find: func(ctx context.Context, repo *worldsendChartRepository, exec repository.Executor) (map[string]*entity.Song, error) {
				swc1, err := repo.FindByDisplayID(ctx, exec, "WE001")
				if err != nil {
					return nil, err
				}
				swc2, err := repo.FindByDisplayID(ctx, exec, "WE002")
				if err != nil {
					return nil, err
				}
				return songsByDisplayID([]*entity.Song{swc1.Song, swc2.Song}), nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			db := setupWorldsendUpdateDB(t)
			defer db.Close()

			_, err := db.Exec(`
				INSERT INTO songs (id, display_id, title, wiki_page_title, artist, genre_id, bpm, released_at, official_idx, jacket, is_worldsend, is_new, is_deleted)
				VALUES
					(1, 'WE001', 'title', ?, 'artist', 1, 180, NULL, 'WEIDX001', NULL, 1, 0, 0),
					(2, 'WE002', 'title2', NULL, 'artist', 1, 180, NULL, 'WEIDX002', NULL, 1, 0, 0)
			`, wikiPageTitle)
			require.NoError(t, err)

			_, err = db.Exec(`
				INSERT INTO worldsend_charts (id, song_id, level_star, attribute, notes, notes_designer)
				VALUES (101, 1, 4, '狂', 1200, NULL), (102, 2, 3, '光', 1000, NULL)
			`)
			require.NoError(t, err)

			// When
			songs, err := tt.find(context.Background(), &worldsendChartRepository{db: db}, db)

			// Then
			require.NoError(t, err)
			require.Contains(t, songs, "WE001")
			require.Contains(t, songs, "WE002")
			assert.Equal(t, &wikiPageTitle, songs["WE001"].WikiPageTitle)
			assert.Nil(t, songs["WE002"].WikiPageTitle)
		})
	}
}
