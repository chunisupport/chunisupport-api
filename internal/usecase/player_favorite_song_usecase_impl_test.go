package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/displayid"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubUserRepoForFav struct {
	user *entity.User
	err  error
}

func (s *stubUserRepoForFav) FindByUsername(ctx context.Context, exec repository.Executor, username string) (*entity.User, error) {
	return s.user, s.err
}

func (s *stubUserRepoForFav) FindByID(ctx context.Context, exec repository.Executor, id int) (*entity.User, error) {
	return nil, errors.New("not implemented")
}

func (s *stubUserRepoForFav) FindByIDForUpdate(ctx context.Context, exec repository.Executor, id int) (*entity.User, error) {
	if s.user != nil && s.user.ID == id {
		return s.user, s.err
	}
	return s.user, s.err
}

func (s *stubUserRepoForFav) FindAllWithPlayer(ctx context.Context, exec repository.Executor, limit int, offset int, searchName string) ([]entity.UserWithPlayer, error) {
	return nil, errors.New("not implemented")
}

func (s *stubUserRepoForFav) FindAllWithPlayerForAdmin(ctx context.Context, exec repository.Executor, limit int, offset int, searchName string) ([]entity.UserWithPlayer, error) {
	return nil, errors.New("not implemented")
}

func (s *stubUserRepoForFav) FindByFirebaseUID(ctx context.Context, exec repository.Executor, uid string) (*entity.User, error) {
	return nil, errors.New("not implemented")
}

func (s *stubUserRepoForFav) LinkFirebaseUID(ctx context.Context, exec repository.Executor, userID int, currentUID *string, newUID string, updatedAt time.Time) error {
	return errors.New("not implemented")
}

func (s *stubUserRepoForFav) DeleteByID(ctx context.Context, exec repository.Executor, id int) error {
	return errors.New("not implemented")
}

func (s *stubUserRepoForFav) Save(ctx context.Context, exec repository.Executor, user *entity.User) error {
	return errors.New("not implemented")
}

type stubPlayerRepoForFav struct {
	player *entity.Player
	err    error
}

func (s *stubPlayerRepoForFav) FindByUserID(ctx context.Context, exec repository.Executor, userID int) (*entity.Player, error) {
	return s.player, s.err
}

func (s *stubPlayerRepoForFav) FindByUserIDForUpdate(ctx context.Context, exec repository.Executor, userID int) (*entity.Player, error) {
	return s.FindByUserID(ctx, exec, userID)
}

func (s *stubPlayerRepoForFav) FindByID(ctx context.Context, exec repository.Executor, id int) (*entity.Player, error) {
	return nil, errors.New("not implemented")
}

func (s *stubPlayerRepoForFav) FindByIDWithHonors(ctx context.Context, exec repository.Executor, id int) (*repository.PlayerWithHonors, error) {
	return nil, errors.New("not implemented")
}

func (s *stubPlayerRepoForFav) FindHonorsByPlayerID(ctx context.Context, exec repository.Executor, playerID int) ([]*entity.PlayerHonor, error) {
	return nil, errors.New("not implemented")
}

func (s *stubPlayerRepoForFav) FindByIDForUpdate(ctx context.Context, exec repository.Executor, id int) (*entity.Player, error) {
	return s.FindByID(ctx, exec, id)
}

func (s *stubPlayerRepoForFav) Save(ctx context.Context, exec repository.Executor, player *entity.Player) error {
	return errors.New("not implemented")
}

func (s *stubPlayerRepoForFav) DeleteByUserID(ctx context.Context, exec repository.Executor, userID int) error {
	return errors.New("not implemented")
}

type stubFavSongRepoForAdd struct {
	count      int
	exists     bool
	saveErr    error
	countErr   error
	existsErr  error
	savedFav   *entity.PlayerFavoriteSong
	deletedPID int
	deletedSID int
}

func (s *stubFavSongRepoForAdd) CountByPlayerID(ctx context.Context, exec repository.Executor, playerID int) (int, error) {
	return s.count, s.countErr
}

func (s *stubFavSongRepoForAdd) Exists(ctx context.Context, exec repository.Executor, playerID int, songID int) (bool, error) {
	return s.exists, s.existsErr
}

