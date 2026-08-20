package repository

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/playername"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestUserDataTransferRepositoryImportSnapshotCommitsAtomically(t *testing.T) {
	db := newTransferRepositoryTestDB(t)
	_, err := db.Exec("INSERT INTO users (id, player_id) VALUES (1, NULL)")
	require.NoError(t, err)
	repo := NewUserDataTransferRepository(db)
	snapshot := emptyTransferRepositorySnapshot(t)

	empty, err := repo.IsDestinationEmpty(context.Background(), 1)
	require.NoError(t, err)
	require.True(t, empty)
	playerID, err := repo.ImportSnapshot(context.Background(), 1, snapshot)

	require.NoError(t, err)
	assert.Positive(t, playerID)
	var linkedID int
	require.NoError(t, db.Get(&linkedID, "SELECT player_id FROM users WHERE id = 1"))
	assert.Equal(t, playerID, linkedID)
	var playerCount int
	require.NoError(t, db.Get(&playerCount, "SELECT COUNT(*) FROM players WHERE user_id = 1"))
	assert.Equal(t, 1, playerCount)
}

func TestUserDataTransferRepositoryImportSnapshotCreatesUnregisteredHonor(t *testing.T) {
	db := newTransferRepositoryTestDB(t)
	_, err := db.Exec("INSERT INTO users (id, player_id) VALUES (1, NULL)")
	require.NoError(t, err)
	_, err = db.Exec("INSERT INTO honor_types (id, name) VALUES (1, 'normal'), (2, 'sp')")
	require.NoError(t, err)
	repo := NewUserDataTransferRepository(db)
	snapshot := emptyTransferRepositorySnapshot(t)
	imageURL := "https://example.com/honors/transfer.png"
	snapshot.Honors = []entity.UserDataTransferHonor{
		{
			Slot:       1,
			Name:       "移行元だけにある通常称号",
			TypeName:   "normal",
			EquippedAt: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			Slot:       2,
			ImageURL:   &imageURL,
			Name:       "移行元だけにあるSP称号",
			TypeName:   "sp",
			EquippedAt: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	playerID, err := repo.ImportSnapshot(context.Background(), 1, snapshot)

	require.NoError(t, err)
	var stored []struct {
		PlayerID int     `db:"player_id"`
		Slot     int     `db:"slot"`
		Name     string  `db:"name"`
		TypeName string  `db:"type_name"`
		ImageURL *string `db:"image_url"`
	}
	require.NoError(t, db.Select(&stored, `SELECT ph.player_id, ph.slot, h.name, ht.name AS type_name, h.image_url
		FROM player_honors ph
		INNER JOIN honors h ON h.id = ph.honor_id
		INNER JOIN honor_types ht ON ht.id = h.honor_type_id
		ORDER BY ph.slot`))
	require.Len(t, stored, 2)
	assert.Equal(t, playerID, stored[0].PlayerID)
	assert.Equal(t, 1, stored[0].Slot)
	assert.Equal(t, "移行元だけにある通常称号", stored[0].Name)
	assert.Equal(t, "normal", stored[0].TypeName)
	assert.Nil(t, stored[0].ImageURL)
	assert.Equal(t, playerID, stored[1].PlayerID)
	assert.Equal(t, 2, stored[1].Slot)
	assert.Equal(t, "移行元だけにあるSP称号", stored[1].Name)
	assert.Equal(t, "sp", stored[1].TypeName)
	require.NotNil(t, stored[1].ImageURL)
	assert.Equal(t, imageURL, *stored[1].ImageURL)
}

func TestUserDataTransferRepositoryImportSnapshotReusesExistingHonor(t *testing.T) {
	db := newTransferRepositoryTestDB(t)
	_, err := db.Exec("INSERT INTO users (id, player_id) VALUES (1, NULL)")
	require.NoError(t, err)
	_, err = db.Exec("INSERT INTO honor_types (id, name) VALUES (1, 'normal')")
	require.NoError(t, err)
	_, err = db.Exec("INSERT INTO honors (id, name, honor_type_id, image_url) VALUES (10, '登録済み称号', 1, NULL)")
	require.NoError(t, err)
	repo := NewUserDataTransferRepository(db)
	snapshot := emptyTransferRepositorySnapshot(t)
	snapshot.Honors = []entity.UserDataTransferHonor{{
		Slot:       1,
		Name:       "登録済み称号",
		TypeName:   "normal",
		EquippedAt: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
	}}

	_, err = repo.ImportSnapshot(context.Background(), 1, snapshot)

	require.NoError(t, err)
	var honorCount int
	require.NoError(t, db.Get(&honorCount, "SELECT COUNT(*) FROM honors"))
	assert.Equal(t, 1, honorCount)
	var assignedHonorID int
	require.NoError(t, db.Get(&assignedHonorID, "SELECT honor_id FROM player_honors"))
	assert.Equal(t, 10, assignedHonorID)
}

func TestUserDataTransferRepositoryImportSnapshotRollsBackWhenDestinationUserDisappears(t *testing.T) {
	db := newTransferRepositoryTestDB(t)
	repo := NewUserDataTransferRepository(db)
	snapshot := emptyTransferRepositorySnapshot(t)

	_, err := repo.ImportSnapshot(context.Background(), 999, snapshot)

	require.Error(t, err)
	var playerCount int
	require.NoError(t, db.Get(&playerCount, "SELECT COUNT(*) FROM players"))
	assert.Zero(t, playerCount)
}

func TestUserDataTransferRepositoryImportSnapshotRollsBackAfterChildSaveFailure(t *testing.T) {
	db := newTransferRepositoryTestDB(t)
	_, err := db.Exec("INSERT INTO users (id, player_id) VALUES (1, NULL)")
	require.NoError(t, err)
	_, err = db.Exec("INSERT INTO honor_types (id, name) VALUES (1, 'normal')")
	require.NoError(t, err)
	repo := NewUserDataTransferRepository(db)
	snapshot := emptyTransferRepositorySnapshot(t)
	snapshot.Honors = []entity.UserDataTransferHonor{{
		Slot:       1,
		Name:       "ロールバック対象称号",
		TypeName:   "normal",
		EquippedAt: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
	}}
	snapshot.RecordFilters = []entity.UserDataTransferRecordFilter{{
		Name:          "失敗確認",
		FilterType:    "normal",
		SchemaVersion: 1,
		Filter:        json.RawMessage(`{}`),
		CreatedAt:     time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:     time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
	}}

	_, err = repo.ImportSnapshot(context.Background(), 1, snapshot)

	require.Error(t, err)
	var playerCount int
	require.NoError(t, db.Get(&playerCount, "SELECT COUNT(*) FROM players"))
	assert.Zero(t, playerCount)
	var linkedCount int
	require.NoError(t, db.Get(&linkedCount, "SELECT COUNT(player_id) FROM users WHERE id = 1"))
	assert.Zero(t, linkedCount)
	var honorCount int
	require.NoError(t, db.Get(&honorCount, "SELECT COUNT(*) FROM honors"))
	assert.Zero(t, honorCount)
}

func TestUserDataTransferRepositoryExportAuxiliaryPlayerDataTreatsEmptyHonorImageURLAsUnset(t *testing.T) {
	db := newTransferRepositoryTestDB(t)
	_, err := db.Exec(`INSERT INTO honor_types (id, name) VALUES (1, 'normal')`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO honors (id, name, honor_type_id, image_url) VALUES (1, '称号', 1, '')`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO player_honors (player_id, honor_id, slot, created_at) VALUES (10, 1, 1, '2026-08-01 00:00:00')`)
	require.NoError(t, err)
	snapshot := emptyTransferRepositorySnapshot(t)
	repo := &userDataTransferRepository{db: db}

	err = repo.exportAuxiliaryPlayerData(context.Background(), db, 10, snapshot)

	require.NoError(t, err)
	require.Len(t, snapshot.Honors, 1)
	assert.Nil(t, snapshot.Honors[0].ImageURL)
	assert.NoError(t, snapshot.Validate())
}

func newTransferRepositoryTestDB(t *testing.T) *sqlx.DB {
	t.Helper()
	db, err := sqlx.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	statements := []string{
		"CREATE TABLE users (id INTEGER PRIMARY KEY, player_id INTEGER NULL, updated_at DATETIME)",
		"CREATE TABLE players (id INTEGER PRIMARY KEY AUTOINCREMENT, user_id INTEGER UNIQUE, player_name TEXT, player_level INTEGER, official_player_rating REAL, calculated_player_rating REAL, new_average_rating REAL, best_average_rating REAL, class_emblem_id INTEGER NULL, class_emblem_base_id INTEGER NULL, last_played_at DATETIME NULL, overpower_value REAL, official_overpower REAL, data_collected_at DATETIME NULL, created_at DATETIME, updated_at DATETIME)",
		"CREATE TABLE goals (id INTEGER PRIMARY KEY, user_id INTEGER, group_id INTEGER)",
		"CREATE TABLE goal_groups (id INTEGER PRIMARY KEY, user_id INTEGER)",
		"CREATE TABLE record_filters (id BLOB PRIMARY KEY, user_id INTEGER)",
		"CREATE TABLE songs (id INTEGER PRIMARY KEY, official_idx TEXT, is_deleted INTEGER)",
		"CREATE TABLE difficulties (id INTEGER PRIMARY KEY, name TEXT)",
		"CREATE TABLE charts (id INTEGER PRIMARY KEY, song_id INTEGER, difficulty_id INTEGER, const REAL)",
		"CREATE TABLE worldsend_charts (id INTEGER PRIMARY KEY, song_id INTEGER)",
		"CREATE TABLE courses (id INTEGER PRIMARY KEY, official_idx TEXT)",
		"CREATE TABLE clear_lamp_types (id INTEGER PRIMARY KEY, name TEXT)",
		"CREATE TABLE combo_lamp_types (id INTEGER PRIMARY KEY, name TEXT)",
		"CREATE TABLE full_chain_types (id INTEGER PRIMARY KEY, name TEXT)",
		"CREATE TABLE slots (id INTEGER PRIMARY KEY, name TEXT)",
		"CREATE TABLE class_emblems (id INTEGER PRIMARY KEY, name TEXT)",
		"CREATE TABLE class_emblem_bases (id INTEGER PRIMARY KEY, name TEXT)",
		"CREATE TABLE achievement_types (id INTEGER PRIMARY KEY, code TEXT)",
		"CREATE TABLE genres (id INTEGER PRIMARY KEY, name TEXT)",
		"CREATE TABLE versions (id INTEGER PRIMARY KEY, name TEXT)",
		"CREATE TABLE honor_types (id INTEGER PRIMARY KEY, name TEXT)",
		"CREATE TABLE honors (id INTEGER PRIMARY KEY, name TEXT, honor_type_id INTEGER, image_url TEXT NULL, UNIQUE(name, honor_type_id), UNIQUE(image_url))",
		"CREATE TABLE player_metric_histories (player_id INTEGER, official_rating REAL, official_overpower REAL, data_collected_at DATETIME)",
		"CREATE TABLE player_course_records (player_id INTEGER, course_id INTEGER, score INTEGER, is_clear INTEGER, combo_lamp_id INTEGER, updated_at DATETIME)",
		"CREATE TABLE player_honors (player_id INTEGER, honor_id INTEGER, slot INTEGER, created_at DATETIME)",
		"CREATE TABLE player_favorite_songs (player_id INTEGER, song_id INTEGER, created_at DATETIME)",
		"CREATE TABLE player_locked_songs (player_id INTEGER, song_id INTEGER, is_ultima INTEGER)",
	}
	for _, statement := range statements {
		_, err := db.Exec(statement)
		require.NoError(t, err, statement)
	}
	return db
}

func emptyTransferRepositorySnapshot(t *testing.T) *entity.UserDataTransferSnapshot {
	t.Helper()
	name, err := playername.NewPlayerName("テスト")
	require.NoError(t, err)
	return &entity.UserDataTransferSnapshot{
		Player:                   entity.UserDataTransferPlayer{Name: name, Level: 1, CreatedAt: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)},
		Records:                  []entity.UserDataTransferRecord{},
		RecordHistories:          []entity.UserDataTransferRecordHistory{},
		WorldsendRecords:         []entity.UserDataTransferWorldsendRecord{},
		WorldsendRecordHistories: []entity.UserDataTransferWorldsendRecordHistory{},
		MetricHistories:          []entity.UserDataTransferMetricHistory{},
		CourseRecords:            []entity.UserDataTransferCourseRecord{},
		Honors:                   []entity.UserDataTransferHonor{},
		FavoriteSongs:            []entity.UserDataTransferFavoriteSong{},
		LockedSongs:              []entity.UserDataTransferLockedSong{},
		Goals:                    entity.UserDataTransferGoals{Groups: []entity.UserDataTransferGoalGroup{}, Ungrouped: []entity.UserDataTransferGoal{}},
		RecordFilters:            []entity.UserDataTransferRecordFilter{},
	}
}
