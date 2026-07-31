package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type adminUserStatisticsQueryServiceStub struct {
	result repository.AdminUserStatistics
	err    error
	cutoff time.Time
}

func (s *adminUserStatisticsQueryServiceStub) Get(_ context.Context, cutoff time.Time) (repository.AdminUserStatistics, error) {
	s.cutoff = cutoff
	return s.result, s.err
}

func TestAdminUserStatisticsUsecase_Get(t *testing.T) {
	// Given
	now := time.Date(2026, 8, 1, 15, 0, 0, 0, time.FixedZone("JST", 9*60*60))
	queryService := &adminUserStatisticsQueryServiceStub{result: repository.AdminUserStatistics{
		TotalUsers:                 100,
		UsersWithPlayerData:        80,
		ActivePlayerDataLast30Days: 50,
	}}
	uc := newAdminUserStatisticsUsecase(queryService, func() time.Time { return now })

	// When
	result, err := uc.Get(context.Background())

	// Then
	require.NoError(t, err)
	assert.Equal(t, now.Add(-30*24*time.Hour), queryService.cutoff)
	assert.Equal(t, 100, result.TotalUsers)
	assert.Equal(t, 80, result.UsersWithPlayerData)
	assert.Equal(t, 50, result.ActivePlayerDataLast30Days)
}

func TestAdminUserStatisticsUsecase_Get_取得失敗を返す(t *testing.T) {
	// Given
	wantErr := errors.New("query failed")
	queryService := &adminUserStatisticsQueryServiceStub{err: wantErr}
	uc := newAdminUserStatisticsUsecase(queryService, time.Now)

	// When
	_, err := uc.Get(context.Background())

	// Then
	assert.ErrorIs(t, err, wantErr)
}
