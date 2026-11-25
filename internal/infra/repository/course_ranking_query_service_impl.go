package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
)

type CourseRankingQueryService struct{}

var _ domainrepo.CourseRankingQueryService = CourseRankingQueryService{}

type courseRankingRow struct {
	UserID     int       `db:"user_id"`
	Username   string    `db:"username"`
	PlayerName string    `db:"player_name"`
	Score      uint32    `db:"score"`
	IsClear    bool      `db:"is_clear"`
	ComboLamp  string    `db:"combo_lamp"`
	UpdatedAt  time.Time `db:"updated_at"`
	Total      int       `db:"total"`
}

const courseRankingRecordColumns = `
	u.id AS user_id, u.username, p.player_name, r.score, r.is_clear, cl.name AS combo_lamp, r.updated_at
`

const courseRankingOrder = `ORDER BY r.score DESC, r.updated_at DESC, u.username ASC`

// ListFriends は本人と相互承認済みフレンドの自己ベストを返します。
func (CourseRankingQueryService) ListFriends(ctx context.Context, exec domainrepo.Executor, courseID, userID int) ([]domainrepo.CourseRankingRecord, error) {
	const query = `SELECT ` + courseRankingRecordColumns + `
		FROM (` + friendRankingUsersSQL + `) ranking_users
		INNER JOIN users u ON u.id = ranking_users.user_id
		INNER JOIN players p ON p.id = u.player_id
		INNER JOIN player_course_records r ON r.player_id = p.id AND r.course_id = ?
		INNER JOIN combo_lamp_types cl ON cl.id = r.combo_lamp_id
	` + courseRankingOrder
	var rows []courseRankingRow
	accepted := entity.FriendshipStatusAccepted
	if err := exec.SelectContext(ctx, &rows, query, userID, accepted, userID, accepted, courseID); err != nil {
		return nil, fmt.Errorf("%w: list friend course rankings: %w", domainrepo.ErrRepositoryOperationFailed, err)
	}
	return courseRankingRecords(rows), nil
}

// ListTop は順位表と、表示上限を適用する前の登録者数を同じクエリで取得します。
func (CourseRankingQueryService) ListTop(ctx context.Context, exec domainrepo.Executor, courseID, limit int) ([]domainrepo.CourseRankingRecord, int, error) {
	const query = `SELECT ` + courseRankingRecordColumns + `, COUNT(*) OVER () AS total
		FROM player_course_records r
		INNER JOIN players p ON p.id = r.player_id
		INNER JOIN users u ON u.player_id = p.id
		INNER JOIN combo_lamp_types cl ON cl.id = r.combo_lamp_id
		WHERE r.course_id = ?
	` + courseRankingOrder + ` LIMIT ?`
	var rows []courseRankingRow
	if err := exec.SelectContext(ctx, &rows, query, courseID, limit); err != nil {
		return nil, 0, fmt.Errorf("%w: list admin course rankings: %w", domainrepo.ErrRepositoryOperationFailed, err)
	}
	total := 0
	if len(rows) > 0 {
		total = rows[0].Total
	}
	return courseRankingRecords(rows), total, nil
}

func courseRankingRecords(rows []courseRankingRow) []domainrepo.CourseRankingRecord {
	records := make([]domainrepo.CourseRankingRecord, 0, len(rows))
	for _, row := range rows {
		records = append(records, domainrepo.CourseRankingRecord{
			UserID: row.UserID, Username: row.Username, PlayerName: row.PlayerName,
			Score: row.Score, IsClear: row.IsClear, ComboLamp: row.ComboLamp, UpdatedAt: row.UpdatedAt,
		})
	}
	return records
}
