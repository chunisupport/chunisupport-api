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
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/notes"
	"github.com/jmoiron/sqlx"
)

// songRepository は SongRepository の実装です。
type songRepository struct {
	db *sqlx.DB
}

// NewSongRepository は SongRepository の実装を生成します。
func NewSongRepository(db *sqlx.DB) repository.SongRepository {
	return &songRepository{db: db}
}

// songRow はDBから取得する楽曲データの行を表します。
type songRow struct {
	ID             int        `db:"id"`
	DisplayID      string     `db:"display_id"`
	Title          string     `db:"title"`
	WikiPageTitle  *string    `db:"wiki_page_title"`
	Reading        *string    `db:"reading"`
	Artist         string     `db:"artist"`
	GenreID        *int       `db:"genre_id"`
	BPM            *int       `db:"bpm"`
	ReleasedAt     *time.Time `db:"released_at"`
	OfficialIdx    string     `db:"official_idx"`
	Jacket         *string    `db:"jacket"`
	IsWorldsend    bool       `db:"is_worldsend"`
	IsNew          bool       `db:"is_new"`
	UnlockRequired bool       `db:"unlock_required"`
	IsDeleted      bool       `db:"is_deleted"`
	UpdatedAt      *time.Time `db:"updated_at"`
}

// chartRow はDBから取得する譜面データの行を表します。
type chartRow struct {
	ID             int        `db:"id"`
	SongID         int        `db:"song_id"`
	DifficultyID   int        `db:"difficulty_id"`
	Const          float64    `db:"const"`
	IsConstUnknown bool       `db:"is_const_unknown"`
	Notes          *int       `db:"notes"`
	NotesDesigner  *string    `db:"notes_designer"`
	UpdatedAt      *time.Time `db:"updated_at"`
}

// FindAllExcludingWorldsend はWORLD'S END以外の全楽曲を取得します。
// includeDeletedがfalseの場合、削除済み楽曲は除外されます。
// N+1問題を回避するため、楽曲と譜面を別々のクエリで取得し、メモリ上で結合します。
func (r *songRepository) FindAllExcludingWorldsend(ctx context.Context, exec repository.Executor, includeDeleted bool) ([]*entity.Song, error) {
	songsQuery := `
		SELECT id, display_id, title, wiki_page_title, reading, artist, genre_id, bpm, released_at, official_idx, jacket, is_worldsend, is_new, unlock_required, is_deleted, updated_at
		FROM songs
		WHERE is_worldsend = 0`
	if !includeDeleted {
		songsQuery += ` AND is_deleted = 0`
	}
	songsQuery += `
		ORDER BY id
	`
	var songRows []songRow
	if err := exec.SelectContext(ctx, &songRows, songsQuery); err != nil {
		return nil, err
	}

	if len(songRows) == 0 {
		return []*entity.Song{}, nil
	}

	songIDs := make([]int, len(songRows))
	songIDToIndex := make(map[int]int, len(songRows))
	for i, s := range songRows {
		songIDs[i] = s.ID
		songIDToIndex[s.ID] = i
	}

	// 3. 該当する楽曲の譜面を一括取得（N+1問題回避）
	chartsQuery, args, err := sqlx.In(`
		SELECT id, song_id, difficulty_id, const, is_const_unknown, notes, notes_designer, updated_at
		FROM charts
		WHERE song_id IN (?)
		ORDER BY song_id, difficulty_id
	`, songIDs)
	if err != nil {
		return nil, err
	}
	chartsQuery = exec.Rebind(chartsQuery)

	var chartRows []chartRow
	if err := exec.SelectContext(ctx, &chartRows, chartsQuery, args...); err != nil {
		return nil, err
	}

	results := make([]*entity.Song, len(songRows))
	for i, sr := range songRows {
		song := r.toSongEntity(&sr)
		song.Charts = []*entity.Chart{}
		results[i] = song
	}

	for _, cr := range chartRows {
		idx, ok := songIDToIndex[cr.SongID]
		if !ok {
			continue
		}
		chart, err := r.toChartEntity(&cr)
		if err != nil {
			return nil, err
		}
		results[idx].Charts = append(results[idx].Charts, chart)
	}

	for _, song := range results {
		service.ApplyAggregation(song)
	}

	return results, nil
}

