package repository

import (
	"context"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestVersionRepository_FindLatest(t *testing.T) {
	db := newVersionRepositoryTestDB(t)
	_, err := db.Exec(`INSERT INTO versions (id, name, released_at) VALUES
		(1, 'CHUNITHM A', '2025-01-01'),
		(2, 'CHUNITHM B', '2026-01-01'),
		(3, 'CHUNITHM C', '2026-01-01')`)
	require.NoError(t, err)
	repo := NewVersionRepository()

	latest, err := repo.FindLatest(context.Background(), db)

	require.NoError(t, err)
	assert.Equal(t, 3, latest.ID)
}

func TestVersionRepository_ExistsSongInRange(t *testing.T) {
	db := newVersionRepositoryTestDB(t)
	_, err := db.Exec(`INSERT INTO songs (released_at) VALUES (NULL), (?), (?)`, repositoryTestDate(2025, 1, 1), repositoryTestDate(2026, 1, 1))
	require.NoError(t, err)
	repo := NewVersionRepository()
	ctx := context.Background()

	tests := []struct {
		name string
		from time.Time
		to   *time.Time
		want bool
	}{
		{name: "下端を含む", from: repositoryTestDate(2025, 1, 1), to: datePointer(repositoryTestDate(2025, 1, 2)), want: true},
		{name: "上端を含まない", from: repositoryTestDate(2025, 1, 2), to: datePointer(repositoryTestDate(2026, 1, 1)), want: false},
		{name: "上限なし", from: repositoryTestDate(2026, 1, 1), want: true},
		{name: "NULLは対象外", from: repositoryTestDate(2027, 1, 1), want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.ExistsSongInRange(ctx, db, tt.from, tt.to)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestVersionRepository_CRUD(t *testing.T) {
	db := newVersionRepositoryTestDB(t)
	repo := NewVersionRepository()
	ctx := context.Background()
	version, err := entity.NewVersion("CHUNITHM VERSE", repositoryTestDate(2025, 1, 1))
	require.NoError(t, err)

	created, err := repo.Create(ctx, db, version)
	require.NoError(t, err)
	created.Name = "CHUNITHM VERSE II"
	require.NoError(t, repo.Save(ctx, db, created))
	found, err := repo.FindByID(ctx, db, created.ID)
	require.NoError(t, err)
	assert.Equal(t, "CHUNITHM VERSE II", found.Name)
	require.NoError(t, repo.Delete(ctx, db, created.ID))
	_, err = repo.FindByID(ctx, db, created.ID)
	assert.ErrorIs(t, err, domainrepo.ErrVersionNotFound)
	assert.ErrorIs(t, repo.Delete(ctx, db, created.ID), domainrepo.ErrVersionNotFound)
}

func TestWrapVersionDuplicateError(t *testing.T) {
	err := &mysql.MySQLError{Number: mysqlDuplicateEntryErrorNumber, Message: "Duplicate entry for key 'versions.name'"}
	assert.ErrorIs(t, wrapVersionDuplicateError(err), domainrepo.ErrVersionConflict)
}

func newVersionRepositoryTestDB(t *testing.T) *sqlx.DB {
	t.Helper()
	db, err := sqlx.Open("sqlite", ":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	_, err = db.Exec(`
		CREATE TABLE versions (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL UNIQUE, released_at DATE NOT NULL);
		CREATE TABLE songs (released_at DATE NULL);
	`)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	return db
}

func repositoryTestDate(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func datePointer(value time.Time) *time.Time {
	return &value
}
