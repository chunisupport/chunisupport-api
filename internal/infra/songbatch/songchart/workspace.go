package songchart

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"log/slog"
	"strings"
	"time"

	apirepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	domainservice "github.com/chunisupport/chunisupport-api/internal/domain/service"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/chunisupport/chunisupport-api/internal/utils"

	"github.com/jmoiron/sqlx"

	_ "modernc.org/sqlite"
)

const (
	defaultWorkspaceDSN = "file:chuni_ws?mode=memory&cache=shared&_pragma=foreign_keys(ON)"
)

//go:embed schema.sql
var workspaceSchema string

// Config はワークスペース生成時のオプションです。
type Config struct {
	DSN string
}

// SyncOptions は MySQL への同期挙動を制御します。
type SyncOptions struct {
	MajorUpdate            bool
	FillMissingReleaseDate bool // 特定フラグ: データソース・MySQL両方に日付がないbrand new楽曲へ実行日(JST)をreleased_at補完
}

// SongChartWorkspace は songs/charts を扱う SQLite ワークスペースを表します。
type SongChartWorkspace struct {
	db *sqlx.DB
}

// NewSongChartWorkspace は SQLite ワークスペースを構築します。
func NewSongChartWorkspace(ctx context.Context, cfg Config) (*SongChartWorkspace, error) {
	dsn := cfg.DSN
	if dsn == "" {
		dsn = defaultWorkspaceDSN
	}

	db, err := sqlx.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite workspace: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to ping sqlite workspace: %w", err)
	}

	ws := &SongChartWorkspace{db: db}
	if err := ws.bootstrapSchema(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	return ws, nil
}

// Close はワークスペース接続を閉じます。
func (w *SongChartWorkspace) Close() error {
	if w == nil || w.db == nil {
		return nil
	}
	return w.db.Close()
}

// DB は内部の *sqlx.DB を返します。
func (w *SongChartWorkspace) DB() *sqlx.DB {
	return w.db
}

func (w *SongChartWorkspace) bootstrapSchema(ctx context.Context) error {
	for _, stmt := range parseSQLStatements(workspaceSchema) {
		if _, err := w.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("failed to initialize workspace schema: %w", err)
		}
	}

	return nil
}

