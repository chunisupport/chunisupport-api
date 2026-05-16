package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/playername"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/username"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserUsecase_GetUserProfile_Success(t *testing.T) {
	now := time.Now()
	un, err := username.NewUserName("tester")
	require.NoError(t, err)
	user := &entity.User{
		ID:       1,
		Username: un,
		PlayerID: intPointer(1),
	}
	player := &entity.Player{
		ID:        1,
		Name:      playername.MustNewPlayerName("テストプレイヤー"),
		Level:     10,
		UpdatedAt: now,
	}
	service := NewUserUsecase(nil, &stubUserRepository{user: user}, &stubPlayerRepository{playerWithHonors: &repository.PlayerWithHonors{Player: player, Honors: []*entity.PlayerHonor{}}}, &stubPlayerRecordRepository{}, nil, nil, nil, nil)

	result, err := service.GetUserProfile(context.Background(), "tester", nil)
	require.NoError(t, err)
	require.NotNil(t, result.Player)
	assert.Equal(t, "tester", result.Username)
	assert.True(t, result.Player.UpdatedAt.Equal(now))
}

func TestUserUsecase_GetUserProfile_PrivateUserBlocked(t *testing.T) {
	un, err := username.NewUserName("privateuser")
	require.NoError(t, err)
	user := &entity.User{
		ID:        1,
		Username:  un,
		PlayerID:  intPointer(1),
		IsPrivate: true,
	}
	player := &entity.Player{
		ID:        1,
		Name:      playername.MustNewPlayerName("プライベ"),
		Level:     1,
		UpdatedAt: time.Now(),
	}
	service := NewUserUsecase(nil, &stubUserRepository{user: user}, &stubPlayerRepository{playerWithHonors: &repository.PlayerWithHonors{Player: player, Honors: []*entity.PlayerHonor{}}}, &stubPlayerRecordRepository{}, nil, nil, nil, nil)

	_, err = service.GetUserProfile(context.Background(), "privateuser", nil)
	assert.ErrorIs(t, err, ErrUserPrivate)
}

func TestUserUsecase_GetUserProfile_UserNotFound(t *testing.T) {
	service := NewUserUsecase(nil, &stubUserRepository{err: repository.ErrUserNotFound}, &stubPlayerRepository{}, &stubPlayerRecordRepository{}, nil, nil, nil, nil)

	_, err := service.GetUserProfile(context.Background(), "nobody", nil)
	assert.ErrorIs(t, err, ErrUserNotFound)
}

func TestUserUsecase_GetUserProfile_PlayerNotLinkedReturnsNilPlayer(t *testing.T) {
	un, err := username.NewUserName("tester")
	require.NoError(t, err)
	user := &entity.User{
		ID:       1,
		Username: un,
	}
	service := NewUserUsecase(nil, &stubUserRepository{user: user}, &stubPlayerRepository{}, &stubPlayerRecordRepository{}, nil, nil, nil, nil)

	result, err := service.GetUserProfile(context.Background(), "tester", nil)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "tester", result.Username)
	assert.Nil(t, result.Player)
}
