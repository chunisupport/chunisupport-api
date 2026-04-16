package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/infra/models"
	"github.com/jmoiron/sqlx"
)

type apiTokenRepository struct {
	db *sqlx.DB
}

// NewAPITokenRepository は新しいAPITokenRepositoryを生成します。
func NewAPITokenRepository(db *sqlx.DB) repository.APITokenRepository {
	return &apiTokenRepository{db: db}
}

// CreateOrReplace はユーザーのAPIトークンを保存し、既存のトークンがあれば置き換えます。
func (r *apiTokenRepository) CreateOrReplace(ctx context.Context, exec repository.Executor, token *entity.APIToken) error {
	query := `
INSERT INTO api_tokens (user_id, hashed_token)
VALUES (?, ?)
ON DUPLICATE KEY UPDATE
    hashed_token = VALUES(hashed_token),
    created_at = CURRENT_TIMESTAMP
`
	result, err := exec.ExecContext(ctx, query, token.UserID, token.HashedToken)
	if err != nil {
		return err
	}

	if id, err := result.LastInsertId(); err == nil && id != 0 {
		token.ID = id
	}
	return nil
}

// FindByUserID はユーザーIDからAPIトークンを取得します。
func (r *apiTokenRepository) FindByUserID(ctx context.Context, exec repository.Executor, userID int) (*entity.APIToken, error) {
	var tokenModel models.APITokenModel
	query := `SELECT id, user_id, hashed_token, created_at FROM api_tokens WHERE user_id = ?`
	if err := exec.GetContext(ctx, &tokenModel, query, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.Join(repository.ErrAPITokenNotFound, err)
		}
		return nil, err
	}
	return tokenModel.ToEntity(), nil
}

// FindByHashedToken はハッシュ値からAPIトークンを取得します。
func (r *apiTokenRepository) FindByHashedToken(ctx context.Context, exec repository.Executor, hashedToken string) (*entity.APIToken, error) {
	var tokenModel models.APITokenModel
	query := `SELECT id, user_id, hashed_token, created_at FROM api_tokens WHERE hashed_token = ?`
	if err := exec.GetContext(ctx, &tokenModel, query, hashedToken); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.Join(repository.ErrAPITokenNotFound, err)
		}
		return nil, err
	}
	return tokenModel.ToEntity(), nil
}

// DeleteByUserID はユーザーIDに紐づくAPIトークンを削除します。
func (r *apiTokenRepository) DeleteByUserID(ctx context.Context, exec repository.Executor, userID int) error {
	query := `DELETE FROM api_tokens WHERE user_id = ?`
	_, err := exec.ExecContext(ctx, query, userID)
	return err
}
