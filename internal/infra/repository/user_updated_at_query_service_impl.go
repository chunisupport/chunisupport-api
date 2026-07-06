package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/infra/models"
)

type userUpdatedAtQueryService struct{}

// NewUserUpdatedAtQueryService はユーザー更新日時専用の読み取りサービスを生成します。
func NewUserUpdatedAtQueryService() domainrepo.UserUpdatedAtQueryService {
	return &userUpdatedAtQueryService{}
}

// FindByUsername はユーザーと関連データの更新日時を1回のDBアクセスで取得します。
func (s *userUpdatedAtQueryService) FindByUsername(ctx context.Context, exec domainrepo.Executor, username string) (*domainrepo.UserUpdatedAtQueryResult, error) {
	const query = `
WITH target_user AS (
	SELECT
		id, username, firebase_uid, created_at, updated_at, player_id,
		account_type_id, is_suspicious, is_private
	FROM users
	WHERE username = ?
),
latest_record_updates AS (
	SELECT (
		SELECT updated_at
		FROM player_records
		WHERE player_id = (SELECT player_id FROM target_user)
		ORDER BY updated_at DESC
		LIMIT 1
	) AS updated_at
	UNION ALL
	SELECT (
		SELECT updated_at
		FROM player_worldsend_records
		WHERE player_id = (SELECT player_id FROM target_user)
		ORDER BY updated_at DESC
		LIMIT 1
	) AS updated_at
)
SELECT
	u.id, u.username, u.firebase_uid, u.created_at, u.updated_at, u.player_id,
	u.account_type_id, u.is_suspicious, u.is_private,
	p.updated_at AS player_updated_at,
	(SELECT MAX(updated_at) FROM latest_record_updates) AS records_updated_at
FROM target_user u
LEFT JOIN players p ON p.id = u.player_id
`

	var row struct {
		models.UserModel
		PlayerUpdatedAt  *time.Time `db:"player_updated_at"`
		RecordsUpdatedAt any        `db:"records_updated_at"`
	}
	if err := exec.GetContext(ctx, &row, query, username); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.Join(domainrepo.ErrUserNotFound, err)
		}
		return nil, err
	}

	user, err := row.UserModel.ToEntity()
	if err != nil {
		return nil, err
	}
	recordsUpdatedAt, err := parseLastScoreUpdate(row.RecordsUpdatedAt)
	if err != nil {
		return nil, err
	}

	return &domainrepo.UserUpdatedAtQueryResult{
		User:             user,
		PlayerUpdatedAt:  row.PlayerUpdatedAt,
		RecordsUpdatedAt: recordsUpdatedAt,
	}, nil
}
