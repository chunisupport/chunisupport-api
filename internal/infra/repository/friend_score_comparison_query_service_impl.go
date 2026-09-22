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

var _ domainrepo.FriendScoreComparisonQueryService = (*FriendScoreComparisonQueryService)(nil)

type FriendScoreComparisonQueryService struct {
	db *sqlx.DB
}

func NewFriendScoreComparisonQueryService(db *sqlx.DB) *FriendScoreComparisonQueryService {
	return &FriendScoreComparisonQueryService{db: db}
}

type friendScoreComparisonUserRow struct {
	SelfUsername     string         `db:"self_username"`
	SelfPlayerName   sql.NullString `db:"self_player_name"`
	SelfPlayerID     *int           `db:"self_player_id"`
	FriendUsername   string         `db:"friend_username"`
	FriendPlayerName sql.NullString `db:"friend_player_name"`
	FriendPlayerID   *int           `db:"friend_player_id"`
}

type friendScoreComparisonChartRow struct {
	SongDisplayID       string                      `db:"song_display_id"`
	SongTitle           string                      `db:"song_title"`
	SongArtist          string                      `db:"song_artist"`
	ChartConst          chartconstant.ChartConstant `db:"chart_const"`
	IsConstUnknown      bool                        `db:"is_const_unknown"`
	LevelStar           *int                        `db:"level_star"`
	Attribute           *string                     `db:"attribute"`
	SelfRecordChartID   *int64                      `db:"self_record_chart_id"`
	FriendRecordChartID *int64                      `db:"friend_record_chart_id"`
	SelfScore           sql.NullInt64               `db:"self_score"`
	SelfClearLamp       sql.NullString              `db:"self_clear_lamp"`
	SelfComboLamp       sql.NullString              `db:"self_combo_lamp"`
	SelfFullChain       sql.NullString              `db:"self_full_chain"`
	SelfUpdatedAt       *time.Time                  `db:"self_updated_at"`
	FriendScore         sql.NullInt64               `db:"friend_score"`
	FriendClearLamp     sql.NullString              `db:"friend_clear_lamp"`
	FriendComboLamp     sql.NullString              `db:"friend_combo_lamp"`
	FriendFullChain     sql.NullString              `db:"friend_full_chain"`
	FriendUpdatedAt     *time.Time                  `db:"friend_updated_at"`
}

func (q *FriendScoreComparisonQueryService) FindAcceptedFriendPair(
	ctx context.Context,
	selfUserID int,
	friendUsername string,
) (*domainrepo.FriendScoreComparisonUsers, error) {
	// 双方向 accepted が2件ある場合だけ返す。不存在・片方向・申請中・自分自身は同じ未検出にする。
	const query = `
		SELECT
			self_u.username AS self_username,
			self_p.player_name AS self_player_name,
			self_u.player_id AS self_player_id,
			friend_u.username AS friend_username,
			friend_p.player_name AS friend_player_name,
			friend_u.player_id AS friend_player_id
		FROM users self_u
		INNER JOIN users friend_u
			ON friend_u.username = ?
		   AND friend_u.id <> self_u.id
		INNER JOIN friendships out_f
			ON out_f.user_id = self_u.id
		   AND out_f.friend_user_id = friend_u.id
		   AND out_f.status_id = ?
		INNER JOIN friendships in_f
			ON in_f.user_id = friend_u.id
		   AND in_f.friend_user_id = self_u.id
		   AND in_f.status_id = ?
		LEFT JOIN players self_p ON self_p.id = self_u.player_id
		LEFT JOIN players friend_p ON friend_p.id = friend_u.player_id
		WHERE self_u.id = ?
	`
	var row friendScoreComparisonUserRow
	err := sqlx.GetContext(
		ctx,
		q.db,
		&row,
		query,
		friendUsername,
		entity.FriendshipStatusAccepted,
		entity.FriendshipStatusAccepted,
		selfUserID,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, wrapFriendScoreComparisonQueryError("find accepted friend pair", err)
	}
	return &domainrepo.FriendScoreComparisonUsers{
		Self: domainrepo.FriendScoreComparisonUser{
			Username:   row.SelfUsername,
			PlayerName: row.SelfPlayerName.String,
			PlayerID:   row.SelfPlayerID,
		},
		Friend: domainrepo.FriendScoreComparisonUser{
			Username:   row.FriendUsername,
			PlayerName: row.FriendPlayerName.String,
			PlayerID:   row.FriendPlayerID,
		},
	}, nil
}

