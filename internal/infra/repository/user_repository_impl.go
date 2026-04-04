package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/playername"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/username"
	"github.com/chunisupport/chunisupport-api/internal/infra/models"
	"github.com/chunisupport/chunisupport-api/internal/utils"
	"github.com/jmoiron/sqlx"
)

// userRepository は UserRepository の実装です。
type userRepository struct {
	db *sqlx.DB
}

// NewUserRepository は UserRepository の実装を生成します。
func NewUserRepository(db *sqlx.DB) repository.UserRepository {
	return &userRepository{db: db}
}

// FindByID はIDでユーザーを検索します。
func (r *userRepository) FindByID(ctx context.Context, exec repository.Executor, id int) (*entity.User, error) {
	var userModel models.UserModel
	query := `SELECT id, username, firebase_uid, password_hash, created_at, updated_at, player_id, account_type_id, is_suspicious, is_deleted, is_private FROM users WHERE id = ?`
	err := exec.GetContext(ctx, &userModel, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.Join(repository.ErrUserNotFound, err)
		}
		return nil, err
	}
	return userModel.ToEntity()
}

// FindByUsername はユーザー名でユーザーを検索します。
func (r *userRepository) FindByUsername(ctx context.Context, exec repository.Executor, username string) (*entity.User, error) {
	var userModel models.UserModel
	query := `SELECT id, username, firebase_uid, password_hash, created_at, updated_at, player_id, account_type_id, is_suspicious, is_deleted, is_private FROM users WHERE username = ?`
	err := exec.GetContext(ctx, &userModel, query, username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.Join(repository.ErrUserNotFound, err)
		}
		return nil, err
	}
	return userModel.ToEntity()
}

// FindByFirebaseUID はFirebase UIDでユーザーを検索します。
func (r *userRepository) FindByFirebaseUID(ctx context.Context, exec repository.Executor, uid string) (*entity.User, error) {
	var userModel models.UserModel
	query := `SELECT id, username, firebase_uid, password_hash, created_at, updated_at, player_id, account_type_id, is_suspicious, is_deleted, is_private FROM users WHERE firebase_uid = ?`
	err := exec.GetContext(ctx, &userModel, query, uid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.Join(repository.ErrUserNotFound, err)
		}
		return nil, err
	}
	return userModel.ToEntity()
}

