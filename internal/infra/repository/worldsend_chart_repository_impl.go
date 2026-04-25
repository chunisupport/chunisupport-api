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

// FindAll は全 WORLD'S END 楽曲を譜面情報付きで取得します。
func (r *worldsendChartRepository) FindAll(ctx context.Context, exec repository.Executor, includeDeleted bool) ([]*entity.WorldsendSongWithChart, error) {
	query := `
		SELECT
			s.id, s.display_id, s.title, s.artist, s.genre_id, s.bpm, s.released_at, s.official_idx, s.jacket, s.is_worldsend, s.is_deleted, s.updated_at,
			wc.id AS 'worldsend_charts.id',
			wc.song_id AS 'worldsend_charts.song_id',
			wc.level_star AS 'worldsend_charts.level_star',
			wc.attribute AS 'worldsend_charts.attribute',
			wc.notes AS 'worldsend_charts.notes',
			wc.notes_designer AS 'worldsend_charts.notes_designer',
			wc.updated_at AS 'worldsend_charts.updated_at'
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

	results := []*entity.WorldsendSongWithChart{}
	for rows.Next() {
		var songModel models.SongModel
		var chartModel models.WorldsendChartModel

		err := rows.Scan(
			&songModel.ID, &songModel.DisplayID, &songModel.Title, &songModel.Artist,
			&songModel.GenreID, &songModel.BPM, &songModel.ReleasedAt, &songModel.OfficialIdx,
			&songModel.Jacket, &songModel.IsWorldsend, &songModel.IsDeleted, &songModel.UpdatedAt,
			&chartModel.ID, &chartModel.SongID, &chartModel.LevelStar, &chartModel.Attribute, &chartModel.Notes, &chartModel.NotesDesigner, &chartModel.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		results = append(results, &entity.WorldsendSongWithChart{
			Song:  songModel.ToEntity(),
			Chart: chartModel.ToEntity(),
		})
	}

	return results, rows.Err()
}

// FindByDisplayID は指定された DisplayID の WORLD'S END 楽曲を取得します。
func (r *worldsendChartRepository) FindByDisplayID(ctx context.Context, exec repository.Executor, displayID string) (*entity.WorldsendSongWithChart, error) {
	query := `
		SELECT
			s.id, s.display_id, s.title, s.artist, s.genre_id, s.bpm, s.released_at, s.official_idx, s.jacket, s.is_worldsend, s.is_deleted, s.updated_at,
			wc.id AS 'worldsend_charts.id',
			wc.song_id AS 'worldsend_charts.song_id',
			wc.level_star AS 'worldsend_charts.level_star',
			wc.attribute AS 'worldsend_charts.attribute',
			wc.notes AS 'worldsend_charts.notes',
			wc.notes_designer AS 'worldsend_charts.notes_designer',
			wc.updated_at AS 'worldsend_charts.updated_at'
		FROM songs s
		INNER JOIN worldsend_charts wc ON s.id = wc.song_id
		WHERE s.display_id = ? AND s.is_worldsend = 1`

	var songModel models.SongModel
	var chartModel models.WorldsendChartModel

	err := exec.QueryRowxContext(ctx, query, displayID).Scan(
		&songModel.ID, &songModel.DisplayID, &songModel.Title, &songModel.Artist,
		&songModel.GenreID, &songModel.BPM, &songModel.ReleasedAt, &songModel.OfficialIdx,
		&songModel.Jacket, &songModel.IsWorldsend, &songModel.IsDeleted, &songModel.UpdatedAt,
		&chartModel.ID, &chartModel.SongID, &chartModel.LevelStar, &chartModel.Attribute, &chartModel.Notes, &chartModel.NotesDesigner, &chartModel.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrSongNotFound
		}
		return nil, err
	}

	return &entity.WorldsendSongWithChart{
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
func (r *worldsendChartRepository) UpdateSongs(ctx context.Context, exec repository.Executor, updates []*repository.WorldsendUpdate) error {
	if len(updates) == 0 {
		return nil
	}

	songs, err := collectSongsFromWorldsendUpdates(updates)
	if err != nil {
		return err
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

	songRowsAffected, err := r.bulkUpdateSongs(ctx, exec, songs, targets)
	if err != nil {
		return err
	}

	chartRowsAffected, expectedChartUpdates, err := r.bulkUpdateCharts(ctx, exec, updates, targets)
	if err != nil {
		return err
	}

	// RowsAffected はドライバごとの差異があるため、不一致時は存在確認クエリで最終判定する。
	rowsAffectedMismatch := songRowsAffected != int64(len(songs)) || chartRowsAffected != int64(expectedChartUpdates)
	// 一部リクエストで譜面更新が無い場合、RowsAffected だけでは songs 更新後に発生した
	// 並行削除(例: 更新対象外 chart の削除)を検知できないため存在確認を行う。
	requiresExistenceCheckForSkippedChartUpdates := expectedChartUpdates < len(updates)

	if rowsAffectedMismatch || requiresExistenceCheckForSkippedChartUpdates {
		exists, err := r.ensureTargetsExist(ctx, exec, targets)
		if err != nil {
			return err
		}
		if !exists {
			return repository.ErrSongNotFound
		}
	}

	return nil
}

func collectSongsFromWorldsendUpdates(updates []*repository.WorldsendUpdate) ([]*entity.Song, error) {
	songs := make([]*entity.Song, 0, len(updates))
	for i, update := range updates {
		if update == nil || update.Song == nil {
			return nil, fmt.Errorf("updates[%d].song is nil", i)
		}
		songs = append(songs, update.Song)
	}

	return songs, nil
}

type worldsendUpdateTarget struct {
	SongID  int
	ChartID int
}

func (r *worldsendChartRepository) findUpdateTargetsByDisplayIDs(ctx context.Context, exec repository.Executor, displayIDs []string) (map[string]worldsendUpdateTarget, error) {
	if len(displayIDs) == 0 {
		return map[string]worldsendUpdateTarget{}, nil
	}

	query, args, err := sqlx.In(`
		SELECT s.display_id, s.id, wc.id
		FROM songs s
		INNER JOIN worldsend_charts wc ON s.id = wc.song_id
		WHERE s.is_worldsend = 1 AND s.is_deleted = 0 AND s.display_id IN (?)
	`, displayIDs)
	if err != nil {
		return nil, err
	}
	query = exec.Rebind(query)

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

func (r *worldsendChartRepository) bulkUpdateSongs(ctx context.Context, exec repository.Executor, songs []*entity.Song, targets map[string]worldsendUpdateTarget) (int64, error) {
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

	result, err := exec.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return rowsAffected, nil
}

func (r *worldsendChartRepository) bulkUpdateCharts(ctx context.Context, exec repository.Executor, updates []*repository.WorldsendUpdate, targets map[string]worldsendUpdateTarget) (int64, int, error) {
	type chartUpdate struct {
		ChartID       int
		LevelStar     *levelstar.LevelStar
		Attribute     *string
		Notes         any
		NotesDesigner *string
	}

	chartUpdates := make([]chartUpdate, 0, len(updates))
	for _, update := range updates {
		if update == nil || update.Chart == nil {
			continue
		}

		target := targets[update.Song.DisplayID]
		chartUpdates = append(chartUpdates, chartUpdate{
			ChartID:       target.ChartID,
			LevelStar:     update.Chart.LevelStar,
			Attribute:     update.Chart.Attribute,
			Notes:         update.Chart.Notes,
			NotesDesigner: update.Chart.NotesDesigner,
		})
	}

	if len(chartUpdates) == 0 {
		return 0, 0, nil
	}

	var levelCases, attributeCases, notesCases, notesDesignerCases []string
	var levelArgs, attributeArgs, notesArgs, notesDesignerArgs []any
	chartIDs := make([]int, 0, len(chartUpdates))

	for _, update := range chartUpdates {
		chartIDs = append(chartIDs, update.ChartID)

		levelCases = append(levelCases, "WHEN id = ? THEN ?")
		levelArgs = append(levelArgs, update.ChartID, update.LevelStar)

		attributeCases = append(attributeCases, "WHEN id = ? THEN ?")
		attributeArgs = append(attributeArgs, update.ChartID, update.Attribute)

		notesCases = append(notesCases, "WHEN id = ? THEN ?")
		notesArgs = append(notesArgs, update.ChartID, update.Notes)

		notesDesignerCases = append(notesDesignerCases, "WHEN id = ? THEN ?")
		notesDesignerArgs = append(notesDesignerArgs, update.ChartID, update.NotesDesigner)
	}

	args := make([]any, 0)
	args = append(args, levelArgs...)
	args = append(args, attributeArgs...)
	args = append(args, notesArgs...)
	args = append(args, notesDesignerArgs...)

	placeholders := make([]string, len(chartIDs))
	for i, id := range chartIDs {
		placeholders[i] = "?"
		args = append(args, id)
	}

	query := fmt.Sprintf(`
		UPDATE worldsend_charts SET
			level_star = CASE %s END,
			attribute = CASE %s END,
			notes = CASE %s END,
			notes_designer = CASE %s END
		WHERE id IN (%s)
	`,
		strings.Join(levelCases, " "),
		strings.Join(attributeCases, " "),
		strings.Join(notesCases, " "),
		strings.Join(notesDesignerCases, " "),
		strings.Join(placeholders, ","),
	)

	result, err := exec.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, 0, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, 0, err
	}

	return rowsAffected, len(chartUpdates), nil
}

func (r *worldsendChartRepository) ensureTargetsExist(ctx context.Context, exec repository.Executor, targets map[string]worldsendUpdateTarget) (bool, error) {
	if len(targets) == 0 {
		return true, nil
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
		WHERE s.is_worldsend = 1 AND s.is_deleted = 0 AND (%s)
	`, strings.Join(pairConditions, " OR "))

	var count int
	if err := exec.QueryRowxContext(ctx, query, args...).Scan(&count); err != nil {
		return false, err
	}

	return count == len(targets), nil
}

