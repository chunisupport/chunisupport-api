package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/username"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type stubFriendshipRepo struct {
	relations map[[2]int]*entity.Friendship
	counts    map[int]int
	exists    map[[2]int]bool
	saved     []*entity.Friendship
	deleted   [][2]int
}

type stubUserRepoForFriendship struct {
	target  *entity.User
	locked  *entity.User
	lockIDs []int
	err     error
}

func (s *stubUserRepoForFriendship) FindByUsername(ctx context.Context, exec repository.Executor, username string) (*entity.User, error) {
	return s.target, s.err
}

func (s *stubUserRepoForFriendship) FindByID(ctx context.Context, exec repository.Executor, id int) (*entity.User, error) {
	return nil, errors.New("not implemented")
}

func (s *stubUserRepoForFriendship) FindByIDForUpdate(ctx context.Context, exec repository.Executor, id int) (*entity.User, error) {
	s.lockIDs = append(s.lockIDs, id)
	if s.locked != nil {
		return s.locked, s.err
	}
	return &entity.User{ID: id}, s.err
}

func (s *stubUserRepoForFriendship) FindAllWithPlayer(ctx context.Context, exec repository.Executor, limit int, offset int, searchName string) ([]entity.UserWithPlayer, error) {
	return nil, errors.New("not implemented")
}

func (s *stubUserRepoForFriendship) FindAllWithPlayerForAdmin(ctx context.Context, exec repository.Executor, limit int, offset int, searchName string) ([]entity.UserWithPlayer, error) {
	return nil, errors.New("not implemented")
}

func (s *stubUserRepoForFriendship) FindByFirebaseUID(ctx context.Context, exec repository.Executor, uid string) (*entity.User, error) {
	return nil, errors.New("not implemented")
}

func (s *stubUserRepoForFriendship) LinkFirebaseUID(ctx context.Context, exec repository.Executor, userID int, currentUID *string, newUID string, updatedAt time.Time) error {
	return errors.New("not implemented")
}

func (s *stubUserRepoForFriendship) DeleteByID(ctx context.Context, exec repository.Executor, id int) error {
	return errors.New("not implemented")
}

func (s *stubUserRepoForFriendship) Save(ctx context.Context, exec repository.Executor, user *entity.User) error {
	return errors.New("not implemented")
}

func newStubFriendshipRepo() *stubFriendshipRepo {
	return &stubFriendshipRepo{
		relations: map[[2]int]*entity.Friendship{},
		counts:    map[int]int{},
		exists:    map[[2]int]bool{},
	}
}

func (s *stubFriendshipRepo) Find(ctx context.Context, exec repository.Executor, userID int, friendUserID int) (*entity.Friendship, error) {
	return s.relations[[2]int{userID, friendUserID}], nil
}

func (s *stubFriendshipRepo) Save(ctx context.Context, exec repository.Executor, friendship *entity.Friendship) error {
	key := [2]int{friendship.UserID, friendship.FriendUserID}
	copied := *friendship
	s.relations[key] = &copied
	s.saved = append(s.saved, &copied)
	return nil
}

func (s *stubFriendshipRepo) Delete(ctx context.Context, exec repository.Executor, userID int, friendUserID int) error {
	delete(s.relations, [2]int{userID, friendUserID})
	s.deleted = append(s.deleted, [2]int{userID, friendUserID})
	return nil
}

func (s *stubFriendshipRepo) DeletePending(ctx context.Context, exec repository.Executor, userID int, friendUserID int) error {
	key := [2]int{userID, friendUserID}
	if s.relations[key] != nil && s.relations[key].StatusID == entity.FriendshipStatusPending {
		delete(s.relations, key)
		s.deleted = append(s.deleted, key)
	}
	return nil
}

func (s *stubFriendshipRepo) DeletePair(ctx context.Context, exec repository.Executor, userID int, friendUserID int) error {
	delete(s.relations, [2]int{userID, friendUserID})
	delete(s.relations, [2]int{friendUserID, userID})
	s.deleted = append(s.deleted, [2]int{userID, friendUserID}, [2]int{friendUserID, userID})
	return nil
}

