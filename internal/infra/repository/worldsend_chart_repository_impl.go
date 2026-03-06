package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/levelstar"
	"github.com/chunisupport/chunisupport-api/internal/infra/models"
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
func (r *worldsendChartRepository) FindAll(ctx context.Context, exec repository.Executor, includeDeleted bool) ([]*repository.WorldsendSongWithChart, error) {
	query := `
		SELECT
			s.id, s.display_id, s.title, s.artist, s.genre_id, s.bpm, s.released_at, s.official_idx, s.jacket, s.is_worldsend, s.is_deleted,
			wc.id AS 'worldsend_charts.id',
			wc.song_id AS 'worldsend_charts.song_id',
			wc.level_star AS 'worldsend_charts.level_star',
			wc.attribute AS 'worldsend_charts.attribute',
			wc.notes AS 'worldsend_charts.notes'
		FROM songs s
		INNER JOIN worldsend_charts wc ON s.id = wc.song_id
		WHERE s.is_worldsend = 1`
	if !includeDeleted {
		query += ` AND s.is_deleted = 0`
	}
	query += ` ORDER BY s.id`

	rows, err := exec.QueryxContext(ctx, query)
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
			&chartModel.ID, &chartModel.SongID, &chartModel.LevelStar, &chartModel.Attribute, &chartModel.Notes,
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
func (r *worldsendChartRepository) FindByDisplayID(ctx context.Context, exec repository.Executor, displayID string) (*repository.WorldsendSongWithChart, error) {
	query := `
		SELECT
			s.id, s.display_id, s.title, s.artist, s.genre_id, s.bpm, s.released_at, s.official_idx, s.jacket, s.is_worldsend, s.is_deleted,
			wc.id AS 'worldsend_charts.id',
			wc.song_id AS 'worldsend_charts.song_id',
			wc.level_star AS 'worldsend_charts.level_star',
			wc.attribute AS 'worldsend_charts.attribute',
			wc.notes AS 'worldsend_charts.notes'
		FROM songs s
		INNER JOIN worldsend_charts wc ON s.id = wc.song_id
		WHERE s.display_id = ? AND s.is_worldsend = 1`

	var songModel models.SongModel
	var chartModel models.WorldsendChartModel

	err := exec.QueryRowxContext(ctx, query, displayID).Scan(
		&songModel.ID, &songModel.DisplayID, &songModel.Title, &songModel.Artist,
		&songModel.GenreID, &songModel.BPM, &songModel.ReleasedAt, &songModel.OfficialIdx,
		&songModel.Jacket, &songModel.IsWorldsend, &songModel.IsDeleted,
		&chartModel.ID, &chartModel.SongID, &chartModel.LevelStar, &chartModel.Attribute, &chartModel.Notes,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrSongNotFound
		}
		return nil, err
	}

	return &repository.WorldsendSongWithChart{
		Song:  songModel.ToEntity(),
		Chart: chartModel.ToEntity(),
	}, nil
}

