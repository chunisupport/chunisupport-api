package repository

import (
	"context"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/master"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/notes"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/score"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlayerRecordRepository_FindByPlayerIDForRating_対象枠だけを順序付きで関連復元する(t *testing.T) {
	// Given: 対象4枠、対象外枠、削除曲、他プレイヤーを混在させる。
	db := setupTestDB(t)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	setupPlayerRecordRepositoryDB(t, db)
	_, err := db.Exec(`
		CREATE TABLE clear_lamp_types (id INTEGER PRIMARY KEY, name TEXT NOT NULL);
		CREATE TABLE combo_lamp_types (id INTEGER PRIMARY KEY, name TEXT NOT NULL);
		CREATE TABLE full_chain_types (id INTEGER PRIMARY KEY, name TEXT NOT NULL);
		CREATE TABLE slots (id INTEGER PRIMARY KEY, name TEXT NOT NULL);
		INSERT INTO clear_lamp_types VALUES (5, 'ABSOLUTE');
		INSERT INTO combo_lamp_types VALUES (3, 'ALL JUSTICE');
		INSERT INTO full_chain_types VALUES (2, 'FULL CHAIN');
		INSERT INTO slots VALUES (1, 'none'), (2, 'best'), (3, 'best_candidate'), (4, 'new'), (5, 'new_candidate');
		INSERT INTO songs (id, display_id, title, artist, genre_id, bpm, official_idx, jacket, is_deleted) VALUES
			(11, 'RATING001', '評価曲', '作曲者', 2, 187, '12345', 'rating.png', 0),
			(12, 'DELETED001', '削除曲', '別作者', 1, NULL, '54321', NULL, 1),
			(13, 'OTHER001', '枠外曲', '作者', 1, NULL, '99999', NULL, 0);
		INSERT INTO charts (id, song_id, difficulty_id, const, is_const_unknown, notes) VALUES
			(101, 11, 4, 14.5, 0, 1234), (102, 11, 3, 13.0, 1, NULL),
			(103, 11, 2, 10.0, 0, 800), (104, 11, 1, 5.0, 0, 400),
			(105, 11, 5, 15.0, 0, 1600), (106, 12, 4, 14.0, 0, 1000), (999, 13, 3, 12.0, 0, 900);
		INSERT INTO player_records VALUES
			(10, 101, 1001234, 5, 3, 2, 2, 2, '2026-11-01 10:00:00'),
			(10, 102, 999999, 5, 3, 2, 2, 1, '2026-11-01 09:00:00'),
			(10, 103, 998000, 5, 3, 2, 3, NULL, '2026-11-01 08:00:00'),
			(10, 104, 997000, 5, 3, 2, 4, 1, '2026-11-01 07:00:00'),
			(10, 105, 996000, 5, 3, 2, 5, NULL, '2026-11-01 06:00:00'),
			(10, 106, 995000, 5, 3, 2, 2, 1, '2026-11-01 05:00:00'),
			(20, 101, 994000, 5, 3, 2, 2, 1, '2026-11-01 04:00:00'),
			(10, 999, 993000, 5, 3, 2, 1, NULL, '2026-11-01 03:00:00');
	`)
	require.NoError(t, err)

	// When
	records, err := NewPlayerRecordRepository(db).FindByPlayerIDForRating(context.Background(), db, 10)

	// Then
	require.NoError(t, err)
	require.Len(t, records, 5)
	ids := make([]int, len(records))
	for i, record := range records {
		ids[i] = record.ChartID
	}
	assert.Equal(t, []int{102, 101, 103, 104, 105}, ids)
	constant, err := chartconstant.NewChartConstant(14.5)
	require.NoError(t, err)
	noteCount, err := notes.NewNotes(1234)
	require.NoError(t, err)
	savedScore, err := score.NewScore(1001234)
	require.NoError(t, err)
	genre, bpm, order, jacket := 2, 187, 2, "rating.png"
	assert.Equal(t, &entity.PlayerRecord{
		PlayerID: 10, ChartID: 101, Score: savedScore,
		ClearLampID: 5, ComboLampID: 3, FullChainID: 2, SlotID: 2, SlotOrder: &order,
		UpdatedAt:       time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC),
		Chart:           &entity.Chart{ID: 101, SongID: 11, DifficultyID: 4, Const: constant, Notes: &noteCount},
		Song:            &entity.Song{ID: 11, DisplayID: "RATING001", Title: "評価曲", Artist: "作曲者", GenreID: &genre, BPM: &bpm, OfficialIdx: "12345", Jacket: &jacket, Charts: []*entity.Chart{}},
		ClearLamp:       &master.ClearLampType{ID: 5, Name: "ABSOLUTE"},
		ComboLamp:       &master.ComboLampType{ID: 3, Name: "ALL JUSTICE"},
		FullChain:       &master.FullChainType{ID: 2, Name: "FULL CHAIN"},
		Slot:            &master.Slot{ID: 2, Name: "best"},
		ChartDifficulty: &master.ChartDifficulty{ID: 4, Name: "MASTER"},
	}, records[1])
	assert.True(t, records[0].Chart.IsConstUnknown)
	assert.Nil(t, records[0].Chart.Notes)
	assert.Nil(t, records[2].SlotOrder)

	empty, err := NewPlayerRecordRepository(db).FindByPlayerIDForRating(context.Background(), db, 999)
	require.NoError(t, err)
	assert.Empty(t, empty)
}
