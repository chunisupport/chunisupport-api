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
	ID          int        `db:"id"`
	DisplayID   string     `db:"display_id"`
	Title       string     `db:"title"`
	Artist      string     `db:"artist"`
	GenreID     *int       `db:"genre_id"`
	BPM         *int       `db:"bpm"`
	ReleasedAt  *time.Time `db:"released_at"`
	OfficialIdx string     `db:"official_idx"`
	Jacket      *string    `db:"jacket"`
	IsWorldsend bool       `db:"is_worldsend"`
	IsDeleted   bool       `db:"is_deleted"`
	UpdatedAt   *time.Time `db:"updated_at"`
}

// chartRow はDBから取得する譜面データの行を表します。
type chartRow struct {
	ID             int        `db:"id"`
	SongID         int        `db:"song_id"`
	DifficultyID   int        `db:"difficulty_id"`
	Const          float64    `db:"const"`
	IsConstUnknown bool       `db:"is_const_unknown"`
	Notes          *int       `db:"notes"`
	UpdatedAt      *time.Time `db:"updated_at"`
}

// FindAllExcludingWorldsend はWORLD'S END以外の全楽曲を取得します。
// includeDeletedがfalseの場合、削除済み楽曲は除外されます。
// N+1問題を回避するため、楽曲と譜面を別々のクエリで取得し、メモリ上で結合します。
func (r *songRepository) FindAllExcludingWorldsend(ctx context.Context, exec repository.Executor, includeDeleted bool) (*repository.SongListResult, error) {
	// 1. WORLD'S END以外の楽曲を取得
	songsQuery := `
		SELECT id, display_id, title, artist, genre_id, bpm, released_at, official_idx, jacket, is_worldsend, is_deleted, updated_at
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
		return &repository.SongListResult{Songs: []*entity.Song{}}, nil
	}

	// 2. 取得した楽曲のIDを収集
	songIDs := make([]int, len(songRows))
	songIDToIndex := make(map[int]int, len(songRows))
	for i, s := range songRows {
		songIDs[i] = s.ID
		songIDToIndex[s.ID] = i
	}

	// 3. 該当する楽曲の譜面を一括取得（N+1問題回避）
	chartsQuery, args, err := sqlx.In(`
		SELECT id, song_id, difficulty_id, const, is_const_unknown, notes, updated_at
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

	// 4. 結果を構築
	results := make([]*entity.Song, len(songRows))
	for i, sr := range songRows {
		song := r.toSongEntity(&sr)
		song.Charts = []*entity.Chart{}
		results[i] = song
	}

	// 5. 譜面を楽曲に紐付け
	for _, cr := range chartRows {
		idx, ok := songIDToIndex[cr.SongID]
		if !ok {
			continue
		}
		chart := r.toChartEntity(&cr)
		results[idx].Charts = append(results[idx].Charts, chart)
	}

	// 6. ドメインサービスで譜面集約を適用（MaxChartConst, IsMaxOPUnknown）
	for _, song := range results {
		service.ApplyAggregation(song)
	}

	return &repository.SongListResult{
		Songs:     results,
		UpdatedAt: maxSongListUpdatedAt(songRows, chartRows),
	}, nil
}

// GetLatestUpdatedAtExcludingWorldsend はWORLD'S END以外の楽曲一覧全体の最終更新日時を返します。
// includeDeleted=false の場合でも songs の updated_at は全楽曲対象とします。
// is_deleted=1 への遷移（削除操作）そのものが公開一覧の内容を変えるため、
// 削除済み楽曲の updated_at も MAX 計算に含める必要があるためです。
// 一方 charts の updated_at は公開楽曲（is_deleted=0）に属するもののみを対象とします。
func (r *songRepository) GetLatestUpdatedAtExcludingWorldsend(ctx context.Context, exec repository.Executor, includeDeleted bool) (*time.Time, error) {
	// songs は includeDeleted に関わらず全楽曲対象（削除操作の検知のため、is_deleted フィルタなし）
	// charts は公開楽曲（is_deleted=0）に属するもののみを対象とする
	var chartsWhereClause string
	if !includeDeleted {
		chartsWhereClause = ` AND s.is_deleted = 0`
	}

	query := fmt.Sprintf(`
		SELECT MAX(updated_at) FROM (
			SELECT s.updated_at AS updated_at
			FROM songs s
			WHERE s.is_worldsend = 0
			UNION ALL
			SELECT c.updated_at AS updated_at
			FROM charts c
			INNER JOIN songs s ON s.id = c.song_id
			WHERE s.is_worldsend = 0%s
		) latest_updates
	`, chartsWhereClause)

	return scanNullableTime(ctx, exec, query)
}

