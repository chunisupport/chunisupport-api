package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainmasterdata "github.com/chunisupport/chunisupport-api/internal/domain/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/levelstar"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/notes"
	"github.com/chunisupport/chunisupport-api/internal/info"
)

const worldsendChartKey = "WORLDSEND"

// UpdateWorldsendChartInput は WORLD'S END 譜面更新入力を表します。
type UpdateWorldsendChartInput struct {
	Attribute *string
	LevelStar *int
	Notes     *int
}

// UpdateWorldsendSongInput は WORLD'S END 楽曲更新入力を表します。
type UpdateWorldsendSongInput struct {
	DisplayID  string
	Title      string
	Artist     string
	Genre      *string
	BPM        *int
	ReleasedAt *time.Time
	Jacket     *string
	Charts     map[string]*UpdateWorldsendChartInput
}

// WorldsendUsecase は WORLD'S END 楽曲に関するユースケースを提供します。
type WorldsendUsecase interface {
	// GetAllWorldsendSongs は全 WORLD'S END 楽曲を取得します。
	// includeDeleted が true かつ requesterAccountTypeID が EDITOR 未満の場合、削除済み楽曲は除外されます。
	GetAllWorldsendSongs(ctx context.Context, includeDeleted bool, requesterAccountTypeID *int) ([]*repository.WorldsendSongWithChart, error)

	// GetWorldsendSongByDisplayID は指定された DisplayID の WORLD'S END 楽曲を取得します。
	// requesterAccountTypeIDがnilまたはEDITOR(2)未満の場合、削除済み楽曲はErrSongNotFoundを返します。
	GetWorldsendSongByDisplayID(ctx context.Context, displayID string, requesterAccountTypeID *int) (*repository.WorldsendSongWithChart, error)

	// DeleteWorldsendSong は指定された DisplayID の WORLD'S END 楽曲を論理削除します。
	DeleteWorldsendSong(ctx context.Context, displayID string) error

	// RestoreWorldsendSong は指定された DisplayID の WORLD'S END 楽曲を復活させます。
	RestoreWorldsendSong(ctx context.Context, displayID string) error

	// UpdateWorldsendSongs は WORLD'S END 楽曲および譜面情報を一括更新します。
	UpdateWorldsendSongs(ctx context.Context, requests []*UpdateWorldsendSongInput, masters *domainmasterdata.SongMasters) error
}

// worldsendUsecase は WorldsendUsecase の実装です。
type worldsendUsecase struct {
	worldsendChartRepo repository.WorldsendChartRepository
	defaultExecutor    repository.Executor
	tm                 TransactionManager
}

// NewWorldsendUsecase は WorldsendUsecase の実装を生成します。
func NewWorldsendUsecase(worldsendChartRepo repository.WorldsendChartRepository, tm TransactionManager, defaultExecutor repository.Executor) WorldsendUsecase {
	return &worldsendUsecase{
		worldsendChartRepo: worldsendChartRepo,
		defaultExecutor:    defaultExecutor,
		tm:                 tm,
	}
}

// GetAllWorldsendSongs は全 WORLD'S END 楽曲を取得します。
// includeDeleted が true かつ requesterAccountTypeID が EDITOR 未満の場合、削除済み楽曲は除外されます。
func (s *worldsendUsecase) GetAllWorldsendSongs(ctx context.Context, includeDeleted bool, requesterAccountTypeID *int) ([]*repository.WorldsendSongWithChart, error) {
	// 削除済み楽曲を含める場合はEDITOR権限が必要
	if includeDeleted {
		if requesterAccountTypeID == nil || *requesterAccountTypeID < info.AccountTypeEditor {
			includeDeleted = false
		}
	}

	songsWithCharts, err := s.worldsendChartRepo.FindAll(ctx, s.defaultExecutor, includeDeleted)
	if err != nil {
		slog.Error("failed to find all worldsend songs", "error", err)
		return nil, err
	}

	return songsWithCharts, nil
}

// GetWorldsendSongByDisplayID は指定された DisplayID の WORLD'S END 楽曲を取得します。
// requesterAccountTypeIDがnilまたはEDITOR(2)未満の場合、削除済み楽曲はErrSongNotFoundを返します。
func (s *worldsendUsecase) GetWorldsendSongByDisplayID(ctx context.Context, displayID string, requesterAccountTypeID *int) (*repository.WorldsendSongWithChart, error) {
	songWithChart, err := s.worldsendChartRepo.FindByDisplayID(ctx, s.defaultExecutor, displayID)
	if err != nil {
		if errors.Is(err, repository.ErrSongNotFound) {
			return nil, repository.ErrSongNotFound
		}
		slog.Error("failed to find worldsend song by display_id", "display_id", displayID, "error", err)
		return nil, err
	}

	// 削除済み楽曲の権限チェック
	if !songWithChart.Song.IsActive() {
		// EDITOR以上の権限を持たない場合は404を返す
		if requesterAccountTypeID == nil || *requesterAccountTypeID < info.AccountTypeEditor {
			return nil, repository.ErrSongNotFound
		}
	}

	return songWithChart, nil
}

