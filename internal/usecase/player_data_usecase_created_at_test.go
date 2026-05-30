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
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type stubPlayerRepositoryForPlayerData struct {
	foundPlayer *entity.Player
	savedPlayer *entity.Player
}

func (s *stubPlayerRepositoryForPlayerData) FindByID(ctx context.Context, exec repository.Executor, id int) (*entity.Player, error) {
	return nil, nil
}

func (s *stubPlayerRepositoryForPlayerData) FindByIDWithHonors(ctx context.Context, exec repository.Executor, id int) (*repository.PlayerWithHonors, error) {
	return nil, nil
}

func (s *stubPlayerRepositoryForPlayerData) FindByUserID(ctx context.Context, exec repository.Executor, userID int) (*entity.Player, error) {
	return s.foundPlayer, nil
}

func (s *stubPlayerRepositoryForPlayerData) FindHonorsByPlayerID(ctx context.Context, exec repository.Executor, playerID int) ([]*entity.PlayerHonor, error) {
	return nil, nil
}

func (s *stubPlayerRepositoryForPlayerData) UpdateCalculatedRatings(ctx context.Context, exec repository.Executor, playerID int, calculatedRating, bestAverage, newAverage float64) error {
	return nil
}

func (s *stubPlayerRepositoryForPlayerData) Save(ctx context.Context, exec repository.Executor, player *entity.Player) error {
	copied := *player
	if copied.ID == 0 {
		copied.ID = 99
		player.ID = copied.ID
	}
	s.savedPlayer = &copied
	return nil
}

func (s *stubPlayerRepositoryForPlayerData) DeleteByUserID(ctx context.Context, exec repository.Executor, userID int) error {
	return nil
}

func TestEnsurePlayer_新規プレイヤー作成時はCreatedAtをゼロ値にしない(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "新規登録時はupdated_atをcreated_atにも設定して保存する",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			playerRepo := &stubPlayerRepositoryForPlayerData{}
			userRepo := new(MockUserRepository)
			uc := &playerDataUsecase{
				playerRepo: playerRepo,
				userRepo:   userRepo,
			}

			user := &entity.User{
				ID:       1,
				Username: username.MustNewUserName("playerdatatest"),
			}
			updatedAt := time.Date(2026, 3, 16, 15, 28, 53, 0, time.FixedZone("JST", 9*60*60))
			before := time.Now()
			officialRating := 16.25
			playerName := playername.MustNewPlayerName("テストプレイヤー")
			summary := &PlayerDataSummaryInput{
				Name:           playerName.String(),
				Level:          42,
				OfficialRating: &officialRating,
			}

			userRepo.
				On("Save", mock.Anything, mock.Anything, mock.MatchedBy(func(savedUser *entity.User) bool {
					return savedUser != nil && savedUser.PlayerID != nil && *savedUser.PlayerID == 99
				})).
				Return(nil).
				Once()

			// When
			playerID, err := uc.ensurePlayer(context.Background(), nil, user, summary, updatedAt)
			after := time.Now()

			// Then
			require.NoError(t, err)
			assert.Equal(t, 99, playerID)
			require.NotNil(t, playerRepo.savedPlayer)
			assert.False(t, playerRepo.savedPlayer.CreatedAt.IsZero())
			assert.False(t, playerRepo.savedPlayer.CreatedAt.Before(before))
			assert.False(t, playerRepo.savedPlayer.CreatedAt.After(after))
			assert.True(t, playerRepo.savedPlayer.UpdatedAt.Equal(updatedAt))
			userRepo.AssertExpectations(t)
		})
	}
}