// FindLatestUpdatedAt は songs, charts, worldsend_charts の updated_at の最大値を返します。
func (r *songRepository) FindLatestUpdatedAt(ctx context.Context, exec repository.Executor) (*time.Time, error) {
	var updatedAtRaw sql.NullString
	if err := exec.GetContext(ctx, &updatedAtRaw, `
		SELECT MAX(updated_at)
		FROM (
			SELECT MAX(updated_at) AS updated_at FROM songs
			UNION ALL
			SELECT MAX(updated_at) AS updated_at FROM charts
			UNION ALL
			SELECT MAX(updated_at) AS updated_at FROM worldsend_charts
		) latest_updated_at
	`); err != nil {
		return nil, err
	}

	if !updatedAtRaw.Valid || updatedAtRaw.String == "" {
		return nil, nil
	}

	parsedUpdatedAt, err := parseLatestUpdatedAt(updatedAtRaw.String)
	if err != nil {
		return nil, err
	}

	return &parsedUpdatedAt, nil
}

func parseLatestUpdatedAt(value string) (time.Time, error) {
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05 -0700 MST",
		time.DateTime,
		time.DateOnly,
	}

	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed, nil
		}
	}

	return time.Time{}, fmt.Errorf("failed to parse latest updated_at: %s", value)
}

func (r *songRepository) toSongEntity(row *songRow) *entity.Song {
	song := entity.NewSong()
	song.ID = row.ID
	song.DisplayID = row.DisplayID
	song.Title = row.Title
	song.WikiPageTitle = row.WikiPageTitle
	song.Reading = row.Reading
	song.Artist = row.Artist
	song.GenreID = row.GenreID
	song.BPM = row.BPM
	song.ReleasedAt = row.ReleasedAt
	song.OfficialIdx = row.OfficialIdx
	song.Jacket = row.Jacket
	song.IsWorldsend = row.IsWorldsend
	song.IsNew = row.IsNew
	song.UnlockRequired = row.UnlockRequired
	song.IsDeleted = row.IsDeleted
	song.UpdatedAt = row.UpdatedAt
	return song
}

func (r *songRepository) toChartEntity(row *chartRow) (*entity.Chart, error) {
	constVal, err := chartconstant.NewChartConstant(row.Const)
	if err != nil {
		return nil, fmt.Errorf("invalid chart constant for chart %d: %w", row.ID, err)
	}

	var notesVal *notes.Notes
	if row.Notes != nil {
		n, err := notes.NewNotes(*row.Notes)
		if err != nil {
			return nil, fmt.Errorf("invalid notes for chart %d: %w", row.ID, err)
		}
		notesVal = &n
	}

	return &entity.Chart{
		ID:             row.ID,
		SongID:         row.SongID,
		DifficultyID:   row.DifficultyID,
		Const:          constVal,
		IsConstUnknown: row.IsConstUnknown,
		Notes:          notesVal,
		NotesDesigner:  row.NotesDesigner,
		UpdatedAt:      row.UpdatedAt,
	}, nil
}