func maxSongListUpdatedAt(songRows []songRow, chartRows []chartRow) *time.Time {
	var maxUpdatedAt *time.Time

	for _, row := range songRows {
		maxUpdatedAt = maxTimePtr(maxUpdatedAt, row.UpdatedAt)
	}
	for _, row := range chartRows {
		maxUpdatedAt = maxTimePtr(maxUpdatedAt, row.UpdatedAt)
	}

	return maxUpdatedAt
}

func maxTimePtr(current *time.Time, candidate *time.Time) *time.Time {
	if candidate == nil {
		return current
	}
	if current == nil || candidate.After(*current) {
		value := *candidate
		return &value
	}

	return current
}

func (r *songRepository) toSongEntity(row *songRow) *entity.Song {
	song := entity.NewSong()
	song.ID = row.ID
	song.DisplayID = row.DisplayID
	song.Title = row.Title
	song.Artist = row.Artist
	song.GenreID = row.GenreID
	song.BPM = row.BPM
	song.ReleasedAt = row.ReleasedAt
	song.OfficialIdx = row.OfficialIdx
	song.Jacket = row.Jacket
	song.IsWorldsend = row.IsWorldsend
	song.IsDeleted = row.IsDeleted
	return song
}

func (r *songRepository) toChartEntity(row *chartRow) *entity.Chart {
	constVal, _ := chartconstant.NewChartConstant(row.Const)

	var notesVal *notes.Notes
	if row.Notes != nil {
		n, _ := notes.NewNotes(*row.Notes)
		notesVal = &n
	}

	return &entity.Chart{
		ID:             row.ID,
		SongID:         row.SongID,
		DifficultyID:   row.DifficultyID,
		Const:          constVal,
		IsConstUnknown: row.IsConstUnknown,
		Notes:          notesVal,
	}
}

