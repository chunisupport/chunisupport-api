package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	domainservice "github.com/chunisupport/chunisupport-api/internal/domain/service"
	"github.com/chunisupport/chunisupport-api/internal/info"
)

var (
	errPlayerLockedSongInputRequired = errors.New("input is required")
	errPlayerLockedSongNilDB         = errors.New("database executor is nil")
	errPlayerLockedSongNilTM         = errors.New("transaction manager is nil")
)

type playerLockedSongUsecase struct {
	db             repository.Executor
	tm             TransactionManager
	userRepo       repository.UserRepository
	playerRepo     repository.PlayerRepository
	playerRecRepo  repository.PlayerRecordRepository
	playerDataRepo repository.PlayerDataRepository
	songRepo       repository.SongRepository
	lockedRepo     repository.PlayerLockedSongRepository
	friendshipRepo repository.FriendshipRepository
	queryService   PlayerLockedSongQueryService
	resolver       PlayerSongIDResolver
}

// SetFriendshipRepository は非公開ユーザー閲覧時のフレンド判定リポジトリを設定します。
func (u *playerLockedSongUsecase) SetFriendshipRepository(friendshipRepo repository.FriendshipRepository) {
	u.friendshipRepo = friendshipRepo
}

func NewPlayerLockedSongUsecase(db repository.Executor, tm TransactionManager, userRepo repository.UserRepository, playerRepo repository.PlayerRepository, playerRecRepo repository.PlayerRecordRepository, playerDataRepo repository.PlayerDataRepository, songRepo repository.SongRepository, lockedRepo repository.PlayerLockedSongRepository, queryService PlayerLockedSongQueryService, resolver PlayerSongIDResolver) (PlayerLockedSongUsecase, error) {
	if db == nil {
		return nil, errPlayerLockedSongNilDB
	}
	if tm == nil {
		return nil, errPlayerLockedSongNilTM
	}
	return &playerLockedSongUsecase{db: db, tm: tm, userRepo: userRepo, playerRepo: playerRepo, playerRecRepo: playerRecRepo, playerDataRepo: playerDataRepo, songRepo: songRepo, lockedRepo: lockedRepo, queryService: queryService, resolver: resolver}, nil
}

func (u *playerLockedSongUsecase) List(ctx context.Context, username string, requester *entity.User) ([]*PlayerLockedSongOutput, error) {
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
	rows, err := u.queryService.ListWithSongDisplayIDAndTitleByPlayerID(ctx, u.db, player.ID)
	if err != nil {
		return nil, err
	}
	items := make([]*PlayerLockedSongOutput, 0, len(rows))
	for _, row := range rows {
		items = append(items, &PlayerLockedSongOutput{DisplayID: row.DisplayID, Title: row.Title, IsUltima: row.IsUltima})
	}
	return items, nil
}

func (u *playerLockedSongUsecase) Lock(ctx context.Context, userID int, input *PlayerLockedSongInput) error {
	if input == nil {
		return errPlayerLockedSongInputRequired
	}
	if u.tm == nil {
		return errPlayerLockedSongNilTM
	}
	return u.tm.Transactional(ctx, func(tx repository.Executor) error {
		player, err := u.findPlayerForUpdate(ctx, tx, userID)
		if err != nil {
			return err
		}
		song, err := u.songRepo.FindByDisplayID(ctx, tx, input.DisplayID.String())
		if err != nil {
			if errors.Is(err, repository.ErrSongNotFound) {
				return repository.ErrSongNotFound
			}
			return err
		}
		if song == nil || song.IsDeleted || song.IsWorldsend {
			return repository.ErrSongNotFound
		}
		if input.IsUltima && !song.HasDifficultyChart(domainservice.DifficultyIDUltima) {
			return ErrChartNotFound
		}
		lockedSong, err := entity.NewPlayerLockedSong(player.ID, song.ID, input.IsUltima)
		if err != nil {
			return err
		}
		if err := u.lockedRepo.Create(ctx, tx, lockedSong); err != nil {
			return err
		}
		return u.recalculatePlayerOverpowerWithTx(ctx, tx, player)
	})
}

func (u *playerLockedSongUsecase) Unlock(ctx context.Context, userID int, input *PlayerLockedSongInput) error {
	if input == nil {
		return errPlayerLockedSongInputRequired
	}
	if u.tm == nil {
		return errPlayerLockedSongNilTM
	}
	return u.tm.Transactional(ctx, func(tx repository.Executor) error {
		player, err := u.findPlayerForUpdate(ctx, tx, userID)
		if err != nil {
			return err
		}
		songID, err := u.resolver.ResolveSongIDByDisplayID(ctx, tx, input.DisplayID.String())
		if err != nil {
			return err
		}
		if songID == nil {
			return nil
		}
		if err := u.lockedRepo.Delete(ctx, tx, player.ID, *songID, input.IsUltima); err != nil {
			return err
		}
		return u.recalculatePlayerOverpowerWithTx(ctx, tx, player)
	})
}

