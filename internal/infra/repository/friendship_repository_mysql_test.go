package repository

import (
	"context"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/testutil/mysqltest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFriendshipRepositoryMySQL_書き込みと判定(t *testing.T) {
	db := mysqltest.NewDatabase(t, "")
	_, err := db.Exec(`INSERT INTO users (id, username) VALUES
		(1, 'friend_one'), (2, 'friend_two'), (3, 'friend_three'), (4, 'friend_four')`)
	require.NoError(t, err)
	ctx := context.Background()
	repo := NewFriendshipRepository()
	requested := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
	accepted := requested.Add(time.Hour)

	t.Run("Saveは新規作成と更新を行い作成日時を維持する", func(t *testing.T) {
		tx, err := db.Beginx()
		require.NoError(t, err)
		defer tx.Rollback()
		request, err := entity.NewFriendRequest(1, 2, requested)
		require.NoError(t, err)
		require.NoError(t, repo.Save(ctx, tx, request))
		got, err := repo.Find(ctx, tx, 1, 2)
		require.NoError(t, err)
		assert.Equal(t, request, got)
		missing, err := repo.Find(ctx, tx, 2, 1)
		require.NoError(t, err)
		assert.Nil(t, missing)

		require.NoError(t, request.Accept(accepted))
		request.RequestedAt = requested.Add(time.Minute)
		request.CreatedAt = requested.Add(2 * time.Hour)
		require.NoError(t, repo.Save(ctx, tx, request))
		got, err = repo.Find(ctx, tx, 1, 2)
		require.NoError(t, err)
		assert.Equal(t, &entity.Friendship{UserID: 1, FriendUserID: 2, StatusID: 2,
			RequestedAt: requested.Add(time.Minute), AcceptedAt: &accepted, CreatedAt: requested, UpdatedAt: accepted}, got)

		// 再申請で承認日時をNULLに戻せることも実際のUPSERTで確認する。
		renewed, err := entity.NewFriendRequest(1, 2, requested.Add(3*time.Hour))
		require.NoError(t, err)
		require.NoError(t, repo.Save(ctx, tx, renewed))
		got, err = repo.Find(ctx, tx, 1, 2)
		require.NoError(t, err)
		renewed.CreatedAt = requested
		assert.Equal(t, renewed, got)
		var count int
		require.NoError(t, tx.Get(&count, `SELECT COUNT(*) FROM friendships`))
		assert.Equal(t, 1, count)
	})

	t.Run("CountOutgoingActiveとExistsMutualAcceptedは方向と状態を区別する", func(t *testing.T) {
		tx, err := db.Beginx()
		require.NoError(t, err)
		defer tx.Rollback()
		_, err = tx.Exec(`INSERT INTO friendships (user_id, friend_user_id, status_id, requested_at, accepted_at) VALUES
			(1, 2, 1, ?, NULL), (1, 3, 3, ?, NULL), (4, 1, 2, ?, ?), (2, 3, 2, ?, ?)`,
			requested, requested, requested, accepted, requested, accepted)
		require.NoError(t, err)
		count, err := repo.CountOutgoingActive(ctx, tx, 1)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
		mutual, err := repo.ExistsMutualAccepted(ctx, tx, 1, 4)
		require.NoError(t, err)
		assert.False(t, mutual, "片方向だけの承認は相互承認ではない")
		mutual, err = repo.ExistsMutualAccepted(ctx, tx, 1, 2)
		require.NoError(t, err)
		assert.False(t, mutual, "対象ペア以外の承認を数えない")

		forward, err := entity.NewAcceptedFriendship(1, 2, requested, accepted)
		require.NoError(t, err)
		reverse, err := entity.NewFriendRequest(2, 1, requested)
		require.NoError(t, err)
		require.NoError(t, repo.Save(ctx, tx, forward))
		require.NoError(t, repo.Save(ctx, tx, reverse))
		mutual, err = repo.ExistsMutualAccepted(ctx, tx, 1, 2)
		require.NoError(t, err)
		assert.False(t, mutual, "逆方向が申請中なら相互承認ではない")
		require.NoError(t, reverse.Accept(accepted))
		require.NoError(t, repo.Save(ctx, tx, reverse))
		mutual, err = repo.ExistsMutualAccepted(ctx, tx, 1, 2)
		require.NoError(t, err)
		assert.True(t, mutual)
		count, err = repo.CountOutgoingActive(ctx, tx, 2)
		require.NoError(t, err)
		assert.Equal(t, 2, count, "承認済みも有効件数に含める")
		count, err = repo.CountOutgoingActive(ctx, tx, 3)
		require.NoError(t, err)
		assert.Zero(t, count, "受信した申請や承認は数えない")
	})

	for _, tt := range []struct {
		name     string
		remove   func(context.Context, domainrepo.Executor, int, int) error
		friendID int
		want     []string
	}{
		{"Deleteは片方向だけを削除する", repo.Delete, 2, []string{"1:3:2", "2:1:2", "2:3:1", "3:1:3"}},
		{"DeletePendingは申請中だけを削除する", repo.DeletePending, 2, []string{"1:3:2", "2:1:2", "2:3:1", "3:1:3"}},
		{"DeletePendingは承認済みを残す", repo.DeletePending, 3, []string{"1:2:1", "1:3:2", "2:1:2", "2:3:1", "3:1:3"}},
		{"DeletePairは両方向だけを削除する", repo.DeletePair, 2, []string{"1:3:2", "2:3:1", "3:1:3"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			tx, err := db.Beginx()
			require.NoError(t, err)
			defer tx.Rollback()
			_, err = tx.Exec(`INSERT INTO friendships (user_id, friend_user_id, status_id, requested_at, accepted_at) VALUES
				(1, 2, 1, ?, NULL), (2, 1, 2, ?, ?), (1, 3, 2, ?, ?), (3, 1, 3, ?, NULL), (2, 3, 1, ?, NULL)`,
				requested, requested, accepted, requested, accepted, requested, requested)
			require.NoError(t, err)
			require.NoError(t, tt.remove(ctx, tx, 1, tt.friendID))
			var got []string
			require.NoError(t, tx.Select(&got, `SELECT CONCAT(user_id, ':', friend_user_id, ':', status_id) FROM friendships ORDER BY user_id, friend_user_id`))
			assert.Equal(t, tt.want, got)
		})
	}
}
