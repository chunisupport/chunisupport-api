package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/infra/models"
	"github.com/jmoiron/sqlx"
)

var _ domainrepo.FriendshipRepository = (*FriendshipRepository)(nil)

type FriendshipRepository struct{}

func NewFriendshipRepository() *FriendshipRepository {
	return &FriendshipRepository{}
}

func (r *FriendshipRepository) Find(ctx context.Context, exec domainrepo.Executor, userID int, friendUserID int) (*entity.Friendship, error) {
	const q = `
		SELECT user_id, friend_user_id, status_id, requested_at, accepted_at, created_at, updated_at
		FROM friendships
		WHERE user_id = ? AND friend_user_id = ?
	`
	var model models.FriendshipModel
	if err := sqlx.GetContext(ctx, exec, &model, q, userID, friendUserID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, wrapFriendshipRepositoryError("find", err)
	}
	return model.ToEntity(), nil
}

func (r *FriendshipRepository) Save(ctx context.Context, exec domainrepo.Executor, friendship *entity.Friendship) error {
	if err := friendship.Validate(); err != nil {
		return err
	}
	model := models.FromFriendshipEntity(friendship)
	const q = `
		INSERT INTO friendships (
			user_id, friend_user_id, status_id, requested_at, accepted_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			status_id = VALUES(status_id),
			requested_at = VALUES(requested_at),
			accepted_at = VALUES(accepted_at),
			updated_at = VALUES(updated_at)
	`
	_, err := exec.ExecContext(ctx, q, model.UserID, model.FriendUserID, model.StatusID, model.RequestedAt, model.AcceptedAt, model.CreatedAt, model.UpdatedAt)
	return wrapFriendshipRepositoryError("save", err)
}

func (r *FriendshipRepository) Delete(ctx context.Context, exec domainrepo.Executor, userID int, friendUserID int) error {
	const q = `DELETE FROM friendships WHERE user_id = ? AND friend_user_id = ?`
	_, err := exec.ExecContext(ctx, q, userID, friendUserID)
	return wrapFriendshipRepositoryError("delete", err)
}

func (r *FriendshipRepository) DeletePending(ctx context.Context, exec domainrepo.Executor, userID int, friendUserID int) error {
	const q = `
		DELETE FROM friendships
		WHERE user_id = ?
		  AND friend_user_id = ?
		  AND status_id = ?
	`
	_, err := exec.ExecContext(ctx, q, userID, friendUserID, entity.FriendshipStatusPending)
	return wrapFriendshipRepositoryError("delete pending", err)
}

func (r *FriendshipRepository) DeletePair(ctx context.Context, exec domainrepo.Executor, userID int, friendUserID int) error {
	const q = `
		DELETE FROM friendships
		WHERE (user_id = ? AND friend_user_id = ?)
		   OR (user_id = ? AND friend_user_id = ?)
	`
	_, err := exec.ExecContext(ctx, q, userID, friendUserID, friendUserID, userID)
	return wrapFriendshipRepositoryError("delete pair", err)
}

func (r *FriendshipRepository) CountOutgoingActive(ctx context.Context, exec domainrepo.Executor, userID int) (int, error) {
	const q = `
		SELECT COUNT(*)
		FROM friendships
		WHERE user_id = ?
		  AND status_id IN (?, ?)
	`
	var count int
	if err := sqlx.GetContext(ctx, exec, &count, q, userID, entity.FriendshipStatusPending, entity.FriendshipStatusAccepted); err != nil {
		return 0, wrapFriendshipRepositoryError("count outgoing active", err)
	}
	return count, nil
}

func (r *FriendshipRepository) ListFriends(ctx context.Context, exec domainrepo.Executor, userID int) ([]*domainrepo.FriendshipWithUserSummary, error) {
	const q = `
		SELECT
			f.user_id, f.friend_user_id, f.status_id, f.requested_at, f.accepted_at, f.created_at, f.updated_at,
			u.id AS summary_user_id, u.username AS summary_username, u.is_private AS summary_is_private,
			p.player_level AS summary_player_level,
			p.player_name AS summary_player_name,
			p.calculated_player_rating AS summary_rating
		FROM friendships f
		INNER JOIN users u ON u.id = f.friend_user_id
		LEFT JOIN players p ON p.id = u.player_id
		WHERE f.user_id = ?
		  AND f.status_id = ?
		ORDER BY f.accepted_at DESC, f.friend_user_id ASC
	`
	return selectFriendshipSummaries(ctx, exec, q, userID, entity.FriendshipStatusAccepted)
}