// FindAllWithPlayer はユーザー一覧をプレイヤー情報付きで取得します。
// 通常のユーザー一覧取得用で、プライベート・削除済み・プレイヤー未紐付けアカウントを除外します。
func (r *userRepository) FindAllWithPlayer(ctx context.Context, exec repository.Executor, limit int, offset int, searchName string) ([]entity.UserWithPlayer, error) {
	query := `
		SELECT
			u.id AS user_id,
			u.username,
			u.player_id AS user_player_id,
			p.id AS player_id,
			p.player_name,
			p.official_player_rating AS player_official_rating,
			p.overpower_value AS player_overpower_value
		FROM users u
		LEFT JOIN players p ON u.player_id = p.id
		WHERE u.is_deleted = FALSE
		AND u.is_private = FALSE
		AND u.player_id IS NOT NULL
	`
	args := []any{}

	if searchName != "" {
		// 前方一致検索
		// ユーザー名 OR プレイヤー名
		query += " AND (u.username LIKE ? OR p.player_name LIKE ?)"
		// LIKE句の特殊文字（%, _, \）をエスケープしてSQLインジェクションを防ぐ
		escapedSearchName := utils.EscapeLike(searchName)
		likePattern := escapedSearchName + "%"
		args = append(args, likePattern, likePattern)
	}

	query += " ORDER BY u.id ASC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := exec.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []entity.UserWithPlayer
	for rows.Next() {
		var row models.UserWithPlayerRow
		if err := rows.StructScan(&row); err != nil {
			return nil, fmt.Errorf("scan error: %w", err)
		}

		// UserWithPlayerに変換
		uname, err := username.NewUserName(row.Username)
		if err != nil {
			return nil, fmt.Errorf("failed to create username: %w", err)
		}

		result := entity.UserWithPlayer{
			User: entity.User{
				ID:       row.UserID,
				Username: uname,
				PlayerID: row.UserPlayerID,
			},
		}

		if row.PlayerID != nil {
			var pl entity.Player
			pl.ID = *row.PlayerID
			if row.PlayerName != nil {
				pl.Name, err = playername.NewPlayerName(*row.PlayerName)
				if err != nil {
					return nil, fmt.Errorf("failed to create player name: %w", err)
				}
			}
			pl.OfficialRating = row.PlayerOfficialRating
			pl.OverpowerValue = row.PlayerOverpowerValue
			result.Player = &pl
		}

		results = append(results, result)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

// FindAllWithPlayerForAdmin はADMIN用にすべてのユーザー一覧をプレイヤー情報付きで取得します。
// プライベート・削除済み・プレイヤー未紐付けアカウントを含みます。
func (r *userRepository) FindAllWithPlayerForAdmin(ctx context.Context, exec repository.Executor, limit int, offset int, searchName string) ([]entity.UserWithPlayer, error) {
	query := `
		SELECT
			u.id AS user_id,
			u.username,
			u.account_type_id AS user_account_type_id,
			u.player_id AS user_player_id,
			u.created_at AS user_created_at,
			u.updated_at AS user_updated_at,
			u.is_suspicious AS user_is_suspicious,
			u.is_private AS user_is_private,
			u.is_deleted AS user_is_deleted,
			p.id AS player_id,
			p.player_name,
			p.official_player_rating AS player_official_rating,
			p.overpower_value AS player_overpower_value
		FROM users u
		LEFT JOIN players p ON u.player_id = p.id
		WHERE 1=1
	`
	args := []any{}

	if searchName != "" {
		// 前方一致検索
		// ユーザー名 OR プレイヤー名
		query += " AND (u.username LIKE ? OR p.player_name LIKE ?)"
		// LIKE句の特殊文字（%, _, \）をエスケープしてSQLインジェクションを防ぐ
		escapedSearchName := utils.EscapeLike(searchName)
		likePattern := escapedSearchName + "%"
		args = append(args, likePattern, likePattern)
	}

	query += " ORDER BY u.id ASC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := exec.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []entity.UserWithPlayer
	for rows.Next() {
		var row struct {
			models.UserWithPlayerRow
			UserAccountTypeID int       `db:"user_account_type_id"`
			UserCreatedAt     time.Time `db:"user_created_at"`
			UserUpdatedAt     time.Time `db:"user_updated_at"`
			UserIsSuspicious  *bool     `db:"user_is_suspicious"`
			UserIsPrivate     *bool     `db:"user_is_private"`
			UserIsDeleted     *bool     `db:"user_is_deleted"`
		}
		if err := rows.StructScan(&row); err != nil {
			return nil, fmt.Errorf("scan error: %w", err)
		}

		// UserWithPlayerに変換
		uname, err := username.NewUserName(row.Username)
		if err != nil {
			return nil, fmt.Errorf("failed to create username: %w", err)
		}

		result := entity.UserWithPlayer{
			User: entity.User{
				ID:            row.UserID,
				Username:      uname,
				AccountTypeID: row.UserAccountTypeID,
				CreatedAt:     row.UserCreatedAt,
				UpdatedAt:     row.UserUpdatedAt,
				PlayerID:      row.UserPlayerID,
				IsSuspicious:  row.UserIsSuspicious != nil && *row.UserIsSuspicious,
				IsPrivate:     row.UserIsPrivate != nil && *row.UserIsPrivate,
				IsDeleted:     row.UserIsDeleted != nil && *row.UserIsDeleted,
			},
		}

		if row.PlayerID != nil {
			var pl entity.Player
			pl.ID = *row.PlayerID
			if row.PlayerName != nil {
				pl.Name, err = playername.NewPlayerName(*row.PlayerName)
				if err != nil {
					return nil, fmt.Errorf("failed to create player name: %w", err)
				}
			}
			pl.OfficialRating = row.PlayerOfficialRating
			pl.OverpowerValue = row.PlayerOverpowerValue
			result.Player = &pl
		}

		results = append(results, result)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

// Create は新しいユーザーをデータベースに保存します。保存後、user.IDに自動採番されたIDが設定されます。
func (r *userRepository) Create(ctx context.Context, exec repository.Executor, user *entity.User) error {
	query := `INSERT INTO users (username, password_hash, account_type_id, is_suspicious) VALUES (?, ?, ?, ?)`
	result, err := exec.ExecContext(ctx, query, user.Username, user.PasswordHash, user.AccountTypeID, user.IsSuspicious)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	user.ID = int(id)
	return nil
}

// Save はユーザーを集約単位で保存します。IDが存在する場合は更新、存在しない場合は作成します。
func (r *userRepository) Save(ctx context.Context, exec repository.Executor, user *entity.User) error {
	userModel := models.FromUserEntity(user)

	if user.ID == 0 {
		// 新規作成
		query := `INSERT INTO users (username, firebase_uid, password_hash, account_type_id, is_suspicious, is_deleted, is_private) VALUES (?, ?, ?, ?, ?, ?, ?)`
		result, err := exec.ExecContext(ctx, query, userModel.Username, userModel.FirebaseUID, userModel.PasswordHash, userModel.AccountTypeID, userModel.IsSuspicious, userModel.IsDeleted, userModel.IsPrivate)
		if err != nil {
			return wrapFirebaseUIDDuplicateError(err)
		}
		id, err := result.LastInsertId()
		if err != nil {
			return err
		}
		user.ID = int(id)
		return nil
	}

	// 更新
	query := `UPDATE users SET username = ?, firebase_uid = ?, password_hash = ?, account_type_id = ?, player_id = ?, is_suspicious = ?, is_deleted = ?, is_private = ?, updated_at = ? WHERE id = ?`
	_, err := exec.ExecContext(ctx, query, userModel.Username, userModel.FirebaseUID, userModel.PasswordHash, userModel.AccountTypeID, userModel.PlayerID, userModel.IsSuspicious, userModel.IsDeleted, userModel.IsPrivate, userModel.UpdatedAt, userModel.ID)
	return wrapFirebaseUIDDuplicateError(err)
}
