package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/displaylevel"
	"github.com/jmoiron/sqlx"
)

func (r *courseRepository) FindTrackCandidates(ctx context.Context, exec domainrepo.Executor, courseID int, tracks []entity.CourseTrack) (map[int][]entity.CourseChartCandidate, error) {
	result := make(map[int][]entity.CourseChartCandidate, len(tracks))
	if len(tracks) == 0 {
		return result, nil
	}

	const regularColumns = `s.display_id AS song_id, d.name AS difficulty, s.title,
		ch.const, ch.is_const_unknown, s.jacket,
		NULL AS attribute, NULL AS level_star, NULL AS notes`
	const regularJoins = ` FROM charts ch
		INNER JOIN songs s ON s.id = ch.song_id
		INNER JOIN difficulties d ON d.id = ch.difficulty_id`
	const regularAvailable = `s.is_deleted = FALSE AND s.is_worldsend = FALSE`
	const worldsendColumns = `s.display_id AS song_id, 'WORLDSEND' AS difficulty, s.title,
		0 AS const, FALSE AS is_const_unknown, s.jacket,
		wch.attribute, wch.level_star, wch.notes`
	const worldsendJoins = ` FROM worldsend_charts wch
		INNER JOIN songs s ON s.id = wch.song_id`
	const worldsendAvailable = `s.is_deleted = FALSE AND s.is_worldsend = TRUE`

	queries := []string{
		`SELECT tc.track_no, ` + regularColumns + regularJoins + `
		INNER JOIN course_track_charts tc ON tc.chart_id = ch.id
		WHERE tc.course_id = ? AND ` + regularAvailable,
		`SELECT tc.track_no, ` + worldsendColumns + worldsendJoins + `
		INNER JOIN course_track_worldsend_charts tc ON tc.worldsend_chart_id = wch.id
		WHERE tc.course_id = ? AND ` + worldsendAvailable,
	}
	args := []any{courseID, courseID}
	for _, track := range tracks {
		if track.SelectionType != entity.CourseTrackRandomFilter {
			continue
		}
		filterClauses, filterArgs := courseTrackFilterPredicates(track.Filter)

		queries = append(queries, `SELECT ? AS track_no, `+regularColumns+regularJoins+`
			INNER JOIN genres g ON g.id = s.genre_id
			WHERE `+regularAvailable+` AND `+strings.Join(filterClauses, " AND "))
		args = append(args, track.Number)
		args = append(args, filterArgs...)
	}
	query, args, err := sqlx.In(strings.Join(queries, ` UNION ALL `)+
		` ORDER BY track_no, song_id, difficulty`, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to bind course candidate filters: %w", err)
	}
	var rows []struct {
		Number         int     `db:"track_no"`
		SongID         string  `db:"song_id"`
		Difficulty     string  `db:"difficulty"`
		Title          string  `db:"title"`
		Const          float64 `db:"const"`
		IsConstUnknown bool    `db:"is_const_unknown"`
		Jacket         *string `db:"jacket"`
		Attribute      *string `db:"attribute"`
		LevelStar      *int    `db:"level_star"`
		Notes          *int    `db:"notes"`
	}
	if err := exec.SelectContext(ctx, &rows, exec.Rebind(query), args...); err != nil {
		return nil, fmt.Errorf("failed to find course candidate charts: %w", err)
	}
	for _, row := range rows {
		result[row.Number] = append(result[row.Number], entity.CourseChartCandidate{
			CourseChartRef: entity.CourseChartRef{SongID: row.SongID, Difficulty: row.Difficulty},
			Title:          row.Title, Const: row.Const, IsConstUnknown: row.IsConstUnknown, Jacket: row.Jacket,
			Attribute: row.Attribute, LevelStar: row.LevelStar, Notes: row.Notes,
		})
	}
	return result, nil
}

func courseTrackFilterPredicates(filter *entity.CourseTrackFilter) ([]string, []any) {

	clauses := make([]string, 0, 4)
	var args []any
	if filter.MinLevel != nil {
		clauses = append(clauses, "ch.const >= ?")
		args = append(args, float64(filter.MinLevel.LowerTenths())/10)
	}
	if filter.MaxLevel != nil {
		clauses = append(clauses, "ch.const <= ?")
		args = append(args, float64(filter.MaxLevel.UpperTenths())/10)
	}
	if filter.MinLevel != nil || filter.MaxLevel != nil {
		// 10未満は表示レベル帯の端点だけを候補にし、song-batchと同じ判定にします。
		clauses = append(clauses, "(ch.const >= ? OR MOD(ch.const * 10, ?) = 0)")
		args = append(args, float64(displaylevel.FineConstThresholdTenths)/10, displaylevel.BandWidthTenths)
	}
	if len(filter.Genres) > 0 {
		clauses = append(clauses, "g.name IN (?)")
		args = append(args, filter.Genres)
	}
	return clauses, args
}