// FindByDisplayIDs は指定されたDisplayIDのリストに該当する通常楽曲（WORLD'S END除く）を取得します。
// 各楽曲には関連する譜面情報が含まれます。
// N+1問題を回避するため、楽曲と譜面を別々のクエリで取得し、メモリ上で結合します。
func (r *songRepository) FindByDisplayIDs(ctx context.Context, exec repository.Executor, displayIDs []string) ([]*entity.Song, error) {
	if len(displayIDs) == 0 {
		return []*entity.Song{}, nil
	}

	query, args, err := sqlx.In(`
		SELECT id, display_id, title, wiki_page_title, reading, artist, genre_id, bpm, released_at, official_idx, jacket, is_worldsend, is_new, unlock_required, is_deleted, updated_at
		FROM songs
		WHERE display_id IN (?)
		  AND is_worldsend = 0
	`, displayIDs)
	if err != nil {
		return nil, err
	}
	query = exec.Rebind(query)

	var songRows []songRow
	if err := exec.SelectContext(ctx, &songRows, query, args...); err != nil {
		return nil, err
	}

	if len(songRows) == 0 {
		return []*entity.Song{}, nil
	}

	songIDs := make([]int, len(songRows))
	songIDToIndex := make(map[int]int, len(songRows))
	for i, s := range songRows {
		songIDs[i] = s.ID
		songIDToIndex[s.ID] = i
	}

	// 3. 該当する楽曲の譜面を一括取得（N+1問題回避）
	chartsQuery, chartArgs, err := sqlx.In(`
		SELECT id, song_id, difficulty_id, const, is_const_unknown, notes, notes_designer, updated_at
		FROM charts
		WHERE song_id IN (?)
		ORDER BY song_id, difficulty_id
	`, songIDs)
	if err != nil {
		return nil, err
	}
	chartsQuery = exec.Rebind(chartsQuery)

	var chartRows []chartRow
	if err := exec.SelectContext(ctx, &chartRows, chartsQuery, chartArgs...); err != nil {
		return nil, err
	}

	songs := make([]*entity.Song, len(songRows))
	for i, sr := range songRows {
		song := r.toSongEntity(&sr)
		song.Charts = []*entity.Chart{}
		songs[i] = song
	}

	for _, cr := range chartRows {
		idx, ok := songIDToIndex[cr.SongID]
		if !ok {
			continue
		}
		chart, err := r.toChartEntity(&cr)
		if err != nil {
			return nil, err
		}
		songs[idx].Charts = append(songs[idx].Charts, chart)
	}

	for _, song := range songs {
		service.ApplyAggregation(song)
	}

	return songs, nil
}

// FindByDisplayID は指定されたDisplayIDの通常楽曲（WORLD'S END除く）を取得します。
// 削除済み楽曲も取得します。
func (r *songRepository) FindByDisplayID(ctx context.Context, exec repository.Executor, displayID string) (*entity.Song, error) {
	return r.findByIdentifier(ctx, exec, "display_id", displayID)
}

// FindByOfficialIdx は指定された公式IDの通常楽曲を取得します。
// 削除済み楽曲も取得します。
func (r *songRepository) FindByOfficialIdx(ctx context.Context, exec repository.Executor, officialIdx string) (*entity.Song, error) {
	return r.findByIdentifier(ctx, exec, "official_idx", officialIdx)
}

// FindByOfficialIdxForChange は指定された公式IDの通常楽曲を変更用にロックして取得します。
func (r *songRepository) FindByOfficialIdxForChange(ctx context.Context, exec repository.Executor, officialIdx string) (*entity.Song, error) {
	return r.findByIdentifierForUpdate(ctx, exec, "official_idx", officialIdx)
}

// FindByDisplayIDForChange は指定されたDisplayIDの通常楽曲を変更用にロックして取得します。
func (r *songRepository) FindByDisplayIDForChange(ctx context.Context, exec repository.Executor, displayID string) (*entity.Song, error) {
	return r.findByIdentifierForUpdate(ctx, exec, "display_id", displayID)
}

// findByIdentifierForUpdate は許可済みの識別カラムで通常楽曲集約をロックして取得します。
func (r *songRepository) findByIdentifierForUpdate(ctx context.Context, exec repository.Executor, column, value string) (*entity.Song, error) {
	if column != "display_id" && column != "official_idx" {
		return nil, fmt.Errorf("unsupported song identifier column: %s", column)
	}

	songQuery := fmt.Sprintf(`
		SELECT id, display_id, title, wiki_page_title, reading, artist, genre_id, bpm, released_at, official_idx, jacket, is_worldsend, is_new, unlock_required, is_deleted, updated_at
		FROM songs
		WHERE %s = ? AND is_worldsend = 0
		FOR UPDATE
	`, column)
	var songRow songRow
	if err := exec.GetContext(ctx, &songRow, songQuery, value); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrSongNotFound
		}
		return nil, err
	}

	song := r.toSongEntity(&songRow)

	chartsQuery := `
		SELECT id, song_id, difficulty_id, const, is_const_unknown, notes, notes_designer, updated_at
		FROM charts
		WHERE song_id = ?
		ORDER BY difficulty_id
		FOR UPDATE
	`
	var chartRows []chartRow
	if err := exec.SelectContext(ctx, &chartRows, chartsQuery, songRow.ID); err != nil {
		return nil, err
	}

	charts := make([]*entity.Chart, len(chartRows))
	for i, cr := range chartRows {
		chart, err := r.toChartEntity(&cr)
		if err != nil {
			return nil, err
		}
		charts[i] = chart
	}

	song.Charts = charts
	service.ApplyAggregation(song)

	return song, nil
}