// FindByDisplayIDs は指定されたDisplayIDのリストに該当する通常楽曲（WORLD'S END除く）を取得します。
// 各楽曲には関連する譜面情報が含まれます。
// N+1問題を回避するため、楽曲と譜面を別々のクエリで取得し、メモリ上で結合します。
func (r *songRepository) FindByDisplayIDs(ctx context.Context, exec repository.Executor, displayIDs []string) ([]*entity.Song, error) {
	if len(displayIDs) == 0 {
		return []*entity.Song{}, nil
	}

	// 1. 楽曲を取得
	query, args, err := sqlx.In(`
		SELECT id, display_id, title, artist, genre_id, bpm, released_at, official_idx, jacket, is_worldsend, is_deleted
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

	// 2. 取得した楽曲のIDを収集
	songIDs := make([]int, len(songRows))
	songIDToIndex := make(map[int]int, len(songRows))
	for i, s := range songRows {
		songIDs[i] = s.ID
		songIDToIndex[s.ID] = i
	}

	// 3. 該当する楽曲の譜面を一括取得（N+1問題回避）
	chartsQuery, chartArgs, err := sqlx.In(`
		SELECT id, song_id, difficulty_id, const, is_const_unknown, notes
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

	// 4. 結果を構築
	songs := make([]*entity.Song, len(songRows))
	for i, sr := range songRows {
		song := r.toSongEntity(&sr)
		song.Charts = []*entity.Chart{}
		songs[i] = song
	}

	// 5. 譜面を楽曲に紐付け
	for _, cr := range chartRows {
		idx, ok := songIDToIndex[cr.SongID]
		if !ok {
			continue
		}
		chart := r.toChartEntity(&cr)
		songs[idx].Charts = append(songs[idx].Charts, chart)
	}

	// 6. ドメインサービスで譜面集約を適用（MaxChartConst, IsMaxOPUnknown）
	for _, song := range songs {
		service.ApplyAggregation(song)
	}

	return songs, nil
}

// FindByDisplayID は指定されたDisplayIDの通常楽曲（WORLD'S END除く）を取得します。
// 削除済み楽曲も取得します。
func (r *songRepository) FindByDisplayID(ctx context.Context, exec repository.Executor, displayID string) (*entity.Song, error) {
	// 1. 楽曲を取得
	songQuery := `
		SELECT id, display_id, title, artist, genre_id, bpm, released_at, official_idx, jacket, is_worldsend, is_deleted
		FROM songs
		WHERE display_id = ? AND is_worldsend = 0
	`
	var songRow songRow
	if err := exec.GetContext(ctx, &songRow, songQuery, displayID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrSongNotFound
		}
		return nil, err
	}

	song := r.toSongEntity(&songRow)

	// 2. 譜面を取得
	chartsQuery := `
		SELECT id, song_id, difficulty_id, const, is_const_unknown, notes
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
		charts[i] = r.toChartEntity(&cr)
	}

	song.Charts = charts

	// 3. ドメインサービスで譜面集約を適用（MaxChartConst, IsMaxOPUnknown）
	service.ApplyAggregation(song)

	return song, nil
}

// Save は楽曲エンティティの現在の状態を永続化します。
// 対象が存在しない場合は ErrSongNotFound を返します。
func (r *songRepository) Save(ctx context.Context, exec repository.Executor, song *entity.Song) error {
	query := `
		UPDATE songs
		SET display_id = ?, title = ?, artist = ?, genre_id = ?, bpm = ?, released_at = ?, official_idx = ?, jacket = ?, is_worldsend = ?, is_deleted = ?
		WHERE id = ?
	`
	result, err := exec.ExecContext(
		ctx,
		query,
		song.DisplayID,
		song.Title,
		song.Artist,
		song.GenreID,
		song.BPM,
		song.ReleasedAt,
		song.OfficialIdx,
		song.Jacket,
		song.IsWorldsend,
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
		return repository.ErrSongNotFound
	}

	return nil
}

// UpdateSongs は楽曲および譜面情報を一括更新します。
// トランザクション管理はUseCase層（TransactionManager経由）で行います。
// PERF-008対策: N+1問題を解消するため、一括更新クエリを使用します。
func (r *songRepository) UpdateSongs(ctx context.Context, exec repository.Executor, songs []*entity.Song) error {
	if len(songs) == 0 {
		return nil
	}

	// 重複display_idは後続のCASE WHEN構築で先出現側が暗黙適用されるため事前に弾く
	displayIDs, err := collectUniqueDisplayIDs(songs)
	if err != nil {
		return err
	}

	// 2. 既存楽曲を一括取得（存在確認とID取得、譜面情報も含む）
	existingSongs, err := r.FindByDisplayIDs(ctx, exec, displayIDs)
	if err != nil {
		return fmt.Errorf("failed to find songs by display IDs: %w", err)
	}

	// 3. DisplayID → SongID のマッピング作成
	displayIDToSongID := make(map[string]int, len(existingSongs))
	for _, song := range existingSongs {
		displayIDToSongID[song.DisplayID] = song.ID
	}

	// 4. リクエスト内の全DisplayIDが存在するか確認
	for _, displayID := range displayIDs {
		if _, ok := displayIDToSongID[displayID]; !ok {
			return fmt.Errorf("song with display_id '%s' not found", displayID)
		}
	}

	// 5. 既存譜面の存在確認用マップを作成: "songID-difficultyID" -> true
	// FindByDisplayIDsで既にChartsが含まれているので、そこから取得する
	chartExistsMap := make(map[string]bool)
	for _, song := range existingSongs {
		for _, chart := range song.Charts {
			key := fmt.Sprintf("%d-%d", chart.SongID, chart.DifficultyID)
			chartExistsMap[key] = true
		}
	}

	// 6. 更新対象の譜面が全て存在するか確認
	for _, song := range songs {
		songID := displayIDToSongID[song.DisplayID]
		for _, chart := range song.Charts {
			key := fmt.Sprintf("%d-%d", songID, chart.DifficultyID)
			if !chartExistsMap[key] {
				return fmt.Errorf("chart not found (song_id=%d, difficulty_id=%d)", songID, chart.DifficultyID)
			}
		}
	}

	// 7. 楽曲を一括更新（CASE式を使用）
	if err := r.bulkUpdateSongs(ctx, exec, songs, displayIDToSongID); err != nil {
		return fmt.Errorf("failed to bulk update songs: %w", err)
	}

	// 8. 譜面を一括更新（CASE式を使用）
	if err := r.bulkUpdateCharts(ctx, exec, songs, displayIDToSongID); err != nil {
		return fmt.Errorf("failed to bulk update charts: %w", err)
	}

	return nil
}

// bulkUpdateSongs は楽曲情報をCASE式で一括更新します。
func (r *songRepository) bulkUpdateSongs(ctx context.Context, exec repository.Executor, songs []*entity.Song, displayIDToSongID map[string]int) error {
	if len(songs) == 0 {
		return nil
	}

	// 更新対象のsongIDリストを作成
	songIDs := make([]int, 0, len(songs))
	for _, song := range songs {
		songIDs = append(songIDs, displayIDToSongID[song.DisplayID])
	}

	// CASE式を構築
	// 注意: SQLの引数順序はCASE式の出現順（title→artist→genre→...→IN句）であるため、
	// 各フィールドの引数を別々に蓄積し、最後に正しい順序で結合する必要がある
	var titleCases, artistCases, genreCases, bpmCases, releasedCases, jacketCases []string
	var titleArgs, artistArgs, genreArgs, bpmArgs, releasedArgs, jacketArgs []any

	for _, song := range songs {
		songID := displayIDToSongID[song.DisplayID]

		titleCases = append(titleCases, "WHEN id = ? THEN ?")
		titleArgs = append(titleArgs, songID, song.Title)

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
	}

	// SQLの引数順序に合わせて結合: title→artist→genre→bpm→released→jacket→IN句
	args := make([]any, 0)
	args = append(args, titleArgs...)
	args = append(args, artistArgs...)
	args = append(args, genreArgs...)
	args = append(args, bpmArgs...)
	args = append(args, releasedArgs...)
	args = append(args, jacketArgs...)

	// IN句用の引数を追加
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
			artist = CASE %s END,
			genre_id = CASE %s END,
			bpm = CASE %s END,
			released_at = CASE %s END,
			jacket = CASE %s END
		WHERE id IN (%s)
		  AND is_worldsend = 0
	`,
		strings.Join(titleCases, " "),
		strings.Join(artistCases, " "),
		strings.Join(genreCases, " "),
		strings.Join(bpmCases, " "),
		strings.Join(releasedCases, " "),
		strings.Join(jacketCases, " "),
		strings.Join(placeholders, ","),
	)

	_, err := exec.ExecContext(ctx, query, args...)
	return err
}

