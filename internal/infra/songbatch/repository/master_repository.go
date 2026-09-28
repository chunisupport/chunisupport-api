package repository

import (
	"context"
	"fmt"
	apirepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/songbatch"
	"strings"

	"github.com/chunisupport/chunisupport-api/internal/domain/songbatch/entity"
	"github.com/chunisupport/chunisupport-api/internal/info"
)

// difficultyRepositoryImpl は DifficultyRepository のインフラ層実装です。
type difficultyRepositoryImpl struct {
	db apirepo.Executor
}

type courseRepositoryImpl struct {
	db apirepo.Executor
}

// NewCourseRepository は CourseRepository の実装を生成します。
func NewCourseRepository(db apirepo.Executor) songbatch.CourseRepository {
	return &courseRepositoryImpl{db: db}
}

type existingCourseRow struct {
	OfficialIdx   string `db:"official_idx"`
	Name          string `db:"name"`
	CourseClassID int    `db:"course_class_id"`
	IsDeleted     bool   `db:"is_deleted"`
}

// SaveAll はクラスと既存コースを一括取得した後、未登録のコースをまとめて登録し、変更のあるコースだけ更新します。
// InnoDB は INSERT ... ON DUPLICATE KEY UPDATE が既存行を更新する場合も AUTO_INCREMENT を消費するため、
// 既存コースにはINSERTを発行しません。
func (r *courseRepositoryImpl) SaveAll(ctx context.Context, courses []entity.Course) error {
	if len(courses) == 0 {
		return nil
	}

	rows, err := r.db.QueryContext(ctx, `SELECT id, name FROM course_classes`)
	if err != nil {
		return fmt.Errorf("failed to query course classes: %w", err)
	}
	defer rows.Close()

	classIDs := make(map[string]int)
	for rows.Next() {
		var id int
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return fmt.Errorf("failed to scan course class: %w", err)
		}
		classIDs[strings.ToLower(strings.TrimSpace(name))] = id
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("error during course classes iteration: %w", err)
	}

	existingRows := []existingCourseRow{}
	if err := r.db.SelectContext(ctx, &existingRows, `SELECT official_idx, name, course_class_id, is_deleted FROM courses`); err != nil {
		return fmt.Errorf("failed to query courses: %w", err)
	}
	existing := make(map[string]existingCourseRow, len(existingRows))
	for _, row := range existingRows {
		existing[row.OfficialIdx] = row
	}

	type courseWithClass struct {
		course  entity.Course
		classID int
	}
	var inserts []courseWithClass
	for _, course := range courses {
		classID, ok := classIDs[strings.ToLower(strings.TrimSpace(course.ClassName))]
		if !ok {
			return fmt.Errorf("course %q references unknown class %q", course.OfficialIdx, course.ClassName)
		}
		row, found := existing[course.OfficialIdx]
		if !found {
			inserts = append(inserts, courseWithClass{course: course, classID: classID})
			continue
		}
		if row.Name == course.Name && row.CourseClassID == classID && !row.IsDeleted {
			continue
		}
		// 既存コースの変更は公式データ更新時のみ発生し件数が限られるため、1件ずつ更新する。
		if _, err := r.db.ExecContext(ctx, `UPDATE courses SET name = ?, course_class_id = ?, is_deleted = 0 WHERE official_idx = ?`,
			course.Name, classID, course.OfficialIdx); err != nil {
			return fmt.Errorf("failed to update course %q: %w", course.OfficialIdx, err)
		}
	}

	for start := 0; start < len(inserts); start += info.SongBatchBulkInsertChunkSize {
		end := min(start+info.SongBatchBulkInsertChunkSize, len(inserts))
		chunk := inserts[start:end]
		values := make([]string, len(chunk))
		args := make([]any, 0, len(chunk)*4)
		for i, item := range chunk {
			values[i] = "(?, ?, ?, ?, 0)"
			args = append(args, item.course.DisplayID.String(), item.course.OfficialIdx, item.course.Name, item.classID)
		}

		query := `INSERT INTO courses (display_id, official_idx, name, course_class_id, is_deleted) VALUES ` + strings.Join(values, ",")
		if _, err := r.db.ExecContext(ctx, query, args...); err != nil {
			return fmt.Errorf("failed to insert courses (%d-%d): %w", start, end, err)
		}
	}

	return nil
}

// NewDifficultyRepository は DifficultyRepository の実装を生成します。
func NewDifficultyRepository(db apirepo.Executor) songbatch.DifficultyRepository {
	return &difficultyRepositoryImpl{db: db}
}

// FindAll は全ての難易度を取得します。
func (r *difficultyRepositoryImpl) FindAll(ctx context.Context) ([]songbatch.Difficulty, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, name FROM difficulties`)
	if err != nil {
		return nil, fmt.Errorf("failed to query difficulties: %w", err)
	}
	defer rows.Close()

	var result []songbatch.Difficulty
	for rows.Next() {
		var d songbatch.Difficulty
		if err := rows.Scan(&d.ID, &d.Name); err != nil {
			return nil, fmt.Errorf("failed to scan difficulty: %w", err)
		}
		d.Name = strings.TrimSpace(d.Name)
		result = append(result, d)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during difficulties iteration: %w", err)
	}

	return result, nil
}

// genreRepositoryImpl は GenreRepository のインフラ層実装です。
type genreRepositoryImpl struct {
	db apirepo.Executor
}

// NewGenreRepository は GenreRepository の実装を生成します。
func NewGenreRepository(db apirepo.Executor) songbatch.GenreRepository {
	return &genreRepositoryImpl{db: db}
}

// FindAll は全てのジャンルを取得します。
func (r *genreRepositoryImpl) FindAll(ctx context.Context) ([]songbatch.Genre, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, name FROM genres`)
	if err != nil {
		return nil, fmt.Errorf("failed to query genres: %w", err)
	}
	defer rows.Close()

	var result []songbatch.Genre
	for rows.Next() {
		var g songbatch.Genre
		if err := rows.Scan(&g.ID, &g.Name); err != nil {
			return nil, fmt.Errorf("failed to scan genre: %w", err)
		}
		g.Name = strings.TrimSpace(g.Name)
		result = append(result, g)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during genres iteration: %w", err)
	}

	return result, nil
}
