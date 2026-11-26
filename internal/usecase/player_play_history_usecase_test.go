package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/username/usernametest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type stubPlayerPlayHistoryQueryService struct {
	entries []entity.PlayerPlayHistory
}

func (s *stubPlayerPlayHistoryQueryService) FindByPlayerID(_ context.Context, _ repository.Executor, playerID int) ([]entity.PlayerPlayHistory, error) {
	if playerID != 10 {
		return nil, assert.AnError
	}
	return s.entries, nil
}

func TestPlayerPlayHistoryUsecase_Get(t *testing.T) {
	// Given
	playerID := 10
	user := &entity.User{ID: 1, Username: usernametest.New(t, "testuser"), PlayerID: &playerID}
	userRepo := new(MockUserRepository)
	userRepo.On("FindByUsername", mock.Anything, mock.Anything, "testuser").Return(user, nil).Once()
	playedAt := time.Date(2026, time.November, 26, 0, 10, 0, 0, time.UTC)
	query := &stubPlayerPlayHistoryQueryService{entries: []entity.PlayerPlayHistory{{Score: 1005000, PlayedAt: playedAt}}}
	usecase := NewPlayerPlayHistoryUsecase(nil, userRepo, nil, query)

	// When
	entries, err := usecase.Get(context.Background(), "testuser", nil)

	// Then
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, 1005000, entries[0].Score)
	assert.Equal(t, playedAt, entries[0].PlayedAt)
	userRepo.AssertExpectations(t)
}

func TestPlayerPlayHistoryUsecase_Get_プレイヤー未連携は空配列(t *testing.T) {
	user := &entity.User{ID: 1, Username: usernametest.New(t, "testuser")}
	userRepo := new(MockUserRepository)
	userRepo.On("FindByUsername", mock.Anything, mock.Anything, "testuser").Return(user, nil).Once()
	usecase := NewPlayerPlayHistoryUsecase(nil, userRepo, nil, &stubPlayerPlayHistoryQueryService{})

	entries, err := usecase.Get(context.Background(), "testuser", nil)

	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestPlayerPlayHistoryUsecase_Get_非公開ユーザーは既存の公開範囲に従う(t *testing.T) {
	playerID := 10
	user := &entity.User{ID: 2, Username: usernametest.New(t, "privateuser"), PlayerID: &playerID, IsPrivate: true}
	userRepo := new(MockUserRepository)
	userRepo.On("FindByUsername", mock.Anything, mock.Anything, "privateuser").Return(user, nil).Once()
	usecase := NewPlayerPlayHistoryUsecase(nil, userRepo, newStubFriendshipRepo(), &stubPlayerPlayHistoryQueryService{})

	entries, err := usecase.Get(context.Background(), "privateuser", nil)

	assert.Nil(t, entries)
	assert.ErrorIs(t, err, ErrUserPrivate)
}
