package usecase

import (
	"context"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/username"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserUsecase_GetPublicOfficialRating(t *testing.T) {
	name, err := username.NewUserName("testuser")
	require.NoError(t, err)
	playerID := 12
	tests := []struct {
		name       string
		user       *entity.User
		player     *entity.Player
		wantRating *float64
		wantErr    error
	}{
		{
			name:       "公開ユーザーの公式RATINGを取得する",
			user:       &entity.User{ID: 1, Username: name, PlayerID: &playerID},
			player:     &entity.Player{OfficialRating: 17.29},
			wantRating: func() *float64 { value := 17.29; return &value }(),
		},
		{
			name: "未連携の場合は値なし",
			user: &entity.User{ID: 1, Username: name},
		},
		{
			name:    "非公開ユーザーは匿名で参照できない",
			user:    &entity.User{ID: 1, Username: name, PlayerID: &playerID, IsPrivate: true},
			player:  &entity.Player{OfficialRating: 17.29},
			wantErr: ErrUserPrivate,
		},
		{
			name:    "存在しないユーザーは見つからない",
			wantErr: ErrUserNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var playerWithHonors *repository.PlayerWithHonors
			if tt.player != nil {
				playerWithHonors = &repository.PlayerWithHonors{Player: tt.player}
			}
			u := NewUserUsecase(nil, &stubUserRepository{user: tt.user}, &stubPlayerRepository{playerWithHonors: playerWithHonors}, nil, nil, nil, nil, nil)

			rating, err := u.GetPublicOfficialRating(context.Background(), "testuser")
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, rating)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantRating, rating)
		})
	}
}
