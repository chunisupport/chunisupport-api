package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/levelstar"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/notes"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type deletingChartExecutor struct {
	baseExecutor
	db        *sqlx.DB
	chartID   int
	execCount int
}

func (e *deletingChartExecutor) QueryxContext(ctx context.Context, query string, args ...any) (*sqlx.Rows, error) {
	return e.db.QueryxContext(ctx, query, args...)
}

func (e *deletingChartExecutor) QueryRowxContext(ctx context.Context, query string, args ...any) *sqlx.Row {
	return e.db.QueryRowxContext(ctx, query, args...)
}

func (e *deletingChartExecutor) Rebind(query string) string {
	return e.db.Rebind(query)
}

func (e *deletingChartExecutor) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	e.execCount++
	result, err := e.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	if e.execCount == 1 {
		if _, deleteErr := e.db.ExecContext(ctx, `DELETE FROM worldsend_charts WHERE id = ?`, e.chartID); deleteErr != nil {
			return nil, deleteErr
		}
	}

	return result, nil
}

var _ domainrepo.Executor = (*deletingChartExecutor)(nil)

type softDeletingSongExecutor struct {
	baseExecutor
	db        *sqlx.DB
	songID    int
	execCount int
}

func (e *softDeletingSongExecutor) QueryxContext(ctx context.Context, query string, args ...any) (*sqlx.Rows, error) {
	return e.db.QueryxContext(ctx, query, args...)
}

func (e *softDeletingSongExecutor) QueryRowxContext(ctx context.Context, query string, args ...any) *sqlx.Row {
	return e.db.QueryRowxContext(ctx, query, args...)
}

func (e *softDeletingSongExecutor) Rebind(query string) string {
	return e.db.Rebind(query)
}

func (e *softDeletingSongExecutor) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	e.execCount++
	result, err := e.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	if e.execCount == 1 {
		if _, deleteErr := e.db.ExecContext(ctx, `UPDATE songs SET is_deleted = 1 WHERE id = ?`, e.songID); deleteErr != nil {
			return nil, deleteErr
		}
	}

	return result, nil
}

var _ domainrepo.Executor = (*softDeletingSongExecutor)(nil)

func setupWorldsendUpdateDB(t *testing.T) *sqlx.DB {
	t.Helper()
	db := setupTestDB(t)

	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS worldsend_charts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			song_id INTEGER NOT NULL,
			level_star INTEGER,
			attribute TEXT,
			notes INTEGER,
			notes_designer TEXT,
			FOREIGN KEY (song_id) REFERENCES songs(id) ON DELETE CASCADE,
			UNIQUE(song_id)
		)
	`)
	require.NoError(t, err)

	return db
}

func TestUpdateSongs_SkipsNilChartAndUpdatesSongOnly(t *testing.T) {
	db := setupWorldsendUpdateDB(t)
	defer db.Close()

	ctx := context.Background()

	_, err := db.Exec(`
		INSERT INTO songs (id, display_id, title, artist, genre_id, bpm, released_at, official_idx, jacket, is_worldsend, is_deleted)
		VALUES (1, 'WE001', 'old title', 'old artist', 1, 180, '2024-01-01', 'WEIDX001', 'old.png', 1, 0)
	`)
	require.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO worldsend_charts (id, song_id, level_star, attribute, notes, notes_designer)
		VALUES (101, 1, 4, '狂', 1200, '旧作者')
	`)
	require.NoError(t, err)

	newGenreID := 2
	newBPM := 200
	releasedAt := time.Date(2025, time.March, 1, 0, 0, 0, 0, time.UTC)
	newJacket := "new.png"

	updates := []*domainrepo.WorldsendUpdate{{
		Song: &entity.Song{
			DisplayID:  "WE001",
			Title:      "new title",
			Artist:     "new artist",
			GenreID:    &newGenreID,
			BPM:        &newBPM,
			ReleasedAt: &releasedAt,
			Jacket:     &newJacket,
		},
		Chart: nil,
	}}

	repo := &worldsendChartRepository{db: db}
	err = repo.UpdateSongs(ctx, db, updates)
	require.NoError(t, err)

	var song struct {
		Title  string `db:"title"`
		Artist string `db:"artist"`
	}
	err = db.Get(&song, `SELECT title, artist FROM songs WHERE id = 1`)
	require.NoError(t, err)
	assert.Equal(t, "new title", song.Title)
	assert.Equal(t, "new artist", song.Artist)

	var chart struct {
		LevelStar     *int    `db:"level_star"`
		Attribute     *string `db:"attribute"`
		Notes         *int    `db:"notes"`
		NotesDesigner *string `db:"notes_designer"`
	}
	err = db.Get(&chart, `SELECT level_star, attribute, notes, notes_designer FROM worldsend_charts WHERE id = 101`)
	require.NoError(t, err)
	if assert.NotNil(t, chart.LevelStar) {
		assert.Equal(t, 4, *chart.LevelStar)
	}
	if assert.NotNil(t, chart.Attribute) {
		assert.Equal(t, "狂", *chart.Attribute)
	}
	if assert.NotNil(t, chart.Notes) {
		assert.Equal(t, 1200, *chart.Notes)
	}
	if assert.NotNil(t, chart.NotesDesigner) {
		assert.Equal(t, "旧作者", *chart.NotesDesigner)
	}
}

