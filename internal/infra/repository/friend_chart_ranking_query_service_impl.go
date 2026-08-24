package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/score"
	"github.com/jmoiron/sqlx"
)

var _ domainrepo.FriendChartRankingQueryService = (*FriendChartRankingQueryService)(nil)

type FriendChartRankingQueryService struct{}

func NewFriendChartRankingQueryService() *FriendChartRankingQueryService {
	return &FriendChartRankingQueryService{}
}

type friendChartRankingChartRow struct {
	SongDisplayID   string                      `db:"song_display_id"`
	SongTitle       string                      `db:"song_title"`
	SongArtist      string                      `db:"song_artist"`
	Difficulty      string                      `db:"difficulty"`
	Const           chartconstant.ChartConstant `db:"chart_const"`
	IsConstUnknown  bool                        `db:"is_const_unknown"`
	LevelStar       *int                        `db:"level_star"`
	Attribute       *string                     `db:"attribute"`
	ChartID         int                         `db:"chart_id"`
	ChartDifficulty int                         `db:"chart_difficulty_id"`
	IsWorldsend     bool                        `db:"is_worldsend"`
}

type friendChartRankingRecordRow struct {
	UserID     int         `db:"user_id"`
	Username   string      `db:"username"`
	PlayerName string      `db:"player_name"`
	Score      score.Score `db:"score"`
	ClearLamp  string      `db:"clear_lamp"`
	ComboLamp  string      `db:"combo_lamp"`
	FullChain  string      `db:"full_chain"`
	UpdatedAt  time.Time   `db:"updated_at"`
}

func (q *FriendChartRankingQueryService) FindChart(ctx context.Context, exec domainrepo.Executor, displayID string, difficulty string) (*domainrepo.FriendChartRankingChart, error) {
	const query = `
		SELECT
			s.display_id AS song_display_id,
			s.title AS song_title,
			s.artist AS song_artist,
			d.name AS difficulty,
			c.const AS chart_const,
			c.is_const_unknown AS is_const_unknown,
			NULL AS level_star,
			NULL AS attribute,
			c.id AS chart_id,
			c.difficulty_id AS chart_difficulty_id,
			0 AS is_worldsend
		FROM songs s
		INNER JOIN charts c ON c.song_id = s.id
		INNER JOIN difficulties d ON d.id = c.difficulty_id
		WHERE s.display_id = ?
		  AND d.name = ?
		  AND s.is_worldsend = 0
		  AND s.is_deleted = 0
	`
	var row friendChartRankingChartRow
	if err := sqlx.GetContext(ctx, exec, &row, query, displayID, difficulty); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, wrapFriendChartRankingQueryError("find chart", err)
	}
	return &domainrepo.FriendChartRankingChart{
		SongDisplayID:   row.SongDisplayID,
		SongTitle:       row.SongTitle,
		SongArtist:      row.SongArtist,
		Difficulty:      row.Difficulty,
		Const:           row.Const,
		IsConstUnknown:  row.IsConstUnknown,
		LevelStar:       row.LevelStar,
		Attribute:       row.Attribute,
		ChartID:         row.ChartID,
		ChartDifficulty: row.ChartDifficulty,
		IsWorldsend:     row.IsWorldsend,
	}, nil
}

func (q *FriendChartRankingQueryService) FindWorldsendChart(ctx context.Context, exec domainrepo.Executor, displayID string) (*domainrepo.FriendChartRankingChart, error) {
	const query = `
		SELECT
			s.display_id AS song_display_id,
			s.title AS song_title,
			s.artist AS song_artist,
			'WORLD''S END' AS difficulty,
			0 AS chart_const,
			1 AS is_const_unknown,
			wc.level_star AS level_star,
			wc.attribute AS attribute,
			wc.id AS chart_id,
			0 AS chart_difficulty_id,
			1 AS is_worldsend
		FROM songs s
		INNER JOIN worldsend_charts wc ON wc.song_id = s.id
		WHERE s.display_id = ?
		  AND s.is_worldsend = 1
		  AND s.is_deleted = 0
	`
	var row friendChartRankingChartRow
	if err := sqlx.GetContext(ctx, exec, &row, query, displayID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, wrapFriendChartRankingQueryError("find worldsend chart", err)
	}
	return &domainrepo.FriendChartRankingChart{
		SongDisplayID:   row.SongDisplayID,
		SongTitle:       row.SongTitle,
		SongArtist:      row.SongArtist,
		Difficulty:      row.Difficulty,
		Const:           row.Const,
		IsConstUnknown:  row.IsConstUnknown,
		LevelStar:       row.LevelStar,
		Attribute:       row.Attribute,
		ChartID:         row.ChartID,
		ChartDifficulty: row.ChartDifficulty,
		IsWorldsend:     row.IsWorldsend,
	}, nil
}