// findByIdentifier は許可済みの識別カラムで通常楽曲集約を取得します。
func (r *songRepository) findByIdentifier(ctx context.Context, exec repository.Executor, column, value string) (*entity.Song, error) {
	if column != "display_id" && column != "official_idx" {
		return nil, fmt.Errorf("unsupported song identifier column: %s", column)
	}

	songQuery := fmt.Sprintf(`
		SELECT id, display_id, title, wiki_page_title, reading, artist, genre_id, bpm, released_at, official_idx, jacket, is_worldsend, is_new, unlock_required, is_deleted, updated_at
		FROM songs
		WHERE %s = ? AND is_worldsend = 0
	`, column)
	var songRow songRow
	if err := exec.GetContext(ctx, &songRow, songQuery, value); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrSongNotFound
		}
		return nil, err
	}

	song := r.toSongEntity(&songRow)

	chartsQuery := `
		SELECT id, song_id, difficulty_id, const, is_const_unknown, notes, notes_designer, updated_at
		FROM charts
		WHERE song_id = ?
		ORDER BY difficulty_id
	`
	var chartRows []chartRow
	if err := exec.SelectContext(ctx, &chartRows, chartsQuery, songRow.ID); err != nil {
		return nil, err
	}

	charts := make([]*entity.Chart, len(chartRows))
	for i, cr := range chartRows {
		chart, err := r.toChartEntity(&cr)
		if err != nil {
			return nil, err
		}
		charts[i] = chart
	}

	song.Charts = charts

	service.ApplyAggregation(song)

	return song, nil
}

// Save は楽曲集約（楽曲本体と既存譜面）の現在の状態を永続化します。
// 譜面の追加・削除は行いません。
// 対象が存在しない場合は ErrSongNotFound を返します。
func (r *songRepository) Save(ctx context.Context, exec repository.Executor, song *entity.Song) error {
	query := `
		UPDATE songs
		SET display_id = ?, title = ?, wiki_page_title = ?, reading = ?, artist = ?, genre_id = ?, bpm = ?, released_at = ?, official_idx = ?, jacket = ?, is_worldsend = ?, is_new = ?, unlock_required = ?, is_deleted = ?
		WHERE id = ?
	`
	result, err := exec.ExecContext(
		ctx,
		query,
		song.DisplayID,
		song.Title,
		song.WikiPageTitle,
		song.Reading,
		song.Artist,
		song.GenreID,
		song.BPM,
		song.ReleasedAt,
		song.OfficialIdx,
		song.Jacket,
		song.IsWorldsend,
		song.IsNew,
		song.UnlockRequired,
		song.IsDeleted,
		song.ID,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		var exists bool
		if err := exec.GetContext(ctx, &exists, `SELECT EXISTS(SELECT 1 FROM songs WHERE id = ?)`, song.ID); err != nil {
			return err
		}
		if !exists {
			return repository.ErrSongNotFound
		}
	}

	return r.bulkUpdateCharts(ctx, exec, []*entity.Song{song}, map[string]int{
		song.DisplayID: song.ID,
	})
}

