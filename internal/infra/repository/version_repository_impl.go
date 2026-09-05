package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/infra/models"
)

type versionRepository struct{}

// NewVersionRepository はVersionRepositoryの実装を生成します。
func NewVersionRepository() domainrepo.VersionRepository {
	return &versionRepository{}
}

func (r *versionRepository) FindAll(ctx context.Context, exec domainrepo.Executor) ([]*entity.Version, error) {
	rows := []models.VersionModel{}
	if err := exec.SelectContext(ctx, &rows, `SELECT id, name, released_at FROM versions ORDER BY released_at, id`); err != nil {
		return nil, err
	}
	versions := make([]*entity.Version, len(rows))
	for i := range rows {
		versions[i] = rows[i].ToEntity()
	}
	return versions, nil
}

func (r *versionRepository) FindByID(ctx context.Context, exec domainrepo.Executor, id int) (*entity.Version, error) {
	return r.findOne(ctx, exec, `SELECT id, name, released_at FROM versions WHERE id = ?`, id)
}

func (r *versionRepository) FindByIDForUpdate(ctx context.Context, exec domainrepo.Executor, id int) (*entity.Version, error) {
	return r.findOne(ctx, exec, `SELECT id, name, released_at FROM versions WHERE id = ? FOR UPDATE`, id)
}

func (r *versionRepository) FindByName(ctx context.Context, exec domainrepo.Executor, name string) (*entity.Version, error) {
	return r.findOne(ctx, exec, `SELECT id, name, released_at FROM versions WHERE name = ?`, strings.TrimSpace(name))
}

func (r *versionRepository) FindLatest(ctx context.Context, exec domainrepo.Executor) (*entity.Version, error) {
	return r.findOne(ctx, exec, `SELECT id, name, released_at FROM versions ORDER BY released_at DESC, id DESC LIMIT 1`)
}

func (r *versionRepository) findOne(ctx context.Context, exec domainrepo.Executor, query string, args ...any) (*entity.Version, error) {
	var row models.VersionModel
	if err := exec.GetContext(ctx, &row, query, args...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domainrepo.ErrVersionNotFound
		}
		return nil, err
	}
	return row.ToEntity(), nil
}

func (r *versionRepository) ExistsSongInRange(ctx context.Context, exec domainrepo.Executor, from time.Time, to *time.Time) (bool, error) {
	query := `SELECT 1 FROM songs WHERE released_at >= ?`
	args := []any{from}
	if to != nil {
		query += ` AND released_at < ?`
		args = append(args, *to)
	}
	query += ` LIMIT 1`

	var exists int
	err := exec.GetContext(ctx, &exists, query, args...)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func (r *versionRepository) Create(ctx context.Context, exec domainrepo.Executor, version *entity.Version) (*entity.Version, error) {
	model := models.FromVersionEntity(version)
	result, err := exec.ExecContext(ctx, `INSERT INTO versions (name, released_at) VALUES (?, ?)`, model.Name, model.ReleasedAt)
	if err != nil {
		return nil, wrapVersionDuplicateError(err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	return r.FindByID(ctx, exec, int(id))
}

func (r *versionRepository) Save(ctx context.Context, exec domainrepo.Executor, version *entity.Version) error {
	model := models.FromVersionEntity(version)
	result, err := exec.ExecContext(ctx, `UPDATE versions SET name = ? WHERE id = ?`, model.Name, model.ID)
	if err != nil {
		return wrapVersionDuplicateError(err)
	}
	return versionRowsAffected(result)
}

func (r *versionRepository) Delete(ctx context.Context, exec domainrepo.Executor, id int) error {
	result, err := exec.ExecContext(ctx, `DELETE FROM versions WHERE id = ?`, id)
	if err != nil {
		return err
	}
	return versionRowsAffected(result)
}

func versionRowsAffected(result sql.Result) error {
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return domainrepo.ErrVersionNotFound
	}
	return nil
}

func wrapVersionDuplicateError(err error) error {
	if !isMySQLDuplicateEntryForKey(err, "name") {
		return err
	}
	return fmt.Errorf("%w: %v", domainrepo.ErrVersionConflict, err)
}
