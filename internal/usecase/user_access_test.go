package usecase

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
)

func TestCanAccessPrivateUser(t *testing.T) {
	publicUser := &entity.User{ID: 1}
	privateUser := &entity.User{ID: 1, IsPrivate: true}
	friend := &entity.User{ID: 2}
	stranger := &entity.User{ID: 3}
	friendshipRepo := &stubFriendshipRepo{exists: map[[2]int]bool{{2, 1}: true}}

	tests := []struct {
		name string
		// Given
		friendshipRepo repository.FriendshipRepository
		target         *entity.User
		requester      *entity.User
		// Then
		expected bool
		wantErr  error
	}{
		{name: "公開ユーザーは誰でも閲覧できる", friendshipRepo: friendshipRepo, target: publicUser, requester: nil, expected: true},
		{name: "非公開ユーザーは未ログインでは閲覧できない", friendshipRepo: friendshipRepo, target: privateUser, requester: nil, expected: false},
		{name: "非公開ユーザーは本人なら閲覧できる", friendshipRepo: friendshipRepo, target: privateUser, requester: &entity.User{ID: 1}, expected: true},
		{name: "非公開ユーザーは相互フレンドなら閲覧できる", friendshipRepo: friendshipRepo, target: privateUser, requester: friend, expected: true},
		{name: "非公開ユーザーはフレンド以外では閲覧できない", friendshipRepo: friendshipRepo, target: privateUser, requester: stranger, expected: false},
		{name: "フレンド判定が必要なときにリポジトリがなければ閲覧不可に倒さずエラーにする", friendshipRepo: nil, target: privateUser, requester: friend, wantErr: errNilFriendshipRepository},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			accessible, err := canAccessPrivateUser(context.Background(), nil, tt.friendshipRepo, tt.target, tt.requester)

			// Then
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expected, accessible)
		})
	}
}
