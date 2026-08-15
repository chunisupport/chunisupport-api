package repository

import (
	"context"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func setupSystemMaintenanceRepositorySQLite(t *testing.T) *sqlx.DB {
	t.Helper()

	db, err := sqlx.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, db.Close())
	})

	_, err = db.Exec(`
		CREATE TABLE system_maintenance (
			id INTEGER NOT NULL PRIMARY KEY CHECK (id = 1),
			enabled BOOLEAN NOT NULL,
			comment TEXT NOT NULL,
			updated_by_user_id INTEGER NULL,
			updated_at DATETIME NOT NULL
		)
	`)
	require.NoError(t, err)

	return db
}

func TestSystemMaintenanceRepository_Find(t *testing.T) {
	// Given
	db := setupSystemMaintenanceRepositorySQLite(t)
	updatedAt := time.Date(2026, 7, 26, 12, 0, 0, 123000000, time.UTC)
	_, err := db.Exec(
		`INSERT INTO system_maintenance (id, enabled, comment, updated_by_user_id, updated_at) VALUES (?, ?, ?, ?, ?)`,
		entity.SystemMaintenanceID,
		true,
		"更新中です",
		10,
		updatedAt,
	)
	require.NoError(t, err)
	repo := NewSystemMaintenanceRepository(db)

	// When
	maintenance, err := repo.Find(context.Background())

	// Then
	require.NoError(t, err)
	assert.Equal(t, entity.SystemMaintenanceID, maintenance.ID)
	assert.True(t, maintenance.Enabled)
	assert.Equal(t, "更新中です", maintenance.Comment.String())
	require.NotNil(t, maintenance.UpdatedByUserID)
	assert.Equal(t, 10, *maintenance.UpdatedByUserID)
	assert.Equal(t, updatedAt, maintenance.UpdatedAt)
}

func TestSystemMaintenanceRepository_Find_行が存在しない場合は専用エラーを返す(t *testing.T) {
	// Given
	db := setupSystemMaintenanceRepositorySQLite(t)
	repo := NewSystemMaintenanceRepository(db)

	// When
	_, err := repo.Find(context.Background())

	// Then
	assert.ErrorIs(t, err, domainrepo.ErrSystemMaintenanceNotFound)
}

func TestSystemMaintenanceRepository_Find_DBエラーと行不在を区別する(t *testing.T) {
	// Given
	db, err := sqlx.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, db.Close())
	})
	repo := NewSystemMaintenanceRepository(db)

	// When
	_, err = repo.Find(context.Background())

	// Then
	require.Error(t, err)
	assert.NotErrorIs(t, err, domainrepo.ErrSystemMaintenanceNotFound)
}

func TestSystemMaintenanceRepository_Find_不正な永続状態を拒否する(t *testing.T) {
	// Given
	db := setupSystemMaintenanceRepositorySQLite(t)
	_, err := db.Exec(
		`INSERT INTO system_maintenance (id, enabled, comment, updated_by_user_id, updated_at) VALUES (?, ?, ?, ?, ?)`,
		entity.SystemMaintenanceID,
		true,
		"",
		nil,
		time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC),
	)
	require.NoError(t, err)
	repo := NewSystemMaintenanceRepository(db)

	// When
	_, err = repo.Find(context.Background())

	// Then
	assert.ErrorIs(t, err, entity.ErrSystemMaintenanceCommentRequired)
}

func TestSystemMaintenanceRepository_Save(t *testing.T) {
	// Given
	db := setupSystemMaintenanceRepositorySQLite(t)
	previousAt := time.Date(2026, 7, 26, 11, 0, 0, 0, time.UTC)
	_, err := db.Exec(
		`INSERT INTO system_maintenance (id, enabled, comment, updated_by_user_id, updated_at) VALUES (?, ?, ?, ?, ?)`,
		entity.SystemMaintenanceID,
		false,
		"",
		nil,
		previousAt,
	)
	require.NoError(t, err)
	maintenance, err := entity.RestoreSystemMaintenance(
		entity.SystemMaintenanceID,
		false,
		"",
		nil,
		previousAt,
	)
	require.NoError(t, err)
	updatedAt := previousAt.Add(time.Hour)
	require.NoError(t, maintenance.Enable("更新中です", 10, updatedAt))
	repo := NewSystemMaintenanceRepository(db)

	// When
	err = repo.Save(context.Background(), maintenance)

	// Then
	require.NoError(t, err)
	var saved struct {
		ID              int       `db:"id"`
		Enabled         bool      `db:"enabled"`
		Comment         string    `db:"comment"`
		UpdatedByUserID *int      `db:"updated_by_user_id"`
		UpdatedAt       time.Time `db:"updated_at"`
	}
	err = db.Get(
		&saved,
		`SELECT id, enabled, comment, updated_by_user_id, updated_at FROM system_maintenance WHERE id = ?`,
		entity.SystemMaintenanceID,
	)
	require.NoError(t, err)
	assert.Equal(t, maintenance.ID, saved.ID)
	assert.Equal(t, maintenance.Enabled, saved.Enabled)
	assert.Equal(t, maintenance.Comment.String(), saved.Comment)
	assert.Equal(t, maintenance.UpdatedByUserID, saved.UpdatedByUserID)
	assert.Equal(t, maintenance.UpdatedAt, saved.UpdatedAt)
}

func TestSystemMaintenanceRepository_Save_行が存在しない場合は専用エラーを返す(t *testing.T) {
	// Given
	db := setupSystemMaintenanceRepositorySQLite(t)
	maintenance, err := entity.RestoreSystemMaintenance(
		entity.SystemMaintenanceID,
		false,
		"",
		nil,
		time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC),
	)
	require.NoError(t, err)
	repo := NewSystemMaintenanceRepository(db)

	// When
	err = repo.Save(context.Background(), maintenance)

	// Then
	assert.ErrorIs(t, err, domainrepo.ErrSystemMaintenanceNotFound)
}

func TestSystemMaintenanceRepository_Save_DBエラーと行不在を区別する(t *testing.T) {
	// Given
	db, err := sqlx.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, db.Close())
	})
	maintenance, err := entity.RestoreSystemMaintenance(
		entity.SystemMaintenanceID,
		false,
		"",
		nil,
		time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC),
	)
	require.NoError(t, err)
	repo := NewSystemMaintenanceRepository(db)

	// When
	err = repo.Save(context.Background(), maintenance)

	// Then
	require.Error(t, err)
	assert.NotErrorIs(t, err, domainrepo.ErrSystemMaintenanceNotFound)
}

func TestSystemMaintenanceRepository_取得カラムを明示する(t *testing.T) {
	assert.Equal(t, "id, enabled, comment, updated_by_user_id, updated_at", systemMaintenanceColumns)
	assert.NotContains(t, systemMaintenanceColumns, "*")
}
