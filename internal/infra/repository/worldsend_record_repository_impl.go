package repository

import (
	"context"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/notes"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/score"
	"github.com/jmoiron/sqlx"
)

// worldsendRecordRepository は WorldsendRecordRepository の実装です。
type worldsendRecordRepository struct {
	db *sqlx.DB
}

// NewWorldsendRecordRepository は WorldsendRecordRepository の実装を生成します。
func NewWorldsendRecordRepository(db *sqlx.DB) repository.WorldsendRecordRepository {
	return &worldsendRecordRepository{db: db}
}

// worldsendRecordRow は DB から WORLD'S END レコード取得用の JOIN 結果をマッピングする構造体です。
type worldsendRecordRow struct {
	PlayerID         int          `db:"player_id"`
	WorldsendChartID int          `db:"worldsend_chart_id"`
	Score            uint32       `db:"score"`
	ClearLampID      int          `db:"clear_lamp_id"`
	ComboLampID      int          `db:"combo_lamp_id"`
	FullChainID      int          `db:"full_chain_id"`
	UpdatedAt        time.Time    `db:"updated_at"`
	ChartSongID      int          `db:"chart_song_id"`
	ChartWeStar      *int         `db:"chart_we_star"`
	ChartWeKanji     *string      `db:"chart_we_kanji"`
	ChartNotes       *notes.Notes `db:"chart_notes"`
	SongID           int          `db:"song_id"`
	SongDisplayID    string       `db:"song_display_id"`
	SongTitle        string       `db:"song_title"`
	SongArtist       string       `db:"song_artist"`
	SongGenreID      *int         `db:"song_genre_id"`
	SongBPM          *int         `db:"song_bpm"`
	SongReleasedAt   *time.Time   `db:"song_released_at"`
	SongOfficialIdx  string       `db:"song_official_idx"`
	SongJacket       *string      `db:"song_jacket"`
	SongIsDeleted    bool         `db:"song_is_deleted"`
	ClearLampName    string       `db:"clear_lamp_name"`
	ComboLampName    string       `db:"combo_lamp_name"`
	FullChainName    string       `db:"full_chain_name"`
}

const worldsendRecordQuery = `
SELECT
    pwr.player_id,
    pwr.worldsend_chart_id,
    pwr.score,
    pwr.clear_lamp_id,
    pwr.combo_lamp_id,
    pwr.full_chain_id,
    pwr.updated_at,
    wc.song_id AS chart_song_id,
    wc.we_star AS chart_we_star,
    wc.we_kanji AS chart_we_kanji,
    wc.notes AS chart_notes,
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
    fc.name AS full_chain_name
FROM player_worldsend_records pwr
INNER JOIN worldsend_charts wc ON pwr.worldsend_chart_id = wc.id
INNER JOIN songs s ON wc.song_id = s.id
INNER JOIN clear_lamp_types cl ON pwr.clear_lamp_id = cl.id
INNER JOIN combo_lamp_types co ON pwr.combo_lamp_id = co.id
INNER JOIN full_chain_types fc ON pwr.full_chain_id = fc.id
WHERE pwr.player_id = ? AND s.is_deleted = 0
ORDER BY pwr.updated_at DESC
`

// FindByPlayerID はプレイヤーID で WORLD'S END レコードを検索し、関連する譜面・楽曲・ランプ情報を含むエンティティを返します。
func (r *worldsendRecordRepository) FindByPlayerID(ctx context.Context, exec repository.Executor, playerID int) ([]*entity.PlayerWorldsendRecord, error) {
	var rows []worldsendRecordRow
	if err := exec.SelectContext(ctx, &rows, worldsendRecordQuery, playerID); err != nil {
		return nil, err
	}

	records := make([]*entity.PlayerWorldsendRecord, 0, len(rows))
	for _, row := range rows {
		s, err := score.NewScore(row.Score)
		if err != nil {
			return nil, err
		}

		record := &entity.PlayerWorldsendRecord{
			PlayerID:         row.PlayerID,
			WorldsendChartID: row.WorldsendChartID,
			Score:            s,
			ClearLampID:      row.ClearLampID,
			ComboLampID:      row.ComboLampID,
			FullChainID:      row.FullChainID,
			UpdatedAt:        row.UpdatedAt,
			WorldsendChart: &entity.WorldsendChart{
				ID:      row.WorldsendChartID,
				SongID:  row.ChartSongID,
				WeStar:  row.ChartWeStar,
				WeKanji: row.ChartWeKanji,
				Notes:   row.ChartNotes,
			},
			// PlayerWorldsendRecord内のSongは楽曲メタデータの参照であり、完全な集約ではない。
			// WORLD'S ENDの譜面情報はWorldsendChartで保持するため、Chartsは空スライスで初期化する。
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
				Charts:      []*entity.Chart{},
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
		}
		records = append(records, record)
	}

	return records, nil
}