// UpdateSongs は楽曲および譜面情報を一括更新します。
// トランザクション管理はUseCase層（TransactionManager経由）で行います。
// PERF-008対策: N+1問題を解消するため、一括更新クエリを使用します。
func (r *songRepository) UpdateSongs(ctx context.Context, exec repository.Executor, updates []*repository.SongUpdate) error {
	if len(updates) == 0 {
		return nil
	}

	songs := make([]*entity.Song, 0, len(updates))
	for i, update := range updates {
		if update == nil || update.Song == nil {
			return fmt.Errorf("updates[%d].song is nil", i)
		}
		songs = append(songs, update.Song)
	}

	// 重複display_idは後続のCASE WHEN構築で先出現側が暗黙適用されるため事前に弾く
	displayIDs, err := collectUniqueDisplayIDs(songs)
	if err != nil {
		return err
	}

	existingSongs, err := r.FindByDisplayIDs(ctx, exec, displayIDs)
	if err != nil {
		return fmt.Errorf("failed to find songs by display IDs: %w", err)
	}

	displayIDToSongID := make(map[string]int, len(existingSongs))
	for _, song := range existingSongs {
		displayIDToSongID[song.DisplayID] = song.ID
	}

	for _, displayID := range displayIDs {
		if _, ok := displayIDToSongID[displayID]; !ok {
			return fmt.Errorf("song with display_id '%s' not found", displayID)
		}
	}

	chartExistsMap := make(map[string]bool)
	for _, song := range existingSongs {
		for _, chart := range song.Charts {
			key := fmt.Sprintf("%d-%d", chart.SongID, chart.DifficultyID)
			chartExistsMap[key] = true
		}
	}

	for _, song := range songs {
		songID := displayIDToSongID[song.DisplayID]
		for _, chart := range song.Charts {
			key := fmt.Sprintf("%d-%d", songID, chart.DifficultyID)
			if !chartExistsMap[key] {
				return fmt.Errorf("chart not found (song_id=%d, difficulty_id=%d)", songID, chart.DifficultyID)
			}
		}
	}

	if err := r.bulkUpdateSongs(ctx, exec, updates, displayIDToSongID); err != nil {
		return fmt.Errorf("failed to bulk update songs: %w", err)
	}

	if err := r.bulkUpdateCharts(ctx, exec, songs, displayIDToSongID); err != nil {
		return fmt.Errorf("failed to bulk update charts: %w", err)
	}

	return nil
}

// keepExistingUpdateExpr は一括更新時に、更新指定のある楽曲だけを書き換える代入式を組み立てます。
// 指定のない楽曲は SQL 上で既存値を維持します。
// 楽曲データ収集バッチによる並行更新を消さないよう、読み取った値の書き戻しは行いません。
func keepExistingUpdateExpr(column string, cases []string) string {
	if len(cases) == 0 {
		return column
	}
	return fmt.Sprintf("CASE %s ELSE %s END", strings.Join(cases, " "), column)
}

