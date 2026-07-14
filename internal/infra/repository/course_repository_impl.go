package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/chunisupport/chunisupport-api/internal/infra/models"
	"github.com/jmoiron/sqlx"
)

type courseRepository struct{ db *sqlx.DB }

func NewCourseRepository(db *sqlx.DB) domainrepo.CourseRepository { return &courseRepository{db: db} }

const courseColumns = `c.id, c.display_id, c.official_idx, c.name, c.course_class_id, c.is_deleted, c.updated_at,
cc.name AS course_class_name, cc.sort_order AS course_class_sort_order`

// FindLatestUpdatedAt は courses.updated_at の最大値を返します。
// 削除済みを含む全コースを対象とし、コースが0件の場合は nil を返します。
func (r *courseRepository) FindLatestUpdatedAt(ctx context.Context, exec domainrepo.Executor) (*time.Time, error) {
	if exec == nil {
		exec = r.db
	}
	var updatedAtRaw sql.NullString
	if err := exec.GetContext(ctx, &updatedAtRaw, `SELECT MAX(updated_at) FROM courses`); err != nil {
		return nil, fmt.Errorf("failed to find latest course updated_at: %w", err)
	}
	if !updatedAtRaw.Valid || updatedAtRaw.String == "" {
		return nil, nil
	}
	parsedUpdatedAt, err := parseLatestUpdatedAt(updatedAtRaw.String)
	if err != nil {
		return nil, err
	}
	return &parsedUpdatedAt, nil
}

func (r *courseRepository) FindAll(ctx context.Context, exec domainrepo.Executor, includeDeleted bool) ([]*entity.Course, error) {
	if exec == nil {
		exec = r.db
	}
	query := `SELECT ` + courseColumns + ` FROM courses c INNER JOIN course_classes cc ON cc.id = c.course_class_id`
	if !includeDeleted {
		query += ` WHERE c.is_deleted = FALSE`
	}
	query += ` ORDER BY cc.sort_order, (c.official_idx + 0), c.official_idx`
	var rows []models.CourseModel
	if err := exec.SelectContext(ctx, &rows, query); err != nil {
		return nil, fmt.Errorf("failed to find courses: %w", err)
	}
	result := make([]*entity.Course, 0, len(rows))
	for _, row := range rows {
		course, err := row.ToEntity()
		if err != nil {
			return nil, fmt.Errorf("failed to restore course: %w", err)
		}
		result = append(result, course)
	}
	return result, nil
}

func (r *courseRepository) FindByDisplayID(ctx context.Context, exec domainrepo.Executor, displayID string, includeDeleted bool) (*entity.Course, error) {
	if exec == nil {
		exec = r.db
	}
	query := `SELECT ` + courseColumns + ` FROM courses c INNER JOIN course_classes cc ON cc.id = c.course_class_id WHERE c.display_id = ?`
	if !includeDeleted {
		query += ` AND c.is_deleted = FALSE`
	}
	var row models.CourseModel
	if err := exec.GetContext(ctx, &row, query, displayID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domainrepo.ErrCourseNotFound
		}
		return nil, fmt.Errorf("failed to find course by display_id: %w", err)
	}
	return row.ToEntity()
}

func (r *courseRepository) FindByOfficialIdx(ctx context.Context, exec domainrepo.Executor, idx string, includeDeleted bool) (*entity.Course, error) {
	if exec == nil {
		exec = r.db
	}
	query := `SELECT ` + courseColumns + ` FROM courses c INNER JOIN course_classes cc ON cc.id = c.course_class_id WHERE c.official_idx = ?`
	if !includeDeleted {
		query += ` AND c.is_deleted = FALSE`
	}
	var row models.CourseModel
	if err := exec.GetContext(ctx, &row, query, idx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domainrepo.ErrCourseNotFound
		}
		return nil, fmt.Errorf("failed to find course: %w", err)
	}
	return row.ToEntity()
}

func (r *courseRepository) FindByOfficialIdxList(ctx context.Context, exec domainrepo.Executor, indexes []string) (map[string]*entity.Course, error) {
	result := make(map[string]*entity.Course, len(indexes))
	if len(indexes) == 0 {
		return result, nil
	}
	if exec == nil {
		exec = r.db
	}
	rows, err := selectModelsInChunks[string, models.CourseModel](ctx, exec, indexes, `SELECT `+courseColumns+` FROM courses c INNER JOIN course_classes cc ON cc.id=c.course_class_id WHERE c.is_deleted=FALSE AND c.official_idx IN (?)`, "courses", func(v []string) []any { return []any{v} })
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		course, err := row.ToEntity()
		if err != nil {
			return nil, fmt.Errorf("failed to restore course: %w", err)
		}
		result[course.OfficialIdx] = course
	}
	return result, nil
}

func (r *courseRepository) FindClassByName(ctx context.Context, exec domainrepo.Executor, name string) (*entity.CourseClass, error) {
	if exec == nil {
		exec = r.db
	}
	var item struct {
		ID        int    `db:"id"`
		Name      string `db:"name"`
		SortOrder int    `db:"sort_order"`
	}
	if err := exec.GetContext(ctx, &item, `SELECT id, name, sort_order FROM course_classes WHERE name = ?`, name); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domainrepo.ErrCourseClassNotFound
		}
		return nil, err
	}
	return &entity.CourseClass{ID: item.ID, Name: item.Name, SortOrder: item.SortOrder}, nil
}

