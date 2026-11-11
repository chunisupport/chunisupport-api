package repository

import (
	"context"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/profilecomment"
	"github.com/chunisupport/chunisupport-api/internal/testutil/mysqltest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSocialRepositoriesMySQL_Saveは所有者と公開同意を保持する(t *testing.T) {
	db := mysqltest.NewDatabase(t, "")
	ctx := context.Background()
	_, err := db.Exec("INSERT INTO users (id,username) VALUES (7,'targetuser'),(8,'otheruser')")
	require.NoError(t, err)
	t.Run("X連携先を変えた場合だけ同意を解除する", func(t *testing.T) {
		repo := NewXAccountRepository()
		account, err := entity.NewXAccount(7, "123", "before")
		require.NoError(t, err)
		require.NoError(t, repo.Save(ctx, db, account))
		var public bool
		require.NoError(t, db.Get(&public, "SELECT is_public FROM user_x_accounts WHERE user_id=7"))
		assert.False(t, public)
		_, err = db.Exec("UPDATE user_x_accounts SET is_public=TRUE WHERE user_id=7")
		require.NoError(t, err)
		for _, tt := range []struct {
			xid, username string
			public        bool
		}{
			{"123", "renamed", true}, {"456", "another", false},
		} {
			account, err = entity.NewXAccount(7, tt.xid, tt.username)
			require.NoError(t, err)
			require.NoError(t, repo.Save(ctx, db, account))
			found, err := repo.FindByUserID(ctx, db, 7)
			require.NoError(t, err)
			assert.Equal(t, account, found)
			require.NoError(t, db.Get(&public, "SELECT is_public FROM user_x_accounts WHERE user_id=7"))
			assert.Equal(t, tt.public, public)
		}
	})
	t.Run("プロフィールを新規保存して同じ所有者だけ更新する", func(t *testing.T) {
		repo := NewUserProfileRepository()
		for _, tt := range []struct {
			id      int
			comment string
		}{
			{8, "他人のコメント"}, {7, "最初のコメント"}, {7, "更新したコメント"},
		} {
			comment, err := profilecomment.NewProfileComment(tt.comment)
			require.NoError(t, err)
			profile, err := entity.NewUserProfile(tt.id, comment)
			require.NoError(t, err)
			require.NoError(t, repo.Save(ctx, db, profile))
			found, err := repo.FindByUserID(ctx, db, tt.id)
			require.NoError(t, err)
			assert.Equal(t, profile, found)
		}
		found, err := repo.FindByUserID(ctx, db, 8)
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, "他人のコメント", found.Comment().String())
	})
}
