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
}

// chartRow はDBから取得する譜面データの行を表します。
type chartRow struct {
	ID             int     `db:"id"`
	SongID         int     `db:"song_id"`
	DifficultyID   int     `db:"difficulty_id"`
	Const          float64 `db:"const"`
	IsConstUnknown bool    `db:"is_const_unknown"`
	Notes          *int    `db:"notes"`
}

// FindAllExcludingWorldsend はWORLD'S END以外の全楽曲を取得します。
// includeDeletedがfalseの場合、削除済み楽曲は除外されます。
// N+1問題を回避するため、楽曲と譜面を別々のクエリで取得し、メモリ上で結合します。
func (r *songRepository) FindAllExcludingWorldsend(ctx context.Context, exec repository.Executor, includeDeleted bool) ([]*entity.Song, error) {
	// 1. WORLD'S END以外の楽曲を取得
	songsQuery := `
		SELECT id, display_id, title, artist, genre_id, bpm, released_at, official_idx, jacket, is_worldsend, is_deleted
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

	// 2. 取得した楽曲のIDを収集
	songIDs := make([]int, len(songRows))
	songIDToIndex := make(map[int]int, len(songRows))
	for i, s := range songRows {
		songIDs[i] = s.ID
		songIDToIndex[s.ID] = i
	}

	// 3. 該当する楽曲の譜面を一括取得（N+1問題回避）
	chartsQuery, args, err := sqlx.In(`
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

	return results, nil
}