// bulkUpdateSongs は楽曲情報をCASE式で一括更新します。
func (r *songRepository) bulkUpdateSongs(ctx context.Context, exec repository.Executor, updates []*repository.SongUpdate, displayIDToSongID map[string]int) error {
	if len(updates) == 0 {
		return nil
	}

	songIDs := make([]int, 0, len(updates))
	for _, update := range updates {
		songIDs = append(songIDs, displayIDToSongID[update.Song.DisplayID])
	}

	// 注意: SQLの引数順序はCASE式の出現順（title→wiki_page_title→reading→artist→genre→...→IN句）であるため、
	// 各フィールドの引数を別々に蓄積し、最後に正しい順序で結合する必要がある
	var titleCases, wikiPageTitleCases, readingCases, artistCases, genreCases, bpmCases, releasedCases, jacketCases, isNewCases, unlockRequiredCases []string
	var titleArgs, wikiPageTitleArgs, readingArgs, artistArgs, genreArgs, bpmArgs, releasedArgs, jacketArgs, isNewArgs, unlockRequiredArgs []any

	for _, update := range updates {
		song := update.Song
		songID := displayIDToSongID[song.DisplayID]

		titleCases = append(titleCases, "WHEN id = ? THEN ?")
		titleArgs = append(titleArgs, songID, song.Title)

		if update.UpdateWikiPageTitle {
			wikiPageTitleCases = append(wikiPageTitleCases, "WHEN id = ? THEN ?")
			wikiPageTitleArgs = append(wikiPageTitleArgs, songID, song.WikiPageTitle)
		}

		readingCases = append(readingCases, "WHEN id = ? THEN ?")
		readingArgs = append(readingArgs, songID, song.Reading)

		artistCases = append(artistCases, "WHEN id = ? THEN ?")
		artistArgs = append(artistArgs, songID, song.Artist)

		genreCases = append(genreCases, "WHEN id = ? THEN ?")
		genreArgs = append(genreArgs, songID, song.GenreID)

		bpmCases = append(bpmCases, "WHEN id = ? THEN ?")
		bpmArgs = append(bpmArgs, songID, song.BPM)

		releasedCases = append(releasedCases, "WHEN id = ? THEN ?")
		releasedArgs = append(releasedArgs, songID, song.ReleasedAt)

		jacketCases = append(jacketCases, "WHEN id = ? THEN ?")
		jacketArgs = append(jacketArgs, songID, song.Jacket)

		isNewCases = append(isNewCases, "WHEN id = ? THEN ?")
		isNewArgs = append(isNewArgs, songID, song.IsNew)

		if update.UnlockRequired != nil {
			unlockRequiredCases = append(unlockRequiredCases, "WHEN id = ? THEN ?")
			unlockRequiredArgs = append(unlockRequiredArgs, songID, *update.UnlockRequired)
		}
	}

	args := make([]any, 0)
	args = append(args, titleArgs...)
	args = append(args, wikiPageTitleArgs...)
	args = append(args, readingArgs...)
	args = append(args, artistArgs...)
	args = append(args, genreArgs...)
	args = append(args, bpmArgs...)
	args = append(args, releasedArgs...)
	args = append(args, jacketArgs...)
	args = append(args, isNewArgs...)
	args = append(args, unlockRequiredArgs...)

	for _, id := range songIDs {
		args = append(args, id)
	}

	placeholders := make([]string, len(songIDs))
	for i := range placeholders {
		placeholders[i] = "?"
	}

	query := fmt.Sprintf(`
		UPDATE songs SET
			title = CASE %s END,
			wiki_page_title = %s,
			reading = CASE %s END,
			artist = CASE %s END,
			genre_id = CASE %s END,
			bpm = CASE %s END,
			released_at = CASE %s END,
			jacket = CASE %s END,
			is_new = CASE %s END,
			unlock_required = %s
		WHERE id IN (%s)
		  AND is_worldsend = 0
	`,
		strings.Join(titleCases, " "),
		keepExistingUpdateExpr("wiki_page_title", wikiPageTitleCases),
		strings.Join(readingCases, " "),
		strings.Join(artistCases, " "),
		strings.Join(genreCases, " "),
		strings.Join(bpmCases, " "),
		strings.Join(releasedCases, " "),
		strings.Join(jacketCases, " "),
		strings.Join(isNewCases, " "),
		keepExistingUpdateExpr("unlock_required", unlockRequiredCases),
		strings.Join(placeholders, ","),
	)

	_, err := exec.ExecContext(ctx, query, args...)
	return err
}

