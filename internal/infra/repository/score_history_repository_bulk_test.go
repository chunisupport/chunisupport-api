package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// historyBulkExecutor は本番SQLをDBへ委譲し、チャンク数と途中失敗後の打ち切りだけを観測する。
type historyBulkExecutor struct {
	domainrepo.Executor
	chunks []int
	failAt int
	cause  error
}

func (e *historyBulkExecutor) NamedExecContext(ctx context.Context, query string, arg any) (sql.Result, error) {
	e.chunks = append(e.chunks, reflect.ValueOf(arg).Len())
	if len(e.chunks) == e.failAt {
		return nil, e.cause
	}
	return e.Executor.NamedExecContext(ctx, query, arg)
}

type historyBulkVariant struct {
	name        string
	table       string
	chartColumn string
	insert      func(context.Context, domainrepo.ScoreHistoryRepository, domainrepo.Executor, []domainrepo.PlayerRecordHistory) error
}

func historyBulkVariants() []historyBulkVariant {
	return []historyBulkVariant{
		{"通常", "player_record_histories", "chart_id", func(ctx context.Context, repo domainrepo.ScoreHistoryRepository, exec domainrepo.Executor, rows []domainrepo.PlayerRecordHistory) error {
			return repo.BulkInsertStandard(ctx, exec, rows)
		}},
		{"WORLDSEND", "player_worldsend_record_histories", "worldsend_chart_id", func(ctx context.Context, repo domainrepo.ScoreHistoryRepository, exec domainrepo.Executor, rows []domainrepo.PlayerRecordHistory) error {
			worldsend := make([]domainrepo.PlayerWorldsendRecordHistory, len(rows))
			for i, row := range rows {
				worldsend[i] = domainrepo.PlayerWorldsendRecordHistory{PlayerID: row.PlayerID, WorldsendChartID: row.ChartID, State: domainrepo.WorldsendRecordState{
					Score: row.State.Score, ClearLampID: row.State.ClearLampID, ComboLampID: row.State.ComboLampID, FullChainID: row.State.FullChainID, UpdatedAt: row.State.UpdatedAt,
				}}
			}
			return repo.BulkInsertWorldsend(ctx, exec, worldsend)
		}},
	}
}

func historyBulkRows(count int) []domainrepo.PlayerRecordHistory {
	rows := make([]domainrepo.PlayerRecordHistory, count)
	for i := range rows {
		rows[i] = domainrepo.PlayerRecordHistory{PlayerID: 7, ChartID: 41, State: domainrepo.PlayerRecordState{
			Score: 900000 + i, ClearLampID: 5, ComboLampID: 3, FullChainID: 2,
			UpdatedAt: time.Date(2026, 11, 1, 0, 0, i, 0, time.UTC),
		}}
	}
	return rows
}

func TestScoreHistoryRepository_BulkInsert_境界と具体保存値(t *testing.T) {
	for _, variant := range historyBulkVariants() {
		for _, tt := range []struct {
			count  int
			chunks []int
		}{
			{0, nil}, {2999, []int{2999}}, {3000, []int{3000}}, {3001, []int{3000, 1}},
		} {
			t.Run(fmt.Sprintf("%s/%d件", variant.name, tt.count), func(t *testing.T) {
				// Given
				db := newTransferRepositoryTestDB(t)
				exec := &historyBulkExecutor{Executor: db}
				repo := NewScoreHistoryRepository(db)

				// When
				err := variant.insert(context.Background(), repo, exec, historyBulkRows(tt.count))

				// Then
				require.NoError(t, err)
				assert.Equal(t, tt.chunks, exec.chunks)
				var count int
				require.NoError(t, db.Get(&count, "SELECT COUNT(*) FROM "+variant.table))
				assert.Equal(t, tt.count, count)
				if tt.count == 0 {
					return
				}
				var last struct {
					PlayerID    int       `db:"player_id"`
					ChartID     int       `db:"chart_id"`
					Score       int       `db:"score"`
					ClearLampID int       `db:"clear_lamp_id"`
					ComboLampID int       `db:"combo_lamp_id"`
					FullChainID int       `db:"full_chain_id"`
					UpdatedAt   time.Time `db:"updated_at"`
				}
				require.NoError(t, db.Get(&last, "SELECT player_id, "+variant.chartColumn+" AS chart_id, score, clear_lamp_id, combo_lamp_id, full_chain_id, updated_at FROM "+variant.table+" ORDER BY updated_at DESC LIMIT 1"))
				assert.Equal(t, 7, last.PlayerID)
				assert.Equal(t, 41, last.ChartID)
				assert.Equal(t, 899999+tt.count, last.Score)
				assert.Equal(t, 5, last.ClearLampID)
				assert.Equal(t, 3, last.ComboLampID)
				assert.Equal(t, 2, last.FullChainID)
				assert.Equal(t, time.Date(2026, 11, 1, 0, 0, tt.count-1, 0, time.UTC), last.UpdatedAt)
			})
		}
	}
}

func TestScoreHistoryRepository_BulkInsert_途中失敗は原因を保持して後続を実行しない(t *testing.T) {
	duplicate := &mysql.MySQLError{Number: 1062, Message: "Duplicate entry"}
	for _, variant := range historyBulkVariants() {
		for _, tt := range []struct {
			name     string
			cause    error
			conflict bool
		}{
			{"キャンセル", context.Canceled, false},
			{"MySQL主キー重複", duplicate, true},
		} {
			t.Run(variant.name+"/"+tt.name, func(t *testing.T) {
				// Given: 3チャンクのうち2番目で失敗させ、最初のSQLは実行する。
				db := newTransferRepositoryTestDB(t)
				exec := &historyBulkExecutor{Executor: db, failAt: 2, cause: tt.cause}

				// When
				err := variant.insert(context.Background(), NewScoreHistoryRepository(db), exec, historyBulkRows(6001))

				// Then
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.cause)
				assert.Equal(t, tt.conflict, errors.Is(err, domainrepo.ErrScoreHistoryTimestampConflict))
				assert.Equal(t, []int{3000, 3000}, exec.chunks)
				var count int
				require.NoError(t, db.Get(&count, "SELECT COUNT(*) FROM "+variant.table))
				assert.Equal(t, 3000, count)
			})
		}
	}
}
