package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/Qman110101/chunisupport-api/internal/domain/entity"
	"github.com/Qman110101/chunisupport-api/internal/domain/repository"
	"github.com/Qman110101/chunisupport-api/internal/domain/vo/chartconstant"
	"github.com/Qman110101/chunisupport-api/internal/domain/vo/notes"
	api_internal "github.com/Qman110101/chunisupport-api/internal/dto/api_internal"
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
func (r *songRepository) FindAllExcludingWorldsend(ctx context.Context, exec repository.Executor, includeDeleted bool) ([]*repository.SongWithCharts, error) {
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
		return []*repository.SongWithCharts{}, nil
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
	results := make([]*repository.SongWithCharts, len(songRows))
	for i, sr := range songRows {
		song := r.toSongEntity(&sr)
		results[i] = &repository.SongWithCharts{
			Song:   song,
			Charts: []*entity.Chart{},
		}
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

// toSongEntity は songRow を entity.Song に変換します。
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

// toChartEntity は chartRow を entity.Chart に変換します。
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
func (r *songRepository) FindByDisplayID(ctx context.Context, exec repository.Executor, displayID string) (*repository.SongWithCharts, error) {
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

	return &repository.SongWithCharts{
		Song:   song,
		Charts: charts,
	}, nil
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
func (r *songRepository) UpdateSongs(ctx context.Context, exec repository.Executor, requests []*api_internal.UpdateSongRequest) error {
	if len(requests) == 0 {
		return nil
	}

	// 1. 全DisplayIDを収集
	displayIDs := make([]string, len(requests))
	for i, req := range requests {
		displayIDs[i] = req.DisplayID
	}

	// 2. 存在確認（Batch Read）
	songs, err := r.FindByDisplayIDs(ctx, exec, displayIDs)
	if err != nil {
		return fmt.Errorf("failed to find songs by display IDs: %w", err)
	}

	// 3. DisplayID → SongID のマッピング作成
	displayIDToSongID := make(map[string]int, len(songs))
	for _, song := range songs {
		displayIDToSongID[song.DisplayID] = song.ID
	}

	// 4. リクエスト内の全DisplayIDが存在するか確認
	for _, displayID := range displayIDs {
		if _, ok := displayIDToSongID[displayID]; !ok {
			return fmt.Errorf("song with display_id '%s' not found", displayID)
		}
	}

	// 5. 楽曲と譜面を更新
	for _, req := range requests {
		songID := displayIDToSongID[req.DisplayID]

		// 楽曲情報を更新
		updateSongQuery := `
			UPDATE songs
			SET title = ?, artist = ?, genre_id = ?, bpm = ?, released_at = ?, jacket = ?
			WHERE id = ?
		`
		_, err := exec.ExecContext(ctx, updateSongQuery,
			req.Title,
			req.Artist,
			req.GenreID,
			req.BPM,
			req.ReleasedAt,
			req.Jacket,
			songID,
		)
		if err != nil {
			return fmt.Errorf("failed to update song (display_id=%s): %w", req.DisplayID, err)
		}

		// 譜面情報を更新
		for _, chart := range req.Charts {
			// 譜面が存在するか確認
			var exists bool
			checkChartQuery := `SELECT EXISTS(SELECT 1 FROM charts WHERE song_id = ? AND difficulty_id = ?)`
			err := exec.GetContext(ctx, &exists, checkChartQuery, songID, chart.DifficultyID)
			if err != nil {
				return fmt.Errorf("failed to check chart existence (song_id=%d, difficulty_id=%d): %w", songID, chart.DifficultyID, err)
			}
			if !exists {
				return fmt.Errorf("chart not found (song_id=%d, difficulty_id=%d)", songID, chart.DifficultyID)
			}

			// Notes のポインタをint型ポインタに変換
			var notesPtr *int
			if chart.Notes != nil {
				notesPtr = chart.Notes
			}

			updateChartQuery := `
				UPDATE charts
				SET const = ?, is_const_unknown = ?, notes = ?
				WHERE song_id = ? AND difficulty_id = ?
			`
			_, err = exec.ExecContext(ctx, updateChartQuery,
				chart.Const,
				chart.IsConstUnknown,
				notesPtr,
				songID,
				chart.DifficultyID,
			)
			if err != nil {
				return fmt.Errorf("failed to update chart (song_id=%d, difficulty_id=%d): %w", songID, chart.DifficultyID, err)
			}
		}
	}

	return nil
}