// SyncToMySQL はワークスペース内容を MySQL songs/charts に反映します。
func (w *SongChartWorkspace) SyncToMySQL(ctx context.Context, mysql apirepo.Executor, opts SyncOptions) error {
	slog.Info("Starting workspace synchronization to MySQL")

	wsSongs, err := w.loadWorkspaceSongs(ctx)
	if err != nil {
		return err
	}
	wsCharts, err := w.loadWorkspaceCharts(ctx)
	if err != nil {
		return err
	}
	wsWorldsendCharts, err := w.loadWorkspaceWorldsendCharts(ctx)
	if err != nil {
		return err
	}

	mysqlSongs, err := loadMySQLSongs(ctx, mysql)
	if err != nil {
		return err
	}
	nameFolderIDs, err := loadMySQLNameFolderIDs(ctx, mysql)
	if err != nil {
		return err
	}
	mysqlCharts, err := loadMySQLCharts(ctx, mysql)
	if err != nil {
		return err
	}
	mysqlWorldsendCharts, err := loadMySQLWorldsendCharts(ctx, mysql)
	if err != nil {
		return err
	}

	// 特定フラグ有効時のための実行日(JST)取得（1回のみ）
	// 条件: データソースから一切日付が来ておらず、かつこの実行開始時点でMySQLに楽曲自体が存在しない場合のみ使用
	var executionDateJST string
	if opts.FillMissingReleaseDate {
		jst := time.FixedZone("JST", 9*60*60)
		executionDateJST = time.Now().In(jst).Format("2006-01-02")
	}

	officialSeen := make(map[string]struct{}, len(wsSongs))
	songsToInsert := make([]songInsertRecord, 0, len(wsSongs))
	songsToUpdate := make([]songUpdateRecord, 0, len(wsSongs))

	for _, song := range wsSongs {
		if song.OfficialIdx == "" {
			slog.Warn("Skipping workspace song without official_idx", "song_id", song.ID)
			continue
		}

		releasedAt := song.ReleasedAt
		// フラグ有効 かつ released_at が空 かつ MySQLにこのofficial_idxが存在しない（= brand new 楽曲）場合のみ実行日で補完
		// これにより「楽曲はあるが日付がnull」の既存曲は一切触らない（ユーザ要件厳守）
		if executionDateJST != "" {
			if !releasedAt.Valid || strings.TrimSpace(releasedAt.String) == "" {
				if _, existsInMySQL := mysqlSongs[song.OfficialIdx]; !existsInMySQL {
					releasedAt = sql.NullString{String: executionDateJST, Valid: true}
					slog.Info("特定フラグにより未補完の新規楽曲へ実行日(JST)をreleased_atとして補完",
						"official_idx", song.OfficialIdx, "date", executionDateJST)
				}
			}
		}

		officialSeen[song.OfficialIdx] = struct{}{}
		nameFolderID, err := resolveSongNameFolderID(song.Title, song.Reading, nameFolderIDs)
		if err != nil {
			return fmt.Errorf("failed to resolve name folder for official_idx %s: %w", song.OfficialIdx, err)
		}
		rec := songInsertRecord{
			DisplayID:      song.DisplayID,
			Title:          song.Title,
			WikiPageTitle:  song.WikiPageTitle,
			Reading:        song.Reading,
			Artist:         song.Artist,
			GenreID:        song.GenreID,
			BPM:            song.BPM,
			ReleasedAt:     releasedAt,
			OfficialIdx:    song.OfficialIdx,
			Jacket:         song.Jacket,
			IsWorldsend:    song.IsWorldsend,
			IsNew:          song.IsNew,
			UnlockRequired: song.UnlockRequired,
			NameFolderID:   nameFolderID,
		}
		if existing, exists := mysqlSongs[song.OfficialIdx]; exists {
			songsToUpdate = append(songsToUpdate, songUpdateRecord{
				ID:     existing.ID,
				record: rec,
			})
		} else {
			songsToInsert = append(songsToInsert, rec)
		}
	}

	slog.Info("Prepared MySQL song synchronization",
		"insert_count", len(songsToInsert),
		"update_count", len(songsToUpdate))

	if err := bulkInsertMySQLSongs(ctx, mysql, songsToInsert, info.SongBatchBulkInsertChunkSize); err != nil {
		return err
	}
	if err := bulkUpdateMySQLSongs(ctx, mysql, songsToUpdate, info.SongBatchBulkInsertChunkSize); err != nil {
		return err
	}
	if err := syncMySQLSongNameFolderIDs(ctx, mysql, nameFolderIDs, info.SongBatchBulkInsertChunkSize); err != nil {
		return fmt.Errorf("failed to synchronize mysql song name folders: %w", err)
	}

	mysqlSongs, err = loadMySQLSongs(ctx, mysql)
	if err != nil {
		return err
	}

	songIDMap := make(map[int]int, len(wsSongs))
	for _, song := range wsSongs {
		if song.OfficialIdx == "" {
			continue
		}
		mysqlSong, ok := mysqlSongs[song.OfficialIdx]
		if !ok {
			return fmt.Errorf("failed to resolve mysql song id for official_idx %s", song.OfficialIdx)
		}
		songIDMap[song.ID] = mysqlSong.ID
	}

	var chartsToInsert []chartInsertRecord
	var chartsToUpdate []chartUpdateRecord

	for _, chart := range wsCharts {
		mysqlSongID, ok := songIDMap[chart.SongID]
		if !ok {
			return fmt.Errorf("workspace chart references unknown song_id %d", chart.SongID)
		}

		key := chartKey(mysqlSongID, chart.DifficultyID)
		existing, exists := mysqlCharts[key]

		finalConst, finalUnknown, finalNotes, finalNotesDesigner, action := resolveChartUpdate(existing, exists, chart, opts)

		switch action {
		case actionInsert:
			chartsToInsert = append(chartsToInsert, chartInsertRecord{
				SongID:        mysqlSongID,
				DifficultyID:  chart.DifficultyID,
				Const:         finalConst,
				TargetUnknown: finalUnknown,
				Notes:         finalNotes,
				NotesDesigner: finalNotesDesigner,
			})
		case actionUpdate:
			chartsToUpdate = append(chartsToUpdate, chartUpdateRecord{
				SongID:        mysqlSongID,
				DifficultyID:  chart.DifficultyID,
				Const:         finalConst,
				TargetUnknown: finalUnknown,
				Notes:         finalNotes,
				NotesDesigner: finalNotesDesigner,
			})
		}
	}

	if err := bulkInsertMySQLCharts(ctx, mysql, chartsToInsert, info.SongBatchBulkInsertChunkSize); err != nil {
		return err
	}
	if err := bulkUpdateMySQLCharts(ctx, mysql, chartsToUpdate, info.SongBatchBulkInsertChunkSize); err != nil {
		return err
	}

	var worldsendChartsToInsert []worldsendChartInsertRecord
	var worldsendChartsToUpdate []worldsendChartUpdateRecord

	for _, weChart := range wsWorldsendCharts {
		mysqlSongID, ok := songIDMap[weChart.SongID]
		if !ok {
			return fmt.Errorf("workspace worldsend_chart references unknown song_id %d", weChart.SongID)
		}

		_, exists := mysqlWorldsendCharts[mysqlSongID]
		if !exists {
			worldsendChartsToInsert = append(worldsendChartsToInsert, worldsendChartInsertRecord{
				SongID:        mysqlSongID,
				LevelStar:     weChart.LevelStar,
				Attribute:     weChart.Attribute,
				Notes:         weChart.Notes,
				NotesDesigner: weChart.NotesDesigner,
			})
		} else {
			worldsendChartsToUpdate = append(worldsendChartsToUpdate, worldsendChartUpdateRecord{
				SongID:        mysqlSongID,
				LevelStar:     weChart.LevelStar,
				Attribute:     weChart.Attribute,
				Notes:         weChart.Notes,
				NotesDesigner: weChart.NotesDesigner,
			})
		}
	}

	if err := bulkInsertMySQLWorldsendCharts(ctx, mysql, worldsendChartsToInsert, info.SongBatchBulkInsertChunkSize); err != nil {
		return err
	}
	if err := bulkUpdateMySQLWorldsendCharts(ctx, mysql, worldsendChartsToUpdate, info.SongBatchBulkInsertChunkSize); err != nil {
		return err
	}

	slog.Info("Workspace synchronization to MySQL completed",
		"songs", len(wsSongs),
		"charts", len(wsCharts),
		"worldsend_charts", len(wsWorldsendCharts))
	return nil
}

