package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/url"
	"path"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/service"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/playername"
	"github.com/chunisupport/chunisupport-api/internal/info"
	api_internal "github.com/chunisupport/chunisupport-api/internal/usecase/playerdataresult"
)

const (
	maxPlayerDataChangeDetails          = 100
	maxScoreValue                       = 1010000
	minScoreValue                       = 0
	playerDataMetricDiffScale           = 10_000
	playerDataOverpowerPercentDiffScale = 100_000
	tokyoLayout                         = "2006/01/02 15:04"
	defaultSlotName                     = "none"
)

var (
	difficultyCodeToName = map[string]string{
		"BAS":      "BASIC",
		"BASIC":    "BASIC",
		"ADV":      "ADVANCED",
		"ADVANCED": "ADVANCED",
		"EXP":      "EXPERT",
		"EXPERT":   "EXPERT",
		"MAS":      "MASTER",
		"MASTER":   "MASTER",
		"ULT":      "ULTIMA",
		"ULTIMA":   "ULTIMA",
	}

	clearLampAlias = map[string]string{
		"":            "FAILED",
		"failed":      "FAILED",
		"clear":       "CLEAR",
		"hard":        "HARD",
		"brave":       "BRAVE",
		"absolute":    "ABSOLUTE",
		"catastrophy": "CATASTROPHY",
	}
)

// validatePlayerDataPayload はプレイヤーデータの事前検証を行い、明らかに不正なデータを検知します。
// トランザクションに入る前に実行し、改ざんや異常なデータを早期に検出します。
func validatePlayerDataPayload(payload *PlayerDataPayload) error {
	if payload == nil {
		return &PlayerDataValidationError{
			Field:   "payload",
			Message: "payload cannot be nil",
		}
	}
	if payload.Rating == nil {
		return &PlayerDataValidationError{Field: "rating", Message: "rating is required"}
	}
	if *payload.Rating < 0 || *payload.Rating > info.MaxOfficialRating || math.IsNaN(*payload.Rating) || math.IsInf(*payload.Rating, 0) {
		return &PlayerDataValidationError{Field: "rating", Message: "rating is out of range"}
	}
	if !hasOfficialMetricPrecision(*payload.Rating) {
		return &PlayerDataValidationError{Field: "rating", Message: "rating must have at most 2 decimal places"}
	}
	if payload.Overpower.Value == nil {
		return &PlayerDataValidationError{Field: "overpower.value", Message: "overpower.value is required"}
	}
	if *payload.Overpower.Value < 0 || *payload.Overpower.Value > info.MaxOfficialOverpower || math.IsNaN(*payload.Overpower.Value) || math.IsInf(*payload.Overpower.Value, 0) {
		return &PlayerDataValidationError{Field: "overpower.value", Message: "overpower.value is out of range"}
	}
	if !hasOfficialMetricPrecision(*payload.Overpower.Value) {
		return &PlayerDataValidationError{Field: "overpower.value", Message: "overpower.value must have at most 2 decimal places"}
	}
	if payload.Overpower.Percentage == nil {
		return &PlayerDataValidationError{Field: "overpower.percentage", Message: "overpower.percentage is required"}
	}
	if *payload.Overpower.Percentage < 0 || *payload.Overpower.Percentage > info.MaxOfficialOverpowerPercent || math.IsNaN(*payload.Overpower.Percentage) || math.IsInf(*payload.Overpower.Percentage, 0) {
		return &PlayerDataValidationError{Field: "overpower.percentage", Message: "overpower.percentage is out of range"}
	}
	if !hasOfficialMetricPrecision(*payload.Overpower.Percentage) {
		return &PlayerDataValidationError{Field: "overpower.percentage", Message: "overpower.percentage must have at most 2 decimal places"}
	}

	// スコアデータの整合性検証
	errorCount := 0
	maxErrorsToReport := 10
	errorMessages := make([]string, 0, maxErrorsToReport)

	// 通常譜面のスコア検証
	for i, entry := range payload.Scores.Standard {
		if errorCount >= maxErrorsToReport {
			break
		}
		if err := validateScoreEntry(&entry, "standard", i); err != nil {
			errorCount++
			errorMessages = append(errorMessages, err.Error())
		}
	}

	// WORLD'S END譜面のスコア検証
	for i, entry := range payload.Scores.Worldsend {
		if errorCount >= maxErrorsToReport {
			break
		}
		if err := validateScoreEntry(&entry, "worldsend", i); err != nil {
			errorCount++
			errorMessages = append(errorMessages, err.Error())
		}
	}
	for i, entry := range payload.Scores.Course {
		if errorCount >= maxErrorsToReport {
			break
		}
		if err := validateCourseScoreEntry(entry, i); err != nil {
			errorCount++
			errorMessages = append(errorMessages, err.Error())
		}
	}

	if errorCount > 0 {
		msg := fmt.Sprintf("detected %d invalid score entries: %s", errorCount, strings.Join(errorMessages, "; "))
		if errorCount >= maxErrorsToReport {
			msg += " (and more...)"
		}
		return &PlayerDataValidationError{
			Field:   "scores",
			Message: msg,
		}
	}

	return nil
}

func hasOfficialMetricPrecision(value float64) bool {
	scaled := value * info.OfficialMetricDecimalScale
	return math.Abs(scaled-math.Round(scaled)) <= info.OfficialMetricDecimalTolerance
}

func normalizeOfficialMetric(value float64) float64 {
	return math.Round(value*info.OfficialMetricDecimalScale) / info.OfficialMetricDecimalScale
}

func validateCourseScoreEntry(entry PlayerDataCourseEntry, index int) error {
	if strings.TrimSpace(entry.Idx) == "" {
		return fmt.Errorf("course[%d]: idx is required", index)
	}
	if entry.Score < 0 || entry.Score > 3030000 {
		return fmt.Errorf("course[%d]: score must be between 0 and 3030000 (idx=%s)", index, entry.Idx)
	}
	if entry.ComboLv < 1 || entry.ComboLv > 3 {
		return fmt.Errorf("course[%d]: unknown combo level: %d (idx=%s)", index, entry.ComboLv, entry.Idx)
	}
	if entry.Score == 3030000 && entry.ComboLv != 3 {
		return fmt.Errorf("course[%d]: score=3030000 without AJ (cmb_lv=3, idx=%s)", index, entry.Idx)
	}
	if entry.ComboLv == 3 && entry.Score < 3000000 {
		return fmt.Errorf("course[%d]: AJ requires score>=3000000 (idx=%s)", index, entry.Idx)
	}
	if entry.IsClear && entry.Score == 0 {
		return fmt.Errorf("course[%d]: cleared course cannot have score=0 (idx=%s)", index, entry.Idx)
	}
	return nil
}

// validateScoreEntry は個別のスコアエントリーを検証します。
// AJ（All Justice: cmb_lv=3）である場合、必ず1,000,000点以上でなければならないという整合性をチェックします。
func validateScoreEntry(entry *PlayerDataScoreEntry, recordType string, index int) error {
	// AJかつ100万点未満は矛盾している
	if entry.ComboLv != nil && *entry.ComboLv == 3 {
		if entry.Score < 1000000 {
			return fmt.Errorf("%s[%d]: inconsistent data - AJ (cmb_lv=3) with score=%d (must be >= 1,000,000, idx=%s)",
				recordType, index, entry.Score, entry.Idx)
		}
	}

	// 1010000点かつAJでないのは矛盾している
	if entry.Score == 1010000 && (entry.ComboLv == nil || *entry.ComboLv != 3) {
		return fmt.Errorf("%s[%d]: inconsistent data - score=1,010,000 without AJ (cmb_lv=3), idx=%s",
			recordType, index, entry.Idx)
	}

	// FULL CHAINは複数人でAJまたはFCを達成したときのランプなので、個人のAJ/FCなしでは成立しない
	if entry.FullChain != nil && (*entry.FullChain == 2 || *entry.FullChain == 3) &&
		(entry.ComboLv == nil || (*entry.ComboLv != 2 && *entry.ComboLv != 3)) {
		return fmt.Errorf("%s[%d]: inconsistent data - FULL CHAIN (fch_lv=%d) without AJ/FC (cmb_lv=2 or 3), idx=%s",
			recordType, index, *entry.FullChain, entry.Idx)
	}

	return nil
}

// playerDataMaster はプレイヤーデータ登録時に使用するマスターデータのキャッシュを保持します。
type playerDataMaster struct {
	*masterdata.PlayerDataMasters
	songs             map[string]entity.PlayerDataSong
	chartsByKey       map[string]entity.PlayerDataChart
	chartsByID        map[int]entity.PlayerDataChart
	worldsendBySongID map[int]entity.PlayerDataWorldsendChart
	courses           map[string]*entity.Course
}