// DeleteWorldsendSong は指定された DisplayID の WORLD'S END 楽曲を論理削除します。
func (s *worldsendUsecase) DeleteWorldsendSong(ctx context.Context, displayID string) error {
	return s.tm.Transactional(ctx, func(tx repository.Executor) error {
		songWithChart, err := s.worldsendChartRepo.FindByDisplayID(ctx, tx, displayID)
		if err != nil {
			return err
		}

		songWithChart.Song.Delete()
		return s.worldsendChartRepo.SaveSong(ctx, tx, songWithChart.Song)
	})
}

// RestoreWorldsendSong は指定された DisplayID の WORLD'S END 楽曲を復活させます。
func (s *worldsendUsecase) RestoreWorldsendSong(ctx context.Context, displayID string) error {
	return s.tm.Transactional(ctx, func(tx repository.Executor) error {
		songWithChart, err := s.worldsendChartRepo.FindByDisplayID(ctx, tx, displayID)
		if err != nil {
			return err
		}

		songWithChart.Song.Restore()
		return s.worldsendChartRepo.SaveSong(ctx, tx, songWithChart.Song)
	})
}

// UpdateWorldsendSongs は WORLD'S END 楽曲および譜面情報を一括更新します。

func (s *worldsendUsecase) UpdateWorldsendSongs(ctx context.Context, requests []*UpdateWorldsendSongInput, masters *domainmasterdata.SongMasters) error {
	if masters == nil {
		return fmt.Errorf("%w: masters is nil", ErrInternalError)
	}

	if len(requests) == 0 {
		return nil
	}

	updates, err := convertWorldsendRequestsToEntities(requests, masters)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidWorldsendInput, err)
	}

	// リポジトリに委譲
	if err := s.tm.Transactional(ctx, func(tx repository.Executor) error {
		return s.worldsendChartRepo.UpdateSongs(ctx, tx, updates)
	}); err != nil {
		if errors.Is(err, repository.ErrDuplicateDisplayID) {
			return fmt.Errorf("%w: %w", ErrInvalidWorldsendInput, err)
		}
		slog.Error("failed to update worldsend songs", "error", err)
		return err
	}

	return nil
}

func convertWorldsendRequestsToEntities(requests []*UpdateWorldsendSongInput, masters *domainmasterdata.SongMasters) ([]*repository.WorldsendUpdate, error) {
	updates := make([]*repository.WorldsendUpdate, 0, len(requests))

	for idx, req := range requests {
		if req == nil {
			return nil, fmt.Errorf("requests[%d]: request is null", idx)
		}

		chartReq, hasChartUpdate, err := validateAndGetWorldsendChartRequest(req.Charts)
		if err != nil {
			return nil, fmt.Errorf("requests[%d].charts: %w", idx, err)
		}

		var genreID *int
		if req.Genre != nil {
			genreMaster, ok := masters.Genres[*req.Genre]
			if !ok {
				return nil, fmt.Errorf("invalid genre: %s", *req.Genre)
			}
			genreID = &genreMaster.ID
		}

		updatedSong := entity.NewSong()
		updatedSong.DisplayID = req.DisplayID
		updatedSong.Title = req.Title
		updatedSong.Artist = req.Artist
		updatedSong.GenreID = genreID
		updatedSong.BPM = req.BPM
		updatedSong.ReleasedAt = req.ReleasedAt
		updatedSong.Jacket = req.Jacket
		updatedSong.IsWorldsend = true

		var updatedChart *entity.WorldsendChart
		if hasChartUpdate {
			var levelStarVO *levelstar.LevelStar
			if chartReq.LevelStar != nil {
				ls, lsErr := levelstar.NewLevelStar(*chartReq.LevelStar)
				if lsErr != nil {
					return nil, fmt.Errorf("requests[%d].charts.%s.level_star: %w", idx, worldsendChartKey, lsErr)
				}
				levelStarVO = &ls
			}

			var notesVO *notes.Notes
			if chartReq.Notes != nil {
				n, nErr := notes.NewNotes(*chartReq.Notes)
				if nErr != nil {
					return nil, fmt.Errorf("requests[%d].charts.%s.notes: %w", idx, worldsendChartKey, nErr)
				}
				notesVO = &n
			}

			updatedChart = &entity.WorldsendChart{
				LevelStar: levelStarVO,
				Attribute: chartReq.Attribute,
				Notes:     notesVO,
			}
		}

		updates = append(updates, &repository.WorldsendUpdate{
			Song:  updatedSong,
			Chart: updatedChart,
		})
	}

	return updates, nil
}

func validateAndGetWorldsendChartRequest(charts map[string]*UpdateWorldsendChartInput) (*UpdateWorldsendChartInput, bool, error) {
	if len(charts) == 0 {
		return nil, false, nil
	}

	if len(charts) > 1 {
		keys := slices.Sorted(maps.Keys(charts))
		return nil, false, fmt.Errorf("only one chart key (%s) is allowed: got %v", worldsendChartKey, keys)
	}

	chart, ok := charts[worldsendChartKey]
	if !ok {
		var invalidKey string
		for k := range charts {
			invalidKey = k
		}
		return nil, false, fmt.Errorf("unsupported chart key: %s", invalidKey)
	}

	if chart == nil {
		return nil, false, fmt.Errorf("chart for %s is null", worldsendChartKey)
	}

	return chart, true, nil
}