type workspaceSong struct {
	ID             int            `db:"id"`
	DisplayID      string         `db:"display_id"`
	Title          string         `db:"title"`
	WikiPageTitle  sql.NullString `db:"wiki_page_title"`
	Reading        sql.NullString `db:"reading"`
	Artist         string         `db:"artist"`
	GenreID        sql.NullInt64  `db:"genre_id"`
	BPM            sql.NullInt64  `db:"bpm"`
	ReleasedAt     sql.NullString `db:"released_at"`
	OfficialIdx    string         `db:"official_idx"`
	Jacket         sql.NullString `db:"jacket"`
	IsWorldsend    int            `db:"is_worldsend"`
	IsNew          int            `db:"is_new"`
	UnlockRequired sql.NullInt64  `db:"unlock_required"`
	IsDeleted      int            `db:"is_deleted"`
}

type workspaceChart struct {
	ID             int            `db:"id"`
	SongID         int            `db:"song_id"`
	DifficultyID   int            `db:"difficulty_id"`
	Const          float64        `db:"const"`
	IsConstUnknown bool           `db:"is_const_unknown"`
	Notes          sql.NullInt64  `db:"notes"`
	NotesDesigner  sql.NullString `db:"notes_designer"`
}

type workspaceWorldsendChart struct {
	ID            int            `db:"id"`
	SongID        int            `db:"song_id"`
	LevelStar     sql.NullInt64  `db:"level_star"`
	Attribute     sql.NullString `db:"attribute"`
	Notes         sql.NullInt64  `db:"notes"`
	NotesDesigner sql.NullString `db:"notes_designer"`
}

type mysqlSong struct {
	ID          int
	OfficialIdx string
}

type mysqlChart struct {
	SongID         int
	DifficultyID   int
	Const          float64
	IsConstUnknown bool
	Notes          sql.NullInt64
	NotesDesigner  sql.NullString
}

type mysqlWorldsendChart struct {
	SongID        int
	LevelStar     sql.NullInt64
	Attribute     sql.NullString
	Notes         sql.NullInt64
	NotesDesigner sql.NullString
}

func (w *SongChartWorkspace) loadWorkspaceSongs(ctx context.Context) ([]workspaceSong, error) {
	var songs []workspaceSong
	if err := w.db.SelectContext(ctx, &songs, `SELECT id, display_id, title, wiki_page_title, reading, artist, genre_id, bpm, released_at, official_idx, jacket, is_worldsend, is_new, unlock_required, is_deleted FROM songs ORDER BY id`); err != nil {
		return nil, fmt.Errorf("failed to load workspace songs: %w", err)
	}

	return songs, nil
}

func (w *SongChartWorkspace) loadWorkspaceCharts(ctx context.Context) ([]workspaceChart, error) {
	var charts []workspaceChart
	if err := w.db.SelectContext(ctx, &charts, `SELECT id, song_id, difficulty_id, const, is_const_unknown, notes, notes_designer FROM charts ORDER BY id`); err != nil {
		return nil, fmt.Errorf("failed to load workspace charts: %w", err)
	}

	return charts, nil
}

func (w *SongChartWorkspace) loadWorkspaceWorldsendCharts(ctx context.Context) ([]workspaceWorldsendChart, error) {
	var charts []workspaceWorldsendChart
	if err := w.db.SelectContext(ctx, &charts, `SELECT id, song_id, level_star, attribute, notes, notes_designer FROM worldsend_charts ORDER BY id`); err != nil {
		return nil, fmt.Errorf("failed to load workspace worldsend_charts: %w", err)
	}

	return charts, nil
}

func loadMySQLSongs(ctx context.Context, mysql apirepo.Executor) (map[string]mysqlSong, error) {
	rows, err := mysql.QueryContext(ctx, `SELECT id, official_idx FROM songs WHERE official_idx IS NOT NULL`)
	if err != nil {
		return nil, fmt.Errorf("failed to load mysql songs: %w", err)
	}
	defer rows.Close()

	result := make(map[string]mysqlSong)
	for rows.Next() {
		var rec mysqlSong
		if err := rows.Scan(&rec.ID, &rec.OfficialIdx); err != nil {
			return nil, fmt.Errorf("failed to scan mysql song: %w", err)
		}
		result[rec.OfficialIdx] = rec
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mysql songs iteration error: %w", err)
	}
	return result, nil
}

func loadMySQLNameFolderIDs(ctx context.Context, mysql apirepo.Executor) (map[string]int, error) {
	rows, err := mysql.QueryContext(ctx, `SELECT id, code FROM name_folders`)
	if err != nil {
		return nil, fmt.Errorf("failed to load mysql name folders: %w", err)
	}
	defer rows.Close()

	result := make(map[string]int)
	for rows.Next() {
		var id int
		var code string
		if err := rows.Scan(&id, &code); err != nil {
			return nil, fmt.Errorf("failed to scan mysql name folder: %w", err)
		}
		result[code] = id
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mysql name folders iteration error: %w", err)
	}
	return result, nil
}

func resolveSongNameFolderID(title string, songReading sql.NullString, idsByCode map[string]int) (int, error) {
	var reading *string
	if songReading.Valid {
		reading = &songReading.String
	}

	code := domainservice.ResolveNameFolderCode(title, reading)
	id, ok := idsByCode[code]
	if !ok {
		return 0, fmt.Errorf("name folder code %q was not found", code)
	}
	return id, nil
}

type mysqlSongNameFolder struct {
	ID           int
	Title        string
	Reading      sql.NullString
	NameFolderID int
}

type songNameFolderUpdate struct {
	ID           int
	NameFolderID int
}

