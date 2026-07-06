package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/jmoiron/sqlx"
)

type playerDataBatchRepository struct {
	db *sqlx.DB
}

func NewPlayerDataBatchRepository(db *sqlx.DB) domainrepo.PlayerDataBatchRepository {
	return &playerDataBatchRepository{db: db}
}

func (r *playerDataBatchRepository) LoadSnapshot(ctx context.Context, operationalDate time.Time) (domainrepo.PlayerDataMasterSnapshot, error) {
	var snapshot domainrepo.PlayerDataMasterSnapshot
	var version batchVersionRow
	if err := r.db.GetContext(ctx, &version, `
		SELECT id, name, released_at
		FROM versions
		WHERE released_at <= ?
		ORDER BY released_at DESC, id DESC
		LIMIT 1`, operationalDate.Format(time.DateOnly)); err != nil {
		return snapshot, err
	}
	snapshot.Version = domainrepo.BatchVersion{ID: version.ID, Name: version.Name, ReleasedAt: version.ReleasedAt}
	var songs []batchSongRow
	if err := r.db.SelectContext(ctx, &songs, `
		SELECT id, released_at, is_deleted, is_worldsend, official_idx
		FROM songs`); err != nil {
		return snapshot, err
	}
	for _, song := range songs {
		snapshot.Songs = append(snapshot.Songs, domainrepo.BatchSong{ID: song.ID, ReleasedAt: song.ReleasedAt, IsDeleted: song.IsDeleted, IsWorldsend: song.IsWorldsend, OfficialIndex: song.OfficialIndex})
	}
	var charts []batchChartRow
	if err := r.db.SelectContext(ctx, &charts, `
		SELECT c.id, c.song_id, c.difficulty_id, d.name AS difficulty_name,
		       c.const AS chart_const, c.is_const_unknown
		FROM charts c
		INNER JOIN difficulties d ON d.id = c.difficulty_id`); err != nil {
		return snapshot, err
	}
	for _, chart := range charts {
		snapshot.Charts = append(snapshot.Charts, domainrepo.BatchChart{ID: chart.ID, SongID: chart.SongID, DifficultyID: chart.DifficultyID, DifficultyName: chart.DifficultyName, ChartConst: chart.ChartConst, IsConstUnknown: chart.IsConstUnknown})
	}
	var slots []struct {
		ID   int    `db:"id"`
		Name string `db:"name"`
	}
	if err := r.db.SelectContext(ctx, &slots, `SELECT id, name FROM slots`); err != nil {
		return snapshot, err
	}
	snapshot.SlotIDs = make(map[string]int, len(slots))
	for _, slot := range slots {
		snapshot.SlotIDs[slot.Name] = slot.ID
	}
	if err := r.db.GetContext(ctx, &snapshot.UpperBound, `SELECT COALESCE(MAX(id), 0) FROM players`); err != nil {
		return snapshot, err
	}
	return snapshot, nil
}

func (r *playerDataBatchRepository) ListPlayerKeys(ctx context.Context, afterID, upperBound, limit int) ([]domainrepo.PlayerBatchKey, error) {
	var rows []playerBatchKeyRow
	err := r.db.SelectContext(ctx, &rows, `
		SELECT id, data_collected_at
		FROM players
		WHERE id > ? AND id <= ?
		ORDER BY id
		LIMIT ?`, afterID, upperBound, limit)
	keys := make([]domainrepo.PlayerBatchKey, 0, len(rows))
	for _, row := range rows {
		keys = append(keys, domainrepo.PlayerBatchKey{ID: row.ID, DataCollectedAt: row.DataCollectedAt})
	}
	return keys, err
}

