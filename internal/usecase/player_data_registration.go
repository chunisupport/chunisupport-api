package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/service"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/playername"
	api_internal "github.com/chunisupport/chunisupport-api/internal/usecase/playerdataresult"
)

const (
	tokyoLayout     = "2006/01/02 15:04"
	defaultSlotName = "none"
)

// playerDataUsecase は PlayerDataUsecase の実装です。
type playerDataUsecase struct {
	tm               TransactionManager
	userRepo         repository.UserRepository
	playerRepo       repository.PlayerRepository
	playerRecRepo    repository.PlayerRecordRepository
	worldsendRecRepo repository.WorldsendRecordRepository
	honorRepo        repository.HonorRepository
	playerDataRepo   repository.PlayerDataRepository
	lockedRepo       repository.PlayerLockedSongRepository
	masterCache      repository.PlayerDataMasterProvider
	scoreHistoryRepo repository.ScoreHistoryRepository
	courseRepo       repository.CourseRepository
}

// NewPlayerDataUsecaseWithScoreHistory はスコア履歴保存を有効にした実装を生成します。
// courseRepo はコース記録の解決と保存に使います。nil のままコース記録を含む登録を受けるとエラーになります。
func NewPlayerDataUsecaseWithScoreHistory(
	tm TransactionManager,
	userRepo repository.UserRepository,
	playerRepo repository.PlayerRepository,
	playerRecRepo repository.PlayerRecordRepository,
	worldsendRecRepo repository.WorldsendRecordRepository,
	honorRepo repository.HonorRepository,
	playerDataRepo repository.PlayerDataRepository,
	lockedRepo repository.PlayerLockedSongRepository,
	masterCache repository.PlayerDataMasterProvider,
	scoreHistoryRepo repository.ScoreHistoryRepository,
	courseRepo repository.CourseRepository,
) PlayerDataUsecase {
	us := newPlayerDataUsecase(
		tm, userRepo, playerRepo, playerRecRepo, worldsendRecRepo,
		honorRepo, playerDataRepo, lockedRepo, masterCache,
	)
	us.scoreHistoryRepo = scoreHistoryRepo
	us.courseRepo = courseRepo
	return us
}

// NewPlayerDataUsecase は PlayerDataUsecase の実装を生成します。
// courseRepo はコース記録の解決と保存に使います。nil のままコース記録を含む登録を受けるとエラーになります。
func NewPlayerDataUsecase(
	tm TransactionManager,
	userRepo repository.UserRepository,
	playerRepo repository.PlayerRepository,
	playerRecRepo repository.PlayerRecordRepository,
	worldsendRecRepo repository.WorldsendRecordRepository,
	honorRepo repository.HonorRepository,
	playerDataRepo repository.PlayerDataRepository,
	lockedRepo repository.PlayerLockedSongRepository,
	masterCache repository.PlayerDataMasterProvider,
	courseRepo repository.CourseRepository,
) PlayerDataUsecase {
	us := newPlayerDataUsecase(
		tm, userRepo, playerRepo, playerRecRepo, worldsendRecRepo,
		honorRepo, playerDataRepo, lockedRepo, masterCache,
	)
	us.courseRepo = courseRepo
	return us
}

func newPlayerDataUsecase(
	tm TransactionManager,
	userRepo repository.UserRepository,
	playerRepo repository.PlayerRepository,
	playerRecRepo repository.PlayerRecordRepository,
	worldsendRecRepo repository.WorldsendRecordRepository,
	honorRepo repository.HonorRepository,
	playerDataRepo repository.PlayerDataRepository,
	lockedRepo repository.PlayerLockedSongRepository,
	masterCache repository.PlayerDataMasterProvider,
) *playerDataUsecase {
	if playerRecRepo == nil {
		panic("player record repository is required")
	}
	if lockedRepo == nil {
		panic("player locked song repository is required")
	}

	return &playerDataUsecase{
		tm:               tm,
		userRepo:         userRepo,
		playerRepo:       playerRepo,
		playerRecRepo:    playerRecRepo,
		worldsendRecRepo: worldsendRecRepo,
		honorRepo:        honorRepo,
		playerDataRepo:   playerDataRepo,
		lockedRepo:       lockedRepo,
		masterCache:      masterCache,
	}
}