// bulkUpdateCharts は譜面情報をCASE式で一括更新します。
func (r *songRepository) bulkUpdateCharts(ctx context.Context, exec repository.Executor, songs []*entity.Song, displayIDToSongID map[string]int) error {
	// 全譜面データを収集
	type chartUpdate struct {
		SongID         int
		DifficultyID   int
		Const          float64
		IsConstUnknown bool
		Notes          *int
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
				Const:          float64(chart.Const),
				IsConstUnknown: chart.IsConstUnknown,
				Notes:          notesPtr,
			})
		}
	}

	if len(updates) == 0 {
		return nil
	}

	// CASE式を構築
	// 注意: SQLの引数順序はCASE式の出現順（const→is_const_unknown→notes→WHERE）であるため、
	// 各フィールドの引数を別々に蓄積し、最後に正しい順序で結合する必要がある
	var constCases, unknownCases, notesCases []string
	var constArgs, unknownArgs, notesArgs, whereArgs []any

	for _, u := range updates {
		constCases = append(constCases, "WHEN song_id = ? AND difficulty_id = ? THEN ?")
		constArgs = append(constArgs, u.SongID, u.DifficultyID, u.Const)

		unknownCases = append(unknownCases, "WHEN song_id = ? AND difficulty_id = ? THEN ?")
		unknownArgs = append(unknownArgs, u.SongID, u.DifficultyID, u.IsConstUnknown)

		notesCases = append(notesCases, "WHEN song_id = ? AND difficulty_id = ? THEN ?")
		notesArgs = append(notesArgs, u.SongID, u.DifficultyID, u.Notes)
	}

	// WHERE句用: (song_id, difficulty_id) の組み合わせ
	var wherePairs []string
	for _, u := range updates {
		wherePairs = append(wherePairs, "(song_id = ? AND difficulty_id = ?)")
		whereArgs = append(whereArgs, u.SongID, u.DifficultyID)
	}

	// SQLの引数順序に合わせて結合: const→is_const_unknown→notes→WHERE
	args := make([]any, 0)
	args = append(args, constArgs...)
	args = append(args, unknownArgs...)
	args = append(args, notesArgs...)
	args = append(args, whereArgs...)

	query := fmt.Sprintf(`
		UPDATE charts SET
			const = CASE %s END,
			is_const_unknown = CASE %s END,
			notes = CASE %s END
		WHERE (%s)
		  AND song_id IN (SELECT id FROM songs WHERE is_worldsend = 0)
	`,
		strings.Join(constCases, " "),
		strings.Join(unknownCases, " "),
		strings.Join(notesCases, " "),
		strings.Join(wherePairs, " OR "),
	)

	_, err := exec.ExecContext(ctx, query, args...)
	return err
}
