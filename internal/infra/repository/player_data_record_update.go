package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/info"
)

type recordUpdateRow struct {
	playerID int
	chartID  int
	values   []any
}

func updateFullRecords(ctx context.Context, exec repository.Executor, records []repository.PlayerRecordForUpsert) error {
	rows := make([]recordUpdateRow, 0, len(records))
	for _, r := range records {
		s := r.State
		rows = append(rows, recordUpdateRow{r.PlayerID, r.ChartID, []any{s.Score, s.ClearLampID, s.ComboLampID, s.FullChainID, s.SlotID, s.SlotOrder, s.UpdatedAt}})
	}
	return updateRecordRows(ctx, exec, "player_records", "chart_id", []string{"score", "clear_lamp_id", "combo_lamp_id", "full_chain_id", "slot_id", "slot_order", "updated_at"}, rows, false)
}

func updateWorldsendRecords(ctx context.Context, exec repository.Executor, records []repository.WorldsendRecordForUpsert) error {
	rows := make([]recordUpdateRow, 0, len(records))
	for _, r := range records {
		s := r.State
		rows = append(rows, recordUpdateRow{r.PlayerID, r.ChartID, []any{s.Score, s.ClearLampID, s.ComboLampID, s.FullChainID, s.UpdatedAt}})
	}
	return updateRecordRows(ctx, exec, "player_worldsend_records", "worldsend_chart_id", []string{"score", "clear_lamp_id", "combo_lamp_id", "full_chain_id", "updated_at"}, rows, false)
}

func updateRecordSlots(ctx context.Context, exec repository.Executor, records []repository.PlayerRecordSlotUpdate) error {
	rows := make([]recordUpdateRow, 0, len(records))
	for _, r := range records {
		rows = append(rows, recordUpdateRow{r.PlayerID, r.ChartID, []any{r.SlotID, r.SlotOrder}})
	}
	return updateRecordRows(ctx, exec, "player_records", "chart_id", []string{"slot_id", "slot_order"}, rows, true)
}

// updateRecordRows はプレイヤー別・チャンク単位のCASE更新でN+1とプレースホルダー上限超過を避けます。
// 識別子はこのファイル内の固定値のみを渡し、値はすべてバインドします。
func updateRecordRows(ctx context.Context, exec repository.Executor, table, key string, columns []string, rows []recordUpdateRow, preserveTimestamp bool) error {
	byPlayer := make(map[int][]recordUpdateRow)
	for _, r := range rows {
		byPlayer[r.playerID] = append(byPlayer[r.playerID], r)
	}
	for playerID, records := range byPlayer {
		for start := 0; start < len(records); start += info.PlayerRecordUpdateChunkSize {
			batch := records[start:min(start+info.PlayerRecordUpdateChunkSize, len(records))]
			var query strings.Builder
			fmt.Fprintf(&query, "UPDATE %s SET ", table)
			args := make([]any, 0, len(batch)*(2*len(columns)+1)+1)
			for i, column := range columns {
				if i > 0 {
					query.WriteString(", ")
				}
				fmt.Fprintf(&query, "%s = CASE %s", column, key)
				for _, r := range batch {
					query.WriteString(" WHEN ? THEN ?")
					args = append(args, r.chartID, r.values[i])
				}
				query.WriteString(" END")
			}
			// 枠だけの更新ではON UPDATE CURRENT_TIMESTAMPを明示的に抑止します。
			if preserveTimestamp {
				query.WriteString(", updated_at = updated_at")
			}
			fmt.Fprintf(&query, " WHERE player_id = ? AND %s IN (", key)
			args = append(args, playerID)
			for i, r := range batch {
				if i > 0 {
					query.WriteString(",")
				}
				query.WriteString("?")
				args = append(args, r.chartID)
			}
			query.WriteString(")")
			if _, err := exec.ExecContext(ctx, query.String(), args...); err != nil {
				return fmt.Errorf("failed to update %s: %w", table, err)
			}
		}
	}
	return nil
}
