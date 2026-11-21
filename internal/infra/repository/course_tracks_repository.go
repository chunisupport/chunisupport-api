package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/displaylevel"
	"github.com/chunisupport/chunisupport-api/internal/info"
)

type courseTrackRow struct {
	CourseID      int     `db:"course_id"`
	Number        int     `db:"track_no"`
	SelectionType string  `db:"selection_type"`
	Filter        []byte  `db:"filter"`
	SongID        *string `db:"song_id"`
	Difficulty    *string `db:"difficulty"`
}

type courseTrackFilterJSON struct {
	MinLevel string   `json:"min_level,omitempty"`
	MaxLevel string   `json:"max_level,omitempty"`
	Genres   []string `json:"genres,omitempty"`
}

func (r *courseRepository) FindTracks(ctx context.Context, exec domainrepo.Executor, courseIDs []int) (map[int][]entity.CourseTrack, error) {
	const query = `SELECT t.course_id, t.track_no, t.selection_type, t.filter,
		s.display_id AS song_id, d.name AS difficulty
		FROM course_tracks t
		LEFT JOIN course_track_charts tc ON tc.course_id = t.course_id AND tc.track_no = t.track_no
		LEFT JOIN charts ch ON ch.id = tc.chart_id
		LEFT JOIN songs s ON s.id = ch.song_id
		LEFT JOIN difficulties d ON d.id = ch.difficulty_id
		WHERE t.course_id IN (?)
		UNION ALL
		SELECT t.course_id, t.track_no, t.selection_type, t.filter,
			s.display_id AS song_id, 'WORLDSEND' AS difficulty
		FROM course_tracks t
		INNER JOIN course_track_worldsend_charts twc
			ON twc.course_id = t.course_id AND twc.track_no = t.track_no
		INNER JOIN worldsend_charts wch ON wch.id = twc.worldsend_chart_id
		INNER JOIN songs s ON s.id = wch.song_id
		WHERE t.course_id IN (?)
		ORDER BY course_id, track_no, song_id, difficulty`
	rows, err := selectModelsInChunks[int, courseTrackRow](ctx, exec, courseIDs, query, "course tracks", func(ids []int) []any {
		return []any{ids, ids}
	})
	if err != nil {
		return nil, err
	}
	result := make(map[int][]entity.CourseTrack, len(courseIDs))
	for _, row := range rows {
		tracks := result[row.CourseID]
		if len(tracks) == 0 || tracks[len(tracks)-1].Number != row.Number {
			filter, err := decodeCourseTrackFilter(row.Filter)
			if err != nil {
				return nil, fmt.Errorf("failed to restore course %d track %d filter: %w", row.CourseID, row.Number, err)
			}
			tracks = append(tracks, entity.CourseTrack{
				Number: row.Number, SelectionType: entity.CourseTrackSelectionType(row.SelectionType),
				Filter: filter,
			})
		}
		if row.SongID != nil && row.Difficulty != nil {
			track := &tracks[len(tracks)-1]
			track.Charts = append(track.Charts, entity.CourseChartRef{SongID: *row.SongID, Difficulty: *row.Difficulty})
		}
		result[row.CourseID] = tracks
	}
	return result, nil
}

func decodeCourseTrackFilter(raw []byte) (*entity.CourseTrackFilter, error) {
	if raw == nil {
		return nil, nil
	}
	var stored courseTrackFilterJSON
	if err := json.Unmarshal(raw, &stored); err != nil {
		return nil, err
	}
	filter := &entity.CourseTrackFilter{Genres: stored.Genres}
	if stored.MinLevel != "" {
		value, err := displaylevel.Parse(stored.MinLevel)
		if err != nil {
			return nil, err
		}
		filter.MinLevel = &value
	}
	if stored.MaxLevel != "" {
		value, err := displaylevel.Parse(stored.MaxLevel)
		if err != nil {
			return nil, err
		}
		filter.MaxLevel = &value
	}
	return filter, nil
}

