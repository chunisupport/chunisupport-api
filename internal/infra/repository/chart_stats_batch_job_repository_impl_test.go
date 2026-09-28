package repository

import (
	"context"
	"testing"
	"time"
	"uuid"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/username"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func setupChartStatsBatchJobRepositorySQLite(t *testing.T) *sqlx.DB {
	t.Helper()

	db, err := sqlx.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, db.Close())
	})

	_, err = db.Exec(`
		CREATE TABLE users (
			id INTEGER NOT NULL PRIMARY KEY,
			username TEXT NOT NULL
		);
		CREATE TABLE chart_stats_batch_jobs (
			id BLOB NOT NULL PRIMARY KEY,
			trigger_type TEXT NOT NULL,
			status TEXT NOT NULL,
			requested_by_user_id INTEGER NULL,
			started_at DATETIME NOT NULL,
			finished_at DATETIME NULL,
			error_message TEXT NULL
		);
		INSERT INTO users (id, username) VALUES (10, 'adminuser');
	`)
	require.NoError(t, err)

	return db
}

var chartStatsBatchJobRepoStartedAt = time.Date(2026, 9, 28, 3, 0, 0, 0, time.UTC)

func TestChartStatsBatchJobRepository_SaveAndFindByID(t *testing.T) {
	finishedAt := chartStatsBatchJobRepoStartedAt.Add(5 * time.Minute)
	requester := entity.ChartStatsBatchJobRequester{UserID: 10, Username: username.MustNewUserName("adminuser")}

	tests := []struct {
		name   string
		newJob func() *entity.ChartStatsBatchJob
	}{
		{
			name: "CLIから開始した実行中のジョブを保存して取得できる",
			newJob: func() *entity.ChartStatsBatchJob {
				return entity.StartChartStatsBatchJobFromCLI(uuid.NewV4(), chartStatsBatchJobRepoStartedAt)
			},
		},
		{
			name: "管理画面から開始して失敗したジョブを要求者付きで取得できる",
			newJob: func() *entity.ChartStatsBatchJob {
				job := entity.StartChartStatsBatchJobFromAdmin(uuid.NewV4(), requester, chartStatsBatchJobRepoStartedAt)
				require.NoError(t, job.Fail("replace chart stats: deadlock", finishedAt))
				return job
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			db := setupChartStatsBatchJobRepositorySQLite(t)
			repo := NewChartStatsBatchJobRepository(db)
			job := tt.newJob()

			// When
			require.NoError(t, repo.Save(context.Background(), job))
			found, err := repo.FindByID(context.Background(), job.ID())

			// Then
			require.NoError(t, err)
			assert.Equal(t, job, found)
		})
	}
}

func TestChartStatsBatchJobRepository_Save_既存ジョブを更新する(t *testing.T) {
	// Given
	db := setupChartStatsBatchJobRepositorySQLite(t)
	repo := NewChartStatsBatchJobRepository(db)
	job := entity.StartChartStatsBatchJobFromCLI(uuid.NewV4(), chartStatsBatchJobRepoStartedAt)
	require.NoError(t, repo.Save(context.Background(), job))
	require.NoError(t, job.Complete(chartStatsBatchJobRepoStartedAt.Add(time.Minute)))

	// When
	err := repo.Save(context.Background(), job)

	// Then
	require.NoError(t, err)
	found, err := repo.FindByID(context.Background(), job.ID())
	require.NoError(t, err)
	assert.Equal(t, entity.ChartStatsBatchJobStatusSucceeded, found.Status())
	var count int
	require.NoError(t, db.Get(&count, `SELECT COUNT(*) FROM chart_stats_batch_jobs`))
	assert.Equal(t, 1, count)
}

func TestChartStatsBatchJobRepository_Save_終了済みのジョブは上書きしない(t *testing.T) {
	// Given: 取り残されたジョブとして中断扱いにした行
	db := setupChartStatsBatchJobRepositorySQLite(t)
	repo := NewChartStatsBatchJobRepository(db)
	id := uuid.NewV4()
	orphan := entity.StartChartStatsBatchJobFromCLI(id, chartStatsBatchJobRepoStartedAt)
	require.NoError(t, repo.Save(context.Background(), orphan))
	require.NoError(t, orphan.Interrupt(chartStatsBatchJobRepoStartedAt.Add(time.Hour)))
	require.NoError(t, repo.Save(context.Background(), orphan))
	late := entity.StartChartStatsBatchJobFromCLI(id, chartStatsBatchJobRepoStartedAt)
	require.NoError(t, late.Complete(chartStatsBatchJobRepoStartedAt.Add(2*time.Hour)))

	// When: ロックを失った元のプロセスが後から成功を記録しようとする
	err := repo.Save(context.Background(), late)

	// Then
	assert.Error(t, err)
	found, err := repo.FindByID(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, entity.ChartStatsBatchJobStatusInterrupted, found.Status())
}

