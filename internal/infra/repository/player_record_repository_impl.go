package repository

import (
	"context"
	"time"

	"github.com/Qman110101/chunisupport-api/internal/domain/entity"
	"github.com/Qman110101/chunisupport-api/internal/domain/repository"
	"github.com/Qman110101/chunisupport-api/internal/domain/vo/chartconstant"
	"github.com/Qman110101/chunisupport-api/internal/domain/vo/notes"
	"github.com/Qman110101/chunisupport-api/internal/domain/vo/score"
	"github.com/jmoiron/sqlx"
)

// playerRecordRepository は PlayerRecordRepository の実装です。
type playerRecordRepository struct {
	db *sqlx.DB
}

// NewPlayerRecordRepository は PlayerRecordRepository の実装を生成します。
func NewPlayerRecordRepository(db *sqlx.DB) repository.PlayerRecordRepository {
	return &playerRecordRepository{db: db}
}

// playerRecordRow はDBからのプレイヤーレコード取得用のJOIN結果をマッピングする構造体です。
type playerRecordRow struct {
	PlayerID            int                         `db:"player_id"`
	ChartID             int                         `db:"chart_id"`
	Score               score.Score                 `db:"score"`
	ClearLampID         int                         `db:"clear_lamp_id"`
	ComboLampID         int                         `db:"combo_lamp_id"`
	FullChainID         int                         `db:"full_chain_id"`
	SlotID              int                         `db:"slot_id"`
	SlotOrder           *int                        `db:"slot_order"`
	UpdatedAt           time.Time                   `db:"updated_at"`
	ChartSongID         int                         `db:"chart_song_id"`
	ChartDifficultyID   int                         `db:"chart_difficulty_id"`
	ChartConst          chartconstant.ChartConstant `db:"chart_const"`
	ChartIsConstUnknown bool                        `db:"chart_is_const_unknown"`
	ChartNotes          *notes.Notes                `db:"chart_notes"`
	SongID              int                         `db:"song_id"`
	SongDisplayID       string                      `db:"song_display_id"`
	SongTitle           string                      `db:"song_title"`
	SongArtist          string                      `db:"song_artist"`
	SongGenreID         *int                        `db:"song_genre_id"`
	SongBPM             *int                        `db:"song_bpm"`
	SongReleasedAt      *time.Time                  `db:"song_released_at"`
	SongOfficialIdx     string                      `db:"song_official_idx"`
	SongJacket          *string                     `db:"song_jacket"`
	SongIsDeleted       bool                        `db:"song_is_deleted"`
	ClearLampName       string                      `db:"clear_lamp_name"`
	ComboLampName       string                      `db:"combo_lamp_name"`
	FullChainName       string                      `db:"full_chain_name"`
	SlotName            string                      `db:"slot_name"`
	DifficultyName      string                      `db:"difficulty_name"`
}

const playerRecordQuery = `
SELECT
    pr.player_id,
    pr.chart_id,
    pr.score,
    pr.clear_lamp_id,
    pr.combo_lamp_id,
    pr.full_chain_id,
    pr.slot_id,
    pr.slot_order,
    pr.updated_at,
    c.song_id AS chart_song_id,
    c.difficulty_id AS chart_difficulty_id,
    c.const AS chart_const,
    c.is_const_unknown AS chart_is_const_unknown,
    c.notes AS chart_notes,
    s.id AS song_id,
    s.display_id AS song_display_id,
    s.title AS song_title,
    s.artist AS song_artist,
    s.genre_id AS song_genre_id,
    s.bpm AS song_bpm,
    s.released_at AS song_released_at,
    s.official_idx AS song_official_idx,
    s.jacket AS song_jacket,
    s.is_deleted AS song_is_deleted,
    cl.name AS clear_lamp_name,
    co.name AS combo_lamp_name,
    fc.name AS full_chain_name,
    sl.name AS slot_name,
    diff.name AS difficulty_name
FROM player_records pr
INNER JOIN charts c ON pr.chart_id = c.id
INNER JOIN songs s ON c.song_id = s.id
INNER JOIN clear_lamp_types cl ON pr.clear_lamp_id = cl.id
INNER JOIN combo_lamp_types co ON pr.combo_lamp_id = co.id
INNER JOIN full_chain_types fc ON pr.full_chain_id = fc.id
INNER JOIN slots sl ON pr.slot_id = sl.id
INNER JOIN difficulties diff ON c.difficulty_id = diff.id
WHERE pr.player_id = ? AND s.is_deleted = 0
ORDER BY sl.id, pr.slot_order IS NULL, pr.slot_order, pr.updated_at DESC
`

