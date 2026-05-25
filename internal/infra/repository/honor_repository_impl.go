package repository

import (
	"context"
	"strings"

	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/jmoiron/sqlx"
)

// honorRepository は HonorRepository の実装です。
type honorRepository struct {
	db *sqlx.DB
}

// NewHonorRepository は HonorRepository の実装を生成します。
func NewHonorRepository(db *sqlx.DB) repository.HonorRepository {
	return &honorRepository{db: db}
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