func (r *songRepository) toSongEntity(row *songRow) *entity.Song {
	return &entity.Song{
		ID:          row.ID,
		DisplayID:   row.DisplayID,
		Title:       row.Title,
		Artist:      row.Artist,
		GenreID:     row.GenreID,
		BPM:         row.BPM,
		ReleasedAt:  row.ReleasedAt,
		OfficialIdx: row.OfficialIdx,
		Jacket:      row.Jacket,
		IsWorldsend: row.IsWorldsend,
		IsDeleted:   row.IsDeleted,
	}
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

// FindByDisplayIDs は指定されたDisplayIDのリストに該当する楽曲を取得します。
func (r *songRepository) FindByDisplayIDs(ctx context.Context, exec repository.Executor, displayIDs []string) ([]*entity.Song, error) {
	if len(displayIDs) == 0 {
		return []*entity.Song{}, nil
	}

	query, args, err := sqlx.In(`
		SELECT id, display_id, title, artist, genre_id, bpm, released_at, official_idx, jacket, is_worldsend, is_deleted
		FROM songs
		WHERE display_id IN (?)
	`, displayIDs)
	if err != nil {
		return nil, err
	}
	query = exec.Rebind(query)

	var songRows []songRow
	if err := exec.SelectContext(ctx, &songRows, query, args...); err != nil {
		return nil, err
	}

	songs := make([]*entity.Song, len(songRows))
	for i, sr := range songRows {
		songs[i] = r.toSongEntity(&sr)
	}

	return songs, nil
}

// FindByDisplayID は指定されたDisplayIDの楽曲を取得します。
// 削除済み楽曲も取得します。
func (r *songRepository) FindByDisplayID(ctx context.Context, exec repository.Executor, displayID string) (*entity.Song, error) {
	// 1. 楽曲を取得
	songQuery := `
		SELECT id, display_id, title, artist, genre_id, bpm, released_at, official_idx, jacket, is_worldsend, is_deleted
		FROM songs
		WHERE display_id = ?
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

	return song, nil
}

// DeleteSong は指定されたDisplayIDの楽曲を論理削除します。
func (r *songRepository) DeleteSong(ctx context.Context, exec repository.Executor, displayID string) error {
	query := `UPDATE songs SET is_deleted = TRUE WHERE display_id = ?`
	_, err := exec.ExecContext(ctx, query, displayID)
	return err
}

// RestoreSong は指定されたDisplayIDの楽曲を復活させます。
func (r *songRepository) RestoreSong(ctx context.Context, exec repository.Executor, displayID string) error {
	query := `UPDATE songs SET is_deleted = FALSE WHERE display_id = ?`
	_, err := exec.ExecContext(ctx, query, displayID)
	return err
}

// UpdateSongs は楽曲および譜面情報を一括更新します。
// トランザクション管理はUseCase層（TransactionManager経由）で行います。
// PERF-008対策: N+1問題を解消するため、一括更新クエリを使用します。
func (r *songRepository) UpdateSongs(ctx context.Context, exec repository.Executor, songs []*entity.Song) error {
	if len(songs) == 0 {
		return nil
	}

	// 1. 全DisplayIDを収集
	displayIDs := make([]string, len(songs))
	for i, song := range songs {
		displayIDs[i] = song.DisplayID
	}

	// 2. 既存楽曲を一括取得（存在確認とID取得）
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

	// 5. 既存譜面を一括取得して存在確認
	songIDs := make([]int, 0, len(displayIDToSongID))
	for _, id := range displayIDToSongID {
		songIDs = append(songIDs, id)
	}
	existingCharts, err := r.findChartsBySongIDs(ctx, exec, songIDs)
	if err != nil {
		return fmt.Errorf("failed to find charts by song IDs: %w", err)
	}

	// 譜面の存在確認用マップ: "songID-difficultyID" -> true
	chartExistsMap := make(map[string]bool)
	for _, chart := range existingCharts {
		key := fmt.Sprintf("%d-%d", chart.SongID, chart.DifficultyID)
		chartExistsMap[key] = true
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

// findChartsBySongIDs は指定されたsongIDリストの譜面を一括取得します。
func (r *songRepository) findChartsBySongIDs(ctx context.Context, exec repository.Executor, songIDs []int) ([]*entity.Chart, error) {
	if len(songIDs) == 0 {
		return []*entity.Chart{}, nil
	}

	query, args, err := sqlx.In(`
		SELECT id, song_id, difficulty_id, const, is_const_unknown, notes
		FROM charts
		WHERE song_id IN (?)
	`, songIDs)
	if err != nil {
		return nil, err
	}
	query = exec.Rebind(query)

	var chartRows []chartRow
	if err := exec.SelectContext(ctx, &chartRows, query, args...); err != nil {
		return nil, err
	}

	charts := make([]*entity.Chart, len(chartRows))
	for i, cr := range chartRows {
		charts[i] = r.toChartEntity(&cr)
	}

	return charts, nil
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
	var titleCases, artistCases, genreCases, bpmCases, releasedCases, jacketCases []string
	args := make([]any, 0)

	for _, song := range songs {
		songID := displayIDToSongID[song.DisplayID]

		titleCases = append(titleCases, "WHEN id = ? THEN ?")
		args = append(args, songID, song.Title)

		artistCases = append(artistCases, "WHEN id = ? THEN ?")
		args = append(args, songID, song.Artist)

		genreCases = append(genreCases, "WHEN id = ? THEN ?")
		args = append(args, songID, song.GenreID)

		bpmCases = append(bpmCases, "WHEN id = ? THEN ?")
		args = append(args, songID, song.BPM)

		releasedCases = append(releasedCases, "WHEN id = ? THEN ?")
		args = append(args, songID, song.ReleasedAt)

		jacketCases = append(jacketCases, "WHEN id = ? THEN ?")
		args = append(args, songID, song.Jacket)
	}

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
	// 複合キー(song_id, difficulty_id)でマッチングするため、条件式を使用
	var constCases, unknownCases, notesCases []string
	args := make([]any, 0)

	for _, u := range updates {
		constCases = append(constCases, "WHEN song_id = ? AND difficulty_id = ? THEN ?")
		args = append(args, u.SongID, u.DifficultyID, u.Const)

		unknownCases = append(unknownCases, "WHEN song_id = ? AND difficulty_id = ? THEN ?")
		args = append(args, u.SongID, u.DifficultyID, u.IsConstUnknown)

		notesCases = append(notesCases, "WHEN song_id = ? AND difficulty_id = ? THEN ?")
		args = append(args, u.SongID, u.DifficultyID, u.Notes)
	}

	// WHERE句用: (song_id, difficulty_id) の組み合わせ
	var wherePairs []string
	for _, u := range updates {
		wherePairs = append(wherePairs, "(song_id = ? AND difficulty_id = ?)")
		args = append(args, u.SongID, u.DifficultyID)
	}

	query := fmt.Sprintf(`
		UPDATE charts SET
			const = CASE %s END,
			is_const_unknown = CASE %s END,
			notes = CASE %s END
		WHERE %s
	`,
		strings.Join(constCases, " "),
		strings.Join(unknownCases, " "),
		strings.Join(notesCases, " "),
		strings.Join(wherePairs, " OR "),
	)

	_, err := exec.ExecContext(ctx, query, args...)
	return err
}