const playerRecordRatingQuery = `
SELECT
    pr.player_id,
    pr.chart_id,
    pr.score,
    pr.clear_lamp_id,
    pr.combo_lamp_id,
    pr.full_chain_id,
    pr.slot_id,
    pr.slot_order,
    pr.updated_at,
    c.song_id AS chart_song_id,
    c.difficulty_id AS chart_difficulty_id,
    c.const AS chart_const,
    c.is_const_unknown AS chart_is_const_unknown,
    c.notes AS chart_notes,
    s.id AS song_id,
    s.display_id AS song_display_id,
    s.title AS song_title,
    s.artist AS song_artist,
    s.genre_id AS song_genre_id,
    s.bpm AS song_bpm,
    s.released_at AS song_released_at,
    s.official_idx AS song_official_idx,
    s.jacket AS song_jacket,
    s.is_deleted AS song_is_deleted,
    cl.name AS clear_lamp_name,
    co.name AS combo_lamp_name,
    fc.name AS full_chain_name,
    sl.name AS slot_name,
    diff.name AS difficulty_name
FROM player_records pr
INNER JOIN charts c ON pr.chart_id = c.id
INNER JOIN songs s ON c.song_id = s.id
INNER JOIN clear_lamp_types cl ON pr.clear_lamp_id = cl.id
INNER JOIN combo_lamp_types co ON pr.combo_lamp_id = co.id
INNER JOIN full_chain_types fc ON pr.full_chain_id = fc.id
INNER JOIN slots sl ON pr.slot_id = sl.id
INNER JOIN difficulties diff ON c.difficulty_id = diff.id
WHERE pr.player_id = ? AND s.is_deleted = 0
  AND sl.name IN ('best', 'best_candidate', 'new', 'new_candidate')
ORDER BY sl.id, pr.slot_order IS NULL, pr.slot_order, pr.updated_at DESC
`

// FindByPlayerID はプレイヤーIDでレコードを検索し、関連する譜面・楽曲・ランプ情報を含むエンティティを返します。
func (r *playerRecordRepository) FindByPlayerID(ctx context.Context, exec repository.Executor, playerID int) ([]*entity.PlayerRecord, error) {
	var rows []playerRecordRow
	if err := exec.SelectContext(ctx, &rows, playerRecordQuery, playerID); err != nil {
		return nil, err
	}

	return buildPlayerRecords(rows), nil
}

// FindByPlayerIDForRating はレーティング対象のレコードのみを取得します。
func (r *playerRecordRepository) FindByPlayerIDForRating(ctx context.Context, exec repository.Executor, playerID int) ([]*entity.PlayerRecord, error) {
	var rows []playerRecordRow
	if err := exec.SelectContext(ctx, &rows, playerRecordRatingQuery, playerID); err != nil {
		return nil, err
	}

	return buildPlayerRecords(rows), nil
}

// GetLastScoreUpdate はプレイヤーのスコア最終更新日時を取得します。
func (r *playerRecordRepository) GetLastScoreUpdate(ctx context.Context, exec repository.Executor, playerID int) (*time.Time, error) {
	const query = `
SELECT MAX(updated_at) AS last_update FROM (
    SELECT updated_at FROM player_records WHERE player_id = ?
    UNION ALL
    SELECT updated_at FROM player_worldsend_records WHERE player_id = ?
) AS combined
`

	var lastUpdate *time.Time
	if err := exec.GetContext(ctx, &lastUpdate, query, playerID, playerID); err != nil {
		return nil, err
	}

	return lastUpdate, nil
}

func buildPlayerRecords(rows []playerRecordRow) []*entity.PlayerRecord {
	records := make([]*entity.PlayerRecord, 0, len(rows))
	for _, row := range rows {
		record := &entity.PlayerRecord{
			PlayerID:    row.PlayerID,
			ChartID:     row.ChartID,
			Score:       row.Score,
			ClearLampID: row.ClearLampID,
			ComboLampID: row.ComboLampID,
			FullChainID: row.FullChainID,
			SlotID:      row.SlotID,
			SlotOrder:   row.SlotOrder,
			UpdatedAt:   row.UpdatedAt,
			Chart: &entity.Chart{
				ID:             row.ChartID,
				SongID:         row.ChartSongID,
				DifficultyID:   row.ChartDifficultyID,
				Const:          row.ChartConst,
				IsConstUnknown: row.ChartIsConstUnknown,
				Notes:          row.ChartNotes,
			},
			Song: &entity.Song{
				ID:          row.SongID,
				DisplayID:   row.SongDisplayID,
				Title:       row.SongTitle,
				Artist:      row.SongArtist,
				GenreID:     row.SongGenreID,
				BPM:         row.SongBPM,
				ReleasedAt:  row.SongReleasedAt,
				OfficialIdx: row.SongOfficialIdx,
				Jacket:      row.SongJacket,
				IsDeleted:   row.SongIsDeleted,
			},
			ClearLamp: &entity.ClearLampType{
				ID:   row.ClearLampID,
				Name: row.ClearLampName,
			},
			ComboLamp: &entity.ComboLampType{
				ID:   row.ComboLampID,
				Name: row.ComboLampName,
			},
			FullChain: &entity.FullChainType{
				ID:   row.FullChainID,
				Name: row.FullChainName,
			},
			Slot: &entity.Slot{
				ID:   row.SlotID,
				Name: row.SlotName,
			},
			ChartDifficulty: &entity.ChartDifficulty{
				ID:   row.ChartDifficultyID,
				Name: row.DifficultyName,
			},
		}

		records = append(records, record)
	}

	return records
}