func TestFindUpdateTargetsByDisplayIDs_ReturnsEmptyWhenDisplayIDsIsEmpty(t *testing.T) {
	db := setupWorldsendUpdateDB(t)
	defer db.Close()

	repo := &worldsendChartRepository{db: db}

	targets, err := repo.findUpdateTargetsByDisplayIDs(context.Background(), db, []string{})
	require.NoError(t, err)
	require.Empty(t, targets)
}

func TestUpdateSongs_UpdatesChartLevelStarUsingValueObject(t *testing.T) {
	db := setupWorldsendUpdateDB(t)
	defer db.Close()

	ctx := context.Background()

	_, err := db.Exec(`
		INSERT INTO songs (id, display_id, title, artist, genre_id, bpm, released_at, official_idx, jacket, is_worldsend, is_deleted)
		VALUES (1, 'WE001', 'old title', 'old artist', 1, 180, '2024-01-01', 'WEIDX001', 'old.png', 1, 0)
	`)
	require.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO worldsend_charts (id, song_id, level_star, attribute, notes, notes_designer)
		VALUES (101, 1, 2, '狂', 1200, '旧作者')
	`)
	require.NoError(t, err)

	n := notes.Notes(1300)
	genreID := 1
	bpm := 180
	releasedAt := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)
	jacket := "old.png"
	updates := []*domainrepo.WorldsendUpdate{{
		Song: &entity.Song{
			DisplayID:  "WE001",
			Title:      "new title",
			Artist:     "new artist",
			GenreID:    &genreID,
			BPM:        &bpm,
			ReleasedAt: &releasedAt,
			Jacket:     &jacket,
		},
		Chart: &entity.WorldsendChart{
			LevelStar:     levelStarPtrForWorldsendUpdateTest(t, 5),
			Attribute:     stringPtrForWorldsendSaveTest("改"),
			Notes:         &n,
			NotesDesigner: stringPtrForWorldsendSaveTest("新作者"),
		},
	}}

	repo := &worldsendChartRepository{db: db}
	err = repo.UpdateSongs(ctx, db, updates)
	require.NoError(t, err)

	var chart struct {
		LevelStar *int `db:"level_star"`
	}
	err = db.Get(&chart, `SELECT level_star FROM worldsend_charts WHERE id = 101`)
	require.NoError(t, err)
	require.NotNil(t, chart.LevelStar)
	assert.Equal(t, 5, *chart.LevelStar)
}

func TestUpdateSongs_ReturnsErrSongNotFoundWhenDisplayIDMissing(t *testing.T) {
	db := setupWorldsendUpdateDB(t)
	defer db.Close()

	ctx := context.Background()
	repo := &worldsendChartRepository{db: db}

	n := notes.Notes(1300)
	updates := []*domainrepo.WorldsendUpdate{{
		Song: &entity.Song{
			DisplayID: "WE999",
			Title:     "title",
			Artist:    "artist",
		},
		Chart: &entity.WorldsendChart{
			LevelStar:     levelStarPtrForWorldsendUpdateTest(t, 5),
			Attribute:     stringPtrForWorldsendSaveTest("狂"),
			Notes:         &n,
			NotesDesigner: stringPtrForWorldsendSaveTest("新作者"),
		},
	}}

	err := repo.UpdateSongs(ctx, db, updates)
	require.Error(t, err)
	assert.ErrorIs(t, err, domainrepo.ErrSongNotFound)
}

func TestUpdateSongs_ReturnsErrDuplicateDisplayIDWhenRequestContainsDuplicates(t *testing.T) {
	db := setupWorldsendUpdateDB(t)
	defer db.Close()

	ctx := context.Background()
	repo := &worldsendChartRepository{db: db}

	_, err := db.Exec(`
		INSERT INTO songs (id, display_id, title, artist, genre_id, bpm, released_at, official_idx, jacket, is_worldsend, is_deleted)
		VALUES (1, 'WE001', 'old title', 'old artist', 1, 180, '2024-01-01', 'WEIDX001', 'old.png', 1, 0)
	`)
	require.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO worldsend_charts (id, song_id, level_star, attribute, notes, notes_designer)
		VALUES (101, 1, 4, '狂', 1200, '旧作者')
	`)
	require.NoError(t, err)

	updates := []*domainrepo.WorldsendUpdate{
		{Song: &entity.Song{DisplayID: "WE001", Title: "first", Artist: "artist1"}, Chart: nil},
		{Song: &entity.Song{DisplayID: "WE001", Title: "second", Artist: "artist2"}, Chart: nil},
	}

	err = repo.UpdateSongs(ctx, db, updates)
	require.Error(t, err)
	assert.ErrorIs(t, err, domainrepo.ErrDuplicateDisplayID)

	var song struct {
		Title string `db:"title"`
	}
	err = db.Get(&song, `SELECT title FROM songs WHERE id = 1`)
	require.NoError(t, err)
	assert.Equal(t, "old title", song.Title)
}

