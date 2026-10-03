package repository

import (
	"context"
	"testing"
	"time"

	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func setupChunirecProfileQueryDB(t *testing.T) *sqlx.DB {
	t.Helper()

	db, err := sqlx.Open("sqlite", ":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { require.NoError(t, db.Close()) })

	for _, statement := range []string{
		`CREATE TABLE players (
			id INTEGER PRIMARY KEY,
			player_name TEXT NOT NULL,
			player_level INTEGER NOT NULL,
			calculated_player_rating REAL,
			class_emblem_id INTEGER,
			class_emblem_base_id INTEGER,
			updated_at DATETIME NOT NULL
		)`,
		`CREATE TABLE player_honors (player_id INTEGER NOT NULL, honor_id INTEGER NOT NULL, slot INTEGER NOT NULL)`,
		`CREATE TABLE honors (id INTEGER PRIMARY KEY, name TEXT NOT NULL, honor_type_id INTEGER NOT NULL)`,
		`CREATE TABLE honor_types (id INTEGER PRIMARY KEY, name TEXT NOT NULL)`,
	} {
		_, err := db.Exec(statement)
		require.NoError(t, err)
	}
	return db
}

func setupChunirecRecordsQueryDB(t *testing.T) *sqlx.DB {
	t.Helper()

	db, err := sqlx.Open("sqlite", ":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { require.NoError(t, db.Close()) })

	for _, statement := range []string{
		`CREATE TABLE players (id INTEGER PRIMARY KEY)`,
		`CREATE TABLE genres (id INTEGER PRIMARY KEY, name TEXT NOT NULL)`,
		`CREATE TABLE difficulties (id INTEGER PRIMARY KEY, name TEXT NOT NULL, sort_order INTEGER NOT NULL)`,
		`CREATE TABLE songs (
			id INTEGER PRIMARY KEY,
			display_id TEXT NOT NULL,
			title TEXT NOT NULL,
			genre_id INTEGER,
			is_worldsend INTEGER NOT NULL,
			is_deleted INTEGER NOT NULL
		)`,
		`CREATE TABLE charts (
			id INTEGER PRIMARY KEY,
			song_id INTEGER NOT NULL,
			difficulty_id INTEGER NOT NULL,
			const REAL NOT NULL,
			is_const_unknown INTEGER NOT NULL
		)`,
		`CREATE TABLE slots (id INTEGER PRIMARY KEY, name TEXT NOT NULL)`,
		`CREATE TABLE clear_lamp_types (id INTEGER PRIMARY KEY, name TEXT)`,
		`CREATE TABLE combo_lamp_types (id INTEGER PRIMARY KEY, name TEXT)`,
		`CREATE TABLE full_chain_types (id INTEGER PRIMARY KEY, name TEXT)`,
		`CREATE TABLE player_records (
			player_id INTEGER NOT NULL,
			chart_id INTEGER NOT NULL,
			score INTEGER NOT NULL,
			clear_lamp_id INTEGER NOT NULL,
			combo_lamp_id INTEGER NOT NULL,
			full_chain_id INTEGER NOT NULL,
			slot_id INTEGER NOT NULL,
			updated_at DATETIME NOT NULL
		)`,
	} {
		_, err := db.Exec(statement)
		require.NoError(t, err)
	}

	_, err = db.Exec(`
		INSERT INTO players (id) VALUES (7);
		INSERT INTO genres (id, name) VALUES (1, 'POPS'), (2, 'ORIGINAL');
		INSERT INTO difficulties (id, name, sort_order) VALUES
			(10, 'ADVANCED', 2), (20, 'BASIC', 1), (30, 'MASTER', 3), (40, 'EXPERT', 4);
		INSERT INTO songs (id, display_id, title, genre_id, is_worldsend, is_deleted) VALUES
			(1, 'SONG1', '曲1', 1, 0, 0),
			(2, 'SONG2', '曲2', NULL, 0, 0),
			(3, 'DELETED', '削除済み', 2, 0, 1),
			(4, 'WORLD', 'WORLD''S END', 2, 1, 0),
			(5, 'ZERO', '更新日時なし', 2, 0, 0);
		INSERT INTO charts (id, song_id, difficulty_id, const, is_const_unknown) VALUES
			(101, 1, 10, 14.5, 0), (102, 1, 20, 13.7, 1), (103, 2, 30, 15.0, 0),
			(104, 3, 10, 12.0, 0), (105, 4, 10, 13.0, 0), (106, 5, 40, 12.5, 0);
		INSERT INTO slots (id, name) VALUES (1, 'best');
		INSERT INTO clear_lamp_types (id, name) VALUES (1, 'CLEAR'), (2, 'NONE'), (3, '');
		INSERT INTO combo_lamp_types (id, name) VALUES (1, 'ALL JUSTICE'), (2, 'none'), (3, '');
		INSERT INTO full_chain_types (id, name) VALUES (1, 'NONE'), (2, 'FULL CHAIN'), (3, '');
	`)
	require.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO player_records (
			player_id, chart_id, score, clear_lamp_id, combo_lamp_id, full_chain_id, slot_id, updated_at
		) VALUES
			(7, 101, 1009000, 1, 1, 1, 1, ?),
			(7, 102, 0, 2, 2, 3, 1, ?),
			(7, 103, 990000, 3, 3, 2, 1, ?),
			(7, 104, 1000000, 1, 1, 2, 1, ?),
			(7, 105, 1000000, 1, 1, 2, 1, ?),
			(7, 106, 1000000, 1, 1, 2, 1, ?)
	`,
		time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC),
		time.Date(2026, 9, 20, 11, 0, 0, 0, time.UTC),
		time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC),
		time.Date(2026, 9, 20, 13, 0, 0, 0, time.UTC),
		time.Date(2026, 9, 20, 14, 0, 0, 0, time.UTC),
		time.Time{},
	)
	require.NoError(t, err)
	return db
}

func TestChunirecQueryService_FindProfileByPlayerID_プロフィールとスロット1の称号だけを返す(t *testing.T) {
	db := setupChunirecProfileQueryDB(t)
	updatedAt := time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC)
	_, err := db.Exec(`
		INSERT INTO players (id, player_name, player_level, calculated_player_rating, class_emblem_id, class_emblem_base_id, updated_at)
		VALUES (1, 'プレイヤー', 100, 16.25, 12, 34, ?);
		INSERT INTO honor_types (id, name) VALUES (1, 'platinum'), (2, 'silver');
		INSERT INTO honors (id, name, honor_type_id) VALUES (10, '先頭称号', 1), (20, '2番目', 2);
		INSERT INTO player_honors (player_id, honor_id, slot) VALUES (1, 10, 1), (1, 20, 2);
	`, updatedAt)
	require.NoError(t, err)

	profile, err := NewChunirecQueryService(db).FindProfileByPlayerID(context.Background(), 1)

	require.NoError(t, err)
	require.NotNil(t, profile)
	assert.Equal(t, "プレイヤー", profile.Name)
	assert.Equal(t, 100, profile.Level)
	require.NotNil(t, profile.Rating)
	assert.InDelta(t, 16.25, *profile.Rating, 0.0001)
	require.NotNil(t, profile.ClassEmblemID)
	assert.Equal(t, 12, *profile.ClassEmblemID)
	require.NotNil(t, profile.ClassEmblemBaseID)
	assert.Equal(t, 34, *profile.ClassEmblemBaseID)
	require.NotNil(t, profile.Title)
	assert.Equal(t, "先頭称号", *profile.Title)
	require.NotNil(t, profile.TitleRarity)
	assert.Equal(t, "platinum", *profile.TitleRarity)
	assert.True(t, updatedAt.Equal(profile.UpdatedAt))
}

func TestChunirecQueryService_FindProfileByPlayerID_未連携ならnilで任意情報もnilを保つ(t *testing.T) {
	db := setupChunirecProfileQueryDB(t)
	_, err := db.Exec(`
		INSERT INTO players (id, player_name, player_level, updated_at)
		VALUES (2, '称号なし', 1, '2026-09-20T10:00:00Z');
		INSERT INTO players (id, player_name, player_level, updated_at)
		VALUES (3, '称号不整合', 1, '2026-09-20T10:00:00Z');
		INSERT INTO honors (id, name, honor_type_id) VALUES (30, 'typeなし', 999);
		INSERT INTO player_honors (player_id, honor_id, slot) VALUES (3, 30, 1);
	`)
	require.NoError(t, err)
	query := NewChunirecQueryService(db)

	missing, err := query.FindProfileByPlayerID(context.Background(), 999)
	require.NoError(t, err)
	assert.Nil(t, missing)

	profile, err := query.FindProfileByPlayerID(context.Background(), 2)
	require.NoError(t, err)
	require.NotNil(t, profile)
	assert.Nil(t, profile.Rating)
	assert.Nil(t, profile.ClassEmblemID)
	assert.Nil(t, profile.ClassEmblemBaseID)
	assert.Nil(t, profile.Title)
	assert.Nil(t, profile.TitleRarity)

	profile, err = query.FindProfileByPlayerID(context.Background(), 3)
	require.NoError(t, err)
	require.NotNil(t, profile)
	assert.Nil(t, profile.Title)
	assert.Nil(t, profile.TitleRarity)
}

func TestChunirecQueryService_ListRecordsByPlayerID_プレイ済みの通常譜面を並び順と互換lampで返す(t *testing.T) {
	db := setupChunirecRecordsQueryDB(t)

	records, err := NewChunirecQueryService(db).ListRecordsByPlayerID(context.Background(), 7)

	require.NoError(t, err)
	require.Len(t, records, 3)
	assert.Equal(t, []string{"SONG1", "SONG1", "SONG2"}, []string{records[0].ID, records[1].ID, records[2].ID})
	assert.Equal(t, []string{"BASIC", "ADVANCED", "MASTER"}, []string{records[0].Difficulty, records[1].Difficulty, records[2].Difficulty})
	assert.Equal(t, "曲1", records[0].Title)
	assert.Equal(t, "POPS", records[0].Genre)
	assert.InDelta(t, 13.7, records[0].Const.Float64(), 0.0001)
	assert.True(t, records[0].IsConstUnknown)
	assert.Zero(t, records[0].Score)
	assert.Nil(t, records[0].ClearLamp)
	assert.Nil(t, records[0].ComboLamp)
	assert.Nil(t, records[0].FullChain)
	require.NotNil(t, records[1].ClearLamp)
	assert.Equal(t, "CLEAR", *records[1].ClearLamp)
	require.NotNil(t, records[1].ComboLamp)
	assert.Equal(t, "ALL JUSTICE", *records[1].ComboLamp)
	assert.Nil(t, records[1].FullChain)
	assert.Equal(t, "", records[2].Genre)
	assert.Nil(t, records[2].ClearLamp)
	assert.Nil(t, records[2].ComboLamp)
	require.NotNil(t, records[2].FullChain)
	assert.Equal(t, "FULL CHAIN", *records[2].FullChain)
	assert.True(t, records[2].UpdatedAt.After(records[1].UpdatedAt))
}

func TestChunirecQueryService_ListRecordsByPlayerID_対象外と更新日時ゼロを除外する(t *testing.T) {
	db := setupChunirecRecordsQueryDB(t)
	query := NewChunirecQueryService(db)

	records, err := query.ListRecordsByPlayerID(context.Background(), 7)

	require.NoError(t, err)
	for _, record := range records {
		assert.NotContains(t, []string{"DELETED", "WORLD", "ZERO"}, record.ID)
	}

	missing, err := query.ListRecordsByPlayerID(context.Background(), 999)
	require.NoError(t, err)
	assert.Empty(t, missing)
}

func TestChunirecQueryService_操作をキャンセルすると原因エラーを保持する(t *testing.T) {
	db := setupChunirecRecordsQueryDB(t)
	query := NewChunirecQueryService(db)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := query.ListRecordsByPlayerID(ctx, 7)

	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
	assert.ErrorIs(t, err, domainrepo.ErrRepositoryOperationFailed)
}
