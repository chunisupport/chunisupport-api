package repository_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/displaylevel"
	infrarepo "github.com/chunisupport/chunisupport-api/internal/infra/repository"
	"github.com/chunisupport/chunisupport-api/internal/infra/transaction"
	"github.com/chunisupport/chunisupport-api/internal/testutil/mysqltest"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCourseTracksMySQL_同名のWorldsend譜面を別々に保存する(t *testing.T) {
	ctx := context.Background()
	db := mysqltest.NewDatabase(t, "")
	uc := usecase.NewCourseUsecase(db, transaction.NewTransactionManager(db), infrarepo.NewCourseRepository(db), nil, nil, nil)
	const courseID = "aaaabbbbccccdddd"
	_, err := db.ExecContext(ctx, "INSERT INTO courses (display_id, official_idx, name, course_class_id) VALUES (?, '100', '検証コース', 1)", courseID)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `
		INSERT INTO songs (id, display_id, title, artist, genre_id, official_idx)
		VALUES (1, '0000000000000001', '通常楽曲', '作曲者', 1, '1');
		INSERT INTO charts (id, song_id, difficulty_id, const) VALUES (100, 1, 4, 14.7)`)
	require.NoError(t, err)

	ordinary := entity.CourseChartRef{SongID: "0000000000000001", Difficulty: "MASTER"}
	var variants []entity.CourseChartRef
	for _, notes := range []int{1563, 1520, 1556, 1598, 1498, 1575} {
		// 通常譜面とWORLD'S ENDの内部譜面IDが重なる場合も、両方を指定できることを検証する。
		chartID := 100 + len(variants)
		songID := 6 + len(variants)
		publicID := fmt.Sprintf("%016x", songID)
		_, err := db.ExecContext(ctx, `
			INSERT INTO songs (id, display_id, title, artist, genre_id, official_idx, is_worldsend)
			VALUES (?, ?, 'Random', '作曲者', 1, ?, TRUE)`, songID, publicID, fmt.Sprint(songID))
		require.NoError(t, err)
		_, err = db.ExecContext(ctx, `
			INSERT INTO worldsend_charts (id, song_id, attribute, level_star, notes) VALUES (?, ?, '分', 3, ?)`,
			chartID, songID, notes)
		require.NoError(t, err)
		variants = append(variants, entity.CourseChartRef{SongID: publicID, Difficulty: "WORLDSEND"})
	}

	level, err := displaylevel.Parse("14+")
	require.NoError(t, err)
	pool := append([]entity.CourseChartRef{ordinary}, variants...)
	require.NoError(t, uc.ReplaceTracks(ctx, courseID, []entity.CourseTrack{
		{Number: 1, SelectionType: entity.CourseTrackFixed, Charts: variants[:1]},
		{Number: 2, SelectionType: entity.CourseTrackRandomPool, Charts: pool},
		{Number: 3, SelectionType: entity.CourseTrackRandomFilter, Filter: &entity.CourseTrackFilter{MinLevel: &level, MaxLevel: &level}},
	}))
	detail, err := uc.Get(ctx, courseID, false)
	require.NoError(t, err)
	require.Len(t, detail.Tracks, 3)
	assert.Equal(t, variants[:1], detail.Tracks[0].Charts)
	require.Len(t, detail.Tracks[0].Candidates, 1)
	candidate := detail.Tracks[0].Candidates[0]
	assert.Equal(t, "Random", candidate.Title)
	require.NotNil(t, candidate.Attribute)
	require.NotNil(t, candidate.LevelStar)
	require.NotNil(t, candidate.Notes)
	assert.Equal(t, "分", *candidate.Attribute)
	assert.Equal(t, 3, *candidate.LevelStar)
	assert.Equal(t, 1563, *candidate.Notes)
	assert.ElementsMatch(t, pool, detail.Tracks[1].Charts)
	assert.Len(t, detail.Tracks[1].Candidates, len(pool))
	require.Len(t, detail.Tracks[2].Candidates, 1)
	assert.Equal(t, ordinary, detail.Tracks[2].Candidates[0].CourseChartRef)

	_, err = db.ExecContext(ctx, "UPDATE songs SET is_deleted = TRUE WHERE display_id = ?", variants[0].SongID)
	require.NoError(t, err)
	detail, err = uc.Get(ctx, courseID, false)
	require.NoError(t, err)
	assert.Equal(t, variants[:1], detail.Tracks[0].Charts)
	assert.Empty(t, detail.Tracks[0].Candidates)
	assert.ElementsMatch(t, pool, detail.Tracks[1].Charts)
	assert.Len(t, detail.Tracks[1].Candidates, len(pool)-1)
	// 削除済みのWORLD'S END譜面は、登録済みのコースには再保存できるが、別のコースへは登録できない。
	const otherCourseID = "aaaabbbbcccc0000"
	_, err = db.ExecContext(ctx, "INSERT INTO courses (display_id, official_idx, name, course_class_id) VALUES (?, '200', '別コース', 1)", otherCourseID)
	require.NoError(t, err)
	deletedFixed := []entity.CourseTrack{
		{Number: 1, SelectionType: entity.CourseTrackFixed, Charts: variants[:1]},
		{Number: 2, SelectionType: entity.CourseTrackRandomPool, Charts: pool},
		{Number: 3, SelectionType: entity.CourseTrackFixed, Charts: []entity.CourseChartRef{ordinary}},
	}
	require.NoError(t, uc.ReplaceTracks(ctx, courseID, deletedFixed))
	assert.ErrorIs(t, uc.ReplaceTracks(ctx, otherCourseID, deletedFixed), usecase.ErrInvalidCourseInput)

	_, err = db.ExecContext(ctx, "DELETE FROM worldsend_charts WHERE id = 100")
	require.Error(t, err)
	require.NoError(t, uc.ReplaceTracks(ctx, courseID, []entity.CourseTrack{}))
	_, err = db.ExecContext(ctx, "DELETE FROM worldsend_charts WHERE id = 100")
	require.NoError(t, err)
}
