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
		SELECT h.id, h.name, h.honor_type_id, ht.name AS type_name, COALESCE(h.image_url, '') AS image_url, h.created_at
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
		SELECT h.id, h.name, h.honor_type_id, ht.name AS type_name, COALESCE(h.image_url, '') AS image_url, h.created_at
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
	`, strings.TrimSpace(honor.Name), honor.HonorTypeID, nullableHonorImageURL(honor.ImageURL))
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
	`, strings.TrimSpace(honor.Name), honor.HonorTypeID, nullableHonorImageURL(honor.ImageURL), honor.ID)
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
func (r *honorRepository) EnsureHonor(ctx context.Context, exec repository.Executor, title string, honorTypeID int, imageURL *string) (repository.HonorEnsureResult, error) {
	storedTitle := strings.TrimSpace(title)
	var storedImageURL any
	if imageURL != nil {
		storedImageURL = nullableHonorImageURL(*imageURL)
	}
	if storedImageURL != nil {
		var existingID int
		err := exec.GetContext(ctx, &existingID, `SELECT id FROM honors WHERE image_url = ?`, storedImageURL)
		if err == nil {
			return repository.HonorEnsureResult{ID: existingID}, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return repository.HonorEnsureResult{}, err
		}
	}
	query := `INSERT INTO honors (name, honor_type_id, image_url) VALUES (?, ?, ?)`
	if storedImageURL == nil && (r.db == nil || r.db.DriverName() != "sqlite") {
		query += ` ON DUPLICATE KEY UPDATE id = LAST_INSERT_ID(id)`
	}
	result, err := exec.ExecContext(ctx, query, storedTitle, honorTypeID, storedImageURL)
	if err != nil {
		if storedImageURL != nil && (isMySQLDuplicateEntryForKey(err, "unique_honor_image_url") ||
			isMySQLDuplicateEntryForKey(err, "unique_honor_name_type")) {
			var existingID int
			// 先行トランザクションのコミット後の行をREPEATABLE READでも取得するためカレントリードする。
			findErr := exec.GetContext(ctx, &existingID, `SELECT id FROM honors WHERE image_url = ? FOR UPDATE`, storedImageURL)
			if findErr == nil {
				return repository.HonorEnsureResult{ID: existingID}, nil
			}
			if !errors.Is(findErr, sql.ErrNoRows) {
				return repository.HonorEnsureResult{}, findErr
			}
		}
		return repository.HonorEnsureResult{}, wrapHonorDuplicateError(err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return repository.HonorEnsureResult{}, err
	}
	return repository.HonorEnsureResult{ID: int(id), ImageURLRegistered: storedImageURL != nil}, nil
}

// nullableHonorImageURL は画像URL未設定の称号を、一意制約で複数保持できるSQL NULLに変換します。
func nullableHonorImageURL(imageURL string) any {
	trimmed := strings.TrimSpace(imageURL)
	if trimmed == "" {
		return nil
	}
	return trimmed
}

// DeletePlayerHonors はプレイヤーの称号割り当てを全て削除します。
func (r *honorRepository) DeletePlayerHonors(ctx context.Context, exec repository.Executor, playerID int) error {
	query := `DELETE FROM player_honors WHERE player_id = ?`
	_, err := exec.ExecContext(ctx, query, playerID)
	return err
}

// DeletePlayerHonorsExceptSlots は指定スロットを保持して称号割り当てを削除します。
func (r *honorRepository) DeletePlayerHonorsExceptSlots(ctx context.Context, exec repository.Executor, playerID int, preservedSlots []int) error {
	if len(preservedSlots) == 0 {
		return r.DeletePlayerHonors(ctx, exec, playerID)
	}

	placeholders := strings.TrimSuffix(strings.Repeat("?, ", len(preservedSlots)), ", ")
	query := `DELETE FROM player_honors WHERE player_id = ? AND slot NOT IN (` + placeholders + `)`
	args := make([]any, 0, len(preservedSlots)+1)
	args = append(args, playerID)
	for _, slot := range preservedSlots {
		args = append(args, slot)
	}
	_, err := exec.ExecContext(ctx, query, args...)
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
