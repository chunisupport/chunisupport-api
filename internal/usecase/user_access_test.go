package usecase

import (
	"context"
	"errors"
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

func TestGetAccessibleUser(t *testing.T) {
	publicUser := &entity.User{ID: 1}
	privateUser := &entity.User{ID: 1, IsPrivate: true}
	friend := &entity.User{ID: 2}
	stranger := &entity.User{ID: 3}
	repoErr := errors.New("db error")

	tests := []struct {
		name string
		// Given
		target    *entity.User
		findErr   error
		requester *entity.User
		// Then
		expected *entity.User
		wantErr  error
	}{
		{name: "公開ユーザーは誰でも取得できる", target: publicUser, requester: nil, expected: publicUser},
		{name: "リポジトリの未発見エラーはユースケースの未発見エラーに変換する", findErr: repository.ErrUserNotFound, wantErr: ErrUserNotFound},
		{name: "nilユーザーは未発見として扱う", target: nil, wantErr: ErrUserNotFound},
		{name: "その他の取得エラーはそのまま返す", findErr: repoErr, wantErr: repoErr},
		{name: "非公開ユーザーは本人なら取得できる", target: privateUser, requester: &entity.User{ID: 1}, expected: privateUser},
		{name: "非公開ユーザーは相互フレンドなら取得できる", target: privateUser, requester: friend, expected: privateUser},
		{name: "非公開ユーザーはフレンド以外には非公開エラーを返す", target: privateUser, requester: stranger, wantErr: ErrUserPrivate},
		{name: "非公開ユーザーは未ログインでは非公開エラーを返す", target: privateUser, requester: nil, wantErr: ErrUserPrivate},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			userRepo := &stubUserRepoForFriendship{target: tt.target, err: tt.findErr}
			friendshipRepo := &stubFriendshipRepo{exists: map[[2]int]bool{{2, 1}: true}}

			// When
			user, err := getAccessibleUser(context.Background(), nil, userRepo, friendshipRepo, "target", tt.requester)

			// Then
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, user)
				return
			}
			require.NoError(t, err)
			assert.Same(t, tt.expected, user)
		})
	}
}
