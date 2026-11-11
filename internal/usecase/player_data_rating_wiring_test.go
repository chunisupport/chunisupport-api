package usecase

import (
	"context"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegister_本枠だけをbestとnewへ振り分けて保存する(t *testing.T) {
	fixture := newRegistrationFixture(t)
	for _, tt := range []struct {
		slot     string
		constant float64
	}{
		{"best", 15}, {"new", 14}, {"best_candidate", 16}, {"new_candidate", 16.5}, {"none", 17},
	} {
		fixture.records.ratingRecords = append(fixture.records.ratingRecords, &entity.PlayerRecord{Score: 1_009_000, Chart: &entity.Chart{Const: chartconstant.ChartConstant(tt.constant)}, Slot: &entity.Slot{Name: tt.slot}})
	}
	result, err := fixture.uc.Register(context.Background(), fixture.user, fixture.payload, "rating-wiring-hash")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, new(0.666), result.Profile.Rating)
	assert.Equal(t, new(0.666), fixture.player.player.CalculatedRating)
	assert.Equal(t, new(17.15), fixture.player.player.BestAverageRating)
	assert.Equal(t, new(16.15), fixture.player.player.NewAverageRating)
	assert.Equal(t, 1, fixture.data.latestUpdateSaveCalls)
}

func TestRegister_レーティング用の関連欠損なら集約値を保存しない(t *testing.T) {
	for _, tt := range []struct {
		name   string
		record *entity.PlayerRecord
	}{
		{"譜面なし", &entity.PlayerRecord{Slot: &entity.Slot{Name: "best"}}},
		{"枠なし", &entity.PlayerRecord{Chart: &entity.Chart{Const: 15}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			fixture := newRegistrationFixture(t)
			fixture.records.ratingRecords = []*entity.PlayerRecord{tt.record}
			result, err := fixture.uc.Register(context.Background(), fixture.user, fixture.payload, "missing-relation-hash")
			assert.Error(t, err)
			assert.Nil(t, result)
			assert.Nil(t, fixture.player.player.CalculatedRating)
			assert.Zero(t, fixture.data.latestUpdateSaveCalls)
		})
	}
}
