package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/service"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/chunisupport/chunisupport-api/internal/infra/models"
	"github.com/jmoiron/sqlx"
)

type playerDataRepository struct {
	db *sqlx.DB
}

// NewPlayerDataRepository は PlayerDataRepository の実装を生成します。
func NewPlayerDataRepository(db *sqlx.DB) repository.PlayerDataRepository {
	return &playerDataRepository{db: db}
}

// LoadMasterData はプレイヤーデータ登録に必要なマスタ情報を取得します。
// songs/charts/worldsend_chartsの読み取りのみのためトランザクション外で呼び出せます。
func (r *playerDataRepository) LoadMasterData(ctx context.Context, officialIdxList []string) (*repository.PlayerDataMaster, error) {
	executor := r.db
	result := &repository.PlayerDataMaster{
		Songs:             make(map[string]entity.PlayerDataSong),
		ChartsByKey:       make(map[string]entity.PlayerDataChart),
		ChartsByID:        make(map[int]entity.PlayerDataChart),
		WorldsendBySongID: make(map[int]entity.PlayerDataWorldsendChart),
	}

	if len(officialIdxList) == 0 {
		return result, nil
	}

	songModels, err := selectModelsInChunks[string, models.PlayerDataSongModel](
		ctx,
		executor,
		officialIdxList,
		`
			SELECT id, display_id, title, artist, genre_id, bpm, released_at, official_idx, jacket, is_deleted
			FROM songs
			WHERE official_idx IN (?)
		`,
		"songs",
	)
	if err != nil {
		return nil, err
	}

	songIDs := make([]int, 0, len(songModels))
	for _, model := range songModels {
		song := model.ToEntity()
		result.Songs[song.OfficialIdx] = *song
		songIDs = append(songIDs, song.ID)
	}

	if len(songIDs) == 0 {
		return result, nil
	}

	chartModels, err := selectModelsInChunks[int, models.PlayerDataChartModel](
		ctx,
		executor,
		songIDs,
		`
			SELECT id, song_id, difficulty_id, const, is_const_unknown, notes
			FROM charts
			WHERE song_id IN (?)
		`,
		"charts",
	)
	if err != nil {
		return nil, err
	}

	for _, model := range chartModels {
		chart, err := model.ToEntity()
		if err != nil {
			return nil, fmt.Errorf("failed to convert chart model to entity: %w", err)
		}
		key := fmt.Sprintf("%d:%d", chart.SongID, chart.DifficultyID)
		result.ChartsByKey[key] = *chart
		result.ChartsByID[chart.ID] = *chart
	}

	worldsendModels, err := selectModelsInChunks[int, models.PlayerDataWorldsendChartModel](
		ctx,
		executor,
		songIDs,
		`
			SELECT id, song_id
			FROM worldsend_charts
			WHERE song_id IN (?)
		`,
		"worldsend_charts",
	)
	if err != nil {
		return nil, err
	}

	for _, model := range worldsendModels {
		chart := model.ToEntity()
		result.WorldsendBySongID[chart.SongID] = *chart
	}

	return result, nil
}

// SavePlayerData はプレイヤーデータを一括で保存します。
// 書き込み操作のため必ずトランザクション内で呼び出してください。exec が nil の場合はエラーを返します。
func (r *playerDataRepository) SavePlayerData(ctx context.Context, exec repository.Executor, input repository.PlayerDataSaveInput) error {
	if exec == nil {
		return fmt.Errorf("SavePlayerData requires a non-nil executor: must be called within a transaction")
	}
	executor := exec

	if err := r.saveFullRecords(ctx, executor, input.FullRecords); err != nil {
		return fmt.Errorf("failed to save player records (count=%d): %w", len(input.FullRecords), err)
	}

	if err := r.saveWorldsendRecords(ctx, executor, input.WorldsendRecords); err != nil {
		return fmt.Errorf("failed to save worldsend records (count=%d): %w", len(input.WorldsendRecords), err)
	}

	return nil
}

