package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/playername"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type ratingPlayerRepository struct {
	repository.PlayerRepository
	t       *testing.T
	tx      repository.Executor
	player  *entity.Player
	lockErr error
	locked  bool
	saved   bool
}

func (r *ratingPlayerRepository) FindByIDForUpdate(_ context.Context, exec repository.Executor, id int) (*entity.Player, error) {
	assert.Same(r.t, r.tx, exec)
	assert.Equal(r.t, r.player.ID, id)
	r.locked = true
	return r.player, r.lockErr
}

func (r *ratingPlayerRepository) Save(_ context.Context, exec repository.Executor, player *entity.Player) error {
	assert.True(r.t, r.locked)
	assert.Same(r.t, r.tx, exec)
	assert.Same(r.t, r.player, player)
	r.saved = true
	return nil
}

type ratingRecordsRepository struct {
	repository.PlayerRecordRepository
	playerRepo *ratingPlayerRepository
}

func (r *ratingRecordsRepository) FindByPlayerIDForRating(_ context.Context, exec repository.Executor, id int) ([]*entity.PlayerRecord, error) {
	assert.True(r.playerRepo.t, r.playerRepo.locked)
	assert.Same(r.playerRepo.t, r.playerRepo.tx, exec)
	return nil, nil
}

func TestCalculateAndUpdateRatings_ロック後の集約を保存し他の値を保持する(t *testing.T) {
	now := time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC)
	overpower := 12345.0
	player := &entity.Player{ID: 10, UserID: 20, Name: playername.MustNewPlayerName("最新"), Level: 50,
		OfficialRating: 17, OfficialOverpower: 12300, OverpowerValue: &overpower, DataCollectedAt: &now, CreatedAt: now, UpdatedAt: now}
	expected := *player
	zero := 0.0
	expected.CalculatedRating, expected.BestAverageRating, expected.NewAverageRating = &zero, &zero, &zero
	tx := &sqlx.DB{}
	playerRepo := &ratingPlayerRepository{t: t, tx: tx, player: player}
	uc := &playerDataUsecase{playerRepo: playerRepo, playerRecRepo: &ratingRecordsRepository{playerRepo: playerRepo}}

	_, err := uc.calculateAndUpdateRatings(context.Background(), tx, player.ID)

	require.NoError(t, err)
	assert.True(t, playerRepo.saved)
	assert.Equal(t, &expected, playerRepo.player)
}

func TestCalculateAndUpdateRatings_ロック失敗後に成績取得や保存をしない(t *testing.T) {
	expected := errors.New("ロック失敗")
	tx := &sqlx.DB{}
	playerRepo := &ratingPlayerRepository{t: t, tx: tx, player: &entity.Player{ID: 10}, lockErr: expected}
	uc := &playerDataUsecase{playerRepo: playerRepo}

	_, err := uc.calculateAndUpdateRatings(context.Background(), tx, 10)

	assert.ErrorIs(t, err, expected)
	assert.False(t, playerRepo.saved)
}
