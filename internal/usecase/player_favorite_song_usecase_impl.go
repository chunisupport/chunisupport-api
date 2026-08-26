package usecase

import (
	"context"
	"errors"
	"log/slog"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/displayid"
	"github.com/chunisupport/chunisupport-api/internal/info"
)

var (
	errPlayerFavoriteSongNilDB = errors.New("database executor is nil")
	errPlayerFavoriteSongNilTM = errors.New("transaction manager is nil")
)

type playerFavoriteSongUsecase struct {
	db             repository.Executor
	tm             TransactionManager
	userRepo       repository.UserRepository
	playerRepo     repository.PlayerRepository
	songRepo       repository.SongRepository
	favoriteRepo   repository.PlayerFavoriteSongRepository
	friendshipRepo repository.FriendshipRepository
	queryService   PlayerFavoriteSongQueryService
	locker         PlayerFavoriteSongLocker
	resolver       PlayerSongIDResolver
}

// SetFriendshipRepository は非公開ユーザー閲覧時のフレンド判定リポジトリを設定します。
func (u *playerFavoriteSongUsecase) SetFriendshipRepository(friendshipRepo repository.FriendshipRepository) {
	u.friendshipRepo = friendshipRepo
}

func NewPlayerFavoriteSongUsecase(
	db repository.Executor,
	tm TransactionManager,
	userRepo repository.UserRepository,
	playerRepo repository.PlayerRepository,
	songRepo repository.SongRepository,
	favoriteRepo repository.PlayerFavoriteSongRepository,
	queryService PlayerFavoriteSongQueryService,
	locker PlayerFavoriteSongLocker,
	resolver PlayerSongIDResolver,
) (PlayerFavoriteSongUsecase, error) {
	if db == nil {
		return nil, errPlayerFavoriteSongNilDB
	}
	if tm == nil {
		return nil, errPlayerFavoriteSongNilTM
	}
	return &playerFavoriteSongUsecase{
		db:           db,
		tm:           tm,
		userRepo:     userRepo,
		playerRepo:   playerRepo,
		songRepo:     songRepo,
		favoriteRepo: favoriteRepo,
		queryService: queryService,
		locker:       locker,
		resolver:     resolver,
	}, nil
}

func (u *playerFavoriteSongUsecase) List(ctx context.Context, username string, requester *entity.User) ([]*PlayerFavoriteSongOutput, error) {
	user, err := u.userRepo.FindByUsername(ctx, u.db, username)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrUserNotFound
		}
		slog.Error("failed to find user by username", "username", username, "error", err)
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	accessible, err := canAccessPrivateUser(ctx, u.db, u.friendshipRepo, user, requester)
	if err != nil {
		return nil, err
	}
	if !accessible {
		return nil, ErrUserPrivate
	}
	player, err := u.playerRepo.FindByUserID(ctx, u.db, user.ID)
	if err != nil {
		return nil, err
	}
	if player == nil {
		return nil, ErrPlayerNotLinked
	}
	rows, err := u.queryService.ListWithSongDetailsByPlayerID(ctx, u.db, player.ID)
	if err != nil {
		return nil, err
	}
	items := make([]*PlayerFavoriteSongOutput, 0, len(rows))
	for _, row := range rows {
		items = append(items, &PlayerFavoriteSongOutput{
			DisplayID:   row.DisplayID,
			Title:       row.Title,
			Jacket:      row.Jacket,
			FavoritedAt: row.FavoritedAt,
		})
	}
	return items, nil
}

func (u *playerFavoriteSongUsecase) Add(ctx context.Context, userID int, displayID displayid.DisplayID) error {
	player, err := u.playerRepo.FindByUserID(ctx, u.db, userID)
	if err != nil {
		return err
	}
	if player == nil {
		return ErrPlayerNotLinked
	}

	return u.tm.Transactional(ctx, func(tx repository.Executor) error {
		song, err := u.songRepo.FindByDisplayIDForChange(ctx, tx, displayID.String())
		if err != nil {
			if errors.Is(err, repository.ErrSongNotFound) {
				return repository.ErrSongNotFound
			}
			return err
		}
		if song == nil || song.IsDeleted || song.IsWorldsend {
			return repository.ErrSongNotFound
		}
		if err := u.locker.LockPlayer(ctx, tx, player.ID); err != nil {
			return err
		}
		exists, err := u.favoriteRepo.Exists(ctx, tx, player.ID, song.ID)
		if err != nil {
			return err
		}
		if exists {
			return nil
		}
		count, err := u.favoriteRepo.CountByPlayerID(ctx, tx, player.ID)
		if err != nil {
			return err
		}
		if count >= info.PlayerFavoriteSongMaxCount {
			return ErrPlayerFavoriteSongLimitExceeded
		}
		favorite, err := entity.NewPlayerFavoriteSong(player.ID, song.ID)
		if err != nil {
			return err
		}
		return u.favoriteRepo.Save(ctx, tx, favorite)
	})
}

func (u *playerFavoriteSongUsecase) Remove(ctx context.Context, userID int, displayID displayid.DisplayID) error {
	player, err := u.playerRepo.FindByUserID(ctx, u.db, userID)
	if err != nil {
		return err
	}
	if player == nil {
		return ErrPlayerNotLinked
	}
	songID, err := u.resolver.ResolveSongIDByDisplayID(ctx, u.db, displayID.String())
	if err != nil {
		return err
	}
	if songID == nil {
		return nil
	}
	return u.favoriteRepo.Delete(ctx, u.db, player.ID, *songID)
}