func (r *FriendshipRepository) ListReceivedRequests(ctx context.Context, exec domainrepo.Executor, userID int) ([]*domainrepo.FriendshipWithUserSummary, error) {
	const q = `
		SELECT
			f.user_id, f.friend_user_id, f.status_id, f.requested_at, f.accepted_at, f.created_at, f.updated_at,
			u.id AS summary_user_id, u.username AS summary_username, u.is_private AS summary_is_private,
			p.player_level AS summary_player_level,
			p.player_name AS summary_player_name,
			p.calculated_player_rating AS summary_rating
		FROM friendships f
		INNER JOIN users u ON u.id = f.user_id
		LEFT JOIN players p ON p.id = u.player_id AND u.is_private = FALSE
		WHERE f.friend_user_id = ?
		  AND f.status_id = ?
		ORDER BY f.requested_at DESC, f.user_id ASC
	`
	return selectFriendshipSummaries(ctx, exec, q, userID, entity.FriendshipStatusPending)
}

func (r *FriendshipRepository) ListSentRequests(ctx context.Context, exec domainrepo.Executor, userID int) ([]*domainrepo.FriendshipWithUserSummary, error) {
	const q = `
		SELECT
			f.user_id, f.friend_user_id, f.status_id, f.requested_at, f.accepted_at, f.created_at, f.updated_at,
			u.id AS summary_user_id, u.username AS summary_username, u.is_private AS summary_is_private,
			p.player_level AS summary_player_level,
			p.player_name AS summary_player_name,
			p.calculated_player_rating AS summary_rating
		FROM friendships f
		INNER JOIN users u ON u.id = f.friend_user_id
		LEFT JOIN players p ON p.id = u.player_id AND u.is_private = FALSE
		WHERE f.user_id = ?
		  AND f.status_id = ?
		ORDER BY f.requested_at DESC, f.friend_user_id ASC
	`
	return selectFriendshipSummaries(ctx, exec, q, userID, entity.FriendshipStatusPending)
}

func (r *FriendshipRepository) ExistsMutualAccepted(ctx context.Context, exec domainrepo.Executor, userID int, friendUserID int) (bool, error) {
	const q = `
		SELECT COUNT(*)
		FROM friendships
		WHERE ((user_id = ? AND friend_user_id = ?) OR (user_id = ? AND friend_user_id = ?))
		  AND status_id = ?
	`
	var count int
	if err := sqlx.GetContext(ctx, exec, &count, q, userID, friendUserID, friendUserID, userID, entity.FriendshipStatusAccepted); err != nil {
		return false, wrapFriendshipRepositoryError("exists mutual accepted", err)
	}
	return count == 2, nil
}

type friendshipSummaryRow struct {
	models.FriendshipModel
	SummaryUserID      int      `db:"summary_user_id"`
	SummaryUsername    string   `db:"summary_username"`
	SummaryPlayerLevel *int     `db:"summary_player_level"`
	SummaryPlayerName  *string  `db:"summary_player_name"`
	SummaryRating      *float64 `db:"summary_rating"`
	SummaryIsPrivate   bool     `db:"summary_is_private"`
}

func selectFriendshipSummaries(ctx context.Context, exec domainrepo.Executor, query string, args ...any) ([]*domainrepo.FriendshipWithUserSummary, error) {
	var rows []friendshipSummaryRow
	if err := sqlx.SelectContext(ctx, exec, &rows, query, args...); err != nil {
		return nil, wrapFriendshipRepositoryError("select summaries", err)
	}
	res := make([]*domainrepo.FriendshipWithUserSummary, 0, len(rows))
	for _, row := range rows {
		res = append(res, &domainrepo.FriendshipWithUserSummary{
			Friendship: row.FriendshipModel.ToEntity(),
			User: &domainrepo.FriendshipUserSummary{
				UserID:      row.SummaryUserID,
				Username:    row.SummaryUsername,
				PlayerLevel: row.SummaryPlayerLevel,
				PlayerName:  row.SummaryPlayerName,
				Rating:      row.SummaryRating,
				IsPrivate:   row.SummaryIsPrivate,
			},
		})
	}
	return res, nil
}

func wrapFriendshipRepositoryError(operation string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %s: %v", domainrepo.ErrRepositoryOperationFailed, operation, err)
}
