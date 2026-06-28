package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
)

var (
	ErrScoreHistoryNotFound              = errors.New("score history not found")
	ErrScoreHistoryUnsupportedDifficulty = errors.New("score history unsupported difficulty")
)

// ScoreHistoryEntry はAPIへ返すスコア履歴の1件を表します。
type ScoreHistoryEntry struct {
	Score     int
	ClearLamp *string
	ComboLamp *string
	FullChain *string
	UpdatedAt time.Time
}

// ScoreHistoryUsecase は譜面単位のスコア履歴取得を提供します。
type ScoreHistoryUsecase interface {
	GetStandard(ctx context.Context, username string, requester *entity.User, displayID, difficulty string) ([]ScoreHistoryEntry, error)
	GetWorldsend(ctx context.Context, username string, requester *entity.User, displayID string) ([]ScoreHistoryEntry, error)
}

type scoreHistoryUsecase struct {
	exec           repository.Executor
	userRepo       repository.UserRepository
	songRepo       repository.SongRepository
	worldsendRepo  repository.WorldsendChartRepository
	historyRepo    repository.ScoreHistoryRepository
	masterProvider repository.PlayerDataMasterProvider
}

// NewScoreHistoryUsecase はスコア履歴取得ユースケースを生成します。
func NewScoreHistoryUsecase(
	exec repository.Executor,
	userRepo repository.UserRepository,
	songRepo repository.SongRepository,
	worldsendRepo repository.WorldsendChartRepository,
	historyRepo repository.ScoreHistoryRepository,
	masterProvider repository.PlayerDataMasterProvider,
) ScoreHistoryUsecase {
	return &scoreHistoryUsecase{
		exec: exec, userRepo: userRepo, songRepo: songRepo, worldsendRepo: worldsendRepo,
		historyRepo: historyRepo, masterProvider: masterProvider,
	}
}

func (us *scoreHistoryUsecase) GetStandard(ctx context.Context, username string, requester *entity.User, displayID, difficulty string) ([]ScoreHistoryEntry, error) {
	if !entity.SupportsScoreHistory(difficulty) {
		return nil, ErrScoreHistoryUnsupportedDifficulty
	}
	user, err := us.findVisibleUser(ctx, username, requester)
	if err != nil {
		return nil, err
	}
	song, err := us.songRepo.FindByDisplayID(ctx, us.exec, displayID)
	if err != nil {
		return nil, err
	}
	masters := us.masterProvider.PlayerDataMasters()
	difficultyMaster, exists := masters.Difficulties[difficulty]
	if !exists {
		return nil, ErrInvalidDifficulty
	}
	var chartID int
	for _, chart := range song.Charts {
		if chart.DifficultyID == difficultyMaster.ID {
			chartID = chart.ID
			break
		}
	}
	if chartID == 0 {
		return nil, ErrChartNotFound
	}
	rows, err := us.historyRepo.FindStandardTimeline(ctx, *user.PlayerID, chartID)
	if err != nil {
		return nil, err
	}
	return us.convertEntries(rows)
}

func (us *scoreHistoryUsecase) GetWorldsend(ctx context.Context, username string, requester *entity.User, displayID string) ([]ScoreHistoryEntry, error) {
	user, err := us.findVisibleUser(ctx, username, requester)
	if err != nil {
		return nil, err
	}
	song, err := us.worldsendRepo.FindByDisplayID(ctx, us.exec, displayID)
	if err != nil {
		return nil, err
	}
	if song.Chart == nil {
		return nil, ErrChartNotFound
	}
	rows, err := us.historyRepo.FindWorldsendTimeline(ctx, *user.PlayerID, song.Chart.ID)
	if err != nil {
		return nil, err
	}
	return us.convertEntries(rows)
}

func (us *scoreHistoryUsecase) findVisibleUser(ctx context.Context, username string, requester *entity.User) (*entity.User, error) {
	user, err := us.userRepo.FindByUsername(ctx, us.exec, username)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	if user.IsPrivate && (requester == nil || requester.ID != user.ID) {
		return nil, ErrUserPrivate
	}
	if user.PlayerID == nil {
		return nil, ErrScoreHistoryNotFound
	}
	return user, nil
}

func (us *scoreHistoryUsecase) convertEntries(rows []entity.ScoreHistoryEntry) ([]ScoreHistoryEntry, error) {
	if len(rows) == 0 {
		return nil, ErrScoreHistoryNotFound
	}
	masters := us.masterProvider.PlayerDataMasters()
	entries := make([]ScoreHistoryEntry, 0, len(rows))
	for _, row := range rows {
		entries = append(entries, ScoreHistoryEntry{
			Score:     row.Score,
			ClearLamp: playerDataLampNamePtr(masters.ClearLampNamesByID[row.ClearLampID], row.ClearLampID, "clear_lamp"),
			ComboLamp: playerDataLampNamePtr(masters.ComboLampNamesByID[row.ComboLampID], row.ComboLampID, "combo_lamp"),
			FullChain: playerDataLampNamePtr(masters.FullChainNamesByID[row.FullChainID], row.FullChainID, "full_chain"),
			UpdatedAt: row.UpdatedAt,
		})
	}
	return entries, nil
}
