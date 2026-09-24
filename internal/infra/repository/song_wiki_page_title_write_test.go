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

// insertSongsForWikiPageTitleTest は Wiki ページタイトル検証用に、既存タイトルを持つ楽曲を2件登録します。
// WORLD'S END の場合は1曲1譜面の前提を満たすため譜面も登録します。
func insertSongsForWikiPageTitleTest(t *testing.T, db *sqlx.DB, isWorldsend bool) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO songs (id, display_id, title, wiki_page_title, artist, genre_id, bpm, released_at, official_idx, jacket, is_worldsend, is_new, is_deleted)
		VALUES
			(1, 'DISPLAY001', 'Song 1', '既存タイトル1', 'Artist', 1, 180, NULL, 'IDX001', NULL, ?, 0, 0),
			(2, 'DISPLAY002', 'Song 2', '既存タイトル2', 'Artist', 1, 180, NULL, 'IDX002', NULL, ?, 0, 0)
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

// selectWikiPageTitle は楽曲の wiki_page_title を取得します。
func selectWikiPageTitle(t *testing.T, db *sqlx.DB, displayID string) *string {
	t.Helper()
	var title *string
	require.NoError(t, db.Get(&title, `SELECT wiki_page_title FROM songs WHERE display_id = ?`, displayID))
	return title
}

// newSongForWikiPageTitleTest は一括更新に渡す楽曲エンティティを生成します。
func newSongForWikiPageTitleTest(displayID string, wikiPageTitle *string) *entity.Song {
	return &entity.Song{
		DisplayID:     displayID,
		Title:         "Song",
		WikiPageTitle: wikiPageTitle,
		Artist:        "Artist",
		GenreID:       new(1),
		Charts:        []*entity.Chart{},
	}
}

// wikiPageTitleBulkUpdateCases は一括更新における wiki_page_title の書き込みケースです。
// DISPLAY001 には更新指定あり、DISPLAY002 には更新指定なしの楽曲を同時に渡し、
// CASE 式の引数順序と既存値の維持を同じリクエスト内で検証します。
var wikiPageTitleBulkUpdateCases = []struct {
	name string
	// Given: DISPLAY001 の更新内容
	updateWikiPageTitle bool
	wikiPageTitle       *string
	// Then: 永続化される値
	expectedSong1 *string
	expectedSong2 *string
}{
	{
		name:                "更新指定がある楽曲は指定値で更新し、指定がない楽曲は既存値を維持する",
		updateWikiPageTitle: true,
		wikiPageTitle:       new("新しいタイトル"),
		expectedSong1:       new("新しいタイトル"),
		expectedSong2:       new("既存タイトル2"),
	},
	{
		name:                "更新指定があり値がnilの楽曲はNULLに更新する",
		updateWikiPageTitle: true,
		wikiPageTitle:       nil,
		expectedSong1:       nil,
		expectedSong2:       new("既存タイトル2"),
	},
	{
		name:                "すべての楽曲で更新指定がない場合は既存値を維持する",
		updateWikiPageTitle: false,
		wikiPageTitle:       new("無視されるタイトル"),
		expectedSong1:       new("既存タイトル1"),
		expectedSong2:       new("既存タイトル2"),
	},
}

func TestSongRepository_UpdateSongs_WikiPageTitle(t *testing.T) {
	for _, tt := range wikiPageTitleBulkUpdateCases {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			db := setupTestDB(t)
			defer db.Close()
			insertSongsForWikiPageTitleTest(t, db, false)
			repo := &songRepository{db: db}

			// When
			err := repo.UpdateSongs(context.Background(), db, []*domainrepo.SongUpdate{
				{Song: newSongForWikiPageTitleTest("DISPLAY001", tt.wikiPageTitle), UpdateWikiPageTitle: tt.updateWikiPageTitle},
				{Song: newSongForWikiPageTitleTest("DISPLAY002", new("無視されるタイトル")), UpdateWikiPageTitle: false},
			})

			// Then
			require.NoError(t, err)
			assert.Equal(t, tt.expectedSong1, selectWikiPageTitle(t, db, "DISPLAY001"))
			assert.Equal(t, tt.expectedSong2, selectWikiPageTitle(t, db, "DISPLAY002"))
		})
	}
}