func (r *playerDataBatchRepository) ProcessPlayer(ctx context.Context, key domainrepo.PlayerBatchKey, buildUpdate func(domainrepo.PlayerBatchData) (domainrepo.PlayerBatchUpdate, error)) (status domainrepo.PlayerBatchProcessStatus, err error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return status, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var playerRow playerBatchDataRow
	if err = tx.GetContext(ctx, &playerRow, `
		SELECT id, last_played_at, data_collected_at
		FROM players
		WHERE id = ?
		FOR UPDATE`, key.ID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			_ = tx.Rollback()
			return domainrepo.PlayerBatchDeleted, nil
		}
		return status, err
	}
	data := domainrepo.PlayerBatchData{ID: playerRow.ID, LastPlayedAt: playerRow.LastPlayedAt, DataCollectedAt: playerRow.DataCollectedAt}
	if !equalNullableTime(data.DataCollectedAt, key.DataCollectedAt) {
		_ = tx.Rollback()
		return domainrepo.PlayerBatchConflict, nil
	}
	var recordRows []playerBatchRecordRow
	if err = tx.SelectContext(ctx, &recordRows, `
		SELECT pr.chart_id, pr.score, pr.combo_lamp_id, sl.name AS slot_name, pr.slot_order
		FROM player_records pr
		INNER JOIN slots sl ON sl.id = pr.slot_id
		WHERE pr.player_id = ?
		ORDER BY pr.chart_id
		FOR UPDATE`, key.ID); err != nil {
		return status, err
	}
	for _, row := range recordRows {
		data.Records = append(data.Records, domainrepo.PlayerBatchRecord{ChartID: row.ChartID, Score: row.Score, ComboLampID: row.ComboLampID, SlotName: row.SlotName, SlotOrder: row.SlotOrder})
	}
	var lockedRows []playerBatchLockedSongRow
	if err = tx.SelectContext(ctx, &lockedRows, `
		SELECT song_id, is_ultima
		FROM player_locked_songs
		WHERE player_id = ?
		ORDER BY song_id, is_ultima`, key.ID); err != nil {
		return status, err
	}
	for _, row := range lockedRows {
		data.LockedSongs = append(data.LockedSongs, domainrepo.PlayerBatchLockedSong{SongID: row.SongID, IsUltima: row.IsUltima})
	}
	update, err := buildUpdate(data)
	if err != nil {
		return status, err
	}
	if update.ResetSlots {
		if _, err = tx.ExecContext(ctx, `
			UPDATE player_records
			SET slot_id = (SELECT id FROM slots WHERE name = 'none'), slot_order = NULL
			WHERE player_id = ?`, key.ID); err != nil {
			return status, err
		}
		if err = assignSlots(ctx, tx, key.ID, update.Assignments); err != nil {
			return status, err
		}
	}
	result, err := tx.ExecContext(ctx, `
		UPDATE players
		SET calculated_player_rating = ?, best_average_rating = ?,
		    new_average_rating = ?, overpower_value = ?
		WHERE id = ? AND data_collected_at <=> ?`,
		update.PlayerRating, update.BestAverage, update.NewAverage, update.Overpower,
		key.ID, key.DataCollectedAt)
	if err != nil {
		return status, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return status, err
	}
	if affected == 0 {
		_ = tx.Rollback()
		return domainrepo.PlayerBatchConflict, nil
	}
	if err = tx.Commit(); err != nil {
		return status, err
	}
	return domainrepo.PlayerBatchUpdated, nil
}

type batchVersionRow struct {
	ID         int       `db:"id"`
	Name       string    `db:"name"`
	ReleasedAt time.Time `db:"released_at"`
}

type batchSongRow struct {
	ID            int        `db:"id"`
	ReleasedAt    *time.Time `db:"released_at"`
	IsDeleted     bool       `db:"is_deleted"`
	IsWorldsend   bool       `db:"is_worldsend"`
	OfficialIndex string     `db:"official_idx"`
}

type batchChartRow struct {
	ID             int     `db:"id"`
	SongID         int     `db:"song_id"`
	DifficultyID   int     `db:"difficulty_id"`
	DifficultyName string  `db:"difficulty_name"`
	ChartConst     float64 `db:"chart_const"`
	IsConstUnknown bool    `db:"is_const_unknown"`
}

type playerBatchKeyRow struct {
	ID              int        `db:"id"`
	DataCollectedAt *time.Time `db:"data_collected_at"`
}

type playerBatchDataRow struct {
	ID              int        `db:"id"`
	LastPlayedAt    *time.Time `db:"last_played_at"`
	DataCollectedAt *time.Time `db:"data_collected_at"`
}

type playerBatchRecordRow struct {
	ChartID     int    `db:"chart_id"`
	Score       uint32 `db:"score"`
	ComboLampID int    `db:"combo_lamp_id"`
	SlotName    string `db:"slot_name"`
	SlotOrder   *int   `db:"slot_order"`
}

type playerBatchLockedSongRow struct {
	SongID   int  `db:"song_id"`
	IsUltima bool `db:"is_ultima"`
}

func assignSlots(ctx context.Context, tx *sqlx.Tx, playerID int, assignments []domainrepo.PlayerBatchSlotAssignment) error {
	if len(assignments) == 0 {
		return nil
	}
	var slotCase, orderCase, placeholders strings.Builder
	slotArgs := make([]any, 0, len(assignments)*2)
	orderArgs := make([]any, 0, len(assignments)*2)
	for i, assignment := range assignments {
		slotCase.WriteString(" WHEN ? THEN ?")
		slotArgs = append(slotArgs, assignment.ChartID, assignment.SlotID)
		orderCase.WriteString(" WHEN ? THEN ?")
		orderArgs = append(orderArgs, assignment.ChartID, assignment.Position)
		if i > 0 {
			placeholders.WriteString(",")
		}
		placeholders.WriteString("?")
	}
	args := append(slotArgs, orderArgs...)
	args = append(args, playerID)
	for _, assignment := range assignments {
		args = append(args, assignment.ChartID)
	}
	query := fmt.Sprintf(`
		UPDATE player_records
		SET slot_id = CASE chart_id%s END,
		    slot_order = CASE chart_id%s END
		WHERE player_id = ? AND chart_id IN (%s)`, slotCase.String(), orderCase.String(), placeholders.String())
	_, err := tx.ExecContext(ctx, query, args...)
	return err
}

func equalNullableTime(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Equal(*b)
}