func syncMySQLSongNameFolderIDs(ctx context.Context, mysql apirepo.Executor, idsByCode map[string]int, chunkSize int) error {
	query := "SELECT id, title, reading, name_folder_id FROM songs ORDER BY id"
	// 読みの同時更新を古いスナップショットの所属で上書きしないよう、MySQLでは最新行をロックして読みます。
	if driver, ok := mysql.(interface{ DriverName() string }); ok && driver.DriverName() == "mysql" {
		query += " FOR UPDATE"
	}
	rows, err := mysql.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to load mysql song name folder data: %w", err)
	}
	defer rows.Close()

	var updates []songNameFolderUpdate
	for rows.Next() {
		var song mysqlSongNameFolder
		if err := rows.Scan(&song.ID, &song.Title, &song.Reading, &song.NameFolderID); err != nil {
			return fmt.Errorf("failed to scan mysql song name folder data: %w", err)
		}
		nameFolderID, err := resolveSongNameFolderID(song.Title, song.Reading, idsByCode)
		if err != nil {
			return fmt.Errorf("failed to resolve name folder for mysql song id %d: %w", song.ID, err)
		}
		if song.NameFolderID != nameFolderID {
			updates = append(updates, songNameFolderUpdate{ID: song.ID, NameFolderID: nameFolderID})
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("mysql song name folder iteration error: %w", err)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("failed to close mysql song name folder rows: %w", err)
	}
	return bulkUpdateMySQLSongNameFolderIDs(ctx, mysql, updates, chunkSize)
}

func buildBulkUpdateMySQLSongNameFolderIDsSQL(n int) string {
	var sb strings.Builder
	sb.WriteString("UPDATE songs SET name_folder_id = CASE id\n")
	for range n {
		sb.WriteString("WHEN ? THEN ?\n")
	}
	sb.WriteString("ELSE name_folder_id END WHERE id IN (")
	for i := range n {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteByte('?')
	}
	sb.WriteByte(')')
	return sb.String()
}

func bulkUpdateMySQLSongNameFolderIDs(ctx context.Context, mysql apirepo.Executor, records []songNameFolderUpdate, chunkSize int) error {
	if len(records) == 0 {
		return nil
	}
	if chunkSize <= 0 {
		chunkSize = len(records)
	}

	for start := 0; start < len(records); start += chunkSize {
		end := min(start+chunkSize, len(records))
		chunk := records[start:end]
		args := make([]any, 0, len(chunk)*3)
		for _, rec := range chunk {
			args = append(args, rec.ID, rec.NameFolderID)
		}
		for _, rec := range chunk {
			args = append(args, rec.ID)
		}
		if _, err := mysql.ExecContext(ctx, buildBulkUpdateMySQLSongNameFolderIDsSQL(len(chunk)), args...); err != nil {
			return fmt.Errorf("failed to bulk update mysql song name folders (%d-%d): %w", start, end, err)
		}
	}
	return nil
}

func loadMySQLCharts(ctx context.Context, mysql apirepo.Executor) (map[string]mysqlChart, error) {
	rows, err := mysql.QueryContext(ctx, `SELECT song_id, difficulty_id, const, is_const_unknown, notes, notes_designer FROM charts`)
	if err != nil {
		return nil, fmt.Errorf("failed to load mysql charts: %w", err)
	}
	defer rows.Close()

	result := make(map[string]mysqlChart)
	for rows.Next() {
		var rec mysqlChart
		if err := rows.Scan(&rec.SongID, &rec.DifficultyID, &rec.Const, &rec.IsConstUnknown, &rec.Notes, &rec.NotesDesigner); err != nil {
			return nil, fmt.Errorf("failed to scan mysql chart: %w", err)
		}
		result[chartKey(rec.SongID, rec.DifficultyID)] = rec
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mysql charts iteration error: %w", err)
	}
	return result, nil
}

func loadMySQLWorldsendCharts(ctx context.Context, mysql apirepo.Executor) (map[int]mysqlWorldsendChart, error) {
	rows, err := mysql.QueryContext(ctx, `SELECT song_id, level_star, attribute, notes, notes_designer FROM worldsend_charts`)
	if err != nil {
		return nil, fmt.Errorf("failed to load mysql worldsend_charts: %w", err)
	}
	defer rows.Close()

	result := make(map[int]mysqlWorldsendChart)
	for rows.Next() {
		var rec mysqlWorldsendChart
		if err := rows.Scan(&rec.SongID, &rec.LevelStar, &rec.Attribute, &rec.Notes, &rec.NotesDesigner); err != nil {
			return nil, fmt.Errorf("failed to scan mysql worldsend_chart: %w", err)
		}
		result[rec.SongID] = rec
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mysql worldsend_charts iteration error: %w", err)
	}
	return result, nil
}

type chartUpdateRecord struct {
	SongID        int
	DifficultyID  int
	Const         float64
	TargetUnknown bool
	Notes         sql.NullInt64
	NotesDesigner sql.NullString
}