// bulkUpdateCharts は譜面情報をCASE式で一括更新します。
func (r *songRepository) bulkUpdateCharts(ctx context.Context, exec repository.Executor, songs []*entity.Song, displayIDToSongID map[string]int) error {
	// chartUpdate は一括更新に必要な譜面の値を保持します。
	type chartUpdate struct {
		SongID         int
		DifficultyID   int
		Const          float64
		IsConstUnknown bool
		Notes          *int
		NotesDesigner  *string
	}

	var updates []chartUpdate
	for _, song := range songs {
		songID := displayIDToSongID[song.DisplayID]
		for _, chart := range song.Charts {
			var notesPtr *int
			if chart.Notes != nil {
				n := int(*chart.Notes)
				notesPtr = &n
			}
			updates = append(updates, chartUpdate{
				SongID:         songID,
				DifficultyID:   chart.DifficultyID,
				Const:          chart.Const.Float64(),
				IsConstUnknown: chart.IsConstUnknown,
				Notes:          notesPtr,
				NotesDesigner:  chart.NotesDesigner,
			})
		}
	}

	if len(updates) == 0 {
		return nil
	}

	// 注意: SQLの引数順序はCASE式の出現順（const→is_const_unknown→notes→WHERE）であるため、
	// 各フィールドの引数を別々に蓄積し、最後に正しい順序で結合する必要がある
	var constCases, unknownCases, notesCases, notesDesignerCases []string
	var constArgs, unknownArgs, notesArgs, notesDesignerArgs, whereArgs []any

	for _, u := range updates {
		constCases = append(constCases, "WHEN song_id = ? AND difficulty_id = ? THEN ?")
		constArgs = append(constArgs, u.SongID, u.DifficultyID, u.Const)

		unknownCases = append(unknownCases, "WHEN song_id = ? AND difficulty_id = ? THEN ?")
		unknownArgs = append(unknownArgs, u.SongID, u.DifficultyID, u.IsConstUnknown)

		notesCases = append(notesCases, "WHEN song_id = ? AND difficulty_id = ? THEN ?")
		notesArgs = append(notesArgs, u.SongID, u.DifficultyID, u.Notes)

		notesDesignerCases = append(notesDesignerCases, "WHEN song_id = ? AND difficulty_id = ? THEN ?")
		notesDesignerArgs = append(notesDesignerArgs, u.SongID, u.DifficultyID, u.NotesDesigner)
	}

	var wherePairs []string
	for _, u := range updates {
		wherePairs = append(wherePairs, "(song_id = ? AND difficulty_id = ?)")
		whereArgs = append(whereArgs, u.SongID, u.DifficultyID)
	}

	args := make([]any, 0)
	args = append(args, constArgs...)
	args = append(args, unknownArgs...)
	args = append(args, notesArgs...)
	args = append(args, notesDesignerArgs...)
	args = append(args, whereArgs...)

	query := fmt.Sprintf(`
		UPDATE charts SET
			const = CASE %s END,
			is_const_unknown = CASE %s END,
			notes = CASE %s END,
			notes_designer = CASE %s END
		WHERE (%s)
		  AND song_id IN (SELECT id FROM songs WHERE is_worldsend = 0)
	`,
		strings.Join(constCases, " "),
		strings.Join(unknownCases, " "),
		strings.Join(notesCases, " "),
		strings.Join(notesDesignerCases, " "),
		strings.Join(wherePairs, " OR "),
	)

	_, err := exec.ExecContext(ctx, query, args...)
	return err
}

// Create は新規楽曲を songs および charts テーブルに追加します。
// display_id は呼び出し元（usecase）で生成済みのものを使用します。
// official_idx 重複時は ErrDuplicateOfficialIdx を返します。
func (r *songRepository) Create(ctx context.Context, exec repository.Executor, song *entity.Song) (*entity.Song, error) {
	songResult, err := exec.ExecContext(ctx, `
		INSERT INTO songs (display_id, title, wiki_page_title, reading, artist, genre_id, bpm, released_at, official_idx, jacket, is_worldsend, is_new, unlock_required, is_deleted)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0, ?, ?, 0)
	`,
		song.DisplayID,
		song.Title,
		song.WikiPageTitle,
		song.Reading,
		song.Artist,
		song.GenreID,
		song.BPM,
		song.ReleasedAt,
		song.OfficialIdx,
		song.Jacket,
		song.IsNew,
		song.UnlockRequired,
	)
	if err != nil {
		if wrapped := wrapOfficialIdxDuplicateError(err); wrapped != err {
			return nil, wrapped
		}
		return nil, err
	}

	songID, err := songResult.LastInsertId()
	if err != nil {
		return nil, err
	}

	for _, chart := range song.Charts {
		constVal, err := chart.Const.Value()
		if err != nil {
			return nil, fmt.Errorf("failed to get chart const value: %w", err)
		}

		var notesVal *int
		if chart.Notes != nil {
			n := int(*chart.Notes)
			notesVal = &n
		}

		if _, err = exec.ExecContext(ctx, `
			INSERT INTO charts (song_id, difficulty_id, const, is_const_unknown, notes, notes_designer)
			VALUES (?, ?, ?, ?, ?, ?)
		`,
			songID,
			chart.DifficultyID,
			constVal,
			chart.IsConstUnknown,
			notesVal,
			chart.NotesDesigner,
		); err != nil {
			return nil, err
		}
	}

	// DB側で付与される更新日時を返却エンティティに反映するため再取得します。
	return r.FindByDisplayID(ctx, exec, song.DisplayID)
}