func (s *stubFavSongRepoForAdd) Save(ctx context.Context, exec repository.Executor, favorite *entity.PlayerFavoriteSong) error {
	s.savedFav = favorite
	return s.saveErr
}

func (s *stubFavSongRepoForAdd) Delete(ctx context.Context, exec repository.Executor, playerID int, songID int) error {
	s.deletedPID = playerID
	s.deletedSID = songID
	return nil
}

func (s *stubFavSongRepoForAdd) DeleteBySongID(ctx context.Context, exec repository.Executor, songID int) error {
	return nil
}

type stubFavoriteQueryService struct {
	models []*PlayerFavoriteSongReadModel
	err    error
}

func (s *stubFavoriteQueryService) ListWithSongDetailsByPlayerID(ctx context.Context, exec repository.Executor, playerID int) ([]*PlayerFavoriteSongReadModel, error) {
	return s.models, s.err
}

type stubFavoriteLocker struct {
	err error
}

func (s *stubFavoriteLocker) LockPlayer(ctx context.Context, exec repository.Executor, playerID int) error {
	return s.err
}

type stubSongRepoForFavAdd struct {
	song *entity.Song
	err  error
}

func (s *stubSongRepoForFavAdd) FindByDisplayIDForChange(ctx context.Context, exec repository.Executor, displayID string) (*entity.Song, error) {
	return s.song, s.err
}

func (s *stubSongRepoForFavAdd) FindAllExcludingWorldsend(ctx context.Context, exec repository.Executor, includeDeleted bool) ([]*entity.Song, error) {
	return nil, errors.New("not implemented")
}

func (s *stubSongRepoForFavAdd) FindByDisplayID(ctx context.Context, exec repository.Executor, displayID string) (*entity.Song, error) {
	return nil, errors.New("not implemented")
}

func (s *stubSongRepoForFavAdd) FindByOfficialIdx(ctx context.Context, exec repository.Executor, officialIdx string) (*entity.Song, error) {
	return nil, errors.New("not implemented")
}

func (s *stubSongRepoForFavAdd) FindByOfficialIdxForChange(ctx context.Context, exec repository.Executor, officialIdx string) (*entity.Song, error) {
	return nil, errors.New("not implemented")
}

func (s *stubSongRepoForFavAdd) FindByDisplayIDs(ctx context.Context, exec repository.Executor, displayIDs []string) ([]*entity.Song, error) {
	return nil, errors.New("not implemented")
}

func (s *stubSongRepoForFavAdd) FindLatestUpdatedAt(ctx context.Context, exec repository.Executor) (*time.Time, error) {
	return nil, errors.New("not implemented")
}

func (s *stubSongRepoForFavAdd) Save(ctx context.Context, exec repository.Executor, song *entity.Song) error {
	return errors.New("not implemented")
}

func (s *stubSongRepoForFavAdd) UpdateSongs(ctx context.Context, exec repository.Executor, songs []*entity.Song) error {
	return errors.New("not implemented")
}

func (s *stubSongRepoForFavAdd) Create(ctx context.Context, exec repository.Executor, song *entity.Song) (*entity.Song, error) {
	return nil, errors.New("not implemented")
}

func TestNewPlayerFavoriteSongUsecase(t *testing.T) {
	db := &MockExecutor{}
	tm := &MockTransactionManager{}

	u, err := NewPlayerFavoriteSongUsecase(db, tm, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, u)

	_, err = NewPlayerFavoriteSongUsecase(nil, tm, nil, nil, nil, nil, nil, nil, nil)
	require.Error(t, err)

	_, err = NewPlayerFavoriteSongUsecase(db, nil, nil, nil, nil, nil, nil, nil, nil)
	require.Error(t, err)
}