// Register はCHUNITHMプレイヤーデータをトランザクション内で登録・更新します。
// プレイヤー情報、称号、スコアの各種データを処理し、結果をPlayerDataResultで返します。
func (us *playerDataUsecase) Register(ctx context.Context, user *entity.User, payload *PlayerDataPayload, bodyHash string) (*api_internal.PlayerDataResult, error) {
	if user == nil {
		return nil, errors.New("invalid request: user is nil")
	}

	nameVO, err := playername.NewPlayerName(payload.Name)
	if err != nil {
		return nil, &PlayerDataValidationError{Field: "name", Message: err.Error(), Err: err}
	}

	lastPlayedAt, updatedAt, err := parsePlayerDataTimes(payload.LastPlayed, payload.UpdatedAt)
	if err != nil {
		return nil, err
	}

	if err := validatePlayerDataPayload(payload); err != nil {
		slog.WarnContext(ctx, "player data validation failed", "user_id", user.ID, "error", err.Error())
		return nil, fmt.Errorf("invalid player data: %w", err)
	}

	summaryInput := &PlayerDataSummaryInput{
		Name:                     nameVO.String(),
		Level:                    payload.Level,
		OfficialRating:           normalizeOfficialMetric(*payload.Rating),
		OfficialOverpower:        normalizeOfficialMetric(*payload.Overpower.Value),
		OfficialOverpowerPercent: normalizeOfficialMetric(*payload.Overpower.Percentage),
		LastPlayedAt:             lastPlayedAt,
	}

	result := &api_internal.PlayerDataResult{
		AppVersion:     payload.AppVersion,
		ImportedAt:     time.Now().UTC(),
		Changes:        []api_internal.PlayerDataRecordChange{},
		SkippedRecords: []api_internal.SkippedRecord{},
	}
	registeredSPHonors := make([]registeredSPHonor, 0, 1)

	err = us.tm.Transactional(ctx, func(tx repository.Executor) error {
		lockedUser, lockErr := us.userRepo.FindByIDForUpdate(ctx, tx, user.ID)
		if lockErr != nil {
			return fmt.Errorf("failed to lock user before player data registration: %w", lockErr)
		}

		masters, loadErr := us.loadMasterData(ctx, payload)
		if loadErr != nil {
			return loadErr
		}

		classID, baseID, classErr := resolveClassEmblemIDs(payload.ClassEmblem, masters)
		if classErr != nil {
			return classErr
		}
		summaryInput.ClassEmblemID = classID
		summaryInput.ClassBaseID = baseID
		summaryInput.PossessionID = resolvePossessionID(payload.Possession, masters)

		playerID, previousPlayer, ensureErr := us.ensurePlayer(ctx, tx, lockedUser, summaryInput, updatedAt)
		if ensureErr != nil {
			return ensureErr
		}
		result.PlayerID = playerID
		if identityErr := us.validatePlayerDataIdentity(ctx, tx, playerID, updatedAt, bodyHash); identityErr != nil {
			return identityErr
		}

		beforeRecords, beforeRecordsErr := us.playerRecRepo.FindByPlayerID(ctx, tx, playerID)
		if beforeRecordsErr != nil {
			return fmt.Errorf("failed to fetch player records before registration: %w", beforeRecordsErr)
		}
		beforeWorldsendRecords, beforeWorldsendRecordsErr := us.worldsendRecRepo.FindByPlayerID(ctx, tx, playerID)
		if beforeWorldsendRecordsErr != nil {
			return fmt.Errorf("failed to fetch player worldsend records before registration: %w", beforeWorldsendRecordsErr)
		}
		beforeStatistics, beforeStatisticsErr := service.CalculatePlayerRecordStatistics(beforeRecords, beforeWorldsendRecords)
		if beforeStatisticsErr != nil {
			return fmt.Errorf("failed to aggregate player records before registration: %w", beforeStatisticsErr)
		}

		skippedRecords := make([]api_internal.SkippedRecord, 0, 4)

		honorSkipped, registeredHonors, honorErr := us.applyHonors(ctx, tx, playerID, payload.Honors, masters)
		if honorErr != nil {
			return honorErr
		}
		registeredSPHonors = registeredHonors
		skippedRecords = append(skippedRecords, honorSkipped...)

		counts, scoreSkipped, changes, statistics, overpowerSummary, scoreErr := us.applyScores(ctx, tx, playerID, payload.Scores, masters, updatedAt, beforeStatistics)
		if scoreErr != nil {
			return scoreErr
		}
		skippedRecords = append(skippedRecords, scoreSkipped...)
		summaryInput.OverpowerValue = overpowerSummary.Value
		summaryInput.OverpowerPercent = overpowerSummary.Percent

		playerID, _, ensureErr = us.ensurePlayer(ctx, tx, lockedUser, summaryInput, updatedAt)
		if ensureErr != nil {
			return ensureErr
		}
		result.PlayerID = playerID

		ratingStats, ratingErr := us.calculateAndUpdateRatings(ctx, tx, playerID)
		if ratingErr != nil {
			return ratingErr
		}

		result.Counts = counts
		result.Counts.HonorsSkipped = len(honorSkipped)
		result.Changes = changes
		result.Profile = api_internal.PlayerDataProfile{
			PlayerID:          playerID,
			Name:              summaryInput.Name,
			Level:             summaryInput.Level,
			Rating:            &ratingStats.PlayerRating,
			ClassEmblemID:     summaryInput.ClassEmblemID,
			ClassEmblemBaseID: summaryInput.ClassBaseID,
			PossessionID:      summaryInput.PossessionID,
			LastPlayedAt:      summaryInput.LastPlayedAt,
			OverpowerValue:    summaryInput.OverpowerValue,
			OverpowerPercent:  summaryInput.OverpowerPercent,
		}
		result.Summary = api_internal.PlayerDataSummary{
			Name:             summaryInput.Name,
			Level:            summaryInput.Level,
			Rating:           &ratingStats.PlayerRating,
			LastPlayedAt:     summaryInput.LastPlayedAt,
			OverpowerValue:   summaryInput.OverpowerValue,
			OverpowerPercent: summaryInput.OverpowerPercent,
		}
		var previousRating *float64
		var previousOverpowerValue *float64
		if previousPlayer != nil {
			previousRating = previousPlayer.CalculatedRating
			previousOverpowerValue = previousPlayer.OverpowerValue
		}
		result.MetricDiffs = api_internal.PlayerDataMetricDiffs{
			Rating:           buildPlayerDataFloat64Diff(previousRating, &ratingStats.PlayerRating),
			OverpowerValue:   buildPlayerDataFloat64Diff(previousOverpowerValue, summaryInput.OverpowerValue),
			OverpowerPercent: buildPlayerDataOverpowerPercentDiff(previousOverpowerValue, summaryInput.OverpowerPercent, overpowerSummary.MaxOverpowerTotal),
		}
		result.Statistics = statistics
		result.SkippedRecords = skippedRecords

		latestUpdate, latestUpdateErr := buildPlayerLatestUpdate(result, updatedAt, bodyHash)
		if latestUpdateErr != nil {
			return latestUpdateErr
		}
		if latestUpdateErr = us.playerDataRepo.SaveLatestUpdate(ctx, tx, latestUpdate); latestUpdateErr != nil {
			if errors.Is(latestUpdateErr, entity.ErrConflictingPlayerDataBody) {
				return &PlayerDataConflictError{Reason: latestUpdateErr.Error(), Err: latestUpdateErr}
			}
			return latestUpdateErr
		}

		return nil
	})
	if err != nil {
		return nil, withPlayerDataRegistrationContext(user.ID, result.PlayerID, err)
	}

	logRegisteredSPHonors(ctx, registeredSPHonors)

	slog.InfoContext(ctx, "player data imported", "user_id", user.ID, "player_id", result.PlayerID, "hash", bodyHash)
	return result, nil
}