func TestUpdateSongs_ReturnsErrSongNotFoundWhenTargetDisappearsDuringUpdate(t *testing.T) {
	db := setupWorldsendUpdateDB(t)
	defer db.Close()

	ctx := context.Background()
	repo := &worldsendChartRepository{db: db}

	_, err := db.Exec(`
		INSERT INTO songs (id, display_id, title, artist, genre_id, bpm, released_at, official_idx, jacket, is_worldsend, is_deleted)
		VALUES (1, 'WE001', 'old title', 'old artist', 1, 180, '2024-01-01', 'WEIDX001', 'old.png', 1, 0)
	`)
	require.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO worldsend_charts (id, song_id, level_star, attribute, notes, notes_designer)
		VALUES (101, 1, 4, '狂', 1200, '旧作者')
	`)
	require.NoError(t, err)

	n := notes.Notes(1300)
	genreID := 1
	bpm := 180
	releasedAt := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)
	jacket := "old.png"
	updates := []*domainrepo.WorldsendUpdate{{
		Song: &entity.Song{
			DisplayID:  "WE001",
			Title:      "new title",
			Artist:     "new artist",
			GenreID:    &genreID,
			BPM:        &bpm,
			ReleasedAt: &releasedAt,
			Jacket:     &jacket,
		},
		Chart: &entity.WorldsendChart{
			LevelStar:     levelStarPtrForWorldsendUpdateTest(t, 5),
			Attribute:     stringPtrForWorldsendSaveTest("改"),
			Notes:         &n,
			NotesDesigner: stringPtrForWorldsendSaveTest("新作者"),
		},
	}}

	exec := &deletingChartExecutor{db: db, chartID: 101}
	err = repo.UpdateSongs(ctx, exec, updates)
	require.Error(t, err)
	assert.ErrorIs(t, err, domainrepo.ErrSongNotFound)
}

func TestUpdateSongs_ReturnsErrSongNotFoundWhenChartDisappearsAndRequestHasNoChartUpdate(t *testing.T) {
	db := setupWorldsendUpdateDB(t)
	defer db.Close()

	ctx := context.Background()
	repo := &worldsendChartRepository{db: db}

	_, err := db.Exec(`
		INSERT INTO songs (id, display_id, title, artist, genre_id, bpm, released_at, official_idx, jacket, is_worldsend, is_deleted)
		VALUES (1, 'WE001', 'old title', 'old artist', 1, 180, '2024-01-01', 'WEIDX001', 'old.png', 1, 0)
	`)
	require.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO worldsend_charts (id, song_id, level_star, attribute, notes)
		VALUES (101, 1, 4, '狂', 1200)
	`)
	require.NoError(t, err)

	genreID := 1
	bpm := 180
	releasedAt := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)
	jacket := "old.png"
	updates := []*domainrepo.WorldsendUpdate{{
		Song: &entity.Song{
			DisplayID:  "WE001",
			Title:      "new title",
			Artist:     "new artist",
			GenreID:    &genreID,
			BPM:        &bpm,
			ReleasedAt: &releasedAt,
			Jacket:     &jacket,
		},
		Chart: nil,
	}}

	exec := &deletingChartExecutor{db: db, chartID: 101}
	err = repo.UpdateSongs(ctx, exec, updates)
	require.Error(t, err)
	assert.ErrorIs(t, err, domainrepo.ErrSongNotFound)
}