func (q *FriendScoreComparisonQueryService) ListChartRecords(
	ctx context.Context,
	selfPlayerID int,
	friendPlayerID int,
	difficulty string,
) ([]*domainrepo.FriendScoreComparisonChartRecord, error) {
	// 譜面マスタを起点に両者のレコードを1クエリでLEFT JOINする。未プレイの疑似レコードは作らない。
	const query = `
		SELECT
			s.display_id AS song_display_id,
			s.title AS song_title,
			s.artist AS song_artist,
			c.const AS chart_const,
			c.is_const_unknown AS is_const_unknown,
			NULL AS level_star,
			NULL AS attribute,
			self_pr.chart_id AS self_record_chart_id,
			friend_pr.chart_id AS friend_record_chart_id,
			self_pr.score AS self_score,
			self_cl.name AS self_clear_lamp,
			self_co.name AS self_combo_lamp,
			self_fc.name AS self_full_chain,
			self_pr.updated_at AS self_updated_at,
			friend_pr.score AS friend_score,
			friend_cl.name AS friend_clear_lamp,
			friend_co.name AS friend_combo_lamp,
			friend_fc.name AS friend_full_chain,
			friend_pr.updated_at AS friend_updated_at
		FROM charts c
		INNER JOIN songs s ON s.id = c.song_id
		INNER JOIN difficulties d ON d.id = c.difficulty_id
		LEFT JOIN player_records self_pr
			ON self_pr.chart_id = c.id
		   AND self_pr.player_id = ?
		LEFT JOIN player_records friend_pr
			ON friend_pr.chart_id = c.id
		   AND friend_pr.player_id = ?
		LEFT JOIN clear_lamp_types self_cl ON self_cl.id = self_pr.clear_lamp_id
		LEFT JOIN combo_lamp_types self_co ON self_co.id = self_pr.combo_lamp_id
		LEFT JOIN full_chain_types self_fc ON self_fc.id = self_pr.full_chain_id
		LEFT JOIN clear_lamp_types friend_cl ON friend_cl.id = friend_pr.clear_lamp_id
		LEFT JOIN combo_lamp_types friend_co ON friend_co.id = friend_pr.combo_lamp_id
		LEFT JOIN full_chain_types friend_fc ON friend_fc.id = friend_pr.full_chain_id
		WHERE s.is_deleted = 0
		  AND s.is_worldsend = 0
		  AND d.name = ?
		ORDER BY s.id ASC
	`
	var rows []friendScoreComparisonChartRow
	if err := sqlx.SelectContext(ctx, q.db, &rows, query, selfPlayerID, friendPlayerID, difficulty); err != nil {
		return nil, wrapFriendScoreComparisonQueryError("list chart records", err)
	}
	return mapFriendScoreComparisonRows(rows, "list chart records")
}