func (u *playerLockedSongUsecase) Batch(ctx context.Context, userID int, input *PlayerLockedSongBatchInput) error {
	if input == nil {
		return errPlayerLockedSongInputRequired
	}
	if u.tm == nil {
		return errPlayerLockedSongNilTM
	}
	return u.tm.Transactional(ctx, func(tx repository.Executor) error {
		player, err := u.findPlayerForUpdate(ctx, tx, userID)
		if err != nil {
			return err
		}

		addDisplayIDs := make([]string, 0, len(input.Add))
		for _, addInput := range input.Add {
			if addInput == nil {
				return errPlayerLockedSongInputRequired
			}
			addDisplayIDs = append(addDisplayIDs, addInput.DisplayID.String())
		}
		songs, err := u.songRepo.FindByDisplayIDs(ctx, tx, addDisplayIDs)
		if err != nil {
			return err
		}
		songByDisplayID := make(map[string]*entity.Song, len(songs))
		for _, song := range songs {
			songByDisplayID[song.DisplayID] = song
		}
		lockedSongsToAdd := make([]*entity.PlayerLockedSong, 0, len(input.Add))
		for _, addInput := range input.Add {
			song, ok := songByDisplayID[addInput.DisplayID.String()]
			if !ok || song == nil || song.IsDeleted || song.IsWorldsend {
				return repository.ErrSongNotFound
			}
			if addInput.IsUltima && !song.HasDifficultyChart(domainservice.DifficultyIDUltima) {
				return ErrChartNotFound
			}
			lockedSong, err := entity.NewPlayerLockedSong(player.ID, song.ID, addInput.IsUltima)
			if err != nil {
				return err
			}
			lockedSongsToAdd = append(lockedSongsToAdd, lockedSong)
		}

		deleteDisplayIDs := make([]string, 0, len(input.Delete))
		for _, deleteInput := range input.Delete {
			if deleteInput == nil {
				return errPlayerLockedSongInputRequired
			}
			deleteDisplayIDs = append(deleteDisplayIDs, deleteInput.DisplayID.String())
		}
		resolvedSongIDs, err := u.resolver.ResolveSongIDsByDisplayIDs(ctx, tx, deleteDisplayIDs)
		if err != nil {
			return err
		}
		songIDsToDelete := make([]int, 0, len(input.Delete))
		isUltimaFlagsToDelete := make([]bool, 0, len(input.Delete))
		deleteKeys := make(map[string]struct{}, len(input.Delete))
		for _, deleteInput := range input.Delete {
			songID, ok := resolvedSongIDs[deleteInput.DisplayID.String()]
			if !ok {
				continue
			}
			key := fmt.Sprintf("%d:%t", songID, deleteInput.IsUltima)
			if _, exists := deleteKeys[key]; exists {
				continue
			}
			deleteKeys[key] = struct{}{}
			songIDsToDelete = append(songIDsToDelete, songID)
			isUltimaFlagsToDelete = append(isUltimaFlagsToDelete, deleteInput.IsUltima)
		}
		if len(lockedSongsToAdd) == 0 && len(songIDsToDelete) == 0 {
			return nil
		}

		if len(lockedSongsToAdd) > 0 {
			if err := u.lockedRepo.BulkCreate(ctx, tx, lockedSongsToAdd); err != nil {
				return err
			}
		}
		if len(songIDsToDelete) > 0 {
			if err := u.lockedRepo.BulkDelete(ctx, tx, player.ID, songIDsToDelete, isUltimaFlagsToDelete); err != nil {
				return err
			}
		}
		return u.recalculatePlayerOverpowerWithTx(ctx, tx, player)
	})
}

func (u *playerLockedSongUsecase) findPlayerForUpdate(ctx context.Context, exec repository.Executor, userID int) (*entity.Player, error) {
	player, err := u.playerRepo.FindByUserIDForUpdate(ctx, exec, userID)
	if err != nil {
		return nil, err
	}
	if player == nil {
		return nil, ErrPlayerNotLinked
	}
	return player, nil
}

func (u *playerLockedSongUsecase) recalculatePlayerOverpowerWithTx(ctx context.Context, exec repository.Executor, player *entity.Player) error {
	if player == nil {
		return ErrPlayerNotLinked
	}
	records, err := u.playerRecRepo.FindByPlayerID(ctx, exec, player.ID)
	if err != nil {
		return err
	}
	lockedSongs, err := u.lockedRepo.ListByPlayerID(ctx, exec, player.ID)
	if err != nil {
		return err
	}
	lockedSet := make(map[string]struct{}, len(lockedSongs))
	for _, lockedSong := range lockedSongs {
		key := lockedSongKey(lockedSong.SongID, lockedSong.IsUltima)
		lockedSet[key] = struct{}{}
	}
	overpowerRecords, err := playerRecordsToOverpowerRecords(records, false, func(record *entity.PlayerRecord) bool {
		if record.ChartDifficulty == nil {
			return false
		}
		_, exists := lockedSet[lockedSongKey(record.Song.ID, record.ChartDifficulty.Name == info.DifficultyNameUltima)]
		return !exists
	})
	if err != nil {
		return err
	}

	stats, err := u.playerDataRepo.GetOverpowerTargetStatsWithExecutor(ctx, exec, repository.OverpowerTargetFilter{ExcludeWorldsend: true, ExcludeDeleted: true, PlayerID: &player.ID})
	if err != nil {
		return err
	}
	value, percent := domainservice.CalcOverpowerSummary(overpowerRecords, stats.MaxOverpowerTotal)
	player.ChangeOverpower(&value, &percent)
	return u.playerRepo.Save(ctx, exec, player)
}

func lockedSongKey(songID int, isUltima bool) string {
	return fmt.Sprintf("%d:%t", songID, isUltima)
}