type calculatedOverpowerSummary struct {
	Value             *float64
	Percent           *float64
	MaxOverpowerTotal float64
}

type registeredSPHonor struct {
	ID            int
	ImageFilename string
	ImageURL      string
}

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
	courseRepos ...repository.CourseRepository,
) PlayerDataUsecase {
	us := newPlayerDataUsecase(
		tm, userRepo, playerRepo, playerRecRepo, worldsendRecRepo,
		honorRepo, playerDataRepo, lockedRepo, masterCache,
	)
	us.scoreHistoryRepo = scoreHistoryRepo
	if len(courseRepos) > 0 {
		us.courseRepo = courseRepos[0]
	}
	return us
}

// NewPlayerDataUsecase は PlayerDataUsecase の実装を生成します。
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
	courseRepos ...repository.CourseRepository,
) PlayerDataUsecase {
	us := newPlayerDataUsecase(
		tm, userRepo, playerRepo, playerRecRepo, worldsendRecRepo,
		honorRepo, playerDataRepo, lockedRepo, masterCache,
	)
	if len(courseRepos) > 0 {
		us.courseRepo = courseRepos[0]
	}
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
		return nil, &PlayerDataValidationError{Field: "name", Message: err.Error()}
	}

	lastPlayedAt, updatedAt, err := parsePlayerDataTimes(payload.LastPlayed, payload.UpdatedAt)
	if err != nil {
		return nil, err
	}

	// トランザクション開始前にペイロードの事前検証を実行
	// 明らかに不正なデータがある場合はここで拒否する
	if err := validatePlayerDataPayload(payload); err != nil {
		slog.Warn("player data validation failed", "user_id", user.ID, "error", err.Error())
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

		// レーティングを再計算して更新
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
			return latestUpdateErr
		}

		return nil
	})
	if err != nil {
		return nil, withPlayerDataRegistrationContext(user.ID, result.PlayerID, err)
	}

	logRegisteredSPHonors(registeredSPHonors)

	slog.Info("player data imported", "user_id", user.ID, "player_id", result.PlayerID, "hash", bodyHash)
	return result, nil
}

func withPlayerDataRegistrationContext(userID, playerID int, err error) error {
	var validationErr *PlayerDataValidationError
	var notFoundErr *PlayerDataNotFoundError
	var conflictErr *PlayerDataConflictError
	if errors.As(err, &validationErr) || errors.As(err, &notFoundErr) || errors.As(err, &conflictErr) {
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
		return &PlayerDataConflictError{Reason: err.Error()}
	}
	return nil
}

// logRegisteredSPHonors はトランザクションのコミット後に、手動対応が必要な新規SP称号を通知用ログへ出力します。
func logRegisteredSPHonors(honors []registeredSPHonor) {
	for _, honor := range honors {
		slog.Warn(
			"unknown SP honor registered",
			"event", info.UnknownSPHonorRegisteredEvent,
			"honor_id", honor.ID,
			"image_filename", honor.ImageFilename,
			"image_url", honor.ImageURL,
		)
	}
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
		}
	}

	return lastPlayedAt, updatedAt.UTC(), nil
}

// loadMasterData はプレイヤーデータ登録に必要なマスターデータをキャッシュおよびDBから読み込みます。
func (us *playerDataUsecase) loadMasterData(ctx context.Context, payload *PlayerDataPayload) (*playerDataMaster, error) {
	if us.masterCache == nil {
		return nil, errors.New("master cache is not initialized")
	}

	baseMasters := us.masterCache.PlayerDataMasters()
	if baseMasters == nil {
		return nil, errors.New("master cache is not initialized")
	}

	masters := &playerDataMaster{
		PlayerDataMasters: baseMasters,
		songs:             make(map[string]entity.PlayerDataSong),
		chartsByKey:       make(map[string]entity.PlayerDataChart),
		chartsByID:        make(map[int]entity.PlayerDataChart),
		worldsendBySongID: make(map[int]entity.PlayerDataWorldsendChart),
		courses:           make(map[string]*entity.Course),
	}

	idxSet := make(map[string]struct{})
	for _, entry := range payload.Scores.Standard {
		idx := strings.TrimSpace(entry.Idx)
		if idx != "" {
			idxSet[idx] = struct{}{}
		}
	}
	for _, entry := range payload.Scores.Worldsend {
		idx := strings.TrimSpace(entry.Idx)
		if idx != "" {
			idxSet[idx] = struct{}{}
		}
	}
	courseIdxSet := make(map[string]struct{}, len(payload.Scores.Course))
	for _, entry := range payload.Scores.Course {
		if idx := strings.TrimSpace(entry.Idx); idx != "" {
			courseIdxSet[idx] = struct{}{}
		}
	}
	if len(courseIdxSet) > 0 {
		if us.courseRepo == nil {
			return nil, errors.New("course repository is not initialized")
		}
		courseIdxList := make([]string, 0, len(courseIdxSet))
		for idx := range courseIdxSet {
			courseIdxList = append(courseIdxList, idx)
		}
		slices.Sort(courseIdxList)
		courses, err := us.courseRepo.FindByOfficialIdxList(ctx, nil, courseIdxList)
		if err != nil {
			return nil, err
		}
		masters.courses = courses
	}
	if len(idxSet) == 0 {
		return masters, nil
	}

	idxList := make([]string, 0, len(idxSet))
	for idx := range idxSet {
		idxList = append(idxList, idx)
	}
	slices.Sort(idxList)

	loaded, err := us.playerDataRepo.LoadMasterData(ctx, idxList)
	if err != nil {
		return nil, err
	}

	masters.songs = loaded.Songs
	masters.chartsByKey = loaded.ChartsByKey
	masters.chartsByID = loaded.ChartsByID
	masters.worldsendBySongID = loaded.WorldsendBySongID

	return masters, nil
}

func resolveClassEmblemIDs(payload PlayerDataClassPayload, masters *playerDataMaster) (*int, *int, error) {
	var classID *int
	var baseID *int

	medalKey := normalizeClassEmblemKey(payload.MedalClass)
	if medalKey != "" {
		if item, ok := masters.ClassEmblems[medalKey]; ok {
			v := item.ID
			classID = &v
		}
		// 見つからなくてもエラーにしない（classIDはnilのまま）
	}

	baseKey := normalizeClassEmblemKey(payload.BaseClass)
	if baseKey != "" {
		if item, ok := masters.ClassEmblemBases[baseKey]; ok {
			v := item.ID
			baseID = &v
		}
		// 見つからなくてもエラーにしない（baseIDはnilのまま）
	}

	return classID, baseID, nil
}

func normalizeClassEmblemKey(raw string) string {
	key := strings.TrimSpace(raw)
	if key == "" {
		return ""
	}

	key = strings.ToLower(key)
	if key == "inf" {
		return key
	}

	key = strings.TrimLeft(key, "0")
	if key == "" {
		return "0"
	}

	if key == "6" {
		return "inf"
	}

	return key
}

// ensurePlayer はユーザーに紐づくプレイヤーの存在を確認し、存在しなければ作成します。
// プレイヤー情報（名前、レベル、レーティング等）を更新し、プレイヤーIDと更新前状態を返します。
func (us *playerDataUsecase) ensurePlayer(ctx context.Context, tx repository.Executor, user *entity.User, summary *PlayerDataSummaryInput, updatedAt time.Time) (int, *entity.Player, error) {
	// ユーザーに紐づくプレイヤーを検索
	existingPlayer, err := us.playerRepo.FindByUserIDForUpdate(ctx, tx, user.ID)
	if err != nil {
		return 0, nil, err
	}

	// PlayerNameのバリデーション
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
	player.ChangeProfile(playerName, summary.Level, summary.ClassEmblemID, summary.ClassBaseID, summary.LastPlayedAt)
	player.ChangeOverpower(summary.OverpowerValue, summary.OverpowerPercent)

	if err := player.ChangeOfficialMetrics(summary.OfficialRating, summary.OfficialOverpower, summary.OfficialOverpowerPercent, updatedAt); err != nil {
		return 0, nil, &PlayerDataConflictError{Reason: err.Error()}
	}

	// 保存（IDがなければINSERT、それ以外はUPDATE）
	if err := us.playerRepo.Save(ctx, tx, player); err != nil {
		return 0, nil, err
	}

	// ユーザーとプレイヤーのリンク
	if user.PlayerID == nil || *user.PlayerID != player.ID {
		user.LinkPlayer(player.ID)
		if err := us.userRepo.Save(ctx, tx, user); err != nil {
			return 0, nil, err
		}
	}

	return player.ID, previousPlayer, nil
}