func (q *FriendScoreComparisonQueryService) ListWorldsendChartRecords(
	ctx context.Context,
	selfPlayerID int,
	friendPlayerID int,
) ([]*domainrepo.FriendScoreComparisonChartRecord, error) {
	const query = `
		SELECT
			s.display_id AS song_display_id,
			s.title AS song_title,
			s.artist AS song_artist,
			0 AS chart_const,
			1 AS is_const_unknown,
			wc.level_star AS level_star,
			wc.attribute AS attribute,
			self_pwr.worldsend_chart_id AS self_record_chart_id,
			friend_pwr.worldsend_chart_id AS friend_record_chart_id,
			self_pwr.score AS self_score,
			self_cl.name AS self_clear_lamp,
			self_co.name AS self_combo_lamp,
			self_fc.name AS self_full_chain,
			self_pwr.updated_at AS self_updated_at,
			friend_pwr.score AS friend_score,
			friend_cl.name AS friend_clear_lamp,
			friend_co.name AS friend_combo_lamp,
			friend_fc.name AS friend_full_chain,
			friend_pwr.updated_at AS friend_updated_at
		FROM worldsend_charts wc
		INNER JOIN songs s ON s.id = wc.song_id
		LEFT JOIN player_worldsend_records self_pwr
			ON self_pwr.worldsend_chart_id = wc.id
		   AND self_pwr.player_id = ?
		LEFT JOIN player_worldsend_records friend_pwr
			ON friend_pwr.worldsend_chart_id = wc.id
		   AND friend_pwr.player_id = ?
		LEFT JOIN clear_lamp_types self_cl ON self_cl.id = self_pwr.clear_lamp_id
		LEFT JOIN combo_lamp_types self_co ON self_co.id = self_pwr.combo_lamp_id
		LEFT JOIN full_chain_types self_fc ON self_fc.id = self_pwr.full_chain_id
		LEFT JOIN clear_lamp_types friend_cl ON friend_cl.id = friend_pwr.clear_lamp_id
		LEFT JOIN combo_lamp_types friend_co ON friend_co.id = friend_pwr.combo_lamp_id
		LEFT JOIN full_chain_types friend_fc ON friend_fc.id = friend_pwr.full_chain_id
		WHERE s.is_deleted = 0
		  AND s.is_worldsend = 1
		ORDER BY s.id ASC
	`
	var rows []friendScoreComparisonChartRow
	if err := sqlx.SelectContext(ctx, q.db, &rows, query, selfPlayerID, friendPlayerID); err != nil {
		return nil, wrapFriendScoreComparisonQueryError("list worldsend chart records", err)
	}
	return mapFriendScoreComparisonRows(rows, "list worldsend chart records")
}

func mapFriendScoreComparisonRows(rows []friendScoreComparisonChartRow, operation string) ([]*domainrepo.FriendScoreComparisonChartRecord, error) {
	records := make([]*domainrepo.FriendScoreComparisonChartRecord, 0, len(rows))
	for _, row := range rows {
		selfPlay, err := comparisonPlay(row.SelfRecordChartID, row.SelfScore, row.SelfClearLamp, row.SelfComboLamp, row.SelfFullChain, row.SelfUpdatedAt)
		if err != nil {
			return nil, wrapFriendScoreComparisonQueryError(operation, err)
		}
		friendPlay, err := comparisonPlay(row.FriendRecordChartID, row.FriendScore, row.FriendClearLamp, row.FriendComboLamp, row.FriendFullChain, row.FriendUpdatedAt)
		if err != nil {
			return nil, wrapFriendScoreComparisonQueryError(operation, err)
		}
		records = append(records, &domainrepo.FriendScoreComparisonChartRecord{
			SongDisplayID:  row.SongDisplayID,
			SongTitle:      row.SongTitle,
			SongArtist:     row.SongArtist,
			ChartConst:     row.ChartConst,
			IsConstUnknown: row.IsConstUnknown,
			LevelStar:      row.LevelStar,
			Attribute:      row.Attribute,
			Self:           selfPlay,
			Friend:         friendPlay,
		})
	}
	return records, nil
}

func comparisonPlay(
	chartID *int64,
	rawScore sql.NullInt64,
	clearLamp sql.NullString,
	comboLamp sql.NullString,
	fullChain sql.NullString,
	updatedAt *time.Time,
) (*domainrepo.FriendScoreComparisonPlay, error) {
	if chartID == nil {
		return nil, nil
	}
	if !rawScore.Valid || updatedAt == nil || rawScore.Int64 < 0 || rawScore.Int64 > int64(^uint32(0)) {
		return nil, fmt.Errorf("player record is incomplete")
	}
	parsed, err := score.NewScore(uint32(rawScore.Int64))
	if err != nil {
		return nil, err
	}
	return &domainrepo.FriendScoreComparisonPlay{
		Score:     uint32(parsed),
		ClearLamp: clearLamp.String,
		ComboLamp: comboLamp.String,
		FullChain: fullChain.String,
		UpdatedAt: *updatedAt,
	}, nil
}

func wrapFriendScoreComparisonQueryError(operation string, err error) error {
	return fmt.Errorf("%w: %s: %v", domainrepo.ErrRepositoryOperationFailed, operation, err)
}