// buildBulkUpdateChartsSQL はバルク更新用の CASE 式を含む SQL 文を生成します。
// text/template を使わず strings.Builder で構築することで、
// テンプレートエンジン経由のインジェクションリスクを排除します。
// データ値はすべてプレースホルダー(?) で渡されます。
func buildBulkUpdateChartsSQL(n int) string {
	var sb strings.Builder

	sb.WriteString("UPDATE charts\nSET\n")

	writeCaseBlock := func(column, elseExpr string) {
		sb.WriteString("\t")
		sb.WriteString(column)
		sb.WriteString(" = CASE\n")
		for range n {
			sb.WriteString("\t\tWHEN song_id = ? AND difficulty_id = ? THEN ?\n")
		}
		sb.WriteString("\t\tELSE ")
		sb.WriteString(elseExpr)
		sb.WriteString("\n\tEND")
	}

	writeCaseBlock("const", "const")
	sb.WriteString(",\n")
	writeCaseBlock("is_const_unknown", "is_const_unknown")
	sb.WriteString(",\n")
	writeCaseBlock("notes", "notes")
	sb.WriteString(",\n")
	writeCaseBlock("notes_designer", "notes_designer")

	sb.WriteString("\nWHERE (song_id, difficulty_id) IN (")
	for i := range n {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString("(?,?)")
	}
	sb.WriteByte(')')

	return sb.String()
}

func bulkUpdateMySQLCharts(ctx context.Context, mysql apirepo.Executor, records []chartUpdateRecord, chunkSize int) error {
	if len(records) == 0 {
		return nil
	}

	if chunkSize <= 0 {
		chunkSize = len(records)
	}

	for start := 0; start < len(records); start += chunkSize {
		end := min(start+chunkSize, len(records))

		chunk := records[start:end]
		query := buildBulkUpdateChartsSQL(len(chunk))

		var args []any
		for _, rec := range chunk {
			args = append(args, rec.SongID, rec.DifficultyID, rec.Const)
		}
		for _, rec := range chunk {
			args = append(args, rec.SongID, rec.DifficultyID, utils.BoolToInt(rec.TargetUnknown))
		}
		for _, rec := range chunk {
			args = append(args, rec.SongID, rec.DifficultyID, nullableInt(rec.Notes))
		}
		for _, rec := range chunk {
			args = append(args, rec.SongID, rec.DifficultyID, nullableString(rec.NotesDesigner))
		}
		for _, rec := range chunk {
			args = append(args, rec.SongID, rec.DifficultyID)
		}

		if _, err := mysql.ExecContext(ctx, query, args...); err != nil {
			return fmt.Errorf("failed to bulk update charts (%d-%d): %w", start, end, err)
		}
	}

	return nil
}

type worldsendChartUpdateRecord struct {
	SongID        int
	LevelStar     sql.NullInt64
	Attribute     sql.NullString
	Notes         sql.NullInt64
	NotesDesigner sql.NullString
}

// buildBulkUpdateWorldsendChartsSQL はバルク更新用の CASE 式を含む SQL 文を生成します。
// text/template を使わず strings.Builder で構築することで、
// テンプレートエンジン経由のインジェクションリスクを排除します。
// データ値はすべてプレースホルダー(?) で渡されます。
func buildBulkUpdateWorldsendChartsSQL(n int) string {
	var sb strings.Builder

	sb.WriteString("UPDATE worldsend_charts\nSET\n")

	writeCaseBlock := func(column, coalesceExpr string) {
		sb.WriteString("\t")
		sb.WriteString(column)
		sb.WriteString(" = CASE\n")
		for range n {
			sb.WriteString("\t\tWHEN song_id = ? THEN ")
			sb.WriteString(coalesceExpr)
			sb.WriteByte('\n')
		}
		sb.WriteString("\t\tELSE ")
		sb.WriteString(column)
		sb.WriteString("\n\tEND")
	}

	writeCaseBlock("level_star", "COALESCE(?, level_star)")
	sb.WriteString(",\n")
	writeCaseBlock("attribute", "COALESCE(?, attribute)")
	sb.WriteString(",\n")
	writeCaseBlock("notes", "COALESCE(?, notes)")
	sb.WriteString(",\n")
	writeCaseBlock("notes_designer", "COALESCE(notes_designer, ?)")

	sb.WriteString("\nWHERE song_id IN (")
	for i := range n {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteByte('?')
	}
	sb.WriteByte(')')

	return sb.String()
}

func bulkUpdateMySQLWorldsendCharts(ctx context.Context, mysql apirepo.Executor, records []worldsendChartUpdateRecord, chunkSize int) error {
	if len(records) == 0 {
		return nil
	}

	if chunkSize <= 0 {
		chunkSize = len(records)
	}

	for start := 0; start < len(records); start += chunkSize {
		end := min(start+chunkSize, len(records))

		chunk := records[start:end]
		query := buildBulkUpdateWorldsendChartsSQL(len(chunk))

		var args []any
		for _, rec := range chunk {
			args = append(args, rec.SongID, nullableInt(rec.LevelStar))
		}
		for _, rec := range chunk {
			args = append(args, rec.SongID, nullableString(rec.Attribute))
		}
		for _, rec := range chunk {
			args = append(args, rec.SongID, nullableInt(rec.Notes))
		}
		for _, rec := range chunk {
			args = append(args, rec.SongID, nullableString(rec.NotesDesigner))
		}
		for _, rec := range chunk {
			args = append(args, rec.SongID)
		}

		if _, err := mysql.ExecContext(ctx, query, args...); err != nil {
			return fmt.Errorf("failed to bulk update worldsend_charts (%d-%d): %w", start, end, err)
		}
	}

	return nil
}

type syncAction int

const (
	actionSkip syncAction = iota
	actionInsert
	actionUpdate
)

const (
	majorUpdateLevelThreshold = 10.0
	majorUpdateLevelRange     = 0.5
)

