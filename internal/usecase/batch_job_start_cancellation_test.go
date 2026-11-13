package usecase

import (
	"context"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/songbatch"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/username/usernametest"
	"github.com/stretchr/testify/assert"
)

func TestSongBatchJobUsecase_開始保存中の切断でもロックを解放する(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	repo := newFakeSongBatchJobRepository()
	repo.beforeSave = cancel
	locks := &fakeSongBatchLockProvider{acquired: true}
	uc := newTestSongBatchJobUsecase(context.Background(), locks, repo, &fakeSongBatchRunner{}, nil)

	_, err := uc.StartFromAdmin(ctx, entity.SongBatchJobRequester{UserID: 7, Username: usernametest.New(t, "adminuser")}, songbatch.NewRunRequest(false, false))

	assert.ErrorIs(t, err, context.Canceled)
	assert.True(t, locks.lock.released)
	assert.True(t, locks.lock.releaseHasDeadline)
}

func TestChartStatsBatchJobUsecase_開始保存中の切断でもロックを解放する(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	repo := newFakeChartStatsBatchJobRepository()
	repo.beforeSave = cancel
	locks := &fakeSongBatchLockProvider{acquired: true}
	uc := newTestChartStatsBatchJobUsecase(context.Background(), locks, repo, &fakeChartStatsBatchRunner{})

	_, err := uc.StartFromAdmin(ctx, newChartStatsBatchRequester(t))

	assert.ErrorIs(t, err, context.Canceled)
	assert.True(t, locks.lock.released)
	assert.True(t, locks.lock.releaseHasDeadline)
}