func TestUpdateSongs_ReturnsErrSongNotFoundWhenMixedChartUpdatesSkipOneAndSkippedChartDisappears(t *testing.T) {
	db := setupWorldsendUpdateDB(t)
	defer db.Close()

	ctx := context.Background()
	repo := &worldsendChartRepository{db: db}

	_, err := db.Exec(`
		INSERT INTO songs (id, display_id, title, artist, genre_id, bpm, released_at, official_idx, jacket, is_worldsend, is_deleted)
		VALUES
			(1, 'WE001', 'old title 1', 'old artist 1', 1, 180, '2024-01-01', 'WEIDX001', 'old1.png', 1, 0),
			(2, 'WE002', 'old title 2', 'old artist 2', 2, 190, '2024-01-02', 'WEIDX002', 'old2.png', 1, 0)
	`)
	require.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO worldsend_charts (id, song_id, level_star, attribute, notes, notes_designer)
		VALUES
			(101, 1, 3, '狂', 1000, '作者1'),
			(102, 2, 4, '止', 1100, '作者2')
	`)
	require.NoError(t, err)

	genreID1 := 1
	bpm1 := 180
	releasedAt1 := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)
	jacket1 := "old1.png"

	genreID2 := 2
	bpm2 := 190
	releasedAt2 := time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC)
	jacket2 := "old2.png"
	n := notes.Notes(1300)

	updates := []*domainrepo.WorldsendUpdate{
		{
			Song: &entity.Song{
				DisplayID:  "WE001",
				Title:      "new title 1",
				Artist:     "new artist 1",
				GenreID:    &genreID1,
				BPM:        &bpm1,
				ReleasedAt: &releasedAt1,
				Jacket:     &jacket1,
			},
			Chart: nil,
		},
		{
			Song: &entity.Song{
				DisplayID:  "WE002",
				Title:      "new title 2",
				Artist:     "new artist 2",
				GenreID:    &genreID2,
				BPM:        &bpm2,
				ReleasedAt: &releasedAt2,
				Jacket:     &jacket2,
			},
			Chart: &entity.WorldsendChart{
				LevelStar:     levelStarPtrForWorldsendUpdateTest(t, 5),
				Attribute:     stringPtrForWorldsendSaveTest("改"),
				Notes:         &n,
				NotesDesigner: stringPtrForWorldsendSaveTest("新作者2"),
			},
		},
	}

	exec := &deletingChartExecutor{db: db, chartID: 101}
	err = repo.UpdateSongs(ctx, exec, updates)
	require.Error(t, err)
	assert.ErrorIs(t, err, domainrepo.ErrSongNotFound)
}

func TestUpdateSongs_ReturnsErrSongNotFoundWhenSongSoftDeletedDuringUpdate(t *testing.T) {
	db := setupWorldsendUpdateDB(t)
	defer db.Close()

	ctx := context.Background()
	repo := &worldsendChartRepository{db: db}

	_, err := db.Exec(`
		INSERT INTO songs (id, display_id, title, artist, genre_id, bpm, released_at, official_idx, jacket, is_worldsend, is_deleted)
		VALUES (1, 'WE001', 'old title', 'old artist', 1, 180, '2024-01-01', 'WEIDX001', 'old.png', 1, 0)
	`)
	require.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO worldsend_charts (id, song_id, level_star, attribute, notes, notes_designer)
		VALUES (101, 1, 4, '狂', 1200, '旧作者')
	`)
	require.NoError(t, err)

	genreID := 1
	bpm := 180
	releasedAt := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)
	jacket := "old.png"
	updates := []*domainrepo.WorldsendUpdate{{
		Song: &entity.Song{
			DisplayID:  "WE001",
			Title:      "new title",
			Artist:     "new artist",
			GenreID:    &genreID,
			BPM:        &bpm,
			ReleasedAt: &releasedAt,
			Jacket:     &jacket,
		},
		Chart: nil,
	}}

	exec := &softDeletingSongExecutor{db: db, songID: 1}
	err = repo.UpdateSongs(ctx, exec, updates)
	require.Error(t, err)
	assert.ErrorIs(t, err, domainrepo.ErrSongNotFound)
}