func resolveChartUpdate(existing mysqlChart, exists bool, chart workspaceChart, opts SyncOptions) (float64, bool, sql.NullInt64, sql.NullString, syncAction) {
	if !exists {
		tConst := chart.Const
		tUnknown := chart.IsConstUnknown
		if opts.MajorUpdate {
			if chart.Const < majorUpdateLevelThreshold {
				tUnknown = false
			} else {
				tUnknown = true
			}
		}
		return tConst, tUnknown, chart.Notes, chart.NotesDesigner, actionInsert
	}

	finalConst := existing.Const
	finalUnknown := existing.IsConstUnknown
	finalNotes := existing.Notes
	finalNotesDesigner := existing.NotesDesigner
	shouldUpdate := false

	if opts.MajorUpdate {
		if chart.Const < majorUpdateLevelThreshold {
			finalConst = chart.Const
			finalUnknown = false
			if existing.Const != finalConst || existing.IsConstUnknown != finalUnknown {
				shouldUpdate = true
			}
		} else {
			finalUnknown = true
			rangeStart := chart.Const
			rangeEnd := chart.Const + majorUpdateLevelRange
			if existing.Const < rangeStart || existing.Const >= rangeEnd {
				finalConst = chart.Const
				shouldUpdate = true
			}

			if existing.IsConstUnknown != finalUnknown {
				shouldUpdate = true
			}
		}

		if !shouldUpdate {
			if !existing.Notes.Valid && chart.Notes.Valid {
				shouldUpdate = true
			}
			if !existing.NotesDesigner.Valid && chart.NotesDesigner.Valid {
				shouldUpdate = true
			}
		}
	} else {

		if existing.IsConstUnknown && !chart.IsConstUnknown {
			finalConst = chart.Const
			finalUnknown = false
			shouldUpdate = true
		} else {
			// 既知 -> 不明 への変更はブロック
			// それ以外（既知 -> 既知、不明 -> 不明）で、定数が一致し、かつノーツが新しい場合のみ更新
			isBlocked := !existing.IsConstUnknown && chart.IsConstUnknown

			if !isBlocked && existing.Const == chart.Const {
				if !existing.Notes.Valid && chart.Notes.Valid {
					shouldUpdate = true
				}
				if !existing.NotesDesigner.Valid && chart.NotesDesigner.Valid {
					shouldUpdate = true
				}
			}
		}
	}

	// ノーツ数の更新制御: 既存値が存在する場合、NULL値での上書きを防止
	if chart.Notes.Valid {
		finalNotes = chart.Notes
	}
	// 既存値がない場合のみ、新しい値（NULLでも）を受け入れる
	if !existing.Notes.Valid && chart.Notes.Valid {
		finalNotes = chart.Notes
	}
	if !existing.NotesDesigner.Valid && chart.NotesDesigner.Valid {
		finalNotesDesigner = chart.NotesDesigner
	}

	if shouldUpdate {
		return finalConst, finalUnknown, finalNotes, finalNotesDesigner, actionUpdate
	}
	return finalConst, finalUnknown, finalNotes, finalNotesDesigner, actionSkip
}

func chartKey(songID, difficultyID int) string {
	return fmt.Sprintf("%d:%d", songID, difficultyID)
}

func nullableInt(value sql.NullInt64) any {
	if value.Valid {
		return value.Int64
	}
	return nil
}

func nullableString(value sql.NullString) any {
	if value.Valid {
		return value.String
	}
	return nil
}

type songInsertRecord struct {
	DisplayID      string
	Title          string
	WikiPageTitle  sql.NullString
	Reading        sql.NullString
	Artist         string
	GenreID        sql.NullInt64
	BPM            sql.NullInt64
	ReleasedAt     sql.NullString
	OfficialIdx    string
	Jacket         sql.NullString
	IsWorldsend    int
	IsNew          int
	UnlockRequired sql.NullInt64
	NameFolderID   int
}

type songUpdateRecord struct {
	ID     int
	record songInsertRecord
}

const songInsertColumnCount = 15