// CreateSong は新規 WORLD'S END 楽曲を songs および worldsend_charts テーブルに追加します。
// worldsend_charts は 1 曲 1 行が必須のため、chart が nil の場合でも空行を挿入します。
// official_idx 重複時は ErrDuplicateOfficialIdx を返します。
func (r *worldsendChartRepository) CreateSong(ctx context.Context, exec repository.Executor, song *entity.Song, chart *entity.WorldsendChart) (*entity.WorldsendSongWithChart, error) {
	// songs テーブルに挿入
	songResult, err := exec.ExecContext(ctx, `
		INSERT INTO songs (display_id, title, artist, genre_id, bpm, released_at, official_idx, jacket, is_worldsend, is_deleted)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, 1, 0)
	`,
		song.DisplayID,
		song.Title,
		song.Artist,
		song.GenreID,
		song.BPM,
		song.ReleasedAt,
		song.OfficialIdx,
		song.Jacket,
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

	// worldsend_charts テーブルに挿入（chart が nil の場合も空行を挿入）
	var attribute *string
	var levelStarVal *int
	var notesVal *int
	var notesDesigner *string
	if chart != nil {
		attribute = chart.Attribute
		if chart.LevelStar != nil {
			v := int(*chart.LevelStar)
			levelStarVal = &v
		}
		if chart.Notes != nil {
			n := int(*chart.Notes)
			notesVal = &n
		}
		notesDesigner = chart.NotesDesigner
	}

	if _, err = exec.ExecContext(ctx, `
		INSERT INTO worldsend_charts (song_id, level_star, attribute, notes, notes_designer)
		VALUES (?, ?, ?, ?, ?)
	`,
		songID,
		levelStarVal,
		attribute,
		notesVal,
		notesDesigner,
	); err != nil {
		return nil, err
	}

	// DB が付与した updated_at を取得するため再フェッチする
	return r.FindByDisplayID(ctx, exec, song.DisplayID)
}