func withPlayerDataRegistrationContext(userID, playerID int, err error) error {
	if _, ok := errors.AsType[*PlayerDataValidationError](err); ok {
		return err
	}
	if _, ok := errors.AsType[*PlayerDataNotFoundError](err); ok {
		return err
	}
	if _, ok := errors.AsType[*PlayerDataConflictError](err); ok {
		return err
	}
	return fmt.Errorf("player data registration failed (user_id=%d, player_id=%d): %w", userID, playerID, err)
}

func (us *playerDataUsecase) validatePlayerDataIdentity(ctx context.Context, tx repository.Executor, playerID int, updatedAt time.Time, bodyHash string) error {
	latestUpdate, err := us.playerDataRepo.FindLatestUpdateByPlayerIDForUpdate(ctx, tx, playerID)
	if err != nil {
		if errors.Is(err, repository.ErrPlayerLatestUpdateNotFound) {
			return nil
		}
		return fmt.Errorf("failed to validate player data identity: %w", err)
	}
	if err := latestUpdate.ValidateInputIdentity(updatedAt, bodyHash); err != nil {
		return &PlayerDataConflictError{Reason: err.Error(), Err: err}
	}
	return nil
}

// parsePlayerDataTimes はゲーム由来の時刻をUTCへ正規化します。
// lastPlayed はCHUNITHM-NETが日本時間の壁時計として出力する仕様であり、API出力用timezoneとは独立しています。
func parsePlayerDataTimes(lastPlayed, updatedAtRaw string) (*time.Time, time.Time, error) {
	loc, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		loc = time.FixedZone("Asia/Tokyo", 9*60*60)
	}

	var lastPlayedAt *time.Time
	if strings.TrimSpace(lastPlayed) != "" {
		parsed, err := time.ParseInLocation(tokyoLayout, lastPlayed, loc)
		if err != nil {
			return nil, time.Time{}, &PlayerDataValidationError{
				Field:   "last_played",
				Message: fmt.Sprintf("must match %s: %v", tokyoLayout, err),
				Err:     err,
			}
		}
		utc := parsed.UTC()
		lastPlayedAt = &utc
	}

	updatedAt, err := time.Parse(time.RFC3339, updatedAtRaw)
	if err != nil {
		return nil, time.Time{}, &PlayerDataValidationError{
			Field:   "updated_at",
			Message: fmt.Sprintf("must be RFC3339: %v", err),
			Err:     err,
		}
	}

	return lastPlayedAt, updatedAt.UTC(), nil
}