func encodeCourseTrackFilter(filter *entity.CourseTrackFilter) (*string, error) {
	if filter == nil {
		return nil, nil
	}
	stored := courseTrackFilterJSON{Genres: filter.Genres}
	if filter.MinLevel != nil {
		stored.MinLevel = filter.MinLevel.String()
	}
	if filter.MaxLevel != nil {
		stored.MaxLevel = filter.MaxLevel.String()
	}
	raw, err := json.Marshal(stored)
	if err != nil {
		return nil, err
	}
	// interpolateParams では []byte がバイナリ文字列になり、MySQL の JSON 列へ書き込めません。
	return new(string(raw)), nil
}

func (r *courseRepository) ReplaceTracks(ctx context.Context, exec domainrepo.Executor, course *entity.Course) error {
	chartIDs, err := resolveCourseChartIDs(ctx, exec, course.ID, course.Tracks)
	if err != nil {
		return err
	}
	if err := validateCourseTrackGenres(ctx, exec, course.Tracks); err != nil {
		return err
	}

	trackRows := make([]map[string]any, 0, len(course.Tracks))
	chartRows := make([]map[string]any, 0, len(chartIDs))
	worldsendChartRows := make([]map[string]any, 0, len(chartIDs))
	for _, track := range course.Tracks {
		filter, err := encodeCourseTrackFilter(track.Filter)
		if err != nil {
			return fmt.Errorf("failed to encode course track filter: %w", err)
		}
		trackRows = append(trackRows, map[string]any{
			"course_id": course.ID, "track_no": track.Number, "selection_type": track.SelectionType,
			"filter": filter,
		})
		for _, chart := range track.Charts {
			row := map[string]any{
				"course_id": course.ID, "track_no": track.Number, "chart_id": chartIDs[chart],
			}
			if chart.Difficulty == entity.CourseTrackDifficultyWorldsend {
				worldsendChartRows = append(worldsendChartRows, row)
			} else {
				chartRows = append(chartRows, row)
			}
		}
	}

	if _, err := exec.ExecContext(ctx, "DELETE FROM course_tracks WHERE course_id = ?", course.ID); err != nil {
		return fmt.Errorf("failed to delete previous course tracks: %w", err)
	}
	if len(trackRows) > 0 {
		const insert = `INSERT INTO course_tracks (course_id, track_no, selection_type, filter)
			VALUES (:course_id, :track_no, :selection_type, :filter)`
		if _, err := exec.NamedExecContext(ctx, insert, trackRows); err != nil {
			return fmt.Errorf("failed to save course tracks: %w", err)
		}
	}
	for rows := range slices.Chunk(chartRows, info.BulkInsertChunkSize) {
		const insert = `INSERT INTO course_track_charts (course_id, track_no, chart_id)
			VALUES (:course_id, :track_no, :chart_id)`
		if _, err := exec.NamedExecContext(ctx, insert, rows); err != nil {
			return fmt.Errorf("failed to save course track charts: %w", err)
		}
	}
	for rows := range slices.Chunk(worldsendChartRows, info.BulkInsertChunkSize) {
		const insert = `INSERT INTO course_track_worldsend_charts (course_id, track_no, worldsend_chart_id)
			VALUES (:course_id, :track_no, :chart_id)`
		if _, err := exec.NamedExecContext(ctx, insert, rows); err != nil {
			return fmt.Errorf("failed to save course WORLD'S END track charts: %w", err)
		}
	}
	if _, err := exec.ExecContext(ctx, "UPDATE courses SET updated_at = CURRENT_TIMESTAMP WHERE id = ?", course.ID); err != nil {
		return fmt.Errorf("failed to update course timestamp: %w", err)
	}
	return nil
}