func (s *stubFriendshipRepo) CountOutgoingActive(ctx context.Context, exec repository.Executor, userID int) (int, error) {
	if count, ok := s.counts[userID]; ok {
		return count, nil
	}
	count := 0
	for key, relation := range s.relations {
		if key[0] == userID && (relation.StatusID == entity.FriendshipStatusPending || relation.StatusID == entity.FriendshipStatusAccepted) {
			count++
		}
	}
	return count, nil
}

func (s *stubFriendshipRepo) ListFriends(ctx context.Context, exec repository.Executor, userID int) ([]*repository.FriendshipWithUserSummary, error) {
	return nil, nil
}

func (s *stubFriendshipRepo) ListReceivedRequests(ctx context.Context, exec repository.Executor, userID int) ([]*repository.FriendshipWithUserSummary, error) {
	return nil, nil
}

func (s *stubFriendshipRepo) ListSentRequests(ctx context.Context, exec repository.Executor, userID int) ([]*repository.FriendshipWithUserSummary, error) {
	return nil, nil
}

func (s *stubFriendshipRepo) ExistsMutualAccepted(ctx context.Context, exec repository.Executor, userID int, friendUserID int) (bool, error) {
	return s.exists[[2]int{userID, friendUserID}], nil
}

func testUser(t *testing.T, id int, name string) *entity.User {
	t.Helper()
	uname, err := username.NewUserName(name)
	require.NoError(t, err)
	return &entity.User{ID: id, Username: uname}
}

func TestFriendshipUsecase_SendRequest(t *testing.T) {
	now := time.Date(2026, 7, 8, 12, 0, 0, 0, time.UTC)

	t.Run("username完全一致で申請を作成する", func(t *testing.T) {
		// Given
		repo := newStubFriendshipRepo()
		u := &friendshipUsecase{
			db:             &MockExecutor{},
			tm:             &spyTransactionManager{executor: &MockExecutor{}},
			userRepo:       &stubUserRepoForFriendship{target: testUser(t, 2, "targetuser")},
			friendshipRepo: repo,
			now:            func() time.Time { return now },
		}

		// When
		err := u.SendRequest(context.Background(), 1, "targetuser")

		// Then
		require.NoError(t, err)
		got := repo.relations[[2]int{1, 2}]
		require.NotNil(t, got)
		assert.Equal(t, entity.FriendshipStatusPending, got.StatusID)
		assert.Equal(t, now, got.RequestedAt)
		assert.Nil(t, got.AcceptedAt)
	})

	t.Run("相手から申請中なら即時に双方向承認する", func(t *testing.T) {
		// Given
		repo := newStubFriendshipRepo()
		incoming, err := entity.NewFriendRequest(2, 1, now.Add(-time.Hour))
		require.NoError(t, err)
		repo.relations[[2]int{2, 1}] = incoming
		u := &friendshipUsecase{
			db:             &MockExecutor{},
			tm:             &spyTransactionManager{executor: &MockExecutor{}},
			userRepo:       &stubUserRepoForFriendship{target: testUser(t, 2, "targetuser")},
			friendshipRepo: repo,
			now:            func() time.Time { return now },
		}

		// When
		err = u.SendRequest(context.Background(), 1, "targetuser")

		// Then
		require.NoError(t, err)
		assert.Equal(t, entity.FriendshipStatusAccepted, repo.relations[[2]int{2, 1}].StatusID)
		assert.Equal(t, entity.FriendshipStatusAccepted, repo.relations[[2]int{1, 2}].StatusID)
		assert.Equal(t, now, *repo.relations[[2]int{1, 2}].AcceptedAt)
	})

	t.Run("申請時は双方ユーザーをID昇順でロックする", func(t *testing.T) {
		// Given
		repo := newStubFriendshipRepo()
		userRepo := &stubUserRepoForFriendship{target: testUser(t, 1, "targetuser")}
		u := &friendshipUsecase{
			db:             &MockExecutor{},
			tm:             &spyTransactionManager{executor: &MockExecutor{}},
			userRepo:       userRepo,
			friendshipRepo: repo,
			now:            func() time.Time { return now },
		}

		// When
		err := u.SendRequest(context.Background(), 2, "targetuser")

		// Then
		require.NoError(t, err)
		assert.Equal(t, []int{1, 2}, userRepo.lockIDs)
	})

	t.Run("外向きpendingとacceptedが100件なら上限エラー", func(t *testing.T) {
		// Given
		repo := newStubFriendshipRepo()
		repo.counts[1] = info.FriendshipMaxOutgoingActive
		u := &friendshipUsecase{
			db:             &MockExecutor{},
			tm:             &spyTransactionManager{executor: &MockExecutor{}},
			userRepo:       &stubUserRepoForFriendship{target: testUser(t, 2, "targetuser")},
			friendshipRepo: repo,
			now:            func() time.Time { return now },
		}

		// When
		err := u.SendRequest(context.Background(), 1, "targetuser")

		// Then
		require.ErrorIs(t, err, ErrFriendshipLimitExceeded)
	})

	t.Run("上限到達時でも重複申請はconflictを優先する", func(t *testing.T) {
		// Given
		repo := newStubFriendshipRepo()
		repo.counts[1] = info.FriendshipMaxOutgoingActive
		existing, err := entity.NewFriendRequest(1, 2, now.Add(-time.Hour))
		require.NoError(t, err)
		repo.relations[[2]int{1, 2}] = existing
		u := &friendshipUsecase{
			db:             &MockExecutor{},
			tm:             &spyTransactionManager{executor: &MockExecutor{}},
			userRepo:       &stubUserRepoForFriendship{target: testUser(t, 2, "targetuser")},
			friendshipRepo: repo,
			now:            func() time.Time { return now },
		}

		// When
		err = u.SendRequest(context.Background(), 1, "targetuser")

		// Then
		require.ErrorIs(t, err, ErrFriendshipAlreadyExists)
	})
}