// applyHonors はプレイヤーの称号情報を更新します。
// 既存の称号を削除し、新しい称号をバルクインサートします。
// 称号は最大3つであるため、EnsureHonorのループ内呼び出しによるN+1問題を許容します。
func (us *playerDataUsecase) applyHonors(ctx context.Context, tx repository.Executor, playerID int, honors map[string]PlayerDataHonorPayload, masters *playerDataMaster) ([]api_internal.SkippedRecord, []registeredSPHonor, error) {
	skipped := make([]api_internal.SkippedRecord, 0, 4)
	registered := make([]registeredSPHonor, 0, 1)
	if honors == nil {
		return skipped, registered, nil
	}
	preservedSlots := randomFavoriteHonorSlots(honors)
	if err := us.honorRepo.DeletePlayerHonorsExceptSlots(ctx, tx, playerID, preservedSlots); err != nil {
		return skipped, registered, err
	}

	// バリデーション済みの称号情報を収集
	assignments := make([]repository.HonorAssignment, 0, len(honors))

	for slotKey, honor := range honors {
		slotKey = strings.TrimSpace(slotKey)
		if slotKey == "" {
			continue
		}
		slot, convErr := strconv.Atoi(slotKey)
		if convErr != nil {
			skipped = append(skipped, api_internal.SkippedRecord{
				RecordType: "honor",
				Reason:     fmt.Sprintf("invalid slot %s", slotKey),
				Details:    convErr.Error(),
			})
			continue
		}
		if slot < 1 || slot > 3 {
			skipped = append(skipped, api_internal.SkippedRecord{
				RecordType: "honor",
				Reason:     fmt.Sprintf("slot out of range: %d", slot),
				Details:    fmt.Sprintf("slot=%d, title=%s", slot, honor.Title),
			})
			continue
		}
		if honor.Title == info.RandomFavoriteHonorTitle {
			continue
		}

		honorTypeKey := strings.ToLower(strings.TrimSpace(honor.Class))
		typeItem, ok := masters.HonorTypes[honorTypeKey]
		if !ok {
			skipped = append(skipped, api_internal.SkippedRecord{
				RecordType: "honor",
				Reason:     fmt.Sprintf("honor_type not found: %s", honorTypeKey),
				Details:    fmt.Sprintf("slot=%d, title=%s", slot, honor.Title),
			})
			continue
		}

		if honorTypeKey == "sp" && (honor.Img == nil || strings.TrimSpace(*honor.Img) == "") {
			skipped = append(skipped, api_internal.SkippedRecord{
				RecordType: "honor",
				Reason:     "sp honor image_url is required",
				Details:    fmt.Sprintf("slot=%d, title=%s", slot, honor.Title),
			})
			continue
		}

		honorTitle := honor.Title
		imageURL := honor.Img
		if honorTypeKey == "sp" {
			honorTitle = honorImageFilename(*imageURL)
		} else {
			// 通常称号の画像は既知の対応表で解決できるため、未判明のSP称号だけを保存する。
			imageURL = nil
		}

		ensureResult, err := us.honorRepo.EnsureHonor(ctx, tx, honorTitle, typeItem.ID, imageURL)
		if err != nil {
			skipped = append(skipped, api_internal.SkippedRecord{
				RecordType: "honor",
				Reason:     "failed to create honor",
				Details:    fmt.Sprintf("slot=%d, title=%s, error=%s", slot, honor.Title, err.Error()),
			})
			continue
		}
		if honorTypeKey == "sp" && ensureResult.ImageURLRegistered {
			registered = append(registered, registeredSPHonor{
				ID:            ensureResult.ID,
				ImageFilename: honorTitle,
				ImageURL:      *imageURL,
			})
		}

		assignments = append(assignments, repository.HonorAssignment{
			PlayerID: playerID,
			HonorID:  ensureResult.ID,
			Slot:     slot,
		})
	}

	// player_honors への一括挿入（Repository経由で実行）
	if len(assignments) > 0 {
		if err := us.honorRepo.BulkAssignHonors(ctx, tx, assignments); err != nil {
			// バルクINSERTが失敗した場合、すべての称号をスキップ扱いにする
			for _, a := range assignments {
				skipped = append(skipped, api_internal.SkippedRecord{
					RecordType: "honor",
					Reason:     "failed to insert player_honor (bulk)",
					Details:    fmt.Sprintf("slot=%d, honor_id=%d, error=%s", a.Slot, a.HonorID, err.Error()),
				})
			}
		}
	}

	return skipped, registered, nil
}

// honorImageFilename はSP称号を一意に識別するため、画像URLからファイル名を取り出します。
func honorImageFilename(imageURL string) string {
	parsedURL, err := url.Parse(strings.TrimSpace(imageURL))
	if err != nil {
		return strings.TrimSpace(imageURL)
	}
	return path.Base(parsedURL.EscapedPath())
}

// randomFavoriteHonorSlots はCHUNITHM-NET側で毎回ランダム選択される称号のスロットを返します。
// この表示用テキストを称号として保存せず、現在の割り当てを維持するために使用します。
func randomFavoriteHonorSlots(honors map[string]PlayerDataHonorPayload) []int {
	slots := make([]int, 0, 3)
	for slotKey, honor := range honors {
		if honor.Title != info.RandomFavoriteHonorTitle {
			continue
		}
		slot, err := strconv.Atoi(strings.TrimSpace(slotKey))
		if err == nil && slot >= 1 && slot <= 3 {
			slots = append(slots, slot)
		}
	}
	return slots
}

