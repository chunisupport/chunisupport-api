package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/username"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type stubMetricHistoryRepository struct {
	entries []entity.PlayerMetricHistoryEntry
}

func (s *stubMetricHistoryRepository) Insert(context.Context, repository.Executor, entity.PlayerMetricHistoryEntry) error {
	return nil
}

func (s *stubMetricHistoryRepository) FindTimeline(_ context.Context, playerID int) ([]entity.PlayerMetricHistoryEntry, error) {
	return s.entries, nil
}

func TestPlayerMetricHistoryUsecase_Get(t *testing.T) {
	playerID := 10
	user := &entity.User{ID: 1, Username: username.MustNewUserName("testuser"), PlayerID: &playerID}
	userRepo := new(MockUserRepository)
	userRepo.On("FindByUsername", mock.Anything, mock.Anything, "testuser").Return(user, nil).Once()
	collectedAt := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	historyRepo := &stubMetricHistoryRepository{entries: []entity.PlayerMetricHistoryEntry{{
		PlayerID: playerID, OfficialRating: 17.25, OfficialOverpower: 12345.67, DataCollectedAt: collectedAt,
	}}}
	us := NewPlayerMetricHistoryUsecase(nil, userRepo, historyRepo, nil)

	entries, err := us.Get(context.Background(), "testuser", nil)

	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, 17.25, entries[0].OfficialRating)
	assert.Equal(t, 12345.67, entries[0].OfficialOverpower)
	userRepo.AssertExpectations(t)
}

func TestPlayerMetricHistoryUsecase_Get_プレイヤー未連携は履歴なし(t *testing.T) {
	user := &entity.User{ID: 1, Username: username.MustNewUserName("testuser")}
	userRepo := new(MockUserRepository)
	userRepo.On("FindByUsername", mock.Anything, mock.Anything, "testuser").Return(user, nil).Once()
	us := NewPlayerMetricHistoryUsecase(nil, userRepo, &stubMetricHistoryRepository{}, nil)

	_, err := us.Get(context.Background(), "testuser", nil)

	assert.ErrorIs(t, err, ErrPlayerMetricHistoryNotFound)
}

func TestPlayerMetricHistoryUsecase_Get_非公開ユーザーは匿名参照できない(t *testing.T) {
	playerID := 10
	user := &entity.User{ID: 2, Username: username.MustNewUserName("privateuser"), PlayerID: &playerID, IsPrivate: true}
	userRepo := new(MockUserRepository)
	userRepo.On("FindByUsername", mock.Anything, mock.Anything, "privateuser").Return(user, nil).Once()
	us := NewPlayerMetricHistoryUsecase(nil, userRepo, &stubMetricHistoryRepository{}, newStubFriendshipRepo())

	_, err := us.Get(context.Background(), "privateuser", nil)

	assert.ErrorIs(t, err, ErrUserPrivate)
}

func TestPlayerMetricHistoryUsecase_Get_非公開ユーザーは承認済みフレンドが参照できる(t *testing.T) {
	playerID := 10
	user := &entity.User{ID: 2, Username: username.MustNewUserName("privateuser"), PlayerID: &playerID, IsPrivate: true}
	userRepo := new(MockUserRepository)
	userRepo.On("FindByUsername", mock.Anything, mock.Anything, "privateuser").Return(user, nil).Once()
	friendshipRepo := newStubFriendshipRepo()
	friendshipRepo.exists[[2]int{1, 2}] = true
	historyRepo := &stubMetricHistoryRepository{entries: []entity.PlayerMetricHistoryEntry{{PlayerID: playerID}}}
	us := NewPlayerMetricHistoryUsecase(nil, userRepo, historyRepo, friendshipRepo)

	entries, err := us.Get(context.Background(), "privateuser", &entity.User{ID: 1})

	require.NoError(t, err)
	assert.Len(t, entries, 1)
}