func TestChartStatsBatchJobRepository_Save_上限を超えた古いジョブを削除する(t *testing.T) {
	// Given: 上限件数まで保存済み
	db := setupChartStatsBatchJobRepositorySQLite(t)
	repo := NewChartStatsBatchJobRepository(db)
	var oldestID uuid.UUID
	for i := range info.ChartStatsBatchJobHistoryLimit {
		startedAt := chartStatsBatchJobRepoStartedAt.Add(time.Duration(i) * time.Hour)
		job := entity.StartChartStatsBatchJobFromCLI(uuid.NewV4(), startedAt)
		require.NoError(t, job.Complete(startedAt.Add(time.Minute)))
		require.NoError(t, repo.Save(context.Background(), job))
		if i == 0 {
			oldestID = job.ID()
		}
	}
	newest := entity.StartChartStatsBatchJobFromCLI(uuid.NewV4(), chartStatsBatchJobRepoStartedAt.Add(time.Duration(info.ChartStatsBatchJobHistoryLimit)*time.Hour))

	// When
	err := repo.Save(context.Background(), newest)

	// Then
	require.NoError(t, err)
	var count int
	require.NoError(t, db.Get(&count, `SELECT COUNT(*) FROM chart_stats_batch_jobs`))
	assert.Equal(t, info.ChartStatsBatchJobHistoryLimit, count)
	_, err = repo.FindByID(context.Background(), oldestID)
	assert.ErrorIs(t, err, domainrepo.ErrChartStatsBatchJobNotFound)
	_, err = repo.FindByID(context.Background(), newest.ID())
	assert.NoError(t, err)
}

func TestChartStatsBatchJobRepository_FindByID_存在しない場合は専用エラーを返す(t *testing.T) {
	// Given
	db := setupChartStatsBatchJobRepositorySQLite(t)
	repo := NewChartStatsBatchJobRepository(db)

	// When
	_, err := repo.FindByID(context.Background(), uuid.NewV4())

	// Then
	assert.ErrorIs(t, err, domainrepo.ErrChartStatsBatchJobNotFound)
}

func TestChartStatsBatchJobRepository_FindRunning(t *testing.T) {
	// Given
	db := setupChartStatsBatchJobRepositorySQLite(t)
	repo := NewChartStatsBatchJobRepository(db)
	running := entity.StartChartStatsBatchJobFromCLI(uuid.NewV4(), chartStatsBatchJobRepoStartedAt)
	finished := entity.StartChartStatsBatchJobFromCLI(uuid.NewV4(), chartStatsBatchJobRepoStartedAt)
	require.NoError(t, finished.Complete(chartStatsBatchJobRepoStartedAt.Add(time.Minute)))
	require.NoError(t, repo.Save(context.Background(), running))
	require.NoError(t, repo.Save(context.Background(), finished))

	// When
	jobs, err := repo.FindRunning(context.Background())

	// Then
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	assert.Equal(t, running.ID(), jobs[0].ID())
}

func TestChartStatsBatchJobRepository_ListRecent_開始日時の新しい順に件数を制限して返す(t *testing.T) {
	// Given
	db := setupChartStatsBatchJobRepositorySQLite(t)
	repo := NewChartStatsBatchJobRepository(db)
	var ids []uuid.UUID
	for i := range 3 {
		job := entity.StartChartStatsBatchJobFromCLI(uuid.NewV4(), chartStatsBatchJobRepoStartedAt.Add(time.Duration(i)*time.Hour))
		require.NoError(t, repo.Save(context.Background(), job))
		ids = append(ids, job.ID())
	}

	// When
	jobs, err := repo.ListRecent(context.Background(), 2)

	// Then
	require.NoError(t, err)
	require.Len(t, jobs, 2)
	assert.Equal(t, ids[2], jobs[0].ID())
	assert.Equal(t, ids[1], jobs[1].ID())
}

func TestChartStatsBatchJobRepository_要求者が削除された場合は要求者なしで復元する(t *testing.T) {
	// Given
	db := setupChartStatsBatchJobRepositorySQLite(t)
	repo := NewChartStatsBatchJobRepository(db)
	requester := entity.ChartStatsBatchJobRequester{UserID: 10, Username: username.MustNewUserName("adminuser")}
	job := entity.StartChartStatsBatchJobFromAdmin(uuid.NewV4(), requester, chartStatsBatchJobRepoStartedAt)
	require.NoError(t, repo.Save(context.Background(), job))
	// MySQL の ON DELETE SET NULL 相当
	_, err := db.Exec(`UPDATE chart_stats_batch_jobs SET requested_by_user_id = NULL`)
	require.NoError(t, err)

	// When
	found, err := repo.FindByID(context.Background(), job.ID())

	// Then
	require.NoError(t, err)
	assert.Equal(t, entity.ChartStatsBatchJobTriggerAdmin, found.Trigger())
	assert.Nil(t, found.Requester())
}