// applyScores はプレイヤーのスコア情報を更新します。
// 通常譜面とWORLD'S END譜面のスコアをUPSERTします。
func (us *playerDataUsecase) applyScores(ctx context.Context, tx repository.Executor, playerID int, scores PlayerDataScorePayload, masters *playerDataMaster, updatedAt time.Time, beforeStatistics service.PlayerRecordStatisticsSnapshot) (api_internal.PlayerDataCounts, []api_internal.SkippedRecord, []api_internal.PlayerDataRecordChange, api_internal.PlayerDataStatistics, calculatedOverpowerSummary, error) {
	counts, skipped, fullRecordsToUpsert := applyFullScores(playerID, scores.Standard, masters, updatedAt)
	worldsendCounts, worldsendSkipped, worldsendRecordsToUpsert := applyWorldsendScores(playerID, scores.Worldsend, masters, updatedAt)
	courseCounts, courseSkipped, courseRecordsToUpsert := applyCourseScores(playerID, scores.Course, masters, updatedAt)
	counts.WorldsendRecordsUpserted = worldsendCounts.WorldsendRecordsUpserted
	counts.WorldsendRecordsSkipped = worldsendCounts.WorldsendRecordsSkipped
	skipped = append(skipped, worldsendSkipped...)
	counts.CourseRecordsUpserted = courseCounts.CourseRecordsUpserted
	counts.CourseRecordsSkipped = courseCounts.CourseRecordsSkipped
	skipped = append(skipped, courseSkipped...)

	fullRecordsToUpsert = normalizeFullRecordsForUpsert(fullRecordsToUpsert)
	worldsendRecordsToUpsert = normalizeWorldsendRecordsForUpsert(worldsendRecordsToUpsert)
	courseRecordsToUpsert = normalizeCourseRecordsForUpsert(courseRecordsToUpsert)

	fullBefore, err := us.playerDataRepo.FindPlayerRecordStatesByChartIDs(ctx, tx, playerID, collectFullChartIDs(fullRecordsToUpsert))
	if err != nil {
		return counts, skipped, nil, api_internal.PlayerDataStatistics{}, calculatedOverpowerSummary{}, err
	}
	worldsendBefore, err := us.playerDataRepo.FindWorldsendRecordStatesByChartIDs(ctx, tx, playerID, collectWorldsendChartIDs(worldsendRecordsToUpsert))
	if err != nil {
		return counts, skipped, nil, api_internal.PlayerDataStatistics{}, calculatedOverpowerSummary{}, err
	}
	courseBefore := make(map[int]repository.CourseRecordState)
	if len(courseRecordsToUpsert) > 0 {
		if us.courseRepo == nil {
			return counts, skipped, nil, api_internal.PlayerDataStatistics{}, calculatedOverpowerSummary{}, errors.New("course repository is not initialized")
		}
		courseBefore, err = us.courseRepo.FindRecordStatesByCourseIDs(ctx, tx, playerID, collectCourseIDs(courseRecordsToUpsert))
		if err != nil {
			return counts, skipped, nil, api_internal.PlayerDataStatistics{}, calculatedOverpowerSummary{}, err
		}
	}

	// 差分は保存前状態とupsert予定値から算出するため、理論上は同一プレイヤーの同時リクエストで正しく出力されない場合がある。
	// ただし通常利用では同時登録が起きない前提のため許容し、発生した場合はユーザの責任として扱う。
	fullRecordChanges := computeFullRecordChanges(ctx, fullBefore, fullRecordsToUpsert, masters)
	worldsendRecordChanges := computeWorldsendRecordChanges(ctx, worldsendBefore, worldsendRecordsToUpsert, masters)
	lampLookup := newLampNameLookup(masters)
	changes := make([]api_internal.PlayerDataRecordChange, 0, len(fullRecordChanges)+len(worldsendRecordChanges))
	changes = append(changes, playerRecordChangesDTO(fullRecordChanges, lampLookup)...)
	changes = append(changes, worldsendRecordChangesDTO(worldsendRecordChanges, lampLookup)...)
	courseChanges := computeCourseRecordChanges(courseBefore, courseRecordsToUpsert, masters, lampLookup)
	changes = append(changes, courseChanges...)
	changes = sortAndLimitRecordChanges(changes)
	counts.FullRecordsActuallyChanged = len(fullRecordChanges)
	counts.WorldsendRecordsActuallyChanged = len(worldsendRecordChanges)
	counts.CourseRecordsActuallyChanged = len(courseChanges)

	standardHistories, standardHistoryChartIDs := buildStandardHistories(playerID, fullBefore, fullRecordsToUpsert, masters)
	worldsendHistories, worldsendHistoryChartIDs := buildWorldsendHistories(playerID, worldsendBefore, worldsendRecordsToUpsert)
	if us.scoreHistoryRepo != nil {
		if err := us.scoreHistoryRepo.BulkInsertStandard(ctx, tx, standardHistories); err != nil {
			return counts, skipped, changes, api_internal.PlayerDataStatistics{}, calculatedOverpowerSummary{}, err
		}
		if err := us.scoreHistoryRepo.BulkInsertWorldsend(ctx, tx, worldsendHistories); err != nil {
			return counts, skipped, changes, api_internal.PlayerDataStatistics{}, calculatedOverpowerSummary{}, err
		}
	}

	if err := us.playerDataRepo.SavePlayerData(ctx, tx, repository.PlayerDataSaveInput{
		FullRecords:      fullRecordsToUpsert,
		WorldsendRecords: worldsendRecordsToUpsert,
	}); err != nil {
		return counts, skipped, changes, api_internal.PlayerDataStatistics{}, calculatedOverpowerSummary{}, err
	}
	if len(courseRecordsToUpsert) > 0 {
		if err := us.courseRepo.UpsertRecords(ctx, tx, courseRecordsToUpsert); err != nil {
			return counts, skipped, changes, api_internal.PlayerDataStatistics{}, calculatedOverpowerSummary{}, err
		}
	}
	if us.scoreHistoryRepo != nil {
		if err := us.scoreHistoryRepo.PruneStandardOverLimit(ctx, tx, playerID, standardHistoryChartIDs); err != nil {
			return counts, skipped, changes, api_internal.PlayerDataStatistics{}, calculatedOverpowerSummary{}, err
		}
		if err := us.scoreHistoryRepo.PruneWorldsendOverLimit(ctx, tx, playerID, worldsendHistoryChartIDs); err != nil {
			return counts, skipped, changes, api_internal.PlayerDataStatistics{}, calculatedOverpowerSummary{}, err
		}
	}

	overpowerTargetStats, err := us.playerDataRepo.GetOverpowerTargetStatsWithExecutor(ctx, tx, repository.OverpowerTargetFilter{
		ExcludeWorldsend: true,
		ExcludeDeleted:   true,
		PlayerID:         &playerID,
	})
	if err != nil {
		return counts, skipped, changes, api_internal.PlayerDataStatistics{}, calculatedOverpowerSummary{}, err
	}

	records, recErr := us.playerRecRepo.FindByPlayerID(ctx, tx, playerID)
	if recErr != nil {
		return counts, skipped, changes, api_internal.PlayerDataStatistics{}, calculatedOverpowerSummary{}, fmt.Errorf("failed to fetch player records for overpower calculation: %w", recErr)
	}
	worldsendRecords, worldsendRecErr := us.worldsendRecRepo.FindByPlayerID(ctx, tx, playerID)
	if worldsendRecErr != nil {
		return counts, skipped, changes, api_internal.PlayerDataStatistics{}, calculatedOverpowerSummary{}, fmt.Errorf("failed to fetch player worldsend records for statistics: %w", worldsendRecErr)
	}
	afterStatistics, statisticsErr := service.CalculatePlayerRecordStatistics(records, worldsendRecords)
	if statisticsErr != nil {
		return counts, skipped, changes, api_internal.PlayerDataStatistics{}, calculatedOverpowerSummary{}, fmt.Errorf("failed to aggregate player records after registration: %w", statisticsErr)
	}
	lockedSongs, lockedErr := us.listLockedSongsForOverpower(ctx, tx, playerID)
	if lockedErr != nil {
		return counts, skipped, changes, api_internal.PlayerDataStatistics{}, calculatedOverpowerSummary{}, fmt.Errorf("failed to fetch locked songs for overpower calculation: %w", lockedErr)
	}
	overpowerSummary, err := calculateOverpowerSummaryFromPlayerRecords(records, lockedSongs, overpowerTargetStats.MaxOverpowerTotal)
	if err != nil {
		return counts, skipped, changes, api_internal.PlayerDataStatistics{}, calculatedOverpowerSummary{}, fmt.Errorf("failed to aggregate overpower from player records: %w", err)
	}

	return counts, skipped, changes, buildPlayerDataStatisticsDiff(beforeStatistics, afterStatistics), overpowerSummary, nil
}

func applyCourseScores(playerID int, entries []PlayerDataCourseEntry, masters *playerDataMaster, updatedAt time.Time) (api_internal.PlayerDataCounts, []api_internal.SkippedRecord, []repository.CourseRecordForUpsert) {
	counts := api_internal.PlayerDataCounts{}
	skipped := make([]api_internal.SkippedRecord, 0)
	records := make([]repository.CourseRecordForUpsert, 0, len(entries))
	for _, entry := range entries {
		counts.CourseRecordsUpserted++
		idx := strings.TrimSpace(entry.Idx)
		course, ok := masters.courses[idx]
		if !ok {
			counts.CourseRecordsSkipped++
			skipped = append(skipped, api_internal.SkippedRecord{RecordType: "course", Reason: "failed to resolve course", Details: "idx=" + idx})
			continue
		}
		combo := entry.ComboLv
		comboID, err := resolveComboLampID(&combo, masters)
		if err != nil {
			counts.CourseRecordsSkipped++
			skipped = append(skipped, api_internal.SkippedRecord{RecordType: "course", Reason: "failed to resolve combo_lamp", Details: fmt.Sprintf("idx=%s, combo_lv=%d, error=%s", idx, combo, err)})
			continue
		}
		records = append(records, repository.CourseRecordForUpsert{PlayerID: playerID, CourseID: course.ID, State: repository.CourseRecordState{Score: entry.Score, IsClear: entry.IsClear, ComboLampID: comboID, UpdatedAt: updatedAt}})
	}
	return counts, skipped, records
}

func normalizeCourseRecordsForUpsert(records []repository.CourseRecordForUpsert) []repository.CourseRecordForUpsert {
	last := make(map[int]repository.CourseRecordForUpsert, len(records))
	order := make([]int, 0, len(records))
	for _, record := range records {
		if _, ok := last[record.CourseID]; !ok {
			order = append(order, record.CourseID)
		}
		last[record.CourseID] = record
	}
	result := make([]repository.CourseRecordForUpsert, 0, len(last))
	for _, id := range order {
		result = append(result, last[id])
	}
	return result
}

func collectCourseIDs(records []repository.CourseRecordForUpsert) []int {
	result := make([]int, 0, len(records))
	for _, r := range records {
		result = append(result, r.CourseID)
	}
	return result
}

func computeCourseRecordChanges(before map[int]repository.CourseRecordState, after []repository.CourseRecordForUpsert, masters *playerDataMaster, lookup lampNameLookup) []api_internal.PlayerDataRecordChange {
	byID := make(map[int]*entity.Course, len(masters.courses))
	for _, course := range masters.courses {
		byID[course.ID] = course
	}
	result := make([]api_internal.PlayerDataRecordChange, 0, len(after))
	for _, record := range after {
		old, exists := before[record.CourseID]
		if exists && old.Score == record.State.Score && old.IsClear == record.State.IsClear && old.ComboLampID == record.State.ComboLampID {
			continue
		}
		course := byID[record.CourseID]
		idx := fmt.Sprint(record.CourseID)
		class := ""
		if course != nil {
			idx = course.OfficialIdx
			if course.CourseClass != nil {
				class = course.CourseClass.Name
			}
		}
		clearAfter := record.State.IsClear
		change := api_internal.PlayerDataRecordChange{RecordType: "course", ChangeType: "new", Idx: idx, CourseClass: class, After: api_internal.PlayerDataRecordState{Score: record.State.Score, ComboLamp: lookup.comboLampName(record.State.ComboLampID), IsClear: &clearAfter}}
		if exists {
			clearBefore := old.IsClear
			change.ChangeType = "updated"
			change.Before = &api_internal.PlayerDataRecordState{Score: old.Score, ComboLamp: lookup.comboLampName(old.ComboLampID), IsClear: &clearBefore}
		}
		result = append(result, change)
	}
	return result
}