// buildBulkUpdateSongsSQL は楽曲バルク更新用の CASE 式を含む SQL 文を生成します。
func buildBulkUpdateSongsSQL(n int) string {
	var sb strings.Builder

	sb.WriteString("UPDATE songs\nSET\n")

	writeDisplayIDBlock := func() {
		sb.WriteString("\tdisplay_id = CASE\n")
		for range n {
			sb.WriteString("\t\tWHEN id = ? THEN CASE WHEN display_id IS NULL OR display_id = '' THEN ? ELSE display_id END\n")
		}
		sb.WriteString("\t\tELSE display_id\n\tEND")
	}

	writeCoalesceBlock := func(column string) {
		sb.WriteString("\t")
		sb.WriteString(column)
		sb.WriteString(" = CASE\n")
		for range n {
			sb.WriteString("\t\tWHEN id = ? THEN COALESCE(?, ")
			sb.WriteString(column)
			sb.WriteString(")\n")
		}
		sb.WriteString("\t\tELSE ")
		sb.WriteString(column)
		sb.WriteString("\n\tEND")
	}

	writeDirectBlock := func(column string) {
		sb.WriteString("\t")
		sb.WriteString(column)
		sb.WriteString(" = CASE\n")
		for range n {
			sb.WriteString("\t\tWHEN id = ? THEN ?\n")
		}
		sb.WriteString("\t\tELSE ")
		sb.WriteString(column)
		sb.WriteString("\n\tEND")
	}

	// 既存値を優先し、MySQL側がNULLの場合のみ補完するブロック。
	// 手動で修正した値がデータソースの値で上書きされ続けないようにするために使います。
	writeKeepExistingBlock := func(column string) {
		sb.WriteString("\t")
		sb.WriteString(column)
		sb.WriteString(" = CASE\n")
		for range n {
			sb.WriteString("\t\tWHEN id = ? THEN COALESCE(")
			sb.WriteString(column)
			sb.WriteString(", ?)\n")
		}
		sb.WriteString("\t\tELSE ")
		sb.WriteString(column)
		sb.WriteString("\n\tEND")
	}

	// 現在値が真(1)の場合のみデータソースの値を反映するブロック。
	// 要解禁フラグは解禁済みの楽曲が要解禁に戻ることがないため、データソースの誤りで
	// 偽→真に書き換わらないよう、真→偽の変更だけを受け付けます。
	writeClearOnlyBlock := func(column string) {
		sb.WriteString("\t")
		sb.WriteString(column)
		sb.WriteString(" = CASE\n")
		for range n {
			sb.WriteString("\t\tWHEN id = ? AND ")
			sb.WriteString(column)
			sb.WriteString(" = 1 THEN COALESCE(?, ")
			sb.WriteString(column)
			sb.WriteString(")\n")
		}
		sb.WriteString("\t\tELSE ")
		sb.WriteString(column)
		sb.WriteString("\n\tEND")
	}

	writeDisplayIDBlock()
	sb.WriteString(",\n")
	writeCoalesceBlock("title")
	sb.WriteString(",\n")
	writeKeepExistingBlock("wiki_page_title")
	sb.WriteString(",\n")
	writeDirectBlock("reading")
	sb.WriteString(",\n")
	writeDirectBlock("name_folder_id")
	sb.WriteString(",\n")
	writeCoalesceBlock("artist")
	sb.WriteString(",\n")
	writeCoalesceBlock("genre_id")
	sb.WriteString(",\n")
	writeCoalesceBlock("bpm")
	sb.WriteString(",\n")
	writeKeepExistingBlock("released_at")
	sb.WriteString(",\n")
	writeCoalesceBlock("jacket")
	sb.WriteString(",\n")
	writeDirectBlock("is_worldsend")
	sb.WriteString(",\n")
	writeDirectBlock("is_new")
	sb.WriteString(",\n")
	writeClearOnlyBlock("unlock_required")

	sb.WriteString("\nWHERE id IN (")
	for i := range n {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteByte('?')
	}
	sb.WriteByte(')')

	return sb.String()
}

// buildBulkInsertSongsSQL は新規楽曲だけを登録するSQL文を生成します。
// 競合時に既存楽曲を誤更新しないよう、UPSERTは使用しません。
func buildBulkInsertSongsSQL(n int) string {
	const queryPrefix = `
INSERT INTO songs (
	display_id, title, wiki_page_title, reading, artist, genre_id, bpm, released_at, official_idx, jacket, is_worldsend, is_new, unlock_required, is_deleted, name_folder_id
) VALUES `

	values := make([]string, n)
	for i := range n {
		values[i] = "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, COALESCE(?, 0), ?, ?)"
	}

	return queryPrefix + strings.Join(values, ",")
}

func bulkInsertMySQLSongs(ctx context.Context, mysql apirepo.Executor, records []songInsertRecord, chunkSize int) error {
	if len(records) == 0 {
		return nil
	}

	if chunkSize <= 0 {
		chunkSize = len(records)
	}

	for start := 0; start < len(records); start += chunkSize {
		end := min(start+chunkSize, len(records))

		chunk := records[start:end]
		args := make([]any, 0, len(chunk)*songInsertColumnCount)
		for _, rec := range chunk {
			args = append(args,
				rec.DisplayID,
				rec.Title,
				nullableString(rec.WikiPageTitle),
				nullableString(rec.Reading),
				rec.Artist,
				nullableInt(rec.GenreID),
				nullableInt(rec.BPM),
				nullableString(rec.ReleasedAt),
				rec.OfficialIdx,
				nullableString(rec.Jacket),
				rec.IsWorldsend,
				rec.IsNew,
				nullableInt(rec.UnlockRequired),
				0,
				rec.NameFolderID,
			)
		}

		query := buildBulkInsertSongsSQL(len(chunk))
		if _, err := mysql.ExecContext(ctx, query, args...); err != nil {
			return fmt.Errorf("failed to bulk insert songs (%d-%d): %w", start, end, err)
		}
	}

	return nil
}

