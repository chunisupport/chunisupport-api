package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Qman110101/chunisupport-api/internal/domain/entity"
	"github.com/Qman110101/chunisupport-api/internal/domain/repository"
	"github.com/Qman110101/chunisupport-api/internal/infra/models"
	"github.com/jmoiron/sqlx"
)

// worldsendChartRepository は WorldsendChartRepository の実装です。
type worldsendChartRepository struct {
	db *sqlx.DB
}

// NewWorldsendChartRepository は WorldsendChartRepository の実装を生成します。
func NewWorldsendChartRepository(db *sqlx.DB) repository.WorldsendChartRepository {
	return &worldsendChartRepository{db: db}
}

// worldsendSongChartRow は DB から WORLD'S END 楽曲と譜面の JOIN 結果をマッピングする構造体です。
type worldsendSongChartRow struct {
	models.SongModel
	models.WorldsendChartModel
}

// FindAll は全 WORLD'S END 楽曲を譜面情報付きで取得します。
func (r *worldsendChartRepository) FindAll(ctx context.Context, includeDeleted bool) ([]*repository.WorldsendSongWithChart, error) {
	query := `
		SELECT
			s.id, s.display_id, s.title, s.artist, s.genre_id, s.bpm, s.released_at, s.official_idx, s.jacket, s.is_worldsend, s.is_deleted,
			wc.id AS 'worldsend_charts.id',
			wc.song_id AS 'worldsend_charts.song_id',
			wc.we_star AS 'worldsend_charts.we_star',
			wc.we_kanji AS 'worldsend_charts.we_kanji',
			wc.notes AS 'worldsend_charts.notes'
		FROM songs s
		INNER JOIN worldsend_charts wc ON s.id = wc.song_id
		WHERE s.is_worldsend = 1`
	if !includeDeleted {
		query += ` AND s.is_deleted = 0`
	}
	query += ` ORDER BY s.id`

	rows, err := r.db.QueryxContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := []*repository.WorldsendSongWithChart{}
	for rows.Next() {
		var songModel models.SongModel
		var chartModel models.WorldsendChartModel

		err := rows.Scan(
			&songModel.ID, &songModel.DisplayID, &songModel.Title, &songModel.Artist,
			&songModel.GenreID, &songModel.BPM, &songModel.ReleasedAt, &songModel.OfficialIdx,
			&songModel.Jacket, &songModel.IsWorldsend, &songModel.IsDeleted,
			&chartModel.ID, &chartModel.SongID, &chartModel.WeStar, &chartModel.WeKanji, &chartModel.Notes,
		)
		if err != nil {
			return nil, err
		}

		results = append(results, &repository.WorldsendSongWithChart{
			Song:  songModel.ToEntity(),
			Chart: chartModel.ToEntity(),
		})
	}

	return results, rows.Err()
}

// FindByDisplayID は指定された DisplayID の WORLD'S END 楽曲を取得します。
func (r *worldsendChartRepository) FindByDisplayID(ctx context.Context, displayID string) (*repository.WorldsendSongWithChart, error) {
	query := `
		SELECT
			s.id, s.display_id, s.title, s.artist, s.genre_id, s.bpm, s.released_at, s.official_idx, s.jacket, s.is_worldsend, s.is_deleted,
			wc.id AS 'worldsend_charts.id',
			wc.song_id AS 'worldsend_charts.song_id',
			wc.we_star AS 'worldsend_charts.we_star',
			wc.we_kanji AS 'worldsend_charts.we_kanji',
			wc.notes AS 'worldsend_charts.notes'
		FROM songs s
		INNER JOIN worldsend_charts wc ON s.id = wc.song_id
		WHERE s.display_id = ? AND s.is_worldsend = 1`

	var songModel models.SongModel
	var chartModel models.WorldsendChartModel

	err := r.db.QueryRowxContext(ctx, query, displayID).Scan(
		&songModel.ID, &songModel.DisplayID, &songModel.Title, &songModel.Artist,
		&songModel.GenreID, &songModel.BPM, &songModel.ReleasedAt, &songModel.OfficialIdx,
		&songModel.Jacket, &songModel.IsWorldsend, &songModel.IsDeleted,
		&chartModel.ID, &chartModel.SongID, &chartModel.WeStar, &chartModel.WeKanji, &chartModel.Notes,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}

	return &repository.WorldsendSongWithChart{
		Song:  songModel.ToEntity(),
		Chart: chartModel.ToEntity(),
	}, nil
}

// DeleteSong は指定された DisplayID の WORLD'S END 楽曲を論理削除します。
func (r *worldsendChartRepository) DeleteSong(ctx context.Context, displayID string) error {
	query := `UPDATE songs SET is_deleted = 1 WHERE display_id = ? AND is_worldsend = 1`
	result, err := r.db.ExecContext(ctx, query, displayID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// RestoreSong は指定された DisplayID の WORLD'S END 楽曲を復活させます。
func (r *worldsendChartRepository) RestoreSong(ctx context.Context, displayID string) error {
	query := `UPDATE songs SET is_deleted = 0 WHERE display_id = ? AND is_worldsend = 1`
	result, err := r.db.ExecContext(ctx, query, displayID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// UpdateSongs は WORLD'S END 楽曲および譜面情報を一括更新します。
// トランザクション管理は呼び出し元で行う必要があります。
func (r *worldsendChartRepository) UpdateSongs(ctx context.Context, songs []*entity.Song, charts []*entity.WorldsendChart) error {
	if len(songs) != len(charts) {
		return fmt.Errorf("songs and charts length mismatch: %d != %d", len(songs), len(charts))
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// 楽曲情報を更新
	songQuery := `
		UPDATE songs
		SET title = ?, artist = ?, genre_id = ?, bpm = ?, released_at = ?, official_idx = ?, jacket = ?
		WHERE id = ? AND is_worldsend = 1`
	for _, song := range songs {
		_, err = tx.ExecContext(ctx, songQuery,
			song.Title, song.Artist, song.GenreID, song.BPM, song.ReleasedAt, song.OfficialIdx, song.Jacket, song.ID)
		if err != nil {
			return err
		}
	}

	// WORLD'S END 譜面情報を更新
	chartQuery := `
		UPDATE worldsend_charts
		SET we_star = ?, we_kanji = ?, notes = ?
		WHERE id = ?`
	for _, chart := range charts {
		_, err = tx.ExecContext(ctx, chartQuery,
			chart.WeStar, chart.WeKanji, chart.Notes, chart.ID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}
