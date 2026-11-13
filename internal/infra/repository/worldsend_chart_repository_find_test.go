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

func TestWorldsendChartRepository_FindAll_削除済みと通常楽曲を絞り込む(t *testing.T) {
	tests := []struct {
		name string
		// Given: 削除済みを含めるかどうか
		includeDeleted bool
		// Then: s.id 昇順で期待する DisplayID の一覧
		expected []string
	}{
		{
			name:           "削除済みを含めない場合は削除済みでないWORLD'S ENDだけをid順に返す",
			includeDeleted: false,
			expected:       []string{"WE002", "WE001"},
		},
		{
			name:           "削除済みを含める場合は削除済みのWORLD'S ENDもid順に返す",
			includeDeleted: true,
			expected:       []string{"WE002", "WE003", "WE001"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			// 通常楽曲にも WORLD'S END 譜面行を用意しているのは、s.is_worldsend = 1 の絞り込みを外すと一覧に混ざることを確かめるため。
			db := setupWorldsendUpdateDB(t)
			defer db.Close()

			_, err := db.Exec(`
				INSERT INTO songs (id, display_id, title, artist, genre_id, official_idx, is_worldsend, is_deleted)
				VALUES
					(1, 'WE002', '曲2', 'artist', 1, 'WEIDX002', 1, 0),
					(2, 'WE003', '削除曲', 'artist', 1, 'WEIDX003', 1, 1),
					(3, 'WE001', '曲1', 'artist', 1, 'WEIDX001', 1, 0),
					(4, 'NORMAL001', '通常曲', 'artist', 1, 'NIDX001', 0, 0)
			`)
			require.NoError(t, err)

			_, err = db.Exec(`
				INSERT INTO worldsend_charts (id, song_id)
				VALUES (101, 1), (102, 2), (103, 3), (104, 4)
			`)
			require.NoError(t, err)

			// When
			got, err := (&worldsendChartRepository{db: db}).FindAll(context.Background(), db, tt.includeDeleted)

			// Then
			require.NoError(t, err)
			gotIDs := make([]string, 0, len(got))
			for _, swc := range got {
				require.NotNil(t, swc.Song)
				gotIDs = append(gotIDs, swc.Song.DisplayID)
			}
			assert.Equal(t, tt.expected, gotIDs)
		})
	}
}

func TestWorldsendChartRepository_FindByDisplayID_削除済みと通常楽曲の扱い(t *testing.T) {
	tests := []struct {
		name string
		// Given: 取得方法と DisplayID
		forChange bool
		displayID string
		// Then
		wantErr         error
		expectedTitle   string
		expectedDeleted bool
	}{
		{
			name:            "削除済みのWORLD'S ENDは取得できる",
			forChange:       false,
			displayID:       "WE002",
			expectedTitle:   "削除曲",
			expectedDeleted: true,
		},
		{
			name:      "通常楽曲は未検出エラーを返す",
			forChange: false,
			displayID: "NORMAL001",
			wantErr:   repository.ErrSongNotFound,
		},
		{
			name:      "変更用取得でも通常楽曲は未検出エラーを返す",
			forChange: true,
			displayID: "NORMAL001",
			wantErr:   repository.ErrSongNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			// 通常楽曲にも WORLD'S END 譜面行を用意しているのは、s.is_worldsend = 1 の絞り込みを外すと取得できてしまうことを確かめるため。
			db := setupWorldsendUpdateDB(t)
			defer db.Close()

			_, err := db.Exec(`
				INSERT INTO songs (id, display_id, title, artist, genre_id, official_idx, is_worldsend, is_deleted)
				VALUES
					(1, 'WE002', '削除曲', 'artist', 1, 'WEIDX002', 1, 1),
					(2, 'NORMAL001', '通常曲', 'artist', 1, 'NIDX001', 0, 0)
			`)
			require.NoError(t, err)

			_, err = db.Exec(`
				INSERT INTO worldsend_charts (id, song_id)
				VALUES (101, 1), (102, 2)
			`)
			require.NoError(t, err)

			repo := &worldsendChartRepository{db: db}

			// When
			var got *entity.WorldsendSongWithChart
			if tt.forChange {
				// SQLite は FOR UPDATE を解釈できないため、行ロック句だけを取り除く実行子を使う。
				got, err = repo.FindByDisplayIDForChange(context.Background(), &worldsendLockQueryExecutor{Executor: db}, tt.displayID)
			} else {
				got, err = repo.FindByDisplayID(context.Background(), db, tt.displayID)
			}

			// Then
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
			require.NotNil(t, got.Song)
			assert.Equal(t, tt.expectedTitle, got.Song.Title)
			assert.Equal(t, tt.expectedDeleted, got.Song.IsDeleted)
		})
	}
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