func (q *FriendChartRankingQueryService) ListRecords(ctx context.Context, exec domainrepo.Executor, userID int, chartID int) ([]*domainrepo.FriendChartRankingRecord, error) {
	const query = `
		SELECT
			u.id AS user_id,
			u.username AS username,
			p.player_name AS player_name,
			pr.score AS score,
			cl.name AS clear_lamp,
			co.name AS combo_lamp,
			fc.name AS full_chain,
			pr.updated_at AS updated_at
		FROM users u
		INNER JOIN players p ON p.id = u.player_id
		INNER JOIN player_records pr ON pr.player_id = p.id
		INNER JOIN clear_lamp_types cl ON cl.id = pr.clear_lamp_id
		INNER JOIN combo_lamp_types co ON co.id = pr.combo_lamp_id
		INNER JOIN full_chain_types fc ON fc.id = pr.full_chain_id
		WHERE pr.chart_id = ?
		  AND (
			u.id = ?
			OR EXISTS (
				SELECT 1
				FROM friendships f
				INNER JOIN friendships rf
					ON rf.user_id = f.friend_user_id
				   AND rf.friend_user_id = f.user_id
				   AND rf.status_id = ?
				WHERE f.user_id = ?
				  AND f.friend_user_id = u.id
				  AND f.status_id = ?
			)
		  )
		ORDER BY pr.score DESC, pr.updated_at DESC, u.username ASC
	`
	var rows []friendChartRankingRecordRow
	if err := sqlx.SelectContext(ctx, exec, &rows, query, chartID, userID, entity.FriendshipStatusAccepted, userID, entity.FriendshipStatusAccepted); err != nil {
		return nil, wrapFriendChartRankingQueryError("list records", err)
	}
	records := make([]*domainrepo.FriendChartRankingRecord, 0, len(rows))
	for _, row := range rows {
		records = append(records, &domainrepo.FriendChartRankingRecord{
			UserID:     row.UserID,
			Username:   row.Username,
			PlayerName: row.PlayerName,
			Score:      uint32(row.Score),
			ClearLamp:  row.ClearLamp,
			ComboLamp:  row.ComboLamp,
			FullChain:  row.FullChain,
			UpdatedAt:  row.UpdatedAt,
		})
	}
	return records, nil
}

func (q *FriendChartRankingQueryService) ListWorldsendRecords(ctx context.Context, exec domainrepo.Executor, userID int, worldsendChartID int) ([]*domainrepo.FriendChartRankingRecord, error) {
	const query = `
		SELECT
			u.id AS user_id,
			u.username AS username,
			p.player_name AS player_name,
			pwr.score AS score,
			cl.name AS clear_lamp,
			co.name AS combo_lamp,
			fc.name AS full_chain,
			pwr.updated_at AS updated_at
		FROM users u
		INNER JOIN players p ON p.id = u.player_id
		INNER JOIN player_worldsend_records pwr ON pwr.player_id = p.id
		INNER JOIN clear_lamp_types cl ON cl.id = pwr.clear_lamp_id
		INNER JOIN combo_lamp_types co ON co.id = pwr.combo_lamp_id
		INNER JOIN full_chain_types fc ON fc.id = pwr.full_chain_id
		WHERE pwr.worldsend_chart_id = ?
		  AND (
			u.id = ?
			OR EXISTS (
				SELECT 1
				FROM friendships f
				INNER JOIN friendships rf
					ON rf.user_id = f.friend_user_id
				   AND rf.friend_user_id = f.user_id
				   AND rf.status_id = ?
				WHERE f.user_id = ?
				  AND f.friend_user_id = u.id
				  AND f.status_id = ?
			)
		  )
		ORDER BY pwr.score DESC, pwr.updated_at DESC, u.username ASC
	`
	var rows []friendChartRankingRecordRow
	if err := sqlx.SelectContext(ctx, exec, &rows, query, worldsendChartID, userID, entity.FriendshipStatusAccepted, userID, entity.FriendshipStatusAccepted); err != nil {
		return nil, wrapFriendChartRankingQueryError("list worldsend records", err)
	}
	records := make([]*domainrepo.FriendChartRankingRecord, 0, len(rows))
	for _, row := range rows {
		records = append(records, &domainrepo.FriendChartRankingRecord{
			UserID:     row.UserID,
			Username:   row.Username,
			PlayerName: row.PlayerName,
			Score:      uint32(row.Score),
			ClearLamp:  row.ClearLamp,
			ComboLamp:  row.ComboLamp,
			FullChain:  row.FullChain,
			UpdatedAt:  row.UpdatedAt,
		})
	}
	return records, nil
}

func wrapFriendChartRankingQueryError(operation string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %s: %v", domainrepo.ErrRepositoryOperationFailed, operation, err)
}
