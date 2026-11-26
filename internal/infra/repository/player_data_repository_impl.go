package repository

import (
	"context"
	"database/sql"
	"errors"
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

var _ repository.PlayerPlayHistoryQueryService = (*playerDataRepository)(nil)

type playerPlayHistoryRow struct {
	Title      *string   `db:"title"`
	Difficulty *string   `db:"difficulty"`
	Score      int       `db:"score"`
	ClearLamp  *string   `db:"clear_lamp"`
	ComboLamp  *string   `db:"combo_lamp"`
	FullChain  *string   `db:"full_chain"`
	PlayedAt   time.Time `db:"played_at"`
}

// FindByPlayerID は譜面に紐付いたタイトルとランプ名を一括で解決してプレイ履歴を返します。
func (r *playerDataRepository) FindByPlayerID(ctx context.Context, exec repository.Executor, playerID int) ([]entity.PlayerPlayHistory, error) {
	const query = `
SELECT
    CASE
        WHEN ppl.chart_id IS NOT NULL THEN standard_song.title
        WHEN ppl.worldsend_chart_id IS NOT NULL THEN worldsend_song.title
    END AS title,
    CASE WHEN ppl.difficulty_id IS NULL THEN 'WORLDSEND' ELSE d.name END AS difficulty,
    ppl.score,
    NULLIF(cl.name, 'NONE') AS clear_lamp,
    NULLIF(co.name, 'NONE') AS combo_lamp,
    NULLIF(fc.name, 'NONE') AS full_chain,
    ppl.played_at
FROM player_play_logs ppl
LEFT JOIN charts c ON c.id = ppl.chart_id
LEFT JOIN songs standard_song ON standard_song.id = c.song_id
LEFT JOIN worldsend_charts wc ON wc.id = ppl.worldsend_chart_id
LEFT JOIN songs worldsend_song ON worldsend_song.id = wc.song_id
LEFT JOIN difficulties d ON d.id = ppl.difficulty_id
INNER JOIN clear_lamp_types cl ON cl.id = ppl.clear_lamp_id
INNER JOIN combo_lamp_types co ON co.id = ppl.combo_lamp_id
INNER JOIN full_chain_types fc ON fc.id = ppl.full_chain_id
WHERE ppl.player_id = ?
ORDER BY ppl.play_order
`
	var rows []playerPlayHistoryRow
	if err := exec.SelectContext(ctx, &rows, query, playerID); err != nil {
		return nil, fmt.Errorf("%w: find player play history: %w", repository.ErrRepositoryOperationFailed, err)
	}
	history := make([]entity.PlayerPlayHistory, 0, len(rows))
	for _, row := range rows {
		history = append(history, entity.PlayerPlayHistory{
			Title: row.Title, Difficulty: row.Difficulty, Score: row.Score,
			ClearLamp: row.ClearLamp, ComboLamp: row.ComboLamp, FullChain: row.FullChain,
			PlayedAt: row.PlayedAt,
		})
	}
	return history, nil
}

// NewPlayerDataRepository は PlayerDataRepository の実装を生成します。
func NewPlayerDataRepository(db *sqlx.DB) repository.PlayerDataRepository {
	return &playerDataRepository{db: db}
}

// NewPlayerPlayHistoryQueryService は保存済みプレイ履歴の読み取りサービスを生成します。
func NewPlayerPlayHistoryQueryService(db *sqlx.DB) repository.PlayerPlayHistoryQueryService {
	return &playerDataRepository{db: db}
}

// FindPlayLogChartCandidates はジャケット名を一括照合し、通常譜面とWORLD'S END譜面の候補を返します。
// プレイ履歴は最大50件に制限されるため、照合は1回のIN句で実行します。
func (r *playerDataRepository) FindPlayLogChartCandidates(ctx context.Context, exec repository.Executor, jackets []string) ([]entity.PlayLogChartCandidate, error) {
	if exec == nil {
		return nil, fmt.Errorf("FindPlayLogChartCandidates requires a non-nil executor")
	}
	if len(jackets) == 0 {
		return nil, nil
	}

	query, args, err := sqlx.In(`
		WITH filtered_songs AS (
			SELECT id, title, jacket
			FROM songs
			WHERE jacket IN (?)
		)
		SELECT c.id, s.title, s.jacket, c.difficulty_id
		FROM filtered_songs s
		INNER JOIN charts c ON c.song_id = s.id
		UNION ALL
		SELECT wc.id, s.title, s.jacket, 0
		FROM filtered_songs s
		INNER JOIN worldsend_charts wc ON wc.song_id = s.id
	`, jackets)
	if err != nil {
		return nil, fmt.Errorf("failed to build play log chart candidate query: %w", err)
	}

	var rows []models.PlayLogChartCandidateModel
	if err := exec.SelectContext(ctx, &rows, exec.Rebind(query), args...); err != nil {
		return nil, fmt.Errorf("failed to find play log chart candidates: %w", err)
	}
	candidates := make([]entity.PlayLogChartCandidate, 0, len(rows))
	for _, row := range rows {
		candidates = append(candidates, row.ToEntity())
	}
	return candidates, nil
}

// ReplacePlayLogs はプレイヤーのプレイ履歴を受信順で置き換えます。
// 削除と登録に同じExecutorを使うことで、スコア登録トランザクションから履歴だけが残る状態を防ぎます。
func (r *playerDataRepository) ReplacePlayLogs(ctx context.Context, exec repository.Executor, playerID int, logs []entity.PlayerPlayLog) error {
	if exec == nil {
		return fmt.Errorf("ReplacePlayLogs requires a non-nil executor")
	}
	if _, err := exec.ExecContext(ctx, `DELETE FROM player_play_logs WHERE player_id = ?`, playerID); err != nil {
		return fmt.Errorf("failed to delete player play logs: %w", err)
	}
	rows := make([]models.PlayerPlayLogModel, len(logs))
	for i, log := range logs {
		rows[i] = models.FromPlayerPlayLogEntity(playerID, i, log)
	}
	return bulkExec(ctx, exec, rows, playerPlayLogInsertQuery, "player play logs")
}

// LoadMasterData はプレイヤーデータ登録に必要なマスタ情報を取得します。
// 登録と同じExecutorで読み取り、別の接続を確保しません。
func (r *playerDataRepository) LoadMasterData(ctx context.Context, executor repository.Executor, officialIdxList []string) (*repository.PlayerDataMaster, error) {
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
			SELECT id, display_id, title, reading, artist, genre_id, bpm, released_at, official_idx, jacket, is_deleted
			FROM songs
			WHERE official_idx IN (?)
		`,
		"songs",
		func(batch []string) []any { return []any{batch} },
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
		func(batch []int) []any { return []any{batch} },
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
		func(batch []int) []any { return []any{batch} },
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
	if err := r.saveFullRecords(ctx, exec, input.FullRecordsToInsert); err != nil {
		return err
	}
	if err := r.saveWorldsendRecords(ctx, exec, input.WorldsendRecordsToInsert); err != nil {
		return err
	}
	if err := updateFullRecords(ctx, exec, input.FullRecordsToUpdate); err != nil {
		return err
	}
	if err := updateWorldsendRecords(ctx, exec, input.WorldsendRecordsToUpdate); err != nil {
		return err
	}
	return updateRecordSlots(ctx, exec, input.FullRecordSlotsToUpdate)
}

// FindRankedPlayerRecordChartIDs は入力外の枠解除対象も同じトランザクションで取得します。
func (r *playerDataRepository) FindRankedPlayerRecordChartIDs(ctx context.Context, exec repository.Executor, playerID int) ([]int, error) {
	if exec == nil {
		return nil, fmt.Errorf("FindRankedPlayerRecordChartIDs requires a non-nil executor")
	}
	var ids []int
	if err := exec.SelectContext(ctx, &ids, `SELECT chart_id FROM player_records
		WHERE player_id = ? AND (slot_id <> (SELECT id FROM slots WHERE name = 'none') OR slot_order IS NOT NULL)
		ORDER BY chart_id`+r.recordLockClause(), playerID); err != nil {
		return nil, fmt.Errorf("failed to find ranked records: %w", err)
	}
	return ids, nil
}

// SaveLatestUpdate は収集日時ごとに登録結果を保存し、最新5件を残します。
func (r *playerDataRepository) SaveLatestUpdate(ctx context.Context, exec repository.Executor, update *entity.PlayerLatestUpdate) error {
	if exec == nil {
		return fmt.Errorf("SaveLatestUpdate requires a non-nil executor: must be called within a transaction")
	}
	if update == nil {
		return fmt.Errorf("SaveLatestUpdate requires a non-nil update")
	}

	model := models.FromPlayerLatestUpdateEntity(update)
	var existingHash string
	err := exec.GetContext(ctx, &existingHash, `
		SELECT body_hash FROM player_latest_updates
		WHERE player_id = ? AND source_updated_at = ?
	`, update.PlayerID(), update.SourceUpdatedAt())
	if err == nil {
		if existingHash != update.BodyHash() {
			return entity.ErrConflictingPlayerDataBody
		}
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("failed to check player update history: %w", err)
	}
	insertQuery := `
		INSERT INTO player_latest_updates (
			player_id, schema_version, result_gzip, source_updated_at, imported_at, body_hash
		) VALUES (
			:player_id, :schema_version, :result_gzip, :source_updated_at, :imported_at, :body_hash
		)
	`
	if _, err := exec.NamedExecContext(ctx, insertQuery, model); err != nil {
		return fmt.Errorf("failed to save player latest update: %w", err)
	}
	var cutoff time.Time
	err = exec.GetContext(ctx, &cutoff, `
		SELECT source_updated_at FROM player_latest_updates
		WHERE player_id = ?
		ORDER BY source_updated_at DESC
		LIMIT 1 OFFSET ?
	`, update.PlayerID(), info.PlayerUpdateHistoryLimit-1)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to find player update history cutoff: %w", err)
	}
	_, err = exec.ExecContext(ctx, `
		DELETE FROM player_latest_updates
		WHERE player_id = ? AND source_updated_at < ?
	`, update.PlayerID(), cutoff)
	if err != nil {
		return fmt.Errorf("failed to trim player update history: %w", err)
	}
	return nil
}

// FindRecentUpdatesByPlayerID は保存済みの最新5件を収集日時順に取得します。
func (r *playerDataRepository) FindRecentUpdatesByPlayerID(ctx context.Context, playerID int) ([]*entity.PlayerLatestUpdate, error) {
	var rows []models.PlayerLatestUpdateModel
	err := r.db.SelectContext(ctx, &rows, `
		SELECT player_id, schema_version, result_gzip, source_updated_at, imported_at, body_hash
		FROM player_latest_updates
		WHERE player_id = ?
		ORDER BY source_updated_at DESC
		LIMIT ?
	`, playerID, info.PlayerUpdateHistoryLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to find player update history: %w", err)
	}
	updates := make([]*entity.PlayerLatestUpdate, 0, len(rows))
	for i := range rows {
		update, err := rows[i].ToEntity()
		if err != nil {
			return nil, fmt.Errorf("failed to restore player update history: %w", err)
		}
		updates = append(updates, update)
	}
	return updates, nil
}

// FindLatestUpdateByPlayerID はプレイヤーIDに対応する最新データ登録結果を取得します。
func (r *playerDataRepository) FindLatestUpdateByPlayerID(ctx context.Context, playerID int) (*entity.PlayerLatestUpdate, error) {
	return r.findLatestUpdateByPlayerID(ctx, r.db, playerID, false)
}

// FindLatestUpdateByPlayerIDForUpdate はプレイヤーIDに対応する最新データ登録結果をロックして取得します。
func (r *playerDataRepository) FindLatestUpdateByPlayerIDForUpdate(ctx context.Context, exec repository.Executor, playerID int) (*entity.PlayerLatestUpdate, error) {
	if exec == nil {
		return nil, fmt.Errorf("FindLatestUpdateByPlayerIDForUpdate requires a non-nil executor")
	}
	return r.findLatestUpdateByPlayerID(ctx, exec, playerID, true)
}

func (r *playerDataRepository) findLatestUpdateByPlayerID(ctx context.Context, exec repository.Executor, playerID int, forUpdate bool) (*entity.PlayerLatestUpdate, error) {
	var model models.PlayerLatestUpdateModel
	query := `
		SELECT player_id, schema_version, result_gzip, source_updated_at, imported_at, body_hash
		FROM player_latest_updates
		WHERE player_id = ?
		ORDER BY source_updated_at DESC
		LIMIT 1
	`
	if forUpdate && r.db.DriverName() == "mysql" {
		query += " FOR UPDATE"
	}
	if err := exec.GetContext(ctx, &model, query, playerID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrPlayerLatestUpdateNotFound
		}
		return nil, fmt.Errorf("failed to find player latest update: %w", err)
	}
	update, err := model.ToEntity()
	if err != nil {
		return nil, fmt.Errorf("failed to restore player latest update: %w", err)
	}
	return update, nil
}

// FindPlayerRecordStatesByChartIDs は保存前の通常譜面レコード状態を譜面IDキーで取得します。
func (r *playerDataRepository) FindPlayerRecordStatesByChartIDs(ctx context.Context, exec repository.Executor, playerID int, chartIDs []int) (map[int]repository.PlayerRecordState, error) {
	if exec == nil {
		return nil, fmt.Errorf("FindPlayerRecordStatesByChartIDs requires a non-nil executor")
	}
	lockClause := r.recordLockClause()
	rows, err := selectModelsInChunks[int, playerDataRecordRow](ctx, exec, chartIDs, `
		SELECT
			player_id, chart_id, score, clear_lamp_id, combo_lamp_id,
			full_chain_id, slot_id, slot_order, updated_at
		FROM player_records
		WHERE player_id = ?
		  AND chart_id IN (?)
	`+lockClause, "player record states", func(batch []int) []any { return []any{playerID, batch} })
	if err != nil {
		return nil, err
	}

	states := make(map[int]repository.PlayerRecordState, len(rows))
	for _, row := range rows {
		states[row.ChartID] = repository.PlayerRecordState{
			Score:       row.Score,
			ClearLampID: row.ClearLampID,
			ComboLampID: row.ComboLampID,
			FullChainID: row.FullChainID,
			SlotID:      row.SlotID,
			SlotOrder:   row.SlotOrder,
			UpdatedAt:   row.UpdatedAt,
		}
	}
	return states, nil
}

// FindWorldsendRecordStatesByChartIDs は保存前のWORLD'S ENDレコード状態を譜面IDキーで取得します。
func (r *playerDataRepository) FindWorldsendRecordStatesByChartIDs(ctx context.Context, exec repository.Executor, playerID int, worldsendChartIDs []int) (map[int]repository.WorldsendRecordState, error) {
	if exec == nil {
		return nil, fmt.Errorf("FindWorldsendRecordStatesByChartIDs requires a non-nil executor")
	}
	lockClause := r.recordLockClause()
	rows, err := selectModelsInChunks[int, playerDataWorldsendRecordRow](ctx, exec, worldsendChartIDs, `
		SELECT
			player_id, worldsend_chart_id, score, clear_lamp_id, combo_lamp_id,
			full_chain_id, updated_at
		FROM player_worldsend_records
		WHERE player_id = ?
		  AND worldsend_chart_id IN (?)
	`+lockClause, "worldsend record states", func(batch []int) []any { return []any{playerID, batch} })
	if err != nil {
		return nil, err
	}

	states := make(map[int]repository.WorldsendRecordState, len(rows))
	for _, row := range rows {
		states[row.WorldsendChartID] = repository.WorldsendRecordState{
			Score:       row.Score,
			ClearLampID: row.ClearLampID,
			ComboLampID: row.ComboLampID,
			FullChainID: row.FullChainID,
			UpdatedAt:   row.UpdatedAt,
		}
	}
	return states, nil
}

func (r *playerDataRepository) recordLockClause() string {
	if r.db.DriverName() == "mysql" {
		return " FOR UPDATE"
	}
	return ""
}

// GetOverpowerTargetStats はOVER POWER割合計算の分母となる対象楽曲の最大OP合計を取得します。
// songs/chartsの読み取りのみのためトランザクション外で呼び出せます。
func (r *playerDataRepository) GetOverpowerTargetStats(ctx context.Context, filter repository.OverpowerTargetFilter) (*repository.OverpowerTargetStats, error) {
	return r.getOverpowerTargetStats(ctx, r.db, filter)
}

// GetOverpowerTargetStatsWithExecutor は指定されたExecutorでOVER POWER割合計算の分母となる対象楽曲の最大OP合計を取得します。
func (r *playerDataRepository) GetOverpowerTargetStatsWithExecutor(ctx context.Context, exec repository.Executor, filter repository.OverpowerTargetFilter) (*repository.OverpowerTargetStats, error) {
	if exec == nil {
		exec = r.db
	}
	return r.getOverpowerTargetStats(ctx, exec, filter)
}

func (r *playerDataRepository) getOverpowerTargetStats(ctx context.Context, executor repository.Executor, filter repository.OverpowerTargetFilter) (*repository.OverpowerTargetStats, error) {

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
		// 未解禁の扱いは service.PlayerLockedSongSet と同じ規則にする。
		// pls_song（通常譜面群の未解禁）がある楽曲は楽曲ごと分母から外し、
		// pls_ultima（ULTIMAだけの未解禁）がある楽曲は ULTIMA の定数だけを NULL にして通常譜面の最大定数を残す。
		// 規則を変えるときは PlayerLockedSongSet と揃える。
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

// selectModelsInChunks はIN句に渡すitemsをチャンク分割してSELECTします。
// buildArgsで各チャンクの引数順を明示することで、IN (?) の後にも別のプレースホルダを置けるようにします。
func selectModelsInChunks[T any, M any](ctx context.Context, exec repository.Executor, items []T, query string, queryName string, buildArgs func([]T) []any) ([]M, error) {
	if len(items) == 0 {
		return nil, nil
	}

	results := make([]M, 0, len(items))
	batchSize := info.BulkInsertChunkSize
	for i := 0; i < len(items); i += batchSize {
		end := min(i+batchSize, len(items))
		batch := items[i:end]
		args := buildArgs(batch)
		batchQuery, batchArgs, err := sqlx.In(query, args...)
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

const fullRecordInsertQuery = `
 INSERT INTO player_records (player_id, chart_id, score, clear_lamp_id, combo_lamp_id, full_chain_id, slot_id, slot_order, updated_at)
 VALUES (:player_id, :chart_id, :score, :clear_lamp_id, :combo_lamp_id, :full_chain_id, :slot_id, :slot_order, :updated_at)
`

const worldsendRecordInsertQuery = `
 INSERT INTO player_worldsend_records (player_id, worldsend_chart_id, score, clear_lamp_id, combo_lamp_id, full_chain_id, updated_at)
 VALUES (:player_id, :worldsend_chart_id, :score, :clear_lamp_id, :combo_lamp_id, :full_chain_id, :updated_at)
`

const playerPlayLogInsertQuery = `
 INSERT INTO player_play_logs (
  player_id, play_order, chart_id, worldsend_chart_id, jacket_img_id,
  difficulty_id, score, clear_lamp_id, combo_lamp_id, full_chain_id, played_at
 ) VALUES (
  :player_id, :play_order, :chart_id, :worldsend_chart_id, :jacket_img_id,
  :difficulty_id, :score, :clear_lamp_id, :combo_lamp_id, :full_chain_id, :played_at
 )
`

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

	return bulkExec(ctx, exec, rows, fullRecordInsertQuery, "player records")
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

	return bulkExec(ctx, exec, rows, worldsendRecordInsertQuery, "worldsend records")
}

func bulkExec[T any](ctx context.Context, exec repository.Executor, rows []T, query string, recordType string) error {
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
