package repository_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/displaylevel"
	infrarepo "github.com/chunisupport/chunisupport-api/internal/infra/repository"
	"github.com/chunisupport/chunisupport-api/internal/infra/transaction"
	"github.com/chunisupport/chunisupport-api/internal/testutil/mysqltest"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCourseTracksMySQL_構成と現在の候補を取得する(t *testing.T) {
	ctx := context.Background()
	if dsn := os.Getenv("TEST_MYSQL_DSN"); dsn != "" {
		cfg, err := mysql.ParseDSN(dsn)
		require.NoError(t, err)
		cfg.InterpolateParams = true
		t.Setenv("TEST_MYSQL_DSN", cfg.FormatDSN())
	}
	db := mysqltest.NewDatabase(t, "")
	uc := usecase.NewCourseUsecase(db, transaction.NewTransactionManager(db), infrarepo.NewCourseRepository(db), nil, nil)
	const courseID = "aaaabbbbccccdddd"
	const emptyCourseID = "aaaabbbbcccc0000"
	_, err := db.ExecContext(ctx, `
		INSERT INTO courses (display_id, official_idx, name, course_class_id, updated_at) VALUES
			(?, '100', '構成あり', 1, '2020-01-01 00:00:00'),
			(?, '200', '構成なし', 1, '2020-01-01 00:00:00')`, courseID, emptyCourseID)
	require.NoError(t, err)

	insertChart := func(id int, genre string, constant float64) entity.CourseChartRef {
		t.Helper()
		songID := fmt.Sprintf("%016x", id)
		_, err := db.ExecContext(ctx, `
			INSERT INTO songs (id, display_id, title, artist, genre_id, official_idx)
			VALUES (?, ?, ?, '作曲者', (SELECT id FROM genres WHERE name = ?), ?)`,
			id, songID, fmt.Sprintf("楽曲%d", id), genre, fmt.Sprint(id))
		require.NoError(t, err)
		_, err = db.ExecContext(ctx, "INSERT INTO charts (id, song_id, difficulty_id, const) VALUES (?, ?, 4, ?)", id, id, constant)
		require.NoError(t, err)
		return entity.CourseChartRef{SongID: songID, Difficulty: "MASTER"}
	}
	fixed := insertChart(11, "ゲキマイ", 14.0)
	bandStart := insertChart(12, "ゲキマイ", 14.5)
	bandEnd := insertChart(13, "ゲキマイ", 14.9)
	otherGenre := insertChart(14, "ORIGINAL", 14.9)
	nextBand := insertChart(15, "ゲキマイ", 15.0)
	level, err := displaylevel.Parse("14+")
	require.NoError(t, err)
	tracks := []entity.CourseTrack{
		{Number: 3, SelectionType: entity.CourseTrackRandomPool, Charts: []entity.CourseChartRef{otherGenre}},
		{Number: 1, SelectionType: entity.CourseTrackFixed, Charts: []entity.CourseChartRef{fixed}},
		{Number: 2, SelectionType: entity.CourseTrackRandomFilter, Filter: &entity.CourseTrackFilter{MinLevel: &level, MaxLevel: &level, Genres: []string{"ゲキマイ"}}},
	}

	previous, err := uc.GetCoursesUpdatedAt(ctx)
	require.NoError(t, err)
	require.NotNil(t, previous)
	require.NoError(t, uc.ReplaceTracks(ctx, courseID, tracks))
	updated, err := uc.GetCoursesUpdatedAt(ctx)
	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.True(t, updated.After(*previous))
	assert.Equal(t, time.Duration(0), updated.Sub(updated.Truncate(time.Second)))

	list, err := uc.List(ctx, false)
	require.NoError(t, err)
	require.Len(t, list, 2)
	assert.Equal(t, emptyCourseID, list[1].DisplayID)
	require.NotNil(t, list[1].Tracks)
	assert.Empty(t, list[1].Tracks)
	require.Len(t, list[0].Tracks, 3)
	assert.Equal(t, []entity.CourseChartRef{fixed}, list[0].Tracks[0].Charts)
	assert.Nil(t, list[0].Tracks[0].Candidates)
	assert.Empty(t, list[0].Tracks[1].Charts)
	assert.Equal(t, tracks[2].Filter, list[0].Tracks[1].Filter)
	assert.Equal(t, []entity.CourseChartRef{otherGenre}, list[0].Tracks[2].Charts)

	candidates := func(expectedFixed, expectedFiltered, expectedPool []entity.CourseChartRef) {
		t.Helper()
		result, err := uc.Get(ctx, courseID, false)
		require.NoError(t, err)
		require.Len(t, result.Tracks, 3)
		for i, expected := range [][]entity.CourseChartRef{expectedFixed, expectedFiltered, expectedPool} {
			require.NotNil(t, result.Tracks[i].Candidates)
			actual := make([]entity.CourseChartRef, 0, len(result.Tracks[i].Candidates))
			for _, candidate := range result.Tracks[i].Candidates {
				actual = append(actual, candidate.CourseChartRef)
				assert.NotEmpty(t, candidate.Title)
			}
			assert.ElementsMatch(t, expected, actual)
		}
	}
	candidates([]entity.CourseChartRef{fixed}, []entity.CourseChartRef{bandStart, bandEnd}, []entity.CourseChartRef{otherGenre})

	newChart := insertChart(16, "ゲキマイ", 14.7)
	_, err = db.ExecContext(ctx, "UPDATE charts SET const = CASE id WHEN 12 THEN 14.4 WHEN 15 THEN 14.8 END WHERE id IN (12,15)")
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, "UPDATE songs SET is_deleted = TRUE WHERE id IN (11,14)")
	require.NoError(t, err)
	candidates(nil, []entity.CourseChartRef{bandEnd, nextBand, newChart}, nil)
	list, err = uc.List(ctx, false)
	require.NoError(t, err)
	assert.Equal(t, []entity.CourseChartRef{fixed}, list[0].Tracks[0].Charts)
	assert.Equal(t, []entity.CourseChartRef{otherGenre}, list[0].Tracks[2].Charts)

	_, err = db.ExecContext(ctx, "DELETE FROM charts WHERE id = 14")
	require.Error(t, err)

	invalidTracks := append([]entity.CourseTrack(nil), tracks...)
	invalidTracks[0].Charts = []entity.CourseChartRef{{SongID: "ffffffffffffffff", Difficulty: "MASTER"}}
	assert.ErrorIs(t, uc.ReplaceTracks(ctx, courseID, invalidTracks), usecase.ErrInvalidCourseInput)
	invalidTracks = append([]entity.CourseTrack(nil), tracks...)
	invalidTracks[2].Filter = &entity.CourseTrackFilter{Genres: []string{"存在しないジャンル"}}
	assert.ErrorIs(t, uc.ReplaceTracks(ctx, courseID, invalidTracks), usecase.ErrInvalidCourseInput)
	// 登録済みの削除済み譜面はそのまま再保存できるが、別のコースへ新たに登録することはできない。
	require.NoError(t, uc.ReplaceTracks(ctx, courseID, tracks))
	assert.ErrorIs(t, uc.ReplaceTracks(ctx, emptyCourseID, tracks), usecase.ErrInvalidCourseInput)
	candidates(nil, []entity.CourseChartRef{bandEnd, nextBand, newChart}, nil)

	_, err = db.ExecContext(ctx, "UPDATE songs SET is_deleted = FALSE WHERE id IN (11,14)")
	require.NoError(t, err)
	candidates([]entity.CourseChartRef{fixed}, []entity.CourseChartRef{bandEnd, nextBand, newChart}, []entity.CourseChartRef{otherGenre})

	lowChart := insertChart(17, "ゲキマイ", 9.5)
	insertChart(18, "ゲキマイ", 9.7)
	upperOnly := append([]entity.CourseTrack(nil), tracks...)
	upperOnly[2].Filter = &entity.CourseTrackFilter{MaxLevel: &level, Genres: []string{"ゲキマイ", "ORIGINAL"}}
	require.NoError(t, uc.ReplaceTracks(ctx, courseID, upperOnly))
	candidates([]entity.CourseChartRef{fixed}, []entity.CourseChartRef{fixed, bandStart, bandEnd, otherGenre, nextBand, newChart, lowChart}, []entity.CourseChartRef{otherGenre})

	require.NoError(t, uc.ReplaceTracks(ctx, courseID, []entity.CourseTrack{}))
	detail, err := uc.Get(ctx, courseID, false)
	require.NoError(t, err)
	require.NotNil(t, detail.Tracks)
	assert.Empty(t, detail.Tracks)

	require.NoError(t, uc.ReplaceTracks(ctx, courseID, tracks))
	_, err = db.ExecContext(ctx, "DELETE FROM courses WHERE display_id = ?", courseID)
	require.NoError(t, err)
	var rowCount int
	require.NoError(t, db.GetContext(ctx, &rowCount, "SELECT COUNT(*) FROM course_track_charts"))
	assert.Zero(t, rowCount)
	require.NoError(t, db.GetContext(ctx, &rowCount, "SELECT COUNT(*) FROM course_tracks"))
	assert.Zero(t, rowCount)
}
