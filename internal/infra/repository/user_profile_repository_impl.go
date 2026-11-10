package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/infra/models"
)

type userProfileRepository struct{}

// NewUserProfileRepository はプロフィールの永続化実装を生成します。
func NewUserProfileRepository() repository.UserProfileRepository { return &userProfileRepository{} }

func (r *userProfileRepository) FindByUserID(ctx context.Context, exec repository.Executor, userID int) (*entity.UserProfile, error) {
	var model models.UserProfileModel
	if err := exec.GetContext(ctx, &model, `SELECT user_id, comment FROM user_profiles WHERE user_id = ?`, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("%w: find user profile: %w", repository.ErrRepositoryOperationFailed, err)
	}
	profile, err := model.ToEntity()
	if err != nil {
		return nil, fmt.Errorf("%w: restore user profile: %w", repository.ErrRepositoryOperationFailed, err)
	}
	return profile, nil
}

// Save はAUTO_INCREMENT列を持たないため、採番を浪費せずUPSERTできます。
func (r *userProfileRepository) Save(ctx context.Context, exec repository.Executor, profile *entity.UserProfile) error {
	var model models.UserProfileModel
	model.FromEntity(profile)
	query := `INSERT INTO user_profiles (user_id, comment, updated_at)
VALUES (?, ?, CURRENT_TIMESTAMP)
ON DUPLICATE KEY UPDATE comment = VALUES(comment), updated_at = CURRENT_TIMESTAMP`
	if _, err := exec.ExecContext(ctx, query, model.UserID, model.Comment); err != nil {
		return fmt.Errorf("%w: save user profile: %w", repository.ErrRepositoryOperationFailed, err)
	}
	return nil
}

func (r *userProfileRepository) DeleteByUserID(ctx context.Context, exec repository.Executor, userID int) error {
	if _, err := exec.ExecContext(ctx, `DELETE FROM user_profiles WHERE user_id = ?`, userID); err != nil {
		return fmt.Errorf("%w: delete user profile: %w", repository.ErrRepositoryOperationFailed, err)
	}
	return nil
}