// GetOverpowerTargetStats はOVER POWER割合計算の分母となる対象楽曲の最大OP合計を取得します。
// songs/chartsの読み取りのみのためトランザクション外で呼び出せます。
func (r *playerDataRepository) GetOverpowerTargetStats(ctx context.Context, filter repository.OverpowerTargetFilter) (*repository.OverpowerTargetStats, error) {
	executor := r.db

	where := make([]string, 0, 2)
	if filter.ExcludeWorldsend {
		where = append(where, "s.is_worldsend = 0")
	}
	if filter.ExcludeDeleted {
		where = append(where, "s.is_deleted = 0")
	}

	maxConstExpr := "MAX(c.const)"
	args := make([]any, 0, 2)
	joins := `
		INNER JOIN charts c ON c.song_id = s.id
	`
	if filter.PlayerID != nil {
		maxConstExpr = `MAX(
			CASE
				WHEN pls_ultima.song_id IS NOT NULL AND d.name = 'ULTIMA' THEN NULL
				ELSE c.const
			END
		)`
		joins += `
			INNER JOIN difficulties d ON d.id = c.difficulty_id
			LEFT JOIN player_locked_songs pls_song
				ON pls_song.song_id = s.id
				AND pls_song.player_id = ?
				AND pls_song.is_ultima = 0
			LEFT JOIN player_locked_songs pls_ultima
				ON pls_ultima.song_id = s.id
				AND pls_ultima.player_id = ?
				AND pls_ultima.is_ultima = 1
		`
		args = append(args, *filter.PlayerID, *filter.PlayerID)
		where = append(where, "pls_song.song_id IS NULL")
	}

	query := `
		SELECT
			s.id AS song_id,
			` + maxConstExpr + ` AS max_const
		FROM songs s
	` + joins
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " GROUP BY s.id"
	if filter.PlayerID != nil {
		query += " HAVING max_const IS NOT NULL"
	}

	var rows []struct {
		SongID   int     `db:"song_id"`
		MaxConst float64 `db:"max_const"`
	}
	if err := executor.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("%w: failed to select overpower target stats: %w", repository.ErrRepositoryOperationFailed, err)
	}

	total := 0.0
	for _, row := range rows {
		total += service.CalcSongMaxOP(row.MaxConst)
	}

	return &repository.OverpowerTargetStats{
		SongCount:         len(rows),
		MaxOverpowerTotal: total,
	}, nil
}

func selectModelsInChunks[T any, M any](ctx context.Context, exec repository.Executor, items []T, query string, queryName string) ([]M, error) {
	if len(items) == 0 {
		return nil, nil
	}

	results := make([]M, 0, len(items))
	batchSize := info.BulkInsertChunkSize
	for i := 0; i < len(items); i += batchSize {
		end := min(i+batchSize, len(items))
		batch := items[i:end]
		batchQuery, batchArgs, err := sqlx.In(query, batch)
		if err != nil {
			return nil, fmt.Errorf("failed to build %s query: %w", queryName, err)
		}
		batchQuery = exec.Rebind(batchQuery)

		var batchModels []M
		if err := exec.SelectContext(ctx, &batchModels, batchQuery, batchArgs...); err != nil {
			return nil, fmt.Errorf("failed to select %s: %w", queryName, err)
		}
		results = append(results, batchModels...)
	}

	return results, nil
}

type playerDataRecordRow struct {
	PlayerID    int       `db:"player_id"`
	ChartID     int       `db:"chart_id"`
	Score       int       `db:"score"`
	ClearLampID int       `db:"clear_lamp_id"`
	ComboLampID int       `db:"combo_lamp_id"`
	FullChainID int       `db:"full_chain_id"`
	SlotID      int       `db:"slot_id"`
	SlotOrder   *int      `db:"slot_order"`
	UpdatedAt   time.Time `db:"updated_at"`
}

type playerDataWorldsendRecordRow struct {
	PlayerID         int       `db:"player_id"`
	WorldsendChartID int       `db:"worldsend_chart_id"`
	Score            int       `db:"score"`
	ClearLampID      int       `db:"clear_lamp_id"`
	ComboLampID      int       `db:"combo_lamp_id"`
	FullChainID      int       `db:"full_chain_id"`
	UpdatedAt        time.Time `db:"updated_at"`
}

const (
	fullRecordChangedCondition = "score <> VALUES(score) OR " +
		"clear_lamp_id <> VALUES(clear_lamp_id) OR " +
		"combo_lamp_id <> VALUES(combo_lamp_id) OR " +
		"full_chain_id <> VALUES(full_chain_id)"

	worldsendRecordChangedCondition = "score <> VALUES(score) OR " +
		"clear_lamp_id <> VALUES(clear_lamp_id) OR " +
		"combo_lamp_id <> VALUES(combo_lamp_id) OR " +
		"full_chain_id <> VALUES(full_chain_id)"

	changedConditionPlaceholder = "{{CHANGED_CONDITION}}"
)