// resolveCourseChartIDs は譜面参照を内部IDに変換します。
// 削除済み楽曲の譜面は新しく登録できませんが、GETで返した構成をそのまま送り返せるよう、
// このコースに登録済みの譜面だけは削除後も保存できます。
func resolveCourseChartIDs(ctx context.Context, exec domainrepo.Executor, courseID int, tracks []entity.CourseTrack) (map[entity.CourseChartRef]int, error) {
	requestedRefs := make(map[entity.CourseChartRef]struct{})
	regularSongs := make(map[string]struct{})
	worldsendSongs := make(map[string]struct{})
	for _, track := range tracks {
		for _, chart := range track.Charts {
			requestedRefs[chart] = struct{}{}
			if chart.Difficulty == entity.CourseTrackDifficultyWorldsend {
				worldsendSongs[chart.SongID] = struct{}{}
			} else {
				regularSongs[chart.SongID] = struct{}{}
			}
		}
	}
	type regularChartRow struct {
		ID         int    `db:"id"`
		SongID     string `db:"song_id"`
		Difficulty string `db:"difficulty"`
	}
	result := make(map[entity.CourseChartRef]int, len(requestedRefs))
	regularSongIDs := make([]string, 0, len(regularSongs))
	for songID := range regularSongs {
		regularSongIDs = append(regularSongIDs, songID)
	}
	const regularQuery = `SELECT ch.id, s.display_id AS song_id, d.name AS difficulty
		FROM charts ch INNER JOIN songs s ON s.id = ch.song_id
		INNER JOIN difficulties d ON d.id = ch.difficulty_id
		WHERE s.is_worldsend = FALSE AND s.display_id IN (?)
			AND (s.is_deleted = FALSE OR EXISTS (
				SELECT 1 FROM course_track_charts tc WHERE tc.course_id = ? AND tc.chart_id = ch.id))`
	regularRows, err := selectModelsInChunks[string, regularChartRow](ctx, exec, regularSongIDs, regularQuery, "course chart references", func(ids []string) []any {
		return []any{ids, courseID}
	})
	if err != nil {
		return nil, err
	}
	for _, row := range regularRows {
		ref := entity.CourseChartRef{SongID: row.SongID, Difficulty: row.Difficulty}
		if _, needed := requestedRefs[ref]; needed {
			result[ref] = row.ID
		}
	}
	type worldsendChartRow struct {
		ID     int    `db:"id"`
		SongID string `db:"song_id"`
	}
	worldsendSongIDs := make([]string, 0, len(worldsendSongs))
	for songID := range worldsendSongs {
		worldsendSongIDs = append(worldsendSongIDs, songID)
	}
	const worldsendQuery = `SELECT wc.id, s.display_id AS song_id
		FROM worldsend_charts wc INNER JOIN songs s ON s.id = wc.song_id
		WHERE s.is_worldsend = TRUE AND s.display_id IN (?)
			AND (s.is_deleted = FALSE OR EXISTS (
				SELECT 1 FROM course_track_worldsend_charts tc WHERE tc.course_id = ? AND tc.worldsend_chart_id = wc.id))`
	worldsendRows, err := selectModelsInChunks[string, worldsendChartRow](ctx, exec, worldsendSongIDs, worldsendQuery, "course WORLD'S END chart references", func(ids []string) []any {
		return []any{ids, courseID}
	})
	if err != nil {
		return nil, err
	}
	for _, row := range worldsendRows {
		ref := entity.CourseChartRef{SongID: row.SongID, Difficulty: entity.CourseTrackDifficultyWorldsend}
		if _, needed := requestedRefs[ref]; needed {
			result[ref] = row.ID
		}
	}
	if len(result) != len(requestedRefs) {
		return nil, fmt.Errorf("%w: 指定された譜面が見つかりません", domainrepo.ErrInvalidCourseTracks)
	}
	return result, nil
}

func validateCourseTrackGenres(ctx context.Context, exec domainrepo.Executor, tracks []entity.CourseTrack) error {
	requested := make(map[string]struct{})
	for _, track := range tracks {
		if track.Filter != nil {
			for _, genre := range track.Filter.Genres {
				requested[genre] = struct{}{}
			}
		}
	}
	if len(requested) == 0 {
		return nil
	}
	var genres []string
	if err := exec.SelectContext(ctx, &genres, "SELECT name FROM genres"); err != nil {
		return fmt.Errorf("failed to load course track genres: %w", err)
	}
	for genre := range requested {
		if !slices.Contains(genres, genre) {
			return fmt.Errorf("%w: ジャンル %q が見つかりません", domainrepo.ErrInvalidCourseTracks, genre)
		}
	}
	return nil
}