func TestPlayerFavoriteSongUsecase_List(t *testing.T) {
	t.Run("公開ユーザーのお気に入り一覧を取得できる", func(t *testing.T) {
		now := time.Now()
		u := &playerFavoriteSongUsecase{
			db:         &MockExecutor{},
			userRepo:   &stubUserRepoForFav{user: &entity.User{ID: 1}},
			playerRepo: &stubPlayerRepoForFav{player: &entity.Player{ID: 10, UserID: 1}},
			queryService: &stubFavoriteQueryService{models: []*PlayerFavoriteSongReadModel{
				{DisplayID: "0000000000000001", Title: "楽曲A", Jacket: strPtrFS("a.jpg"), FavoritedAt: now},
			}},
		}

		got, err := u.List(context.Background(), "testuser", nil)
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, "0000000000000001", got[0].DisplayID)
		assert.Equal(t, "楽曲A", got[0].Title)
		assert.Equal(t, "a.jpg", *got[0].Jacket)
		assert.Equal(t, now, got[0].FavoritedAt)
	})

	t.Run("非公開ユーザーを本人が取得できる", func(t *testing.T) {
		u := &playerFavoriteSongUsecase{
			db:         &MockExecutor{},
			userRepo:   &stubUserRepoForFav{user: &entity.User{ID: 1, IsPrivate: true}},
			playerRepo: &stubPlayerRepoForFav{player: &entity.Player{ID: 10, UserID: 1}},
			queryService: &stubFavoriteQueryService{models: []*PlayerFavoriteSongReadModel{
				{DisplayID: "0000000000000001", Title: "楽曲A"},
			}},
		}

		got, err := u.List(context.Background(), "private", &entity.User{ID: 1})
		require.NoError(t, err)
		require.Len(t, got, 1)
	})

	t.Run("非公開ユーザーを承認済みフレンドが取得できる", func(t *testing.T) {
		repo := newStubFriendshipRepo()
		repo.exists[[2]int{2, 1}] = true
		u := &playerFavoriteSongUsecase{
			db:             &MockExecutor{},
			userRepo:       &stubUserRepoForFav{user: &entity.User{ID: 1, IsPrivate: true}},
			playerRepo:     &stubPlayerRepoForFav{player: &entity.Player{ID: 10, UserID: 1}},
			friendshipRepo: repo,
			queryService: &stubFavoriteQueryService{models: []*PlayerFavoriteSongReadModel{
				{DisplayID: "0000000000000001", Title: "楽曲A"},
			}},
		}

		got, err := u.List(context.Background(), "private", &entity.User{ID: 2})
		require.NoError(t, err)
		require.Len(t, got, 1)
	})

	t.Run("非公開ユーザーを他人が取得できない", func(t *testing.T) {
		u := &playerFavoriteSongUsecase{
			db:       &MockExecutor{},
			userRepo: &stubUserRepoForFav{user: &entity.User{ID: 1, IsPrivate: true}},
		}

		_, err := u.List(context.Background(), "private", &entity.User{ID: 2})
		require.ErrorIs(t, err, ErrUserPrivate)
	})

	t.Run("ユーザー未検出でエラー", func(t *testing.T) {
		u := &playerFavoriteSongUsecase{
			db:       &MockExecutor{},
			userRepo: &stubUserRepoForFav{err: repository.ErrUserNotFound},
		}

		_, err := u.List(context.Background(), "nonexistent", nil)
		require.ErrorIs(t, err, ErrUserNotFound)
	})

	t.Run("プレイヤー未連携でエラー", func(t *testing.T) {
		u := &playerFavoriteSongUsecase{
			db:         &MockExecutor{},
			userRepo:   &stubUserRepoForFav{user: &entity.User{ID: 1}},
			playerRepo: &stubPlayerRepoForFav{player: nil},
		}

		_, err := u.List(context.Background(), "testuser", nil)
		require.ErrorIs(t, err, ErrPlayerNotLinked)
	})
}

type spyTransactionManager struct {
	executor repository.Executor
	err      error
}

func (s *spyTransactionManager) Transactional(ctx context.Context, f func(repository.Executor) error) error {
	if s.err != nil {
		return s.err
	}
	return f(s.executor)
}