func buildStandardHistories(
	playerID int,
	before map[int]repository.PlayerRecordState,
	after []repository.PlayerRecordForUpsert,
	masters *playerDataMaster,
) ([]repository.PlayerRecordHistory, []int) {
	rows := make([]repository.PlayerRecordHistory, 0, len(after))
	chartIDs := make([]int, 0, len(after))
	seenChartIDs := make(map[int]struct{}, len(after))
	for _, record := range after {
		beforeState, exists := before[record.ChartID]
		if !exists || !playerRecordMeaningfullyChanged(beforeState, record.State) {
			continue
		}
		chart, exists := masters.chartsByID[record.ChartID]
		if !exists || !entity.SupportsScoreHistory(masters.DifficultyNamesByID[chart.DifficultyID]) {
			continue
		}
		rows = append(rows, repository.PlayerRecordHistory{
			PlayerID: playerID,
			ChartID:  record.ChartID,
			State:    beforeState,
		})
		if _, exists := seenChartIDs[record.ChartID]; !exists {
			seenChartIDs[record.ChartID] = struct{}{}
			chartIDs = append(chartIDs, record.ChartID)
		}
	}
	return rows, chartIDs
}

func buildWorldsendHistories(
	playerID int,
	before map[int]repository.WorldsendRecordState,
	after []repository.WorldsendRecordForUpsert,
) ([]repository.PlayerWorldsendRecordHistory, []int) {
	rows := make([]repository.PlayerWorldsendRecordHistory, 0, len(after))
	chartIDs := make([]int, 0, len(after))
	seenChartIDs := make(map[int]struct{}, len(after))
	for _, record := range after {
		beforeState, exists := before[record.ChartID]
		if !exists || !worldsendRecordMeaningfullyChanged(beforeState, record.State) {
			continue
		}
		rows = append(rows, repository.PlayerWorldsendRecordHistory{
			PlayerID:         playerID,
			WorldsendChartID: record.ChartID,
			State:            beforeState,
		})
		if _, exists := seenChartIDs[record.ChartID]; !exists {
			seenChartIDs[record.ChartID] = struct{}{}
			chartIDs = append(chartIDs, record.ChartID)
		}
	}
	return rows, chartIDs
}

func buildPlayerDataStatisticsDiff(before service.PlayerRecordStatisticsSnapshot, after service.PlayerRecordStatisticsSnapshot) api_internal.PlayerDataStatistics {
	statistics := api_internal.PlayerDataStatistics{
		Overall:      buildPlayerDataStatisticsGroupDiff(before.Overall, after.Overall),
		ByDifficulty: make(map[string]api_internal.PlayerDataStatisticsGroup, len(service.PlayerRecordStatisticsGroupNames())),
	}
	for _, difficulty := range service.PlayerRecordStatisticsGroupNames() {
		statistics.ByDifficulty[difficulty] = buildPlayerDataStatisticsGroupDiff(before.ByDifficulty[difficulty], after.ByDifficulty[difficulty])
	}
	return statistics
}

// buildPlayerDataFloat64Diff は登録前後の値が揃う場合だけ小数差分を計算します。
func buildPlayerDataFloat64Diff(before *float64, after *float64) api_internal.PlayerDataFloat64Diff {
	return buildPlayerDataFloat64DiffWithScale(before, after, playerDataMetricDiffScale)
}

// buildPlayerDataOverpowerPercentDiff は更新前OP値を今回と同じ分母で割合へ変換し、OP%のポイント差を計算します。
func buildPlayerDataOverpowerPercentDiff(beforeValue *float64, afterPercent *float64, maxOverpowerTotal float64) api_internal.PlayerDataFloat64Diff {
	var beforePercent *float64
	if beforeValue != nil {
		beforePercentValue := service.CalcOverpowerPercent(*beforeValue, maxOverpowerTotal)
		beforePercent = &beforePercentValue
	}
	return buildPlayerDataFloat64DiffWithScale(beforePercent, afterPercent, playerDataOverpowerPercentDiffScale)
}

// buildPlayerDataFloat64DiffWithScale は指定精度で登録前後の小数差分を計算します。
func buildPlayerDataFloat64DiffWithScale(before *float64, after *float64, scale float64) api_internal.PlayerDataFloat64Diff {
	diff := api_internal.PlayerDataFloat64Diff{
		Before: cloneFloat64Pointer(before),
		After:  cloneFloat64Pointer(after),
	}
	if before != nil && after != nil {
		delta := math.Round((*after-*before)*scale) / scale
		diff.Delta = &delta
	}
	return diff
}

// cloneFloat64Pointer は呼び出し元の値と共有しないfloat64ポインタを返します。
func cloneFloat64Pointer(value *float64) *float64 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func buildPlayerDataStatisticsGroupDiff(before service.PlayerRecordStatistics, after service.PlayerRecordStatistics) api_internal.PlayerDataStatisticsGroup {
	intDiff := func(beforeValue int, afterValue int) api_internal.PlayerDataIntDiff {
		return api_internal.PlayerDataIntDiff{Before: beforeValue, After: afterValue, Delta: afterValue - beforeValue}
	}
	return api_internal.PlayerDataStatisticsGroup{
		TotalHighScore: api_internal.PlayerDataInt64Diff{Before: before.TotalHighScore, After: after.TotalHighScore, Delta: after.TotalHighScore - before.TotalHighScore},
		RecordStatistics: api_internal.PlayerDataRecordStatisticsDiff{
			AJ: intDiff(before.Achievements.AJ, after.Achievements.AJ), FC: intDiff(before.Achievements.FC, after.Achievements.FC),
			CLR: intDiff(before.Achievements.CLR, after.Achievements.CLR), FCH: intDiff(before.Achievements.FCH, after.Achievements.FCH),
			MAX: intDiff(before.Achievements.MAX, after.Achievements.MAX), SSSPlus: intDiff(before.Achievements.SSSPlus, after.Achievements.SSSPlus),
			SSS: intDiff(before.Achievements.SSS, after.Achievements.SSS), SSPlus: intDiff(before.Achievements.SSPlus, after.Achievements.SSPlus),
			SS: intDiff(before.Achievements.SS, after.Achievements.SS), SPlus: intDiff(before.Achievements.SPlus, after.Achievements.SPlus),
			S: intDiff(before.Achievements.S, after.Achievements.S),
		},
	}
}

func (us *playerDataUsecase) listLockedSongsForOverpower(ctx context.Context, tx repository.Executor, playerID int) ([]*entity.PlayerLockedSong, error) {
	if us.lockedRepo == nil {
		return nil, nil
	}
	return us.lockedRepo.ListByPlayerID(ctx, tx, playerID)
}

type resolvedLampIDs struct {
	clearLampID int
	comboLampID int
	fullChainID int
}

func applyFullScores(playerID int, entries []PlayerDataScoreEntry, masters *playerDataMaster, updatedAt time.Time) (api_internal.PlayerDataCounts, []api_internal.SkippedRecord, []repository.PlayerRecordForUpsert) {
	counts := api_internal.PlayerDataCounts{}
	skipped := make([]api_internal.SkippedRecord, 0, len(entries))
	fullRecordsToUpsert := make([]repository.PlayerRecordForUpsert, 0, len(entries))

	for _, entry := range entries {
		counts.FullRecordsUpserted++

		chart, song, _, err := resolveChart(entry, masters)
		if err != nil {
			counts.FullRecordsSkipped++
			skipped = append(skipped, api_internal.SkippedRecord{
				RecordType: "standard",
				Reason:     "failed to resolve chart",
				Details:    fmt.Sprintf("idx=%s, diff=%s, error=%s", entry.Idx, entry.Diff, err.Error()),
			})
			continue
		}

		if skippedRecord, ok := validateScoreRange("standard", entry, song); ok {
			counts.FullRecordsSkipped++
			skipped = append(skipped, skippedRecord)
			continue
		}

		lampIDs, skippedRecord := resolveCommonLampIDs("standard", entry, song, masters)
		if skippedRecord != nil {
			counts.FullRecordsSkipped++
			skipped = append(skipped, *skippedRecord)
			continue
		}

		slotID, err := resolveSlotID(entry.Slot, masters)
		if err != nil {
			counts.FullRecordsSkipped++
			skipped = append(skipped, newResolveSkippedRecord("standard", "slot", "slot", entry, song, optionalStringValue(entry.Slot), err))
			continue
		}

		fullRecordsToUpsert = append(fullRecordsToUpsert, repository.PlayerRecordForUpsert{
			PlayerID: playerID,
			ChartID:  chart.ID,
			State: repository.PlayerRecordState{
				Score:       entry.Score,
				ClearLampID: lampIDs.clearLampID,
				ComboLampID: lampIDs.comboLampID,
				FullChainID: lampIDs.fullChainID,
				SlotID:      slotID,
				SlotOrder:   entry.Order,
				UpdatedAt:   updatedAt,
			},
		})
	}

	return counts, skipped, fullRecordsToUpsert
}

