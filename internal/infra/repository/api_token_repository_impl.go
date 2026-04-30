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

// Create はユーザーのAPIトークンを保存します。
func (r *apiTokenRepository) Create(ctx context.Context, exec repository.Executor, token *entity.APIToken) error {
	query := `
INSERT INTO api_tokens (user_id, name, hashed_token, created_at)
VALUES (?, ?, ?, ?)
`
	result, err := exec.ExecContext(ctx, query, token.UserID, token.Name, token.HashedToken, token.CreatedAt)
	if err != nil {
		return err
	}

	if id, err := result.LastInsertId(); err == nil && id != 0 {
		token.ID = id
	}
	return nil
}

// FindByUserID はユーザーIDからAPIトークン一覧を取得します。
func (r *apiTokenRepository) FindByUserID(ctx context.Context, exec repository.Executor, userID int) ([]*entity.APIToken, error) {
	var tokenModels []models.APITokenModel
	query := `SELECT id, user_id, name, hashed_token, created_at FROM api_tokens WHERE user_id = ? ORDER BY created_at DESC, id DESC`
	if err := exec.SelectContext(ctx, &tokenModels, query, userID); err != nil {
		return nil, err
	}

	tokens := make([]*entity.APIToken, 0, len(tokenModels))
	for i := range tokenModels {
		tokens = append(tokens, tokenModels[i].ToEntity())
	}
	return tokens, nil
}

// FindByHashedToken はハッシュ値からAPIトークンを取得します。
func (r *apiTokenRepository) FindByHashedToken(ctx context.Context, exec repository.Executor, hashedToken string) (*entity.APIToken, error) {
	var tokenModel models.APITokenModel
	query := `SELECT id, user_id, name, hashed_token, created_at FROM api_tokens WHERE hashed_token = ?`
	if err := exec.GetContext(ctx, &tokenModel, query, hashedToken); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.Join(repository.ErrAPITokenNotFound, err)
		}
		return nil, err
	}
	return tokenModel.ToEntity(), nil
}

// CountByUserID はユーザーIDに紐づくAPIトークン数を取得します。
func (r *apiTokenRepository) CountByUserID(ctx context.Context, exec repository.Executor, userID int) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM api_tokens WHERE user_id = ?`
	if err := exec.GetContext(ctx, &count, query, userID); err != nil {
		return 0, err
	}
	return count, nil
}

// DeleteByID はユーザーIDとトークンIDに紐づくAPIトークンを削除します。
func (r *apiTokenRepository) DeleteByID(ctx context.Context, exec repository.Executor, userID int, tokenID int64) error {
	query := `DELETE FROM api_tokens WHERE user_id = ? AND id = ?`
	_, err := exec.ExecContext(ctx, query, userID, tokenID)
	return err
}

// DeleteByUserID はユーザーIDに紐づくAPIトークンを削除します。
func (r *apiTokenRepository) DeleteByUserID(ctx context.Context, exec repository.Executor, userID int) error {
	query := `DELETE FROM api_tokens WHERE user_id = ?`
	_, err := exec.ExecContext(ctx, query, userID)
	return err
}