// SaveSong は WORLD'S END 楽曲エンティティの現在の状態を永続化します。
// 対象が存在しない場合は ErrSongNotFound を返します。
func (r *worldsendChartRepository) SaveSong(ctx context.Context, exec repository.Executor, song *entity.Song) error {
	query := `
		UPDATE songs
		SET display_id = ?, title = ?, artist = ?, genre_id = ?, bpm = ?, released_at = ?, official_idx = ?, jacket = ?, is_deleted = ?
		WHERE id = ? AND is_worldsend = 1
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

// UpdateSongs は WORLD'S END 楽曲および譜面情報を一括更新します。
// トランザクション管理は呼び出し元で行う必要があります。
func (r *worldsendChartRepository) UpdateSongs(ctx context.Context, exec repository.Executor, songs []*entity.Song, charts []*entity.WorldsendChart) error {
	if len(songs) == 0 {
		return nil
	}

	displayIDs, err := collectUniqueDisplayIDs(songs)
	if err != nil {
		return err
	}

	targets, err := r.findUpdateTargetsByDisplayIDs(ctx, exec, displayIDs)
	if err != nil {
		return err
	}

	for _, displayID := range displayIDs {
		if _, ok := targets[displayID]; !ok {
			return fmt.Errorf("%w: display_id=%s", repository.ErrSongNotFound, displayID)
		}
	}

	if err := r.bulkUpdateSongs(ctx, exec, songs, targets); err != nil {
		return err
	}

	if err := r.bulkUpdateCharts(ctx, exec, songs, charts, targets); err != nil {
		return err
	}

	if err := r.ensureTargetsExist(ctx, exec, targets); err != nil {
		return err
	}

	return nil
}

type worldsendUpdateTarget struct {
	SongID  int
	ChartID int
}

func (r *worldsendChartRepository) findUpdateTargetsByDisplayIDs(ctx context.Context, exec repository.Executor, displayIDs []string) (map[string]worldsendUpdateTarget, error) {
	placeholders := make([]string, len(displayIDs))
	args := make([]any, 0, len(displayIDs))
	for i, displayID := range displayIDs {
		placeholders[i] = "?"
		args = append(args, displayID)
	}

	query := fmt.Sprintf(`
		SELECT s.display_id, s.id, wc.id
		FROM songs s
		INNER JOIN worldsend_charts wc ON s.id = wc.song_id
		WHERE s.is_worldsend = 1 AND s.display_id IN (%s)
	`, strings.Join(placeholders, ","))

	rows, err := exec.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	targets := make(map[string]worldsendUpdateTarget)
	for rows.Next() {
		var displayID string
		var songID, chartID int
		if err := rows.Scan(&displayID, &songID, &chartID); err != nil {
			return nil, err
		}
		targets[displayID] = worldsendUpdateTarget{SongID: songID, ChartID: chartID}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return targets, nil
}

func (r *worldsendChartRepository) bulkUpdateSongs(ctx context.Context, exec repository.Executor, songs []*entity.Song, targets map[string]worldsendUpdateTarget) error {
	var titleCases, artistCases, genreCases, bpmCases, releasedCases, jacketCases []string
	var titleArgs, artistArgs, genreArgs, bpmArgs, releasedArgs, jacketArgs []any
	songIDs := make([]int, 0, len(songs))

	for _, song := range songs {
		target := targets[song.DisplayID]
		songIDs = append(songIDs, target.SongID)

		titleCases = append(titleCases, "WHEN id = ? THEN ?")
		titleArgs = append(titleArgs, target.SongID, song.Title)

		artistCases = append(artistCases, "WHEN id = ? THEN ?")
		artistArgs = append(artistArgs, target.SongID, song.Artist)

		genreCases = append(genreCases, "WHEN id = ? THEN ?")
		genreArgs = append(genreArgs, target.SongID, song.GenreID)

		bpmCases = append(bpmCases, "WHEN id = ? THEN ?")
		bpmArgs = append(bpmArgs, target.SongID, song.BPM)

		releasedCases = append(releasedCases, "WHEN id = ? THEN ?")
		releasedArgs = append(releasedArgs, target.SongID, song.ReleasedAt)

		jacketCases = append(jacketCases, "WHEN id = ? THEN ?")
		jacketArgs = append(jacketArgs, target.SongID, song.Jacket)
	}

	args := make([]any, 0)
	args = append(args, titleArgs...)
	args = append(args, artistArgs...)
	args = append(args, genreArgs...)
	args = append(args, bpmArgs...)
	args = append(args, releasedArgs...)
	args = append(args, jacketArgs...)

	placeholders := make([]string, len(songIDs))
	for i, id := range songIDs {
		placeholders[i] = "?"
		args = append(args, id)
	}

	query := fmt.Sprintf(`
		UPDATE songs SET
			title = CASE %s END,
			artist = CASE %s END,
			genre_id = CASE %s END,
			bpm = CASE %s END,
			released_at = CASE %s END,
			jacket = CASE %s END
		WHERE is_worldsend = 1 AND id IN (%s)
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

func (r *worldsendChartRepository) bulkUpdateCharts(ctx context.Context, exec repository.Executor, songs []*entity.Song, charts []*entity.WorldsendChart, targets map[string]worldsendUpdateTarget) error {
	type chartUpdate struct {
		ChartID   int
		LevelStar *levelstar.LevelStar
		Attribute *string
		Notes     any
	}

	updates := make([]chartUpdate, 0, len(charts))
	for idx, chart := range charts {
		if chart == nil {
			continue
		}

		target := targets[songs[idx].DisplayID]
		updates = append(updates, chartUpdate{
			ChartID:   target.ChartID,
			LevelStar: chart.LevelStar,
			Attribute: chart.Attribute,
			Notes:     chart.Notes,
		})
	}

	if len(updates) == 0 {
		return nil
	}

	var levelCases, attributeCases, notesCases []string
	var levelArgs, attributeArgs, notesArgs []any
	chartIDs := make([]int, 0, len(updates))

	for _, update := range updates {
		chartIDs = append(chartIDs, update.ChartID)

		levelCases = append(levelCases, "WHEN id = ? THEN ?")
		levelArgs = append(levelArgs, update.ChartID, update.LevelStar)

		attributeCases = append(attributeCases, "WHEN id = ? THEN ?")
		attributeArgs = append(attributeArgs, update.ChartID, update.Attribute)

		notesCases = append(notesCases, "WHEN id = ? THEN ?")
		notesArgs = append(notesArgs, update.ChartID, update.Notes)
	}

	args := make([]any, 0)
	args = append(args, levelArgs...)
	args = append(args, attributeArgs...)
	args = append(args, notesArgs...)

	placeholders := make([]string, len(chartIDs))
	for i, id := range chartIDs {
		placeholders[i] = "?"
		args = append(args, id)
	}

	query := fmt.Sprintf(`
		UPDATE worldsend_charts SET
			level_star = CASE %s END,
			attribute = CASE %s END,
			notes = CASE %s END
		WHERE id IN (%s)
	`,
		strings.Join(levelCases, " "),
		strings.Join(attributeCases, " "),
		strings.Join(notesCases, " "),
		strings.Join(placeholders, ","),
	)

	_, err := exec.ExecContext(ctx, query, args...)
	return err
}

func (r *worldsendChartRepository) ensureTargetsExist(ctx context.Context, exec repository.Executor, targets map[string]worldsendUpdateTarget) error {
	if len(targets) == 0 {
		return nil
	}

	pairConditions := make([]string, 0, len(targets))
	args := make([]any, 0, len(targets)*2)
	for _, target := range targets {
		pairConditions = append(pairConditions, "(s.id = ? AND wc.id = ?)")
		args = append(args, target.SongID, target.ChartID)
	}

	query := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM songs s
		INNER JOIN worldsend_charts wc ON s.id = wc.song_id
		WHERE s.is_worldsend = 1 AND (%s)
	`, strings.Join(pairConditions, " OR "))

	var count int
	if err := exec.QueryRowxContext(ctx, query, args...).Scan(&count); err != nil {
		return err
	}
	if count != len(targets) {
		return repository.ErrSongNotFound
	}

	return nil
}