func applyWorldsendScores(playerID int, entries []PlayerDataScoreEntry, masters *playerDataMaster, updatedAt time.Time) (api_internal.PlayerDataCounts, []api_internal.SkippedRecord, []repository.WorldsendRecordForUpsert) {
	counts := api_internal.PlayerDataCounts{}
	skipped := make([]api_internal.SkippedRecord, 0, len(entries))
	worldsendRecordsToUpsert := make([]repository.WorldsendRecordForUpsert, 0, len(entries))

	for _, entry := range entries {
		counts.WorldsendRecordsUpserted++

		chart, song, err := resolveWorldsendChart(entry, masters)
		if err != nil {
			counts.WorldsendRecordsSkipped++
			skipped = append(skipped, api_internal.SkippedRecord{
				RecordType: "worldsend",
				Reason:     "failed to resolve worldsend chart",
				Details:    fmt.Sprintf("idx=%s, error=%s", entry.Idx, err.Error()),
			})
			continue
		}

		if skippedRecord, ok := validateScoreRange("worldsend", entry, song); ok {
			counts.WorldsendRecordsSkipped++
			skipped = append(skipped, skippedRecord)
			continue
		}

		lampIDs, skippedRecord := resolveCommonLampIDs("worldsend", entry, song, masters)
		if skippedRecord != nil {
			counts.WorldsendRecordsSkipped++
			skipped = append(skipped, *skippedRecord)
			continue
		}

		worldsendRecordsToUpsert = append(worldsendRecordsToUpsert, repository.WorldsendRecordForUpsert{
			PlayerID: playerID,
			ChartID:  chart.ID,
			State: repository.WorldsendRecordState{
				Score:       entry.Score,
				ClearLampID: lampIDs.clearLampID,
				ComboLampID: lampIDs.comboLampID,
				FullChainID: lampIDs.fullChainID,
				UpdatedAt:   updatedAt,
			},
		})
	}

	return counts, skipped, worldsendRecordsToUpsert
}

func normalizeFullRecordsForUpsert(records []repository.PlayerRecordForUpsert) []repository.PlayerRecordForUpsert {
	positions := make(map[int]int, len(records))
	normalized := make([]repository.PlayerRecordForUpsert, 0, len(records))
	for _, record := range records {
		if pos, ok := positions[record.ChartID]; ok {
			normalized[pos] = record
			continue
		}
		positions[record.ChartID] = len(normalized)
		normalized = append(normalized, record)
	}
	return normalized
}

func normalizeWorldsendRecordsForUpsert(records []repository.WorldsendRecordForUpsert) []repository.WorldsendRecordForUpsert {
	positions := make(map[int]int, len(records))
	normalized := make([]repository.WorldsendRecordForUpsert, 0, len(records))
	for _, record := range records {
		if pos, ok := positions[record.ChartID]; ok {
			normalized[pos] = record
			continue
		}
		positions[record.ChartID] = len(normalized)
		normalized = append(normalized, record)
	}
	return normalized
}

func collectFullChartIDs(records []repository.PlayerRecordForUpsert) []int {
	ids := make([]int, 0, len(records))
	for _, record := range records {
		ids = append(ids, record.ChartID)
	}
	slices.Sort(ids)
	return ids
}

func collectWorldsendChartIDs(records []repository.WorldsendRecordForUpsert) []int {
	ids := make([]int, 0, len(records))
	for _, record := range records {
		ids = append(ids, record.ChartID)
	}
	slices.Sort(ids)
	return ids
}

func computeFullRecordChanges(ctx context.Context, before map[int]repository.PlayerRecordState, after []repository.PlayerRecordForUpsert, masters *playerDataMaster) []playerDataRecordChange[repository.PlayerRecordState] {
	return computeRecordChanges(
		ctx,
		before,
		after,
		masters,
		func(record repository.PlayerRecordForUpsert) int { return record.ChartID },
		func(record repository.PlayerRecordForUpsert) repository.PlayerRecordState { return record.State },
		func(ctx context.Context, record repository.PlayerRecordForUpsert, lookup recordDisplayLookup) (string, string) {
			return fullRecordDisplayKeys(ctx, record.ChartID, masters, lookup)
		},
		playerRecordMeaningfullyChanged,
		"standard",
	)
}

func computeWorldsendRecordChanges(ctx context.Context, before map[int]repository.WorldsendRecordState, after []repository.WorldsendRecordForUpsert, masters *playerDataMaster) []playerDataRecordChange[repository.WorldsendRecordState] {
	return computeRecordChanges(
		ctx,
		before,
		after,
		masters,
		func(record repository.WorldsendRecordForUpsert) int { return record.ChartID },
		func(record repository.WorldsendRecordForUpsert) repository.WorldsendRecordState { return record.State },
		func(ctx context.Context, record repository.WorldsendRecordForUpsert, lookup recordDisplayLookup) (string, string) {
			return worldsendRecordDisplayKeys(record.ChartID, lookup)
		},
		worldsendRecordMeaningfullyChanged,
		"worldsend",
	)
}

type playerDataRecordChange[State any] struct {
	RecordType string
	ChangeType string
	Idx        string
	Diff       string
	Before     *State
	After      State
}

// computeRecordChanges は通常譜面とWORLD'S ENDで共通する差分生成の流れを集約します。
// 差分計算そのものをAPI DTOに固定しないため、レスポンス変換は呼び出し側で行います。
func computeRecordChanges[State any, Record any](
	ctx context.Context,
	before map[int]State,
	after []Record,
	masters *playerDataMaster,
	getChartID func(Record) int,
	getState func(Record) State,
	getDisplayKeys func(context.Context, Record, recordDisplayLookup) (string, string),
	meaningfullyChanged func(State, State) bool,
	recordType string,
) []playerDataRecordChange[State] {
	lookup := newRecordDisplayLookup(masters)
	changes := make([]playerDataRecordChange[State], 0, len(after))
	for _, record := range after {
		chartID := getChartID(record)
		afterState := getState(record)
		idx, diff := getDisplayKeys(ctx, record, lookup)
		beforeState, exists := before[chartID]
		if !exists {
			changes = append(changes, playerDataRecordChange[State]{RecordType: recordType, ChangeType: "new", Idx: idx, Diff: diff, After: afterState})
			continue
		}
		if !meaningfullyChanged(beforeState, afterState) {
			continue
		}
		beforeCopy := beforeState
		changes = append(changes, playerDataRecordChange[State]{RecordType: recordType, ChangeType: "updated", Idx: idx, Diff: diff, Before: &beforeCopy, After: afterState})
	}
	return changes
}

type recordDisplayLookup struct {
	songsByID          map[int]string
	difficultiesByID   map[int]string
	worldsendByChartID map[int]entity.PlayerDataWorldsendChart
}

func newRecordDisplayLookup(masters *playerDataMaster) recordDisplayLookup {
	lookup := recordDisplayLookup{
		songsByID:          make(map[int]string, len(masters.songs)),
		difficultiesByID:   make(map[int]string, len(masters.Difficulties)),
		worldsendByChartID: make(map[int]entity.PlayerDataWorldsendChart, len(masters.worldsendBySongID)),
	}
	for _, song := range masters.songs {
		lookup.songsByID[song.ID] = song.OfficialIdx
	}
	for _, difficulty := range masters.Difficulties {
		lookup.difficultiesByID[difficulty.ID] = difficulty.Name
	}
	for _, chart := range masters.worldsendBySongID {
		lookup.worldsendByChartID[chart.ID] = chart
	}
	return lookup
}

func fullRecordDisplayKeys(ctx context.Context, chartID int, masters *playerDataMaster, lookup recordDisplayLookup) (string, string) {
	chart, ok := masters.chartsByID[chartID]
	if !ok {
		return fmt.Sprintf("%d", chartID), ""
	}
	idx, ok := lookup.songsByID[chart.SongID]
	if !ok {
		idx = fmt.Sprintf("%d", chart.SongID)
	}
	diff, ok := lookup.difficultiesByID[chart.DifficultyID]
	if !ok {
		slog.WarnContext(ctx, "difficulty not found for player data change display", "difficulty_id", chart.DifficultyID, "chart_id", chartID)
		diff = fmt.Sprintf("%d", chart.DifficultyID)
	}
	return idx, diff
}