func (r *courseRepository) Create(ctx context.Context, exec domainrepo.Executor, course *entity.Course) error {
	if exec == nil {
		exec = r.db
	}
	result, err := exec.ExecContext(ctx, `INSERT INTO courses (display_id, official_idx, name, course_class_id, is_deleted) VALUES (?, ?, ?, ?, ?)`, course.DisplayID.String(), course.OfficialIdx, course.Name, course.CourseClassID, course.IsDeleted)
	if err != nil {
		if wrapped := wrapOfficialIdxDuplicateError(err); wrapped != err {
			return wrapped
		}
		return fmt.Errorf("failed to create course: %w", err)
	}
	id, err := result.LastInsertId()
	if err == nil {
		course.ID = int(id)
	}
	return nil
}

func (r *courseRepository) Save(ctx context.Context, exec domainrepo.Executor, course *entity.Course) error {
	if exec == nil {
		exec = r.db
	}
	_, err := exec.ExecContext(ctx, `UPDATE courses SET name=?, course_class_id=?, is_deleted=? WHERE id=?`, course.Name, course.CourseClassID, course.IsDeleted, course.ID)
	if err != nil {
		return fmt.Errorf("failed to save course: %w", err)
	}
	return nil
}

func (r *courseRepository) FindRecordsByPlayerID(ctx context.Context, exec domainrepo.Executor, playerID int, includeDeleted, includeNoPlay bool) ([]*entity.PlayerCourseRecord, error) {
	if exec == nil {
		exec = r.db
	}
	join := `INNER JOIN player_course_records pcr ON pcr.course_id=c.id AND pcr.player_id=?`
	if includeNoPlay {
		join = `LEFT JOIN player_course_records pcr ON pcr.course_id=c.id AND pcr.player_id=?`
	}
	query := `SELECT COALESCE(pcr.player_id, ?) player_id, c.id course_id, COALESCE(pcr.score,0) score, COALESCE(pcr.is_clear,FALSE) is_clear, COALESCE(pcr.combo_lamp_id,1) combo_lamp_id, pcr.updated_at,
	c.display_id, c.official_idx, c.name course_name, c.course_class_id, cc.name course_class_name, cc.sort_order course_class_sort_order, cl.name combo_lamp_name
	FROM courses c INNER JOIN course_classes cc ON cc.id=c.course_class_id ` + join + ` LEFT JOIN combo_lamp_types cl ON cl.id=pcr.combo_lamp_id`
	if !includeDeleted {
		query += ` WHERE c.is_deleted=FALSE`
	}
	query += ` ORDER BY cc.sort_order, (c.official_idx + 0), c.official_idx`
	var rows []models.PlayerCourseRecordModel
	if err := exec.SelectContext(ctx, &rows, query, playerID, playerID); err != nil {
		return nil, err
	}
	result := make([]*entity.PlayerCourseRecord, 0, len(rows))
	for _, row := range rows {
		record, err := row.ToEntity()
		if err != nil {
			return nil, fmt.Errorf("failed to restore course record: %w", err)
		}
		result = append(result, record)
	}
	return result, nil
}

func (r *courseRepository) FindRecordStatesByCourseIDs(ctx context.Context, exec domainrepo.Executor, playerID int, ids []int) (map[int]domainrepo.CourseRecordState, error) {
	result := make(map[int]domainrepo.CourseRecordState, len(ids))
	if len(ids) == 0 {
		return result, nil
	}
	rows, err := selectModelsInChunks[int, struct {
		CourseID    int       `db:"course_id"`
		Score       int       `db:"score"`
		IsClear     bool      `db:"is_clear"`
		ComboLampID int       `db:"combo_lamp_id"`
		UpdatedAt   time.Time `db:"updated_at"`
	}](ctx, exec, ids, `SELECT course_id,score,is_clear,combo_lamp_id,updated_at FROM player_course_records WHERE player_id=? AND course_id IN (?)`, "course record states", func(v []int) []any { return []any{playerID, v} })
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.CourseID] = domainrepo.CourseRecordState{Score: row.Score, IsClear: row.IsClear, ComboLampID: row.ComboLampID, UpdatedAt: row.UpdatedAt}
	}
	return result, nil
}

func (r *courseRepository) UpsertRecords(ctx context.Context, exec domainrepo.Executor, records []domainrepo.CourseRecordForUpsert) error {
	if len(records) == 0 {
		return nil
	}
	rows := make([]map[string]any, 0, len(records))
	for _, v := range records {
		rows = append(rows, map[string]any{"player_id": v.PlayerID, "course_id": v.CourseID, "score": v.State.Score, "is_clear": v.State.IsClear, "combo_lamp_id": v.State.ComboLampID, "updated_at": v.State.UpdatedAt})
	}
	query := `INSERT INTO player_course_records (player_id,course_id,score,is_clear,combo_lamp_id,updated_at) VALUES (:player_id,:course_id,:score,:is_clear,:combo_lamp_id,:updated_at) ON DUPLICATE KEY UPDATE updated_at=IF(score<>VALUES(score) OR is_clear<>VALUES(is_clear) OR combo_lamp_id<>VALUES(combo_lamp_id),VALUES(updated_at),updated_at),score=VALUES(score),is_clear=VALUES(is_clear),combo_lamp_id=VALUES(combo_lamp_id)`
	for i := 0; i < len(rows); i += info.BulkInsertChunkSize {
		end := min(i+info.BulkInsertChunkSize, len(rows))
		if _, err := exec.NamedExecContext(ctx, query, rows[i:end]); err != nil {
			return fmt.Errorf("failed to upsert course records: %w", err)
		}
	}
	return nil
}