// ensurePlayer はユーザーに紐づくプレイヤーの存在を確認し、存在しなければ作成します。
// プレイヤー情報（名前、レベル、レーティング等）を更新し、プレイヤーIDと更新前状態を返します。
func (us *playerDataUsecase) ensurePlayer(ctx context.Context, tx repository.Executor, user *entity.User, summary *PlayerDataSummaryInput, updatedAt time.Time) (int, *entity.Player, error) {
	existingPlayer, err := us.playerRepo.FindByUserIDForUpdate(ctx, tx, user.ID)
	if err != nil {
		return 0, nil, err
	}

	playerName, err := playername.NewPlayerName(summary.Name)
	if err != nil {
		return 0, nil, fmt.Errorf("invalid player name: %w", err)
	}

	player := existingPlayer
	var previousPlayer *entity.Player
	if player == nil {
		player = entity.NewPlayer(user.ID, playerName)
	} else {
		previous := *player
		previousPlayer = &previous
	}
	player.ChangeProfileWithPossession(playerName, summary.Level, summary.ClassEmblemID, summary.ClassBaseID, summary.PossessionID, summary.LastPlayedAt)
	player.ChangeOverpower(summary.OverpowerValue, summary.OverpowerPercent)

	if err := player.ChangeOfficialMetrics(summary.OfficialRating, summary.OfficialOverpower, summary.OfficialOverpowerPercent, updatedAt); err != nil {
		return 0, nil, &PlayerDataConflictError{Reason: err.Error(), Err: err}
	}

	if err := us.playerRepo.Save(ctx, tx, player); err != nil {
		return 0, nil, err
	}

	if user.PlayerID == nil || *user.PlayerID != player.ID {
		user.LinkPlayer(player.ID)
		if err := us.userRepo.Save(ctx, tx, user); err != nil {
			return 0, nil, err
		}
	}

	return player.ID, previousPlayer, nil
}

func (us *playerDataUsecase) Delete(ctx context.Context, user *entity.User) error {
	if user == nil {
		return errors.New("invalid request")
	}

	return us.tm.Transactional(ctx, func(tx repository.Executor) error {
		lockedUser, err := us.userRepo.FindByIDForUpdate(ctx, tx, user.ID)
		if err != nil {
			return fmt.Errorf("failed to lock user before deleting player data: %w", err)
		}

		if err := us.playerRepo.DeleteByUserID(ctx, tx, user.ID); err != nil {
			return fmt.Errorf("failed to delete player data: %w", err)
		}

		if !lockedUser.HasLinkedPlayer() {
			return nil
		}

		lockedUser.UnlinkPlayer()
		if err := us.userRepo.Save(ctx, tx, lockedUser); err != nil {
			return fmt.Errorf("failed to unlink player from user: %w", err)
		}

		return nil
	})
}