func TestFriendshipUsecase_AcceptAndReject(t *testing.T) {
	now := time.Date(2026, 7, 8, 12, 0, 0, 0, time.UTC)

	t.Run("受信申請を承認して双方向acceptedにする", func(t *testing.T) {
		// Given
		repo := newStubFriendshipRepo()
		incoming, err := entity.NewFriendRequest(2, 1, now.Add(-time.Hour))
		require.NoError(t, err)
		repo.relations[[2]int{2, 1}] = incoming
		u := &friendshipUsecase{
			db:             &MockExecutor{},
			tm:             &spyTransactionManager{executor: &MockExecutor{}},
			userRepo:       &stubUserRepoForFriendship{target: testUser(t, 2, "requester"), locked: testUser(t, 1, "selfuser")},
			friendshipRepo: repo,
			now:            func() time.Time { return now },
		}

		// When
		err = u.AcceptRequest(context.Background(), 1, "requester")

		// Then
		require.NoError(t, err)
		assert.Equal(t, entity.FriendshipStatusAccepted, repo.relations[[2]int{2, 1}].StatusID)
		assert.Equal(t, entity.FriendshipStatusAccepted, repo.relations[[2]int{1, 2}].StatusID)
	})

	t.Run("承認時に外向きpendingとacceptedが100件なら上限エラー", func(t *testing.T) {
		// Given
		repo := newStubFriendshipRepo()
		repo.counts[1] = info.FriendshipMaxOutgoingActive
		incoming, err := entity.NewFriendRequest(2, 1, now.Add(-time.Hour))
		require.NoError(t, err)
		repo.relations[[2]int{2, 1}] = incoming
		u := &friendshipUsecase{
			db:             &MockExecutor{},
			tm:             &spyTransactionManager{executor: &MockExecutor{}},
			userRepo:       &stubUserRepoForFriendship{target: testUser(t, 2, "requester")},
			friendshipRepo: repo,
			now:            func() time.Time { return now },
		}

		// When
		err = u.AcceptRequest(context.Background(), 1, "requester")

		// Then
		require.ErrorIs(t, err, ErrFriendshipLimitExceeded)
	})

	t.Run("受信申請を拒否するとpendingを削除する", func(t *testing.T) {
		// Given
		repo := newStubFriendshipRepo()
		incoming, err := entity.NewFriendRequest(2, 1, now.Add(-time.Hour))
		require.NoError(t, err)
		repo.relations[[2]int{2, 1}] = incoming
		u := &friendshipUsecase{
			db:             &MockExecutor{},
			tm:             &spyTransactionManager{executor: &MockExecutor{}},
			userRepo:       &stubUserRepoForFriendship{target: testUser(t, 2, "requester")},
			friendshipRepo: repo,
		}

		// When
		err = u.RejectRequest(context.Background(), 1, "requester")

		// Then
		require.NoError(t, err)
		assert.Nil(t, repo.relations[[2]int{2, 1}])
		assert.Contains(t, repo.deleted, [2]int{2, 1})
	})

	t.Run("送信済み申請を取り消すと自分から相手へのpendingを削除する", func(t *testing.T) {
		// Given
		repo := newStubFriendshipRepo()
		outgoing, err := entity.NewFriendRequest(1, 2, now.Add(-time.Hour))
		require.NoError(t, err)
		repo.relations[[2]int{1, 2}] = outgoing
		u := &friendshipUsecase{
			db:             &MockExecutor{},
			tm:             &spyTransactionManager{executor: &MockExecutor{}},
			userRepo:       &stubUserRepoForFriendship{target: testUser(t, 2, "targetuser")},
			friendshipRepo: repo,
		}

		// When
		err = u.CancelRequest(context.Background(), 1, "targetuser")

		// Then
		require.NoError(t, err)
		assert.Nil(t, repo.relations[[2]int{1, 2}])
		assert.Contains(t, repo.deleted, [2]int{1, 2})
	})

	t.Run("送信済み申請がない場合はnot foundを返す", func(t *testing.T) {
		// Given
		repo := newStubFriendshipRepo()
		u := &friendshipUsecase{
			db:             &MockExecutor{},
			tm:             &spyTransactionManager{executor: &MockExecutor{}},
			userRepo:       &stubUserRepoForFriendship{target: testUser(t, 2, "targetuser")},
			friendshipRepo: repo,
		}

		// When
		err := u.CancelRequest(context.Background(), 1, "targetuser")

		// Then
		require.ErrorIs(t, err, ErrFriendRequestNotFound)
	})
}