func worldsendRecordDisplayKeys(chartID int, lookup recordDisplayLookup) (string, string) {
	chart, ok := lookup.worldsendByChartID[chartID]
	if !ok {
		return fmt.Sprintf("%d", chartID), "WE"
	}
	idx, ok := lookup.songsByID[chart.SongID]
	if !ok {
		idx = fmt.Sprintf("%d", chart.SongID)
	}
	return idx, "WE"
}

func playerRecordChangesDTO(changes []playerDataRecordChange[repository.PlayerRecordState], lookup lampNameLookup) []api_internal.PlayerDataRecordChange {
	return recordChangesDTO(changes, func(state repository.PlayerRecordState) api_internal.PlayerDataRecordState {
		return playerRecordStateDTO(state, lookup)
	})
}

func worldsendRecordChangesDTO(changes []playerDataRecordChange[repository.WorldsendRecordState], lookup lampNameLookup) []api_internal.PlayerDataRecordChange {
	return recordChangesDTO(changes, func(state repository.WorldsendRecordState) api_internal.PlayerDataRecordState {
		return worldsendRecordStateDTO(state, lookup)
	})
}

func recordChangesDTO[State any](changes []playerDataRecordChange[State], stateDTO func(State) api_internal.PlayerDataRecordState) []api_internal.PlayerDataRecordChange {
	dtos := make([]api_internal.PlayerDataRecordChange, 0, len(changes))
	for _, change := range changes {
		dto := api_internal.PlayerDataRecordChange{
			RecordType: change.RecordType,
			ChangeType: change.ChangeType,
			Idx:        change.Idx,
			Diff:       change.Diff,
			After:      stateDTO(change.After),
		}
		if change.Before != nil {
			before := stateDTO(*change.Before)
			dto.Before = &before
		}
		dtos = append(dtos, dto)
	}
	return dtos
}

func sortAndLimitRecordChanges(changes []api_internal.PlayerDataRecordChange) []api_internal.PlayerDataRecordChange {
	slices.SortStableFunc(changes, comparePlayerDataRecordChange)
	if len(changes) <= maxPlayerDataChangeDetails {
		return changes
	}
	return changes[:maxPlayerDataChangeDetails]
}

func comparePlayerDataRecordChange(a, b api_internal.PlayerDataRecordChange) int {
	aIdx, aOK := parseChangeIdx(a.Idx)
	bIdx, bOK := parseChangeIdx(b.Idx)
	if aOK != bOK {
		if aOK {
			return -1
		}
		return 1
	}
	if aOK && aIdx != bIdx {
		return aIdx - bIdx
	}
	if a.Idx != b.Idx {
		return strings.Compare(a.Idx, b.Idx)
	}
	if a.RecordType != b.RecordType {
		return strings.Compare(a.RecordType, b.RecordType)
	}
	if a.Diff != b.Diff {
		return strings.Compare(a.Diff, b.Diff)
	}
	return strings.Compare(a.ChangeType, b.ChangeType)
}

func parseChangeIdx(idx string) (int, bool) {
	value, err := strconv.Atoi(idx)
	return value, err == nil
}

type lampNameLookup struct {
	clearLamps map[int]string
	comboLamps map[int]string
	fullChains map[int]string
}

func newLampNameLookup(masters *playerDataMaster) lampNameLookup {
	lookup := lampNameLookup{
		clearLamps: make(map[int]string, len(masters.ClearLamps)),
		comboLamps: make(map[int]string, len(masters.ComboLamps)),
		fullChains: make(map[int]string, len(masters.FullChains)),
	}
	for _, item := range masters.ClearLamps {
		lookup.clearLamps[item.ID] = item.Name
	}
	for _, item := range masters.ComboLamps {
		lookup.comboLamps[item.ID] = item.Name
	}
	for _, item := range masters.FullChains {
		lookup.fullChains[item.ID] = item.Name
	}
	return lookup
}

func (l lampNameLookup) clearLampName(id int) *string {
	return playerDataLampNamePtr(l.clearLamps[id], id, "clear_lamp")
}

func (l lampNameLookup) comboLampName(id int) *string {
	return playerDataLampNamePtr(l.comboLamps[id], id, "combo_lamp")
}

func (l lampNameLookup) fullChainName(id int) *string {
	return playerDataLampNamePtr(l.fullChains[id], id, "full_chain")
}

// playerDataLampNamePtr は none 相当およびマスタ未解決を null として返します。
func playerDataLampNamePtr(name string, id int, resource string) *string {
	if name == "" {
		if id != 0 {
			slog.Warn("lamp name not found for player data change display", "resource", resource, "id", id)
		}
		return nil
	}
	if strings.EqualFold(name, "none") {
		return nil
	}
	return &name
}

func playerRecordStateDTO(state repository.PlayerRecordState, lookup lampNameLookup) api_internal.PlayerDataRecordState {
	return api_internal.PlayerDataRecordState{
		Score:     state.Score,
		ClearLamp: lookup.clearLampName(state.ClearLampID),
		ComboLamp: lookup.comboLampName(state.ComboLampID),
		FullChain: lookup.fullChainName(state.FullChainID),
	}
}

func worldsendRecordStateDTO(state repository.WorldsendRecordState, lookup lampNameLookup) api_internal.PlayerDataRecordState {
	return api_internal.PlayerDataRecordState{
		Score:     state.Score,
		ClearLamp: lookup.clearLampName(state.ClearLampID),
		ComboLamp: lookup.comboLampName(state.ComboLampID),
		FullChain: lookup.fullChainName(state.FullChainID),
	}
}

// playerRecordMeaningfullyChanged はDB側の fullRecordChangedCondition と同じ比較対象だけを差分として扱います。
func playerRecordMeaningfullyChanged(before, after repository.PlayerRecordState) bool {
	return before.Score != after.Score ||
		before.ClearLampID != after.ClearLampID ||
		before.ComboLampID != after.ComboLampID ||
		before.FullChainID != after.FullChainID
}

// worldsendRecordMeaningfullyChanged はDB側の worldsendRecordChangedCondition と同じ比較対象だけを差分として扱います。
func worldsendRecordMeaningfullyChanged(before, after repository.WorldsendRecordState) bool {
	return before.Score != after.Score ||
		before.ClearLampID != after.ClearLampID ||
		before.ComboLampID != after.ComboLampID ||
		before.FullChainID != after.FullChainID
}

func validateScoreRange(recordType string, entry PlayerDataScoreEntry, song entity.PlayerDataSong) (api_internal.SkippedRecord, bool) {
	if entry.Score >= minScoreValue && entry.Score <= maxScoreValue {
		return api_internal.SkippedRecord{}, false
	}

	return api_internal.SkippedRecord{
		RecordType: recordType,
		Reason:     fmt.Sprintf("score out of range: %d", entry.Score),
		Details:    fmt.Sprintf("idx=%s (%s), score=%d", entry.Idx, song.Title, entry.Score),
	}, true
}

func resolveCommonLampIDs(recordType string, entry PlayerDataScoreEntry, song entity.PlayerDataSong, masters *playerDataMaster) (resolvedLampIDs, *api_internal.SkippedRecord) {
	clearLampID, err := resolveClearLampID(entry.ClearLamp, masters)
	if err != nil {
		skipped := newResolveSkippedRecord(recordType, "clear_lamp", "clear_lamp", entry, song, optionalStringValue(entry.ClearLamp), err)
		return resolvedLampIDs{}, &skipped
	}

	comboLampID, err := resolveComboLampID(entry.ComboLv, masters)
	if err != nil {
		skipped := newResolveSkippedRecord(recordType, "combo_lamp", "combo_lv", entry, song, optionalIntValue(entry.ComboLv), err)
		return resolvedLampIDs{}, &skipped
	}

	fullChainID, err := resolveFullChainID(entry.FullChain, masters)
	if err != nil {
		skipped := newResolveSkippedRecord(recordType, "full_chain", "full_chain", entry, song, optionalIntValue(entry.FullChain), err)
		return resolvedLampIDs{}, &skipped
	}

	return resolvedLampIDs{
		clearLampID: clearLampID,
		comboLampID: comboLampID,
		fullChainID: fullChainID,
	}, nil
}

func newResolveSkippedRecord(recordType, reasonField, detailField string, entry PlayerDataScoreEntry, song entity.PlayerDataSong, value string, err error) api_internal.SkippedRecord {
	return api_internal.SkippedRecord{
		RecordType: recordType,
		Reason:     fmt.Sprintf("failed to resolve %s", reasonField),
		Details:    fmt.Sprintf("idx=%s (%s), %s=%s, error=%s", entry.Idx, song.Title, detailField, value, err.Error()),
	}
}