func TestUpdateSongs_ReturnsErrorWhenUpdateOrSongIsNil(t *testing.T) {
	tests := []struct {
		name    string
		updates []*domainrepo.WorldsendUpdate
	}{
		{
			name:    "updateがnil",
			updates: []*domainrepo.WorldsendUpdate{nil},
		},
		{
			name:    "songがnil",
			updates: []*domainrepo.WorldsendUpdate{{Song: nil, Chart: nil}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &worldsendChartRepository{}

			err := repo.UpdateSongs(context.Background(), nil, tt.updates)
			require.Error(t, err)
			assert.ErrorContains(t, err, "song is nil")
		})
	}
}

func TestUpdateSongs_UpdatesMultipleRecordsWithMixedChartPresence(t *testing.T) {
	db := setupWorldsendUpdateDB(t)
	defer db.Close()

	ctx := context.Background()

	_, err := db.Exec(`
		INSERT INTO songs (id, display_id, title, artist, genre_id, bpm, released_at, official_idx, jacket, is_worldsend, is_deleted)
		VALUES
			(1, 'WE001', 'old title 1', 'old artist 1', 1, 180, '2024-01-01', 'WEIDX001', 'old1.png', 1, 0),
			(2, 'WE002', 'old title 2', 'old artist 2', 2, 190, '2024-01-02', 'WEIDX002', 'old2.png', 1, 0)
	`)
	require.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO worldsend_charts (id, song_id, level_star, attribute, notes, notes_designer)
		VALUES
			(101, 1, 3, '狂', 1000, '作者1'),
			(102, 2, 4, '止', 1100, '作者2')
	`)
	require.NoError(t, err)

	genreID1 := 10
	bpm1 := 200
	releasedAt1 := time.Date(2025, time.January, 10, 0, 0, 0, 0, time.UTC)
	jacket1 := "new1.png"

	genreID2 := 20
	bpm2 := 210
	releasedAt2 := time.Date(2025, time.February, 10, 0, 0, 0, 0, time.UTC)
	jacket2 := "new2.png"
	n := notes.Notes(2200)

	updates := []*domainrepo.WorldsendUpdate{
		{
			Song: &entity.Song{
				DisplayID:  "WE001",
				Title:      "new title 1",
				Artist:     "new artist 1",
				GenreID:    &genreID1,
				BPM:        &bpm1,
				ReleasedAt: &releasedAt1,
				Jacket:     &jacket1,
			},
			Chart: nil,
		},
		{
			Song: &entity.Song{
				DisplayID:  "WE002",
				Title:      "new title 2",
				Artist:     "new artist 2",
				GenreID:    &genreID2,
				BPM:        &bpm2,
				ReleasedAt: &releasedAt2,
				Jacket:     &jacket2,
			},
			Chart: &entity.WorldsendChart{
				LevelStar:     levelStarPtrForWorldsendUpdateTest(t, 5),
				Attribute:     stringPtrForWorldsendSaveTest("改"),
				Notes:         &n,
				NotesDesigner: stringPtrForWorldsendSaveTest("新作者2"),
			},
		},
	}

	repo := &worldsendChartRepository{db: db}
	err = repo.UpdateSongs(ctx, db, updates)
	require.NoError(t, err)

	var song1 struct {
		Title  string `db:"title"`
		Artist string `db:"artist"`
	}
	err = db.Get(&song1, `SELECT title, artist FROM songs WHERE id = 1`)
	require.NoError(t, err)
	assert.Equal(t, "new title 1", song1.Title)
	assert.Equal(t, "new artist 1", song1.Artist)

	var song2 struct {
		Title  string `db:"title"`
		Artist string `db:"artist"`
	}
	err = db.Get(&song2, `SELECT title, artist FROM songs WHERE id = 2`)
	require.NoError(t, err)
	assert.Equal(t, "new title 2", song2.Title)
	assert.Equal(t, "new artist 2", song2.Artist)

	var chart1 struct {
		LevelStar     *int    `db:"level_star"`
		Attribute     *string `db:"attribute"`
		Notes         *int    `db:"notes"`
		NotesDesigner *string `db:"notes_designer"`
	}
	err = db.Get(&chart1, `SELECT level_star, attribute, notes, notes_designer FROM worldsend_charts WHERE id = 101`)
	require.NoError(t, err)
	require.NotNil(t, chart1.LevelStar)
	assert.Equal(t, 3, *chart1.LevelStar)
	require.NotNil(t, chart1.Attribute)
	assert.Equal(t, "狂", *chart1.Attribute)
	require.NotNil(t, chart1.Notes)
	assert.Equal(t, 1000, *chart1.Notes)
	require.NotNil(t, chart1.NotesDesigner)
	assert.Equal(t, "作者1", *chart1.NotesDesigner)

	var chart2 struct {
		LevelStar     *int    `db:"level_star"`
		Attribute     *string `db:"attribute"`
		Notes         *int    `db:"notes"`
		NotesDesigner *string `db:"notes_designer"`
	}
	err = db.Get(&chart2, `SELECT level_star, attribute, notes, notes_designer FROM worldsend_charts WHERE id = 102`)
	require.NoError(t, err)
	require.NotNil(t, chart2.LevelStar)
	assert.Equal(t, 5, *chart2.LevelStar)
	require.NotNil(t, chart2.Attribute)
	assert.Equal(t, "改", *chart2.Attribute)
	require.NotNil(t, chart2.Notes)
	assert.Equal(t, 2200, *chart2.Notes)
	require.NotNil(t, chart2.NotesDesigner)
	assert.Equal(t, "新作者2", *chart2.NotesDesigner)
}

func levelStarPtrForWorldsendUpdateTest(t *testing.T, value int) *levelstar.LevelStar {
	t.Helper()

	ls, err := levelstar.NewLevelStar(value)
	require.NoError(t, err)

	return &ls
}