func TestPlayerFavoriteSongUsecase_Add(t *testing.T) {
	did, err := displayid.NewDisplayID("0000000000000001")
	require.NoError(t, err)

	t.Run("通常楽曲を登録できる", func(t *testing.T) {
		favRepo := &stubFavSongRepoForAdd{count: 0, exists: false}
		u := &playerFavoriteSongUsecase{
			db:           &MockExecutor{},
			tm:           &spyTransactionManager{executor: &MockExecutor{}},
			playerRepo:   &stubPlayerRepoForFav{player: &entity.Player{ID: 10, UserID: 1}},
			songRepo:     &stubSongRepoForFavAdd{song: &entity.Song{ID: 100, DisplayID: "0000000000000001"}},
			favoriteRepo: favRepo,
			locker:       &stubFavoriteLocker{},
		}

		err := u.Add(context.Background(), 1, did)
		require.NoError(t, err)
		require.NotNil(t, favRepo.savedFav)
		assert.Equal(t, 10, favRepo.savedFav.PlayerID)
		assert.Equal(t, 100, favRepo.savedFav.SongID)
	})

	t.Run("プレイヤー未連携でエラー", func(t *testing.T) {
		u := &playerFavoriteSongUsecase{
			db:         &MockExecutor{},
			playerRepo: &stubPlayerRepoForFav{player: nil},
		}

		err := u.Add(context.Background(), 1, did)
		require.ErrorIs(t, err, ErrPlayerNotLinked)
	})

	t.Run("存在しない楽曲は登録できない", func(t *testing.T) {
		u := &playerFavoriteSongUsecase{
			db:         &MockExecutor{},
			tm:         &spyTransactionManager{executor: &MockExecutor{}},
			playerRepo: &stubPlayerRepoForFav{player: &entity.Player{ID: 10, UserID: 1}},
			songRepo:   &stubSongRepoForFavAdd{err: repository.ErrSongNotFound},
		}

		err := u.Add(context.Background(), 1, did)
		require.ErrorIs(t, err, repository.ErrSongNotFound)
	})

	t.Run("論理削除曲は登録できない", func(t *testing.T) {
		u := &playerFavoriteSongUsecase{
			db:         &MockExecutor{},
			tm:         &spyTransactionManager{executor: &MockExecutor{}},
			playerRepo: &stubPlayerRepoForFav{player: &entity.Player{ID: 10, UserID: 1}},
			songRepo:   &stubSongRepoForFavAdd{song: &entity.Song{ID: 100, IsDeleted: true}},
		}

		err := u.Add(context.Background(), 1, did)
		require.ErrorIs(t, err, repository.ErrSongNotFound)
	})

	t.Run("WORLD'S ENDは登録できない", func(t *testing.T) {
		u := &playerFavoriteSongUsecase{
			db:         &MockExecutor{},
			tm:         &spyTransactionManager{executor: &MockExecutor{}},
			playerRepo: &stubPlayerRepoForFav{player: &entity.Player{ID: 10, UserID: 1}},
			songRepo:   &stubSongRepoForFavAdd{song: &entity.Song{ID: 100, IsWorldsend: true}},
		}

		err := u.Add(context.Background(), 1, did)
		require.ErrorIs(t, err, repository.ErrSongNotFound)
	})

	t.Run("重複登録は成功する", func(t *testing.T) {
		favRepo := &stubFavSongRepoForAdd{exists: true}
		u := &playerFavoriteSongUsecase{
			db:           &MockExecutor{},
			tm:           &spyTransactionManager{executor: &MockExecutor{}},
			playerRepo:   &stubPlayerRepoForFav{player: &entity.Player{ID: 10, UserID: 1}},
			songRepo:     &stubSongRepoForFavAdd{song: &entity.Song{ID: 100}},
			favoriteRepo: favRepo,
			locker:       &stubFavoriteLocker{},
		}

		err := u.Add(context.Background(), 1, did)
		require.NoError(t, err)
		assert.Nil(t, favRepo.savedFav)
	})

	t.Run("99件から100件目を登録できる", func(t *testing.T) {
		favRepo := &stubFavSongRepoForAdd{count: info.PlayerFavoriteSongMaxCount - 1, exists: false}
		u := &playerFavoriteSongUsecase{
			db:           &MockExecutor{},
			tm:           &spyTransactionManager{executor: &MockExecutor{}},
			playerRepo:   &stubPlayerRepoForFav{player: &entity.Player{ID: 10, UserID: 1}},
			songRepo:     &stubSongRepoForFavAdd{song: &entity.Song{ID: 100}},
			favoriteRepo: favRepo,
			locker:       &stubFavoriteLocker{},
		}

		err := u.Add(context.Background(), 1, did)
		require.NoError(t, err)
		require.NotNil(t, favRepo.savedFav)
	})

	t.Run("100件超えは上限エラー", func(t *testing.T) {
		favRepo := &stubFavSongRepoForAdd{count: info.PlayerFavoriteSongMaxCount, exists: false}
		u := &playerFavoriteSongUsecase{
			db:           &MockExecutor{},
			tm:           &spyTransactionManager{executor: &MockExecutor{}},
			playerRepo:   &stubPlayerRepoForFav{player: &entity.Player{ID: 10, UserID: 1}},
			songRepo:     &stubSongRepoForFavAdd{song: &entity.Song{ID: 100}},
			favoriteRepo: favRepo,
			locker:       &stubFavoriteLocker{},
		}

		err := u.Add(context.Background(), 1, did)
		require.ErrorIs(t, err, ErrPlayerFavoriteSongLimitExceeded)
	})

	t.Run("100件でも重複登録は成功", func(t *testing.T) {
		favRepo := &stubFavSongRepoForAdd{count: info.PlayerFavoriteSongMaxCount, exists: true}
		u := &playerFavoriteSongUsecase{
			db:           &MockExecutor{},
			tm:           &spyTransactionManager{executor: &MockExecutor{}},
			playerRepo:   &stubPlayerRepoForFav{player: &entity.Player{ID: 10, UserID: 1}},
			songRepo:     &stubSongRepoForFavAdd{song: &entity.Song{ID: 100}},
			favoriteRepo: favRepo,
			locker:       &stubFavoriteLocker{},
		}

		err := u.Add(context.Background(), 1, did)
		require.NoError(t, err)
		assert.Nil(t, favRepo.savedFav)
	})
}

