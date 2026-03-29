package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/chunisupport/chunisupport-api/internal/infra/models"
	"github.com/jmoiron/sqlx"
)

type recoveryCodeRepository struct {
	db *sqlx.DB
}

// NewRecoveryCodeRepository は新しいRecoveryCodeRepositoryを生成します。
func NewRecoveryCodeRepository(db *sqlx.DB) repository.RecoveryCodeRepository {
	return &recoveryCodeRepository{db: db}
}

// CreateBatch はリカバリーコードをまとめて保存します。
func (r *recoveryCodeRepository) CreateBatch(ctx context.Context, exec repository.Executor, codes []*entity.RecoveryCode) error {
	if len(codes) == 0 {
		return nil
	}

	batchSize := info.BulkInsertChunkSize
	for i := 0; i < len(codes); i += batchSize {
		end := min(i+batchSize, len(codes))
		batch := codes[i:end]

		valueStrings := make([]string, len(batch))
		valueArgs := make([]any, 0, len(batch)*2)
		for j, code := range batch {
			valueStrings[j] = "(?, ?)"
			valueArgs = append(valueArgs, code.UserID, code.CodeHash)
		}

		query := `INSERT INTO user_recovery_codes (user_id, code_hash) VALUES ` + strings.Join(valueStrings, ",")
		query = exec.Rebind(query)
		if _, err := exec.ExecContext(ctx, query, valueArgs...); err != nil {
			return err
		}
	}

	return nil
}

// DeleteByUserID は指定ユーザーのリカバリーコードを削除します。
func (r *recoveryCodeRepository) DeleteByUserID(ctx context.Context, exec repository.Executor, userID int) error {
	query := `DELETE FROM user_recovery_codes WHERE user_id = ?`
	_, err := exec.ExecContext(ctx, query, userID)
	return err
}

// DeleteByID はリカバリーコードを削除します。
func (r *recoveryCodeRepository) DeleteByID(ctx context.Context, exec repository.Executor, id uint32) error {
	query := `DELETE FROM user_recovery_codes WHERE id = ?`
	_, err := exec.ExecContext(ctx, query, id)
	return err
}

// FindByHash はハッシュでリカバリーコードを検索します。
func (r *recoveryCodeRepository) FindByHash(ctx context.Context, exec repository.Executor, codeHash []byte) (*entity.RecoveryCode, error) {
	var model models.RecoveryCodeModel
	query := `SELECT id, user_id, code_hash, created_at FROM user_recovery_codes WHERE code_hash = ?`
	if err := exec.GetContext(ctx, &model, query, codeHash); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.Join(repository.ErrRecoveryCodeNotFound, err)
		}
		return nil, err
	}
	return model.ToEntity(), nil
}

// FindByHashForUpdate はハッシュでリカバリーコードを検索し、ロックします。
func (r *recoveryCodeRepository) FindByHashForUpdate(ctx context.Context, exec repository.Executor, codeHash []byte) (*entity.RecoveryCode, error) {
	var model models.RecoveryCodeModel
	query := `SELECT id, user_id, code_hash, created_at FROM user_recovery_codes WHERE code_hash = ? FOR UPDATE`
	if err := exec.GetContext(ctx, &model, query, codeHash); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.Join(repository.ErrRecoveryCodeNotFound, err)
		}
		return nil, err
	}
	return model.ToEntity(), nil
}
