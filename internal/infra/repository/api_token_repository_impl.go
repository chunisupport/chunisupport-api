package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/infra/models"
	"github.com/jmoiron/sqlx"
)

// カラム名の列挙であり、認証情報をハードコードしているものではありません。
const apiTokenColumns = "id, user_id, name, hashed_token, token_prefix, last_used_at, created_at" // #nosec G101

type apiTokenRepository struct {
	db *sqlx.DB
}

// NewAPITokenRepository は新しいAPITokenRepositoryを生成します。
func NewAPITokenRepository(db *sqlx.DB) repository.APITokenRepository {
	return &apiTokenRepository{db: db}
}

// Save はAPIトークンを新規保存または更新します。
func (r *apiTokenRepository) Save(ctx context.Context, exec repository.Executor, token *entity.APIToken) error {
	model := models.FromAPITokenEntity(token)
	if token.ID == 0 {
		result, err := exec.ExecContext(ctx, `
INSERT INTO api_tokens (user_id, name, hashed_token, token_prefix)
VALUES (?, ?, ?, ?)
`, model.UserID, model.Name, model.HashedToken, model.TokenPrefix)
		if err != nil {
			return wrapAPITokenDuplicateError(err)
		}
		id, err := result.LastInsertId()
		if err != nil {
			return err
		}
		if id <= 0 {
			return fmt.Errorf("api_tokens.id out of range: %d", id)
		}
		token.ID = uint64(id)
		return nil
	}

	result, err := exec.ExecContext(ctx, `
UPDATE api_tokens
SET name = ?, last_used_at = ?
WHERE id = ? AND user_id = ?
`, model.Name, model.LastUsedAt, model.ID, model.UserID)
	if err != nil {
		return wrapAPITokenDuplicateError(err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		var exists int
		if err := exec.GetContext(ctx, &exists, `SELECT COUNT(*) FROM api_tokens WHERE id = ? AND user_id = ?`, model.ID, model.UserID); err != nil {
			return err
		}
		if exists == 0 {
			return repository.ErrAPITokenNotFound
		}
	}
	return nil
}

// ListByUserID はユーザーが所有するAPIトークンを新しい順で取得します。
func (r *apiTokenRepository) ListByUserID(ctx context.Context, exec repository.Executor, userID int) ([]*entity.APIToken, error) {
	var rows []*models.APITokenModel
	query := `SELECT ` + apiTokenColumns + ` FROM api_tokens WHERE user_id = ? ORDER BY created_at DESC, id DESC`
	if err := exec.SelectContext(ctx, &rows, query, userID); err != nil {
		return nil, err
	}
	return apiTokenModelsToEntities(rows)
}

// FindByIDAndUserID は所有者を限定してAPIトークンを取得します。
func (r *apiTokenRepository) FindByIDAndUserID(ctx context.Context, exec repository.Executor, id uint64, userID int) (*entity.APIToken, error) {
	return r.findByIDAndUserID(ctx, exec, id, userID, false)
}

// FindByIDAndUserIDForUpdate は所有者を限定し、更新用行ロック付きでAPIトークンを取得します。
func (r *apiTokenRepository) FindByIDAndUserIDForUpdate(ctx context.Context, exec repository.Executor, id uint64, userID int) (*entity.APIToken, error) {
	return r.findByIDAndUserID(ctx, exec, id, userID, true)
}

func (r *apiTokenRepository) findByIDAndUserID(ctx context.Context, exec repository.Executor, id uint64, userID int, forUpdate bool) (*entity.APIToken, error) {
	var row models.APITokenModel
	query := `SELECT ` + apiTokenColumns + ` FROM api_tokens WHERE id = ? AND user_id = ?`
	if forUpdate {
		query += ` FOR UPDATE`
	}
	if err := exec.GetContext(ctx, &row, query, id, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrAPITokenNotFound
		}
		return nil, err
	}
	return row.ToEntity()
}

// FindByHashedToken はハッシュ値からAPIトークンを取得します。
func (r *apiTokenRepository) FindByHashedToken(ctx context.Context, exec repository.Executor, hashedToken string) (*entity.APIToken, error) {
	return r.findByHashedToken(ctx, exec, hashedToken, false)
}

// FindByHashedTokenForUpdate はハッシュ値から、更新用行ロック付きでAPIトークンを取得します。
func (r *apiTokenRepository) FindByHashedTokenForUpdate(ctx context.Context, exec repository.Executor, hashedToken string) (*entity.APIToken, error) {
	return r.findByHashedToken(ctx, exec, hashedToken, true)
}

func (r *apiTokenRepository) findByHashedToken(ctx context.Context, exec repository.Executor, hashedToken string, forUpdate bool) (*entity.APIToken, error) {
	var row models.APITokenModel
	query := `SELECT ` + apiTokenColumns + ` FROM api_tokens WHERE hashed_token = ?`
	if forUpdate {
		query += ` FOR UPDATE`
	}
	if err := exec.GetContext(ctx, &row, query, hashedToken); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrAPITokenNotFound
		}
		return nil, err
	}
	return row.ToEntity()
}

// CountByUserID はユーザーが所有するAPIトークン数を返します。
func (r *apiTokenRepository) CountByUserID(ctx context.Context, exec repository.Executor, userID int) (int, error) {
	var count int
	if err := exec.GetContext(ctx, &count, `SELECT COUNT(*) FROM api_tokens WHERE user_id = ?`, userID); err != nil {
		return 0, err
	}
	return count, nil
}

// DeleteByIDAndUserID は所有者を限定してAPIトークンを削除します。
func (r *apiTokenRepository) DeleteByIDAndUserID(ctx context.Context, exec repository.Executor, id uint64, userID int) error {
	result, err := exec.ExecContext(ctx, `DELETE FROM api_tokens WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return repository.ErrAPITokenNotFound
	}
	return nil
}

func apiTokenModelsToEntities(rows []*models.APITokenModel) ([]*entity.APIToken, error) {
	tokens := make([]*entity.APIToken, 0, len(rows))
	for _, row := range rows {
		token, err := row.ToEntity()
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
	}
	return tokens, nil
}