func TestUserUsecase_PrivateUserAccessibleByFriend(t *testing.T) {
	// Given
	privateUser := testUser(t, 2, "privateuser")
	privateUser.IsPrivate = true
	repo := newStubFriendshipRepo()
	repo.exists[[2]int{1, 2}] = true
	u := &userUsecase{
		db:             &MockExecutor{},
		userRepo:       &stubUserRepoForFav{user: privateUser},
		playerRepo:     &stubPlayerRepoForFav{},
		friendshipRepo: repo,
	}

	// When
	got, err := u.GetUserProfile(context.Background(), "privateuser", &entity.User{ID: 1})

	// Then
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "privateuser", got.Username)
}

func TestScoreHistoryUsecase_PrivateUserAccessibleByFriend(t *testing.T) {
	// Given
	playerID := 10
	privateUser := testUser(t, 2, "privateuser")
	privateUser.IsPrivate = true
	privateUser.PlayerID = &playerID
	userRepo := new(MockUserRepository)
	userRepo.On("FindByUsername", mock.Anything, mock.Anything, "privateuser").Return(privateUser, nil).Once()
	repo := newStubFriendshipRepo()
	repo.exists[[2]int{1, 2}] = true
	u := &scoreHistoryUsecase{
		exec:           &MockExecutor{},
		userRepo:       userRepo,
		friendshipRepo: repo,
	}

	// When
	got, err := u.findVisibleUser(context.Background(), "privateuser", &entity.User{ID: 1})

	// Then
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "privateuser", got.Username.String())
	userRepo.AssertExpectations(t)
}