var fullRecordUpsertQuery = replaceQueryPlaceholder(`
		INSERT INTO player_records (
			player_id, chart_id, score, clear_lamp_id, combo_lamp_id,
			full_chain_id, slot_id, slot_order, updated_at
		) VALUES (
			:player_id, :chart_id, :score, :clear_lamp_id, :combo_lamp_id,
			:full_chain_id, :slot_id, :slot_order, :updated_at
		)
		ON DUPLICATE KEY UPDATE
			updated_at = IF(
				{{CHANGED_CONDITION}},
				VALUES(updated_at),
				updated_at
			),
			score = VALUES(score),
			clear_lamp_id = VALUES(clear_lamp_id),
			combo_lamp_id = VALUES(combo_lamp_id),
			full_chain_id = VALUES(full_chain_id),
			slot_id = VALUES(slot_id),
			slot_order = VALUES(slot_order)
	`, changedConditionPlaceholder, fullRecordChangedCondition)

var worldsendRecordUpsertQuery = replaceQueryPlaceholder(`
		INSERT INTO player_worldsend_records (
			player_id, worldsend_chart_id, score, clear_lamp_id,
			combo_lamp_id, full_chain_id, updated_at
		) VALUES (
			:player_id, :worldsend_chart_id, :score, :clear_lamp_id,
			:combo_lamp_id, :full_chain_id, :updated_at
		)
		ON DUPLICATE KEY UPDATE
			updated_at = IF(
				{{CHANGED_CONDITION}},
				VALUES(updated_at),
				updated_at
			),
			score = VALUES(score),
			clear_lamp_id = VALUES(clear_lamp_id),
			combo_lamp_id = VALUES(combo_lamp_id),
			full_chain_id = VALUES(full_chain_id)
	`, changedConditionPlaceholder, worldsendRecordChangedCondition)

func replaceQueryPlaceholder(query string, placeholder string, replacement string) string {
	return strings.ReplaceAll(query, placeholder, replacement)
}

func (r *playerDataRepository) saveFullRecords(ctx context.Context, exec repository.Executor, records []repository.PlayerRecordForUpsert) error {
	rows := make([]playerDataRecordRow, 0, len(records))
	for _, record := range records {
		rows = append(rows, playerDataRecordRow{
			PlayerID:    record.PlayerID,
			ChartID:     record.ChartID,
			Score:       record.State.Score,
			ClearLampID: record.State.ClearLampID,
			ComboLampID: record.State.ComboLampID,
			FullChainID: record.State.FullChainID,
			SlotID:      record.State.SlotID,
			SlotOrder:   record.State.SlotOrder,
			UpdatedAt:   record.State.UpdatedAt,
		})
	}

	return bulkUpsert(ctx, exec, rows, fullRecordUpsertQuery, "player records")
}

func (r *playerDataRepository) saveWorldsendRecords(ctx context.Context, exec repository.Executor, records []repository.WorldsendRecordForUpsert) error {
	rows := make([]playerDataWorldsendRecordRow, 0, len(records))
	for _, record := range records {
		rows = append(rows, playerDataWorldsendRecordRow{
			PlayerID:         record.PlayerID,
			WorldsendChartID: record.ChartID,
			Score:            record.State.Score,
			ClearLampID:      record.State.ClearLampID,
			ComboLampID:      record.State.ComboLampID,
			FullChainID:      record.State.FullChainID,
			UpdatedAt:        record.State.UpdatedAt,
		})
	}

	return bulkUpsert(ctx, exec, rows, worldsendRecordUpsertQuery, "worldsend records")
}

func bulkUpsert[T any](ctx context.Context, exec repository.Executor, rows []T, query string, recordType string) error {
	if len(rows) == 0 {
		return nil
	}

	batchSize := info.BulkInsertChunkSize
	for i := 0; i < len(rows); i += batchSize {
		end := min(i+batchSize, len(rows))
		batch := rows[i:end]
		if _, err := exec.NamedExecContext(ctx, query, batch); err != nil {
			return fmt.Errorf("failed to save %s: %w", recordType, err)
		}
	}

	return nil
}