func bulkUpdateMySQLSongs(ctx context.Context, mysql apirepo.Executor, records []songUpdateRecord, chunkSize int) error {
	if len(records) == 0 {
		return nil
	}

	if chunkSize <= 0 {
		chunkSize = len(records)
	}

	for start := 0; start < len(records); start += chunkSize {
		end := min(start+chunkSize, len(records))

		chunk := records[start:end]
		query := buildBulkUpdateSongsSQL(len(chunk))
		args := make([]any, 0, len(chunk)*27)

		for _, rec := range chunk {
			args = append(args, rec.ID, rec.record.DisplayID)
		}
		for _, rec := range chunk {
			args = append(args, rec.ID, rec.record.Title)
		}
		for _, rec := range chunk {
			args = append(args, rec.ID, nullableString(rec.record.WikiPageTitle))
		}
		for _, rec := range chunk {
			args = append(args, rec.ID, nullableString(rec.record.Reading))
		}
		for _, rec := range chunk {
			args = append(args, rec.ID, rec.record.NameFolderID)
		}
		for _, rec := range chunk {
			args = append(args, rec.ID, rec.record.Artist)
		}
		for _, rec := range chunk {
			args = append(args, rec.ID, nullableInt(rec.record.GenreID))
		}
		for _, rec := range chunk {
			args = append(args, rec.ID, nullableInt(rec.record.BPM))
		}
		for _, rec := range chunk {
			args = append(args, rec.ID, nullableString(rec.record.ReleasedAt))
		}
		for _, rec := range chunk {
			args = append(args, rec.ID, nullableString(rec.record.Jacket))
		}
		for _, rec := range chunk {
			args = append(args, rec.ID, rec.record.IsWorldsend)
		}
		for _, rec := range chunk {
			args = append(args, rec.ID, rec.record.IsNew)
		}
		for _, rec := range chunk {
			args = append(args, rec.ID, nullableInt(rec.record.UnlockRequired))
		}
		for _, rec := range chunk {
			args = append(args, rec.ID)
		}

		if _, err := mysql.ExecContext(ctx, query, args...); err != nil {
			return fmt.Errorf("failed to bulk update songs (%d-%d): %w", start, end, err)
		}
	}

	return nil
}

type chartInsertRecord struct {
	SongID        int
	DifficultyID  int
	Const         float64
	TargetUnknown bool
	Notes         sql.NullInt64
	NotesDesigner sql.NullString
}

const chartInsertColumnCount = 6

// buildBulkInsertChartsSQL は新規譜面だけを登録するSQL文を生成します。
// 既存譜面は呼び出し元で UPDATE に振り分け済みのため、UPSERTは使用しません。
// InnoDB は ON DUPLICATE KEY UPDATE が既存行を更新する場合も AUTO_INCREMENT を消費するためです。
func buildBulkInsertChartsSQL(n int) string {
	const queryPrefix = "INSERT INTO charts (song_id, difficulty_id, const, is_const_unknown, notes, notes_designer) VALUES "

	values := make([]string, n)
	for i := range n {
		values[i] = "(?, ?, ?, ?, ?, ?)"
	}

	return queryPrefix + strings.Join(values, ",")
}

func bulkInsertMySQLCharts(ctx context.Context, mysql apirepo.Executor, records []chartInsertRecord, chunkSize int) error {
	if len(records) == 0 {
		return nil
	}

	if chunkSize <= 0 {
		chunkSize = len(records)
	}

	for start := 0; start < len(records); start += chunkSize {
		end := min(start+chunkSize, len(records))

		chunk := records[start:end]
		args := make([]any, 0, len(chunk)*chartInsertColumnCount)
		for _, rec := range chunk {
			args = append(args,
				rec.SongID,
				rec.DifficultyID,
				rec.Const,
				utils.BoolToInt(rec.TargetUnknown),
				nullableInt(rec.Notes),
				nullableString(rec.NotesDesigner),
			)
		}

		query := buildBulkInsertChartsSQL(len(chunk))
		if _, err := mysql.ExecContext(ctx, query, args...); err != nil {
			return fmt.Errorf("failed to bulk insert charts (%d-%d): %w", start, end, err)
		}
	}

	return nil
}

type worldsendChartInsertRecord struct {
	SongID        int
	LevelStar     sql.NullInt64
	Attribute     sql.NullString
	Notes         sql.NullInt64
	NotesDesigner sql.NullString
}

const worldsendChartInsertColumnCount = 5

// buildBulkInsertWorldsendChartsSQL は新規WORLD'S END譜面だけを登録するSQL文を生成します。
// 既存譜面は呼び出し元で UPDATE に振り分け済みのため、AUTO_INCREMENT を消費するUPSERTは使用しません。
func buildBulkInsertWorldsendChartsSQL(n int) string {
	const queryPrefix = "INSERT INTO worldsend_charts (song_id, level_star, attribute, notes, notes_designer) VALUES "

	values := make([]string, n)
	for i := range n {
		values[i] = "(?, ?, ?, ?, ?)"
	}

	return queryPrefix + strings.Join(values, ",")
}

func bulkInsertMySQLWorldsendCharts(ctx context.Context, mysql apirepo.Executor, records []worldsendChartInsertRecord, chunkSize int) error {
	if len(records) == 0 {
		return nil
	}

	if chunkSize <= 0 {
		chunkSize = len(records)
	}

	for start := 0; start < len(records); start += chunkSize {
		end := min(start+chunkSize, len(records))

		chunk := records[start:end]
		args := make([]any, 0, len(chunk)*worldsendChartInsertColumnCount)
		for _, rec := range chunk {
			args = append(args,
				rec.SongID,
				nullableInt(rec.LevelStar),
				nullableString(rec.Attribute),
				nullableInt(rec.Notes),
				nullableString(rec.NotesDesigner),
			)
		}

		query := buildBulkInsertWorldsendChartsSQL(len(chunk))
		if _, err := mysql.ExecContext(ctx, query, args...); err != nil {
			return fmt.Errorf("failed to bulk insert worldsend_charts (%d-%d): %w", start, end, err)
		}
	}

	return nil
}

func parseSQLStatements(sql string) []string {
	parts := strings.Split(sql, ";")
	statements := make([]string, 0, len(parts))
	for _, part := range parts {
		stmt := strings.TrimSpace(part)
		if stmt == "" {
			continue
		}
		statements = append(statements, stmt)
	}
	return statements
}
