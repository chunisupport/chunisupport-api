package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func setupRecordFilterRepositorySQLite(t *testing.T) *sqlx.DB {
	t.Helper()

	db, err := sqlx.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, db.Close())
	})

	_, err = db.Exec(`
		CREATE TABLE record_filters (
			id BLOB NOT NULL PRIMARY KEY,
			user_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			filter_value_gzip BLOB NOT NULL,
			is_worldsend BOOLEAN NOT NULL DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX idx_record_filters_user_id ON record_filters (user_id);
	`)
	require.NoError(t, err)

	return db
}

func TestRecordFilterRepository_SaveFindListCountAndDelete(t *testing.T) {
	ctx := context.Background()
	db := setupRecordFilterRepositorySQLite(t)
	repo := NewRecordFilterRepository(db)
	id := []byte("1234567890123456")
	filter, err := entity.NewRecordFilter(id, 10, "通常枠", []byte{0x1f, 0x8b, 0x08}, false)
	require.NoError(t, err)

	err = repo.Save(ctx, filter)
	require.NoError(t, err)

	count, err := repo.CountByUserID(ctx, 10)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	found, err := repo.FindByIDAndUserID(ctx, id, 10)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, id, found.ID())
	assert.Equal(t, 10, found.UserID())
	assert.Equal(t, "通常枠", found.Name())
	assert.Equal(t, []byte{0x1f, 0x8b, 0x08}, found.FilterValueGzip())
	assert.False(t, found.IsWorldsend())
	assert.False(t, found.CreatedAt().IsZero())
	assert.False(t, found.UpdatedAt().IsZero())

	require.NoError(t, found.ChangeName("ワールズエンド枠"))
	require.NoError(t, found.ChangeFilterValueGzip([]byte{0x1f, 0x8b, 0x09}))
	found.ChangeWorldsend(true)
	err = repo.Save(ctx, found)
	require.NoError(t, err)

	filters, err := repo.ListByUserID(ctx, 10)
	require.NoError(t, err)
	require.Len(t, filters, 1)
	assert.Equal(t, "ワールズエンド枠", filters[0].Name())
	assert.Equal(t, []byte{0x1f, 0x8b, 0x09}, filters[0].FilterValueGzip())
	assert.True(t, filters[0].IsWorldsend())

	err = repo.DeleteByIDAndUserID(ctx, id, 10)
	require.NoError(t, err)

	_, err = repo.FindByIDAndUserID(ctx, id, 10)
	assert.True(t, errors.Is(err, domainrepo.ErrRecordFilterNotFound))
}

func TestRecordFilterRepository_UserIsolation(t *testing.T) {
	ctx := context.Background()
	db := setupRecordFilterRepositorySQLite(t)
	repo := NewRecordFilterRepository(db)
	id := []byte("1234567890123456")
	filter, err := entity.NewRecordFilter(id, 10, "自分のフィルタ", []byte{0x1f, 0x8b, 0x08}, false)
	require.NoError(t, err)
	require.NoError(t, repo.Save(ctx, filter))

	_, err = repo.FindByIDAndUserID(ctx, id, 20)
	assert.True(t, errors.Is(err, domainrepo.ErrRecordFilterNotFound))

	otherUserFilter, err := entity.NewRecordFilter(id, 20, "他ユーザーの更新", []byte{0x1f, 0x8b, 0x09}, true)
	require.NoError(t, err)
	err = repo.Save(ctx, otherUserFilter)
	assert.ErrorIs(t, err, domainrepo.ErrRecordFilterNotFound)

	found, err := repo.FindByIDAndUserID(ctx, id, 10)
	require.NoError(t, err)
	assert.Equal(t, "自分のフィルタ", found.Name())
	assert.False(t, found.IsWorldsend())
}

func TestRecordFilterRepository_FindByIDAndUserID_ReturnsErrorWhenStoredDataIsInvalid(t *testing.T) {
	ctx := context.Background()
	db := setupRecordFilterRepositorySQLite(t)
	repo := NewRecordFilterRepository(db)
	id := []byte("1234567890123456")

	_, err := db.Exec(
		`INSERT INTO record_filters (id, user_id, name, filter_value_gzip) VALUES (?, ?, ?, ?)`,
		id,
		10,
		"",
		[]byte{0x1f, 0x8b, 0x08},
	)
	require.NoError(t, err)

	_, err = repo.FindByIDAndUserID(ctx, id, 10)
	assert.ErrorIs(t, err, domainrepo.ErrRepositoryOperationFailed)
	assert.ErrorIs(t, err, entity.ErrRecordFilterNameRequired)
}
