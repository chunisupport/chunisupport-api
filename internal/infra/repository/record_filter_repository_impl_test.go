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

func TestRecordFilterRepository_CreateFindListCountUpdateAndDelete(t *testing.T) {
	ctx := context.Background()
	db := setupRecordFilterRepositorySQLite(t)
	repo := NewRecordFilterRepository()
	id := []byte("1234567890123456")
	filter, err := entity.NewRecordFilter(id, 10, "通常枠", []byte{0x1f, 0x8b, 0x08}, false)
	require.NoError(t, err)

	err = repo.Create(ctx, db, filter)
	require.NoError(t, err)

	count, err := repo.CountByUserID(ctx, db, 10)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	found, err := repo.FindByIDAndUserID(ctx, db, id, 10)
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
	err = repo.Update(ctx, db, found)
	require.NoError(t, err)

	filters, err := repo.ListByUserID(ctx, db, 10)
	require.NoError(t, err)
	require.Len(t, filters, 1)
	assert.Equal(t, "ワールズエンド枠", filters[0].Name())
	assert.Equal(t, []byte{0x1f, 0x8b, 0x09}, filters[0].FilterValueGzip())
	assert.True(t, filters[0].IsWorldsend())

	err = repo.DeleteByIDAndUserID(ctx, db, id, 10)
	require.NoError(t, err)

	_, err = repo.FindByIDAndUserID(ctx, db, id, 10)
	assert.True(t, errors.Is(err, domainrepo.ErrRecordFilterNotFound))
}

func TestRecordFilterRepository_UserIsolation(t *testing.T) {
	ctx := context.Background()
	db := setupRecordFilterRepositorySQLite(t)
	repo := NewRecordFilterRepository()
	id := []byte("1234567890123456")
	filter, err := entity.NewRecordFilter(id, 10, "自分のフィルタ", []byte{0x1f, 0x8b, 0x08}, false)
	require.NoError(t, err)
	require.NoError(t, repo.Create(ctx, db, filter))

	_, err = repo.FindByIDAndUserID(ctx, db, id, 20)
	assert.True(t, errors.Is(err, domainrepo.ErrRecordFilterNotFound))

	otherUserFilter, err := entity.NewRecordFilter(id, 20, "他ユーザーの更新", []byte{0x1f, 0x8b, 0x09}, true)
	require.NoError(t, err)
	err = repo.Update(ctx, db, otherUserFilter)
	assert.ErrorIs(t, err, domainrepo.ErrRecordFilterNotFound)

	found, err := repo.FindByIDAndUserID(ctx, db, id, 10)
	require.NoError(t, err)
	assert.Equal(t, "自分のフィルタ", found.Name())
	assert.False(t, found.IsWorldsend())
}

func TestRecordFilterRepository_UserIsolation_ListCountAndDelete(t *testing.T) {
	// Given: 2人のユーザーがそれぞれフィルターを持つ。自分の分は通常とWORLD'S ENDの両方の種類を含める
	ctx := context.Background()
	db := setupRecordFilterRepositorySQLite(t)
	repo := NewRecordFilterRepository()
	ownNormalID := []byte("1234567890123456")
	ownWorldsendID := []byte("2345678901234567")
	otherID := []byte("3456789012345678")
	ownNormal, err := entity.NewRecordFilter(ownNormalID, 10, "自分の通常フィルタ", []byte{0x1f, 0x8b, 0x08}, false)
	require.NoError(t, err)
	ownWorldsend, err := entity.NewRecordFilter(ownWorldsendID, 10, "自分のWORLD'S ENDフィルタ", []byte{0x1f, 0x8b, 0x09}, true)
	require.NoError(t, err)
	other, err := entity.NewRecordFilter(otherID, 20, "他人のフィルタ", []byte{0x1f, 0x8b, 0x0a}, false)
	require.NoError(t, err)
	require.NoError(t, repo.Create(ctx, db, ownNormal))
	require.NoError(t, repo.Create(ctx, db, ownWorldsend))
	require.NoError(t, repo.Create(ctx, db, other))

	// When: 自分の一覧を求める
	filters, err := repo.ListByUserID(ctx, db, 10)

	// Then: 種類によらず自分の2件だけが返り、他人の分は混ざらない
	require.NoError(t, err)
	require.Len(t, filters, 2)
	for _, f := range filters {
		assert.Equal(t, 10, f.UserID())
	}
	names := []string{filters[0].Name(), filters[1].Name()}
	assert.Contains(t, names, "自分の通常フィルタ")
	assert.Contains(t, names, "自分のWORLD'S ENDフィルタ")

	// When: 自分の件数を求める
	count, err := repo.CountByUserID(ctx, db, 10)

	// Then: 他人の分を数えず自分の2件だけになる
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	// When: 他人のフィルターのIDを自分のユーザーIDで削除する
	err = repo.DeleteByIDAndUserID(ctx, db, otherID, 10)

	// Then: 対象なしとして失敗し、他人のフィルターは残っている
	assert.ErrorIs(t, err, domainrepo.ErrRecordFilterNotFound)
	remaining, err := repo.FindByIDAndUserID(ctx, db, otherID, 20)
	require.NoError(t, err)
	assert.Equal(t, "他人のフィルタ", remaining.Name())
	otherCount, err := repo.CountByUserID(ctx, db, 20)
	require.NoError(t, err)
	assert.Equal(t, 1, otherCount)
}

func TestRecordFilterRepository_UpdateDoesNotRecreateDeletedFilter(t *testing.T) {
	// Given: 更新前に取得したフィルタが、別リクエストにより削除されている
	ctx := context.Background()
	db := setupRecordFilterRepositorySQLite(t)
	repo := NewRecordFilterRepository()
	id := []byte("1234567890123456")
	filter, err := entity.NewRecordFilter(id, 10, "削除前", []byte{0x1f, 0x8b, 0x08}, false)
	require.NoError(t, err)
	require.NoError(t, repo.Create(ctx, db, filter))
	found, err := repo.FindByIDAndUserID(ctx, db, id, 10)
	require.NoError(t, err)
	require.NoError(t, repo.DeleteByIDAndUserID(ctx, db, id, 10))
	require.NoError(t, found.ChangeName("削除後の更新"))

	// When
	err = repo.Update(ctx, db, found)

	// Then: 削除済みとして扱い、同じIDで作り直さない
	assert.ErrorIs(t, err, domainrepo.ErrRecordFilterNotFound)
	count, err := repo.CountByUserID(ctx, db, 10)
	require.NoError(t, err)
	assert.Zero(t, count)
}

func TestRecordFilterRepository_FindByIDAndUserID_ReturnsErrorWhenStoredDataIsInvalid(t *testing.T) {
	ctx := context.Background()
	db := setupRecordFilterRepositorySQLite(t)
	repo := NewRecordFilterRepository()
	id := []byte("1234567890123456")

	_, err := db.Exec(
		`INSERT INTO record_filters (id, user_id, name, filter_value_gzip) VALUES (?, ?, ?, ?)`,
		id,
		10,
		"",
		[]byte{0x1f, 0x8b, 0x08},
	)
	require.NoError(t, err)

	_, err = repo.FindByIDAndUserID(ctx, db, id, 10)
	assert.ErrorIs(t, err, domainrepo.ErrRepositoryOperationFailed)
	assert.ErrorIs(t, err, entity.ErrRecordFilterNameRequired)
}
