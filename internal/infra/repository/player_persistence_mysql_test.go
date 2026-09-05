package repository

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/playername"
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupPlayerPersistenceMySQL(t *testing.T) *sqlx.DB {
	t.Helper()
	dsn := os.Getenv("PLAYER_PERSISTENCE_MYSQL_DSN")
	if dsn == "" {
		t.Skip("PLAYER_PERSISTENCE_MYSQL_DSN が未設定です")
	}
	cfg, err := mysql.ParseDSN(dsn)
	require.NoError(t, err)
	cfg.DBName = ""
	cfg.ParseTime = true
	admin, err := sqlx.Connect("mysql", cfg.FormatDSN())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, admin.Close()) })
	name := fmt.Sprintf("player_persistence_%d", time.Now().UnixNano())
	_, err = admin.Exec("CREATE DATABASE " + name)
	require.NoError(t, err)
	t.Cleanup(func() { _, err := admin.Exec("DROP DATABASE " + name); require.NoError(t, err) })
	cfg.DBName = name
	db, err := sqlx.Connect("mysql", cfg.FormatDSN())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	setupPlayerRepositorySchema(t, db)
	setupPlayerBatchSchema(t, db)
	seedPlayerWithHonors(t, db, 1, false)
	return db
}

func waitForPlayerLock(t *testing.T, ctx context.Context, db *sqlx.DB) {
	t.Helper()
	require.Eventually(t, func() bool {
		var count int
		err := db.GetContext(ctx, &count, `SELECT COUNT(*) FROM performance_schema.data_lock_waits w
   INNER JOIN performance_schema.data_locks l ON l.ENGINE_LOCK_ID = w.REQUESTING_ENGINE_LOCK_ID AND l.ENGINE = w.ENGINE
   WHERE l.OBJECT_SCHEMA = DATABASE() AND l.OBJECT_NAME = 'players'`)
		return err == nil && count > 0
	}, 5*time.Second, 10*time.Millisecond)
}

func TestPlayerPersistenceMySQL_通常更新の完了を待ってバッチが競合判定する(t *testing.T) {
	db := setupPlayerPersistenceMySQL(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	batch := NewPlayerDataBatchRepository(db)
	keys, err := batch.ListPlayerKeys(ctx, 0, 1, 10)
	require.NoError(t, err)
	require.Len(t, keys, 1)
	repo := NewPlayerRepository(db)
	tx, err := db.BeginTxx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback()
	player, err := repo.FindByUserIDForUpdate(ctx, tx, 20)
	require.NoError(t, err)
	require.NotNil(t, player)
	require.NoError(t, player.ChangeOfficialMetrics(17, 15000, 99, player.DataCollectedAt.Add(time.Hour)))
	require.NoError(t, repo.Save(ctx, tx, player))

	type result struct {
		status domainrepo.PlayerBatchProcessStatus
		err    error
	}
	done := make(chan result, 1)
	go func() {
		status, err := batch.ProcessPlayer(ctx, keys[0], func(domainrepo.PlayerBatchData) (domainrepo.PlayerBatchUpdate, error) {
			return domainrepo.PlayerBatchUpdate{PlayerRating: 1}, nil
		})
		done <- result{status, err}
	}()
	waitForPlayerLock(t, ctx, db)
	require.NoError(t, tx.Commit())
	outcome := <-done
	require.NoError(t, outcome.err)
	assert.Equal(t, domainrepo.PlayerBatchConflict, outcome.status)
	saved, err := repo.FindByID(ctx, db, 1)
	require.NoError(t, err)
	assert.Equal(t, player.DataCollectedAt, saved.DataCollectedAt)
	assert.Equal(t, 17.0, saved.OfficialRating)
	assert.Nil(t, saved.CalculatedRating)
}

func TestPlayerPersistenceMySQL_バッチ後の通常更新が再計算値を保持する(t *testing.T) {
	db := setupPlayerPersistenceMySQL(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	batch := NewPlayerDataBatchRepository(db)
	keys, err := batch.ListPlayerKeys(ctx, 0, 1, 10)
	require.NoError(t, err)
	require.Len(t, keys, 1)
	locked := make(chan struct{})
	release := make(chan struct{}, 1)
	defer close(release)
	batchDone := make(chan error, 1)
	go func() {
		_, err := batch.ProcessPlayer(ctx, keys[0], func(domainrepo.PlayerBatchData) (domainrepo.PlayerBatchUpdate, error) {
			close(locked)
			select {
			case <-release:
			case <-ctx.Done():
				return domainrepo.PlayerBatchUpdate{}, ctx.Err()
			}
			return domainrepo.PlayerBatchUpdate{PlayerRating: 16.5, BestAverage: 16.7, NewAverage: 16.2, Overpower: 12345}, nil
		})
		batchDone <- err
	}()
	select {
	case <-locked:
	case <-ctx.Done():
		require.NoError(t, ctx.Err())
	}
	normalDone := make(chan error, 1)
	go func() {
		repo := NewPlayerRepository(db)
		tx, err := db.BeginTxx(ctx, nil)
		if err != nil {
			normalDone <- err
			return
		}
		defer tx.Rollback()
		player, err := repo.FindByUserIDForUpdate(ctx, tx, 20)
		if err == nil {
			player.ChangeProfile(playername.MustNewPlayerName("更新後"), 40, nil, nil, nil)
			err = repo.Save(ctx, tx, player)
		}
		if err == nil {
			err = tx.Commit()
		}
		normalDone <- err
	}()
	waitForPlayerLock(t, ctx, db)
	release <- struct{}{}
	require.NoError(t, <-batchDone)
	require.NoError(t, <-normalDone)
	saved, err := NewPlayerRepository(db).FindByID(ctx, db, 1)
	require.NoError(t, err)
	assert.Equal(t, "更新後", saved.Name.String())
	require.NotNil(t, saved.CalculatedRating)
	assert.Equal(t, 16.5, *saved.CalculatedRating)
	require.NotNil(t, saved.OverpowerValue)
	assert.Equal(t, 12345.0, *saved.OverpowerValue)
}
