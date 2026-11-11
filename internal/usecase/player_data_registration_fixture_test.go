package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainmasterdata "github.com/chunisupport/chunisupport-api/internal/domain/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/playername/playernametest"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
)

// registrationFixture はRegisterの観測に必要な依存先を揃え、内部処理の分割には依存しないようにする。
type registrationFixture struct {
	uc      *playerDataUsecase
	user    *entity.User
	payload *PlayerDataPayload
	data    *registrationDataRepository
	player  *registrationPlayerRepository
	records *registrationRecordRepository
	tx      repository.Executor
}

func newRegistrationFixture(t *testing.T) *registrationFixture {
	t.Helper()
	tx := &sqlx.DB{}
	masters := newApplyScoresTestMasters()
	now := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	user := &entity.User{ID: 7, PlayerID: new(77)}
	player := &registrationPlayerRepository{t: t, tx: tx, player: &entity.Player{ID: 77, UserID: 7, Name: playernametest.New(t, "テスト"), Level: 42, CreatedAt: now.Add(-24 * time.Hour)}}
	data := &registrationDataRepository{stubPlayerDataRepositoryForApplyScoresTest: &stubPlayerDataRepositoryForApplyScoresTest{latestUpdateErr: repository.ErrPlayerLatestUpdateNotFound}, masters: &repository.PlayerDataMaster{Songs: masters.songs, ChartsByKey: masters.chartsByKey, ChartsByID: masters.chartsByID, WorldsendBySongID: masters.worldsendBySongID}}
	records := &registrationRecordRepository{stubPlayerRecordRepositoryForApplyScoresTest: &stubPlayerRecordRepositoryForApplyScoresTest{}, t: t, tx: tx}
	uc := &playerDataUsecase{tm: &trackingTM{passedFloor: tx}, userRepo: &registrationUserRepository{t: t, tx: tx, user: user}, playerRepo: player, playerRecRepo: records, worldsendRecRepo: &stubWorldsendRecordRepositoryForApplyScoresTest{}, playerDataRepo: data, masterCache: registrationMasterProvider{masters.PlayerDataMasters}}
	return &registrationFixture{uc: uc, user: user, payload: &PlayerDataPayload{Name: "テスト", Level: 42, Rating: new(16.0), Overpower: PlayerDataOverpowerPayload{Value: new(100.0), Percentage: new(1.0)}, UpdatedAt: now.Format(time.RFC3339)}, data: data, player: player, records: records, tx: tx}
}

type registrationMasterProvider struct {
	masters *domainmasterdata.PlayerDataMasters
}

func (p registrationMasterProvider) PlayerDataMasters() *domainmasterdata.PlayerDataMasters {
	return p.masters
}

type registrationUserRepository struct {
	repository.UserRepository
	t    *testing.T
	tx   repository.Executor
	user *entity.User
}

func (r *registrationUserRepository) FindByIDForUpdate(_ context.Context, tx repository.Executor, id int) (*entity.User, error) {
	assert.Same(r.t, r.tx, tx)
	assert.Equal(r.t, r.user.ID, id)
	return r.user, nil
}

type registrationPlayerRepository struct {
	repository.PlayerRepository
	t      *testing.T
	tx     repository.Executor
	player *entity.Player
	saved  bool
}

func (r *registrationPlayerRepository) FindByUserIDForUpdate(_ context.Context, tx repository.Executor, id int) (*entity.Player, error) {
	assert.Same(r.t, r.tx, tx)
	assert.Equal(r.t, r.player.UserID, id)
	return r.player, nil
}
func (r *registrationPlayerRepository) FindByIDForUpdate(_ context.Context, tx repository.Executor, id int) (*entity.Player, error) {
	assert.Same(r.t, r.tx, tx)
	assert.Equal(r.t, r.player.ID, id)
	return r.player, nil
}
func (r *registrationPlayerRepository) Save(_ context.Context, tx repository.Executor, player *entity.Player) error {
	assert.Same(r.t, r.tx, tx)
	r.player = player
	r.saved = true
	return nil
}

type registrationDataRepository struct {
	*stubPlayerDataRepositoryForApplyScoresTest
	masters *repository.PlayerDataMaster
}

func (r *registrationDataRepository) LoadMasterData(context.Context, []string) (*repository.PlayerDataMaster, error) {
	return r.masters, nil
}

type registrationRecordRepository struct {
	*stubPlayerRecordRepositoryForApplyScoresTest
	t             *testing.T
	tx            repository.Executor
	ratingRecords []*entity.PlayerRecord
}

func (r *registrationRecordRepository) FindByPlayerIDForRating(_ context.Context, tx repository.Executor, id int) ([]*entity.PlayerRecord, error) {
	assert.Same(r.t, r.tx, tx)
	assert.Equal(r.t, 77, id)
	return r.ratingRecords, nil
}