func optionalStringValue(value *string) string {
	if value == nil {
		return "nil"
	}
	return *value
}

func optionalIntValue(value *int) string {
	if value == nil {
		return "nil"
	}
	return fmt.Sprintf("%d", *value)
}

func calculateOverpowerSummaryFromPlayerRecords(records []*entity.PlayerRecord, lockedSongs []*entity.PlayerLockedSong, maxOverpowerTotal float64) (calculatedOverpowerSummary, error) {
	lockedSet := make(map[string]struct{}, len(lockedSongs))
	for _, lockedSong := range lockedSongs {
		if lockedSong == nil {
			continue
		}
		lockedSet[lockedSongKey(lockedSong.SongID, lockedSong.IsUltima)] = struct{}{}
	}
	overpowerRecords, skippedRecords, err := playerRecordsToOverpowerRecordsWithSkipped(records, false, func(record *entity.PlayerRecord) (bool, string) {
		if len(lockedSet) == 0 {
			return true, ""
		}
		if record.ChartDifficulty == nil {
			return false, "chart_difficulty_nil"
		}
		_, exists := lockedSet[lockedSongKey(record.Song.ID, record.ChartDifficulty.Name == info.DifficultyNameUltima)]
		if exists {
			return false, "locked_song"
		}
		return true, ""
	})
	if err != nil {
		return calculatedOverpowerSummary{}, err
	}
	unexpectedSkippedRecords := make([]skippedOverpowerRecord, 0, len(skippedRecords))
	for _, skippedRecord := range skippedRecords {
		if skippedRecord.Reason != "locked_song" {
			unexpectedSkippedRecords = append(unexpectedSkippedRecords, skippedRecord)
		}
	}
	if len(unexpectedSkippedRecords) > 0 {
		slog.Warn("skipped player records during overpower recalculation", "total_records", len(records), "aggregated_records", len(overpowerRecords), "skipped_records", unexpectedSkippedRecords)
	}
	value, percent := service.CalcOverpowerSummary(overpowerRecords, maxOverpowerTotal)
	return calculatedOverpowerSummary{Value: &value, Percent: &percent, MaxOverpowerTotal: maxOverpowerTotal}, nil
}

func resolveChart(entry PlayerDataScoreEntry, masters *playerDataMaster) (entity.PlayerDataChart, entity.PlayerDataSong, string, error) {
	diffCode := strings.ToUpper(strings.TrimSpace(entry.Diff))
	diffName, ok := difficultyCodeToName[diffCode]
	if !ok {
		diffName = diffCode
	}
	diffItem, ok := masters.Difficulties[strings.ToUpper(diffName)]
	if !ok {
		return entity.PlayerDataChart{}, entity.PlayerDataSong{}, "", &PlayerDataNotFoundError{Resource: "difficulty", Key: diffName}
	}

	songKey := strings.TrimSpace(entry.Idx)
	song, ok := masters.songs[songKey]
	if !ok {
		return entity.PlayerDataChart{}, entity.PlayerDataSong{}, "", &PlayerDataNotFoundError{Resource: "song", Key: songKey}
	}

	key := fmt.Sprintf("%d:%d", song.ID, diffItem.ID)
	chart, ok := masters.chartsByKey[key]
	if !ok {
		return entity.PlayerDataChart{}, entity.PlayerDataSong{}, "", &PlayerDataNotFoundError{Resource: "chart", Key: fmt.Sprintf("%s-%s", songKey, diffName)}
	}

	return chart, song, diffName, nil
}

func resolveWorldsendChart(entry PlayerDataScoreEntry, masters *playerDataMaster) (entity.PlayerDataWorldsendChart, entity.PlayerDataSong, error) {
	songKey := strings.TrimSpace(entry.Idx)
	song, ok := masters.songs[songKey]
	if !ok {
		return entity.PlayerDataWorldsendChart{}, entity.PlayerDataSong{}, &PlayerDataNotFoundError{Resource: "song", Key: songKey}
	}

	ws, ok := masters.worldsendBySongID[song.ID]
	if !ok {
		return entity.PlayerDataWorldsendChart{}, entity.PlayerDataSong{}, &PlayerDataNotFoundError{Resource: "worldsend_chart", Key: songKey}
	}

	return ws, song, nil
}

func resolveClearLampID(clearLamp *string, masters *playerDataMaster) (int, error) {
	key := ""
	if clearLamp != nil {
		key = strings.ToLower(strings.TrimSpace(*clearLamp))
	}
	mapped, ok := clearLampAlias[key]
	if !ok {
		mapped = strings.ToUpper(key)
	}
	item, ok := masters.ClearLamps[strings.ToLower(mapped)]
	if !ok {
		return 0, &PlayerDataNotFoundError{Resource: "clear_lamp", Key: mapped}
	}
	return item.ID, nil
}

func resolveComboLampID(combo *int, masters *playerDataMaster) (int, error) {
	value := 1
	if combo != nil {
		value = *combo
	}
	var name string
	switch value {
	case 1:
		name = "none"
	case 2:
		name = "full combo"
	case 3:
		name = "all justice"
	default:
		return 0, &PlayerDataValidationError{Field: "cmb_lv", Message: fmt.Sprintf("unknown combo level: %d", value)}
	}
	item, ok := masters.ComboLamps[name]
	if !ok {
		return 0, &PlayerDataNotFoundError{Resource: "combo_lamp", Key: name}
	}
	return item.ID, nil
}

func resolveFullChainID(fullChain *int, masters *playerDataMaster) (int, error) {
	value := 1
	if fullChain != nil {
		value = *fullChain
	}
	var name string
	// 外部プレイヤーデータ側の過去実装との後方互換性を維持するため、
	// fch_lv の 2/3 は一般的な GOLD/PLATINUM の順序と逆で解釈する。
	// - 2 -> FULL CHAIN PLATINUM
	// - 3 -> FULL CHAIN GOLD
	switch value {
	case 1:
		name = "none"
	case 2:
		name = "full chain platinum"
	case 3:
		name = "full chain gold"
	default:
		return 0, &PlayerDataValidationError{Field: "fch_lv", Message: fmt.Sprintf("unknown full chain level: %d", value)}
	}
	item, ok := masters.FullChains[name]
	if !ok {
		return 0, &PlayerDataNotFoundError{Resource: "full_chain", Key: name}
	}
	return item.ID, nil
}

func resolveSlotID(slot *string, masters *playerDataMaster) (int, error) {
	name := defaultSlotName
	if slot != nil {
		trimmed := strings.TrimSpace(*slot)
		if trimmed != "" {
			name = trimmed
		}
	}
	item, ok := masters.Slots[strings.ToLower(name)]
	if !ok {
		return 0, &PlayerDataNotFoundError{Resource: "slot", Key: name}
	}
	return item.ID, nil
}

// calculateAndUpdateRatings はプレイヤーのレーティングを再計算してDBに保存します。
// ベスト枠30曲 + 新曲枠20曲から計算したレーティングを保存します。
func (us *playerDataUsecase) calculateAndUpdateRatings(ctx context.Context, tx repository.Executor, playerID int) (service.RatingStats, error) {
	player, err := us.playerRepo.FindByIDForUpdate(ctx, tx, playerID)
	if err != nil {
		return service.RatingStats{}, err
	}

	// レーティング計算対象のレコードを取得（slot='none'のレコードは除外）
	records, err := us.playerRecRepo.FindByPlayerIDForRating(ctx, tx, playerID)
	if err != nil {
		return service.RatingStats{}, fmt.Errorf("failed to fetch player records: %w", err)
	}

	bestRecords := make([]service.RatingSlotRecord, 0, 30)
	newRecords := make([]service.RatingSlotRecord, 0, 20)
	for _, rec := range records {
		if rec.Chart == nil || rec.Slot == nil {
			return service.RatingStats{}, fmt.Errorf("rating record relation is missing: chart_id=%d", rec.ChartID)
		}
		ratingRecord := service.RatingSlotRecord{ChartID: rec.ChartID, Score: uint32(rec.Score), ChartConst: rec.Chart.Const.Float64()} // #nosec G115
		switch rec.Slot.Name {
		case "best":
			bestRecords = append(bestRecords, ratingRecord)
		case "new":
			newRecords = append(newRecords, ratingRecord)
		}
	}

	stats := service.AggregateOfficialRating(bestRecords, newRecords)

	player.ChangeCalculatedRatings(stats.PlayerRating, stats.BestAverage, stats.NewAverage)
	if err := us.playerRepo.Save(ctx, tx, player); err != nil {
		return service.RatingStats{}, err
	}

	return stats, nil
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
