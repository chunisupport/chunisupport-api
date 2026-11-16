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

// CourseScoreHistoryEntry はAPIへ返すコーススコア履歴の1件を表します。
type CourseScoreHistoryEntry struct {
	Score     int
	IsClear   bool
	ComboLamp *string
	UpdatedAt time.Time
}

// ScoreHistoryUsecase は譜面単位のスコア履歴取得を提供します。
type ScoreHistoryUsecase interface {
	GetStandard(ctx context.Context, username string, requester *entity.User, displayID, difficulty string) ([]ScoreHistoryEntry, error)
	GetWorldsend(ctx context.Context, username string, requester *entity.User, displayID string) ([]ScoreHistoryEntry, error)
	GetCourse(ctx context.Context, username string, requester *entity.User, displayID string) ([]CourseScoreHistoryEntry, error)
}

type scoreHistoryUsecase struct {
	exec           repository.Executor
	userRepo       repository.UserRepository
	songRepo       repository.SongRepository
	worldsendRepo  repository.WorldsendChartRepository
	courseRepo     repository.CourseRepository
	historyRepo    repository.ScoreHistoryRepository
	friendshipRepo repository.FriendshipRepository
	masterProvider repository.PlayerDataMasterProvider
}

// NewScoreHistoryUsecase はスコア履歴取得ユースケースを生成します。
// friendshipRepo は非公開ユーザーをフレンドが閲覧する際の認可に使います。
func NewScoreHistoryUsecase(
	exec repository.Executor,
	userRepo repository.UserRepository,
	songRepo repository.SongRepository,
	worldsendRepo repository.WorldsendChartRepository,
	courseRepo repository.CourseRepository,
	historyRepo repository.ScoreHistoryRepository,
	friendshipRepo repository.FriendshipRepository,
	masterProvider repository.PlayerDataMasterProvider,
) ScoreHistoryUsecase {
	return &scoreHistoryUsecase{
		exec: exec, userRepo: userRepo, songRepo: songRepo, worldsendRepo: worldsendRepo, courseRepo: courseRepo,
		historyRepo: historyRepo, friendshipRepo: friendshipRepo, masterProvider: masterProvider,
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
	if song == nil || song.IsWorldsend || !song.IsActive() {
		return nil, repository.ErrSongNotFound
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
	return us.convertEntries(ctx, rows)
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
	if song == nil || song.Song == nil || !song.Song.IsWorldsend || !song.Song.IsActive() {
		return nil, repository.ErrSongNotFound
	}
	if song.Chart == nil {
		return nil, ErrChartNotFound
	}
	rows, err := us.historyRepo.FindWorldsendTimeline(ctx, *user.PlayerID, song.Chart.ID)
	if err != nil {
		return nil, err
	}
	return us.convertEntries(ctx, rows)
}

// GetCourse はコースのスコア履歴を、現行ベストを先頭に過去ベストの新しい順で返します。
// 公開範囲は既存のコースレコード取得と同じで、削除済みコースは存在しないものとして扱います。
func (us *scoreHistoryUsecase) GetCourse(ctx context.Context, username string, requester *entity.User, displayID string) ([]CourseScoreHistoryEntry, error) {
	user, err := us.findVisibleUser(ctx, username, requester)
	if err != nil {
		return nil, err
	}
	course, err := us.courseRepo.FindByDisplayID(ctx, us.exec, displayID, false)
	if err != nil {
		return nil, err
	}
	rows, err := us.historyRepo.FindCourseTimeline(ctx, *user.PlayerID, course.ID)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, ErrScoreHistoryNotFound
	}
	masters := us.masterProvider.PlayerDataMasters()
	entries := make([]CourseScoreHistoryEntry, 0, len(rows))
	for _, row := range rows {
		entries = append(entries, CourseScoreHistoryEntry{
			Score:     row.Score,
			IsClear:   row.IsClear,
			ComboLamp: playerDataLampNamePtr(ctx, masters.ComboLampNamesByID[row.ComboLampID], row.ComboLampID, "combo_lamp"),
			UpdatedAt: row.UpdatedAt,
		})
	}
	return entries, nil
}

func (us *scoreHistoryUsecase) findVisibleUser(ctx context.Context, username string, requester *entity.User) (*entity.User, error) {
	user, err := getAccessibleUser(ctx, us.exec, us.userRepo, us.friendshipRepo, username, requester)
	if err != nil {
		return nil, err
	}
	if user.PlayerID == nil {
		return nil, ErrScoreHistoryNotFound
	}
	return user, nil
}

func (us *scoreHistoryUsecase) convertEntries(ctx context.Context, rows []entity.ScoreHistoryEntry) ([]ScoreHistoryEntry, error) {
	if len(rows) == 0 {
		return nil, ErrScoreHistoryNotFound
	}
	masters := us.masterProvider.PlayerDataMasters()
	entries := make([]ScoreHistoryEntry, 0, len(rows))
	for _, row := range rows {
		entries = append(entries, ScoreHistoryEntry{
			Score:     row.Score,
			ClearLamp: playerDataLampNamePtr(ctx, masters.ClearLampNamesByID[row.ClearLampID], row.ClearLampID, "clear_lamp"),
			ComboLamp: playerDataLampNamePtr(ctx, masters.ComboLampNamesByID[row.ComboLampID], row.ComboLampID, "combo_lamp"),
			FullChain: playerDataLampNamePtr(ctx, masters.FullChainNamesByID[row.FullChainID], row.FullChainID, "full_chain"),
			UpdatedAt: row.UpdatedAt,
		})
	}
	return entries, nil
}
