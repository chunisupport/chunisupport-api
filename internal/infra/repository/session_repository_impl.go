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

type sessionRepository struct {
	db *sqlx.DB
}

// NewSessionRepository は新しいSessionRepositoryを生成します。
func NewSessionRepository(db *sqlx.DB) repository.SessionRepository {
	return &sessionRepository{db: db}
}

// Create は新しいセッションをデータベースに保存します。
func (r *sessionRepository) Create(ctx context.Context, exec repository.Executor, session *entity.Session) error {
	model, err := models.FromSessionEntity(session)
	if err != nil {
		return err
	}
	query := `INSERT INTO sessions (id, user_id, expires_at) VALUES (?, ?, ?)`
	_, err = exec.ExecContext(ctx, query, model.ID, model.UserID, model.ExpiresAt)
	return err
}

// FindByID はIDでセッションを検索します。
func (r *sessionRepository) FindByID(ctx context.Context, exec repository.Executor, id string) (*entity.Session, error) {
	model, err := models.FromSessionEntity(&entity.Session{ID: id})
	if err != nil {
		return nil, err
	}

	var sessionModel models.SessionModel
	query := `SELECT id, user_id, expires_at, created_at FROM sessions WHERE id = ?`
	if err := exec.GetContext(ctx, &sessionModel, query, model.ID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.Join(repository.ErrSessionNotFound, err)
		}
		return nil, err
	}
	return sessionModel.ToEntity()
}

// Delete はIDでセッションを削除します。
func (r *sessionRepository) Delete(ctx context.Context, exec repository.Executor, id string) error {
	model, err := models.FromSessionEntity(&entity.Session{ID: id})
	if err != nil {
		return err
	}
	query := `DELETE FROM sessions WHERE id = ?`
	_, err = exec.ExecContext(ctx, query, model.ID)
	return err
}

// CountByUserID は指定されたユーザーのセッション数を取得します。
func (r *sessionRepository) CountByUserID(ctx context.Context, exec repository.Executor, userID int) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM sessions WHERE user_id = ? AND expires_at > NOW()`
	if err := exec.GetContext(ctx, &count, query, userID); err != nil {
		return 0, err
	}
	return count, nil
}

// DeleteByUserID は指定されたユーザーのセッションを全て削除します。
func (r *sessionRepository) DeleteByUserID(ctx context.Context, exec repository.Executor, userID int) error {
	query := `DELETE FROM sessions WHERE user_id = ?`
	_, err := exec.ExecContext(ctx, query, userID)
	return err
}

// DeleteByUserIDExcept は指定されたセッションID以外のユーザーのセッションを全て削除します。
func (r *sessionRepository) DeleteByUserIDExcept(ctx context.Context, exec repository.Executor, userID int, excludeSessionID string) error {
	model, err := models.FromSessionEntity(&entity.Session{ID: excludeSessionID})
	if err != nil {
		return err
	}
	query := `DELETE FROM sessions WHERE user_id = ? AND id != ?`
	_, err = exec.ExecContext(ctx, query, userID, model.ID)
	return err
}

// DeleteOldestSessionsOverLimit は指定されたユーザーのセッション数が上限を超えている場合、古い順に削除します。
// maxCountより新しいセッションを残し、それより古いセッションをcreated_atの昇順で削除します。
func (r *sessionRepository) DeleteOldestSessionsOverLimit(ctx context.Context, exec repository.Executor, userID int, maxCount int) error {
	// 有効なセッションのみを対象とし、created_atで降順ソートして最新maxCount件を残すサブクエリを作成
	query := `
		DELETE FROM sessions 
		WHERE user_id = ? 
		  AND expires_at > NOW()
		  AND id NOT IN (
			SELECT id FROM (
			  SELECT id FROM sessions 
			  WHERE user_id = ? AND expires_at > NOW() 
			  ORDER BY created_at DESC 
			  LIMIT ?
			) AS keep_sessions
		  )`
	_, err := exec.ExecContext(ctx, query, userID, userID, maxCount)
	return err
}
