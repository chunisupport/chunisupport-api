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
		INSERT INTO worldsend_charts (id, song_id, level_star, attribute, notes)
		VALUES (101, 1, 4, '狂', 1200)
	`)
	require.NoError(t, err)

	newGenreID := 2
	newBPM := 200
	releasedAt := time.Date(2025, time.March, 1, 0, 0, 0, 0, time.UTC)
	newJacket := "new.png"

	songs := []*entity.Song{{
		DisplayID:  "WE001",
		Title:      "new title",
		Artist:     "new artist",
		GenreID:    &newGenreID,
		BPM:        &newBPM,
		ReleasedAt: &releasedAt,
		Jacket:     &newJacket,
	}}
	charts := []*entity.WorldsendChart{nil}

	repo := &worldsendChartRepository{db: db}
	err = repo.UpdateSongs(ctx, db, songs, charts)
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
		LevelStar *int    `db:"level_star"`
		Attribute *string `db:"attribute"`
		Notes     *int    `db:"notes"`
	}
	err = db.Get(&chart, `SELECT level_star, attribute, notes FROM worldsend_charts WHERE id = 101`)
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
		INSERT INTO worldsend_charts (id, song_id, level_star, attribute, notes)
		VALUES (101, 1, 2, '狂', 1200)
	`)
	require.NoError(t, err)

	n := notes.Notes(1300)
	genreID := 1
	bpm := 180
	releasedAt := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)
	jacket := "old.png"
	songs := []*entity.Song{{
		DisplayID:  "WE001",
		Title:      "new title",
		Artist:     "new artist",
		GenreID:    &genreID,
		BPM:        &bpm,
		ReleasedAt: &releasedAt,
		Jacket:     &jacket,
	}}
	charts := []*entity.WorldsendChart{{
		LevelStar: levelStarPtrForWorldsendUpdateTest(t, 5),
		Attribute: stringPtrForWorldsendSaveTest("改"),
		Notes:     &n,
	}}

	repo := &worldsendChartRepository{db: db}
	err = repo.UpdateSongs(ctx, db, songs, charts)
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
	songs := []*entity.Song{{
		DisplayID: "WE999",
		Title:     "title",
		Artist:    "artist",
	}}
	charts := []*entity.WorldsendChart{{
		LevelStar: levelStarPtrForWorldsendUpdateTest(t, 5),
		Attribute: stringPtrForWorldsendSaveTest("狂"),
		Notes:     &n,
	}}

	err := repo.UpdateSongs(ctx, db, songs, charts)
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
		INSERT INTO worldsend_charts (id, song_id, level_star, attribute, notes)
		VALUES (101, 1, 4, '狂', 1200)
	`)
	require.NoError(t, err)

	songs := []*entity.Song{
		{DisplayID: "WE001", Title: "first", Artist: "artist1"},
		{DisplayID: "WE001", Title: "second", Artist: "artist2"},
	}
	charts := []*entity.WorldsendChart{nil, nil}

	err = repo.UpdateSongs(ctx, db, songs, charts)
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
		INSERT INTO worldsend_charts (id, song_id, level_star, attribute, notes)
		VALUES (101, 1, 4, '狂', 1200)
	`)
	require.NoError(t, err)

	n := notes.Notes(1300)
	genreID := 1
	bpm := 180
	releasedAt := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)
	jacket := "old.png"
	songs := []*entity.Song{{
		DisplayID:  "WE001",
		Title:      "new title",
		Artist:     "new artist",
		GenreID:    &genreID,
		BPM:        &bpm,
		ReleasedAt: &releasedAt,
		Jacket:     &jacket,
	}}
	charts := []*entity.WorldsendChart{{
		LevelStar: levelStarPtrForWorldsendUpdateTest(t, 5),
		Attribute: stringPtrForWorldsendSaveTest("改"),
		Notes:     &n,
	}}

	exec := &deletingChartExecutor{db: db, chartID: 101}
	err = repo.UpdateSongs(ctx, exec, songs, charts)
	require.Error(t, err)
	assert.ErrorIs(t, err, domainrepo.ErrSongNotFound)
}

func levelStarPtrForWorldsendUpdateTest(t *testing.T, value int) *levelstar.LevelStar {
	t.Helper()

	ls, err := levelstar.NewLevelStar(value)
	require.NoError(t, err)

	return &ls
}
