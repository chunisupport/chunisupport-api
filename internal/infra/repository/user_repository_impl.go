package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/constants"
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
	query := `SELECT id, username, firebase_uid, created_at, updated_at, player_id, account_type_id, is_suspicious, is_private FROM users WHERE id = ?`
	err := exec.GetContext(ctx, &userModel, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.Join(repository.ErrUserNotFound, err)
		}
		return nil, err
	}
	return userModel.ToEntity()
}

// FindByIDForUpdate はIDでユーザーを検索し、更新用に行ロックします。
func (r *userRepository) FindByIDForUpdate(ctx context.Context, exec repository.Executor, id int) (*entity.User, error) {
	var userModel models.UserModel
	query := `SELECT id, username, firebase_uid, created_at, updated_at, player_id, account_type_id, is_suspicious, is_private FROM users WHERE id = ? FOR UPDATE`
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
	query := `SELECT id, username, firebase_uid, created_at, updated_at, player_id, account_type_id, is_suspicious, is_private FROM users WHERE username = ?`
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
	query := `SELECT id, username, firebase_uid, created_at, updated_at, player_id, account_type_id, is_suspicious, is_private FROM users WHERE firebase_uid = ?`
	err := exec.GetContext(ctx, &userModel, query, uid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.Join(repository.ErrUserNotFound, err)
		}
		return nil, err
	}
	return userModel.ToEntity()
}

// LinkFirebaseUID は現在の Firebase UID が一致する場合のみ更新します。
func (r *userRepository) LinkFirebaseUID(ctx context.Context, exec repository.Executor, userID int, currentUID *string, newUID string, updatedAt time.Time) error {
	whereClause, whereArgs := userFirebaseUIDWhereClause(currentUID)
	query := fmt.Sprintf("UPDATE users SET firebase_uid = ?, updated_at = ? WHERE id = ? AND %s", whereClause)
	args := []any{newUID, updatedAt, userID}
	args = append(args, whereArgs...)

	result, err := exec.ExecContext(ctx, query, args...)
	if err != nil {
		return wrapFirebaseUIDDuplicateError(err)
	}

	return r.validateSingleUserUpdate(ctx, exec, userID, result)
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
			p.calculated_player_rating AS player_calculated_rating,
			p.overpower_value AS player_overpower_value
		FROM users u
		LEFT JOIN players p ON u.player_id = p.id
		WHERE u.is_private = FALSE
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
			pl.CalculatedRating = row.PlayerCalculatedRating
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
			u.firebase_uid,
			u.account_type_id AS user_account_type_id,
			u.player_id AS user_player_id,
			u.created_at AS user_created_at,
			u.updated_at AS user_updated_at,
			u.is_suspicious AS user_is_suspicious,
			u.is_private AS user_is_private,
			p.id AS player_id,
			p.player_name,
			p.calculated_player_rating AS player_calculated_rating,
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
				FirebaseUID:   row.FirebaseUID,
				AccountTypeID: row.UserAccountTypeID,
				CreatedAt:     row.UserCreatedAt,
				UpdatedAt:     row.UserUpdatedAt,
				PlayerID:      row.UserPlayerID,
				IsSuspicious:  row.UserIsSuspicious != nil && *row.UserIsSuspicious,
				IsPrivate:     row.UserIsPrivate != nil && *row.UserIsPrivate,
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
			pl.CalculatedRating = row.PlayerCalculatedRating
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

// Save はユーザーを集約単位で保存します。IDが存在する場合は更新、存在しない場合は作成します。
func (r *userRepository) Save(ctx context.Context, exec repository.Executor, user *entity.User) error {
	userModel := models.FromUserEntity(user)

	if user.ID == 0 {
		// 新規作成
		query := `INSERT INTO users (username, firebase_uid, created_at, updated_at, player_id, account_type_id, is_suspicious, is_private) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
		result, err := exec.ExecContext(ctx, query, userModel.Username, userModel.FirebaseUID, userModel.CreatedAt, userModel.UpdatedAt, userModel.PlayerID, userModel.AccountTypeID, userModel.IsSuspicious, userModel.IsPrivate)
		if err != nil {
			err = wrapFirebaseUIDDuplicateError(err)
			err = wrapUsernameDuplicateError(err)
			return err
		}
		id, err := result.LastInsertId()
		if err != nil {
			return err
		}
		user.ID = int(id)
		return nil
	}

	// 更新。ユーザー集約の状態を保存するため、権限を含む変更可能項目をまとめて更新します。
	if !constants.IsKnownAccountType(userModel.AccountTypeID) {
		return repository.ErrUserConflict
	}
	whereClause, whereArgs := userFirebaseUIDWhereClause(userModel.FirebaseUID)
	originalAccountTypeID := user.OriginalAccountTypeID
	if originalAccountTypeID == 0 {
		originalAccountTypeID = userModel.AccountTypeID
	}
	query := "UPDATE users SET player_id = ?, account_type_id = ?, is_suspicious = ?, is_private = ?, updated_at = ? WHERE id = ? AND username = ? AND account_type_id = ? AND " + whereClause
	args := []any{userModel.PlayerID, userModel.AccountTypeID, userModel.IsSuspicious, userModel.IsPrivate, userModel.UpdatedAt, userModel.ID, userModel.Username, originalAccountTypeID}
	args = append(args, whereArgs...)

	result, err := exec.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}

	if err := r.validateSingleUserUpdate(ctx, exec, userModel.ID, result); err != nil {
		return err
	}

	// 保存成功後、OriginalAccountTypeIDを最新の値に更新して次回の保存時の競合検出基準を同期
	user.OriginalAccountTypeID = userModel.AccountTypeID
	return nil
}

// DeleteByID はユーザーを物理削除します。
func (r *userRepository) DeleteByID(ctx context.Context, exec repository.Executor, id int) error {
	result, err := exec.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, id)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return repository.ErrUserNotFound
	}

	return nil
}

func userFirebaseUIDWhereClause(firebaseUID *string) (string, []any) {
	if firebaseUID == nil {
		return "firebase_uid IS NULL", nil
	}

	return "firebase_uid = ?", []any{*firebaseUID}
}

func (r *userRepository) validateSingleUserUpdate(ctx context.Context, exec repository.Executor, userID int, result sql.Result) error {
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected > 0 {
		return nil
	}

	var exists int
	err = exec.GetContext(ctx, &exists, `SELECT 1 FROM users WHERE id = ?`, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return repository.ErrUserNotFound
		}
		return err
	}

	return repository.ErrUserConflict
}