func TestPlayerFavoriteSongUsecase_Remove(t *testing.T) {
	did, err := displayid.NewDisplayID("0000000000000001")
	require.NoError(t, err)

	t.Run("登録済みを解除できる", func(t *testing.T) {
		songID := 100
		favRepo := &stubFavSongRepoForAdd{}
		u := &playerFavoriteSongUsecase{
			db:           &MockExecutor{},
			playerRepo:   &stubPlayerRepoForFav{player: &entity.Player{ID: 10, UserID: 1}},
			favoriteRepo: favRepo,
			resolver:     &stubPlayerSongIDResolver{songID: &songID},
		}

		err := u.Remove(context.Background(), 1, did)
		require.NoError(t, err)
		assert.Equal(t, 10, favRepo.deletedPID)
		assert.Equal(t, 100, favRepo.deletedSID)
	})

	t.Run("未登録解除が成功する", func(t *testing.T) {
		songID := 100
		favRepo := &stubFavSongRepoForAdd{}
		u := &playerFavoriteSongUsecase{
			db:           &MockExecutor{},
			playerRepo:   &stubPlayerRepoForFav{player: &entity.Player{ID: 10, UserID: 1}},
			favoriteRepo: favRepo,
			resolver:     &stubPlayerSongIDResolver{songID: &songID},
		}

		err := u.Remove(context.Background(), 1, did)
		require.NoError(t, err)
	})

	t.Run("存在しない楽曲の解除が成功する", func(t *testing.T) {
		favRepo := &stubFavSongRepoForAdd{}
		u := &playerFavoriteSongUsecase{
			db:           &MockExecutor{},
			playerRepo:   &stubPlayerRepoForFav{player: &entity.Player{ID: 10, UserID: 1}},
			favoriteRepo: favRepo,
			resolver:     &stubPlayerSongIDResolver{songID: nil},
		}

		err := u.Remove(context.Background(), 1, did)
		require.NoError(t, err)
		assert.Equal(t, 0, favRepo.deletedPID)
	})

	t.Run("プレイヤー未連携でエラー", func(t *testing.T) {
		u := &playerFavoriteSongUsecase{
			db:         &MockExecutor{},
			playerRepo: &stubPlayerRepoForFav{player: nil},
		}

		err := u.Remove(context.Background(), 1, did)
		require.ErrorIs(t, err, ErrPlayerNotLinked)
	})
}

func strPtrFS(s string) *string {
	return &s
}
