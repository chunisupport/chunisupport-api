package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/jmoiron/sqlx"
)

// honorRepository は HonorRepository の実装です。
type honorRepository struct {
	db *sqlx.DB
}

type honorRow struct {
	ID          int        `db:"id"`
	Name        string     `db:"name"`
	HonorTypeID int        `db:"honor_type_id"`
	TypeName    string     `db:"type_name"`
	ImageURL    string     `db:"image_url"`
	CreatedAt   *time.Time `db:"created_at"`
}

// NewHonorRepository は HonorRepository の実装を生成します。
func NewHonorRepository(db *sqlx.DB) repository.HonorRepository {
	return &honorRepository{db: db}
}

// FindAll は称号をID昇順で全件取得します。
func (r *honorRepository) FindAll(ctx context.Context, exec repository.Executor) ([]*entity.Honor, error) {
	rows := []honorRow{}
	if err := exec.SelectContext(ctx, &rows, `
		SELECT h.id, h.name, h.honor_type_id, ht.name AS type_name, h.image_url, h.created_at
		FROM honors h
		INNER JOIN honor_types ht ON h.honor_type_id = ht.id
		ORDER BY h.id
	`); err != nil {
		return nil, err
	}

	honors := make([]*entity.Honor, len(rows))
	for i := range rows {
		honors[i] = toHonorEntity(&rows[i])
	}
	return honors, nil
}

// FindByID は指定IDの称号を取得します。
func (r *honorRepository) FindByID(ctx context.Context, exec repository.Executor, id int) (*entity.Honor, error) {
	var row honorRow
	if err := exec.GetContext(ctx, &row, `
		SELECT h.id, h.name, h.honor_type_id, ht.name AS type_name, h.image_url, h.created_at
		FROM honors h
		INNER JOIN honor_types ht ON h.honor_type_id = ht.id
		WHERE h.id = ?
	`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrHonorNotFound
		}
		return nil, err
	}
	return toHonorEntity(&row), nil
}

// Create は称号を新規登録します。
func (r *honorRepository) Create(ctx context.Context, exec repository.Executor, honor *entity.Honor) (*entity.Honor, error) {
	result, err := exec.ExecContext(ctx, `
		INSERT INTO honors (name, honor_type_id, image_url)
		VALUES (?, ?, ?)
	`, strings.TrimSpace(honor.Name), honor.HonorTypeID, strings.TrimSpace(honor.ImageURL))
	if err != nil {
		return nil, wrapHonorDuplicateError(err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	return r.FindByID(ctx, exec, int(id))
}

// Save は称号を更新します。
func (r *honorRepository) Save(ctx context.Context, exec repository.Executor, honor *entity.Honor) error {
	result, err := exec.ExecContext(ctx, `
		UPDATE honors
		SET name = ?, honor_type_id = ?, image_url = ?
		WHERE id = ?
	`, strings.TrimSpace(honor.Name), honor.HonorTypeID, strings.TrimSpace(honor.ImageURL), honor.ID)
	if err != nil {
		return wrapHonorDuplicateError(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return repository.ErrHonorNotFound
	}
	return nil
}

// Delete は称号を物理削除します。
func (r *honorRepository) Delete(ctx context.Context, exec repository.Executor, id int) error {
	result, err := exec.ExecContext(ctx, `DELETE FROM honors WHERE id = ?`, id)
	if err != nil {
		return wrapHonorReferencedError(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return repository.ErrHonorNotFound
	}
	return nil
}

func toHonorEntity(row *honorRow) *entity.Honor {
	return &entity.Honor{
		ID:          row.ID,
		Name:        row.Name,
		HonorTypeID: row.HonorTypeID,
		TypeName:    row.TypeName,
		ImageURL:    row.ImageURL,
		CreatedAt:   row.CreatedAt,
	}
}

// EnsureHonor は称号を登録または既存のIDを取得します。
// 称号が存在しなければ登録され、存在すれば既存のIDが返されます。
func (r *honorRepository) EnsureHonor(ctx context.Context, exec repository.Executor, title string, honorTypeID int, imageURL *string) (int, error) {
	storedTitle := strings.TrimSpace(title)
	storedImageURL := ""
	if imageURL != nil {
		storedImageURL = strings.TrimSpace(*imageURL)
	}
	query := `INSERT INTO honors (name, honor_type_id, image_url) VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE id = LAST_INSERT_ID(id)`
	result, err := exec.ExecContext(ctx, query, storedTitle, honorTypeID, storedImageURL)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(id), nil
}

// DeletePlayerHonors はプレイヤーの称号割り当てを全て削除します。
func (r *honorRepository) DeletePlayerHonors(ctx context.Context, exec repository.Executor, playerID int) error {
	query := `DELETE FROM player_honors WHERE player_id = ?`
	_, err := exec.ExecContext(ctx, query, playerID)
	return err
}

// BulkAssignHonors はプレイヤーに称号を一括で割り当てます。
// 大量の割り当てはチャンク分割して実行されます。
func (r *honorRepository) BulkAssignHonors(ctx context.Context, exec repository.Executor, assignments []repository.HonorAssignment) error {
	if len(assignments) == 0 {
		return nil
	}

	batchSize := info.BulkInsertChunkSize
	for i := 0; i < len(assignments); i += batchSize {
		end := min(i+batchSize, len(assignments))
		batch := assignments[i:end]

		query := `INSERT INTO player_honors (player_id, honor_id, slot) VALUES `
		values := make([]any, 0, len(batch)*3)
		placeholders := make([]string, 0, len(batch))

		for _, a := range batch {
			placeholders = append(placeholders, "(?, ?, ?)")
			values = append(values, a.PlayerID, a.HonorID, a.Slot)
		}

		query += strings.Join(placeholders, ", ")
		_, err := exec.ExecContext(ctx, query, values...)
		if err != nil {
			return err
		}
	}

	return nil
}