func TestWorldsendChartRepository_UpdateSongs_WikiPageTitle(t *testing.T) {
	for _, tt := range wikiPageTitleBulkUpdateCases {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			db := setupWorldsendUpdateDB(t)
			defer db.Close()
			insertSongsForWikiPageTitleTest(t, db, true)
			repo := &worldsendChartRepository{db: db}

			// When
			err := repo.UpdateSongs(context.Background(), db, []*domainrepo.WorldsendUpdate{
				{Song: newSongForWikiPageTitleTest("DISPLAY001", tt.wikiPageTitle), UpdateWikiPageTitle: tt.updateWikiPageTitle},
				{Song: newSongForWikiPageTitleTest("DISPLAY002", new("無視されるタイトル")), UpdateWikiPageTitle: false},
			})

			// Then
			require.NoError(t, err)
			assert.Equal(t, tt.expectedSong1, selectWikiPageTitle(t, db, "DISPLAY001"))
			assert.Equal(t, tt.expectedSong2, selectWikiPageTitle(t, db, "DISPLAY002"))
		})
	}
}

func TestSongRepository_Save_WritesWikiPageTitle(t *testing.T) {
	// Given
	db := setupTestDB(t)
	defer db.Close()
	insertSongsForWikiPageTitleTest(t, db, false)
	repo := &songRepository{db: db}
	song, err := repo.FindByDisplayID(context.Background(), db, "DISPLAY001")
	require.NoError(t, err)
	song.WikiPageTitle = new("保存後タイトル")

	// When
	err = repo.Save(context.Background(), db, song)

	// Then
	require.NoError(t, err)
	assert.Equal(t, new("保存後タイトル"), selectWikiPageTitle(t, db, "DISPLAY001"))
}

func TestWorldsendChartRepository_SaveSong_WritesWikiPageTitle(t *testing.T) {
	// Given
	db := setupWorldsendUpdateDB(t)
	defer db.Close()
	insertSongsForWikiPageTitleTest(t, db, true)
	repo := &worldsendChartRepository{db: db}
	swc, err := repo.FindByDisplayID(context.Background(), db, "DISPLAY001")
	require.NoError(t, err)
	swc.Song.WikiPageTitle = new("保存後タイトル")

	// When
	err = repo.SaveSong(context.Background(), db, swc.Song)

	// Then
	require.NoError(t, err)
	assert.Equal(t, new("保存後タイトル"), selectWikiPageTitle(t, db, "DISPLAY001"))
}

func TestSongRepository_Create_WritesWikiPageTitle(t *testing.T) {
	// Given
	db := setupTestDB(t)
	defer db.Close()
	repo := &songRepository{db: db}
	song := newSongForWikiPageTitleTest("DISPLAY001", new("作成時タイトル"))
	song.OfficialIdx = "IDX001"

	// When
	_, err := repo.Create(context.Background(), db, song)

	// Then
	require.NoError(t, err)
	assert.Equal(t, new("作成時タイトル"), selectWikiPageTitle(t, db, "DISPLAY001"))
}

func TestWorldsendChartRepository_CreateSong_WritesWikiPageTitle(t *testing.T) {
	// Given
	db := setupWorldsendUpdateDB(t)
	defer db.Close()
	repo := &worldsendChartRepository{db: db}
	song := newSongForWikiPageTitleTest("DISPLAY001", new("作成時タイトル"))
	song.OfficialIdx = "IDX001"
	song.IsWorldsend = true

	// When
	_, err := repo.CreateSong(context.Background(), db, song, nil)

	// Then
	require.NoError(t, err)
	assert.Equal(t, new("作成時タイトル"), selectWikiPageTitle(t, db, "DISPLAY001"))
}
