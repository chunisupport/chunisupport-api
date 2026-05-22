package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/service"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/playername"
	"github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/info"
)

const (
	maxScoreValue   = 1010000
	minScoreValue   = 1
	tokyoLayout     = "2006/01/02 15:04"
	defaultSlotName = "none"
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

	// アプリバージョンのバリデーション
	if !slices.Contains(info.SupportedAppVersions, payload.AppVersion) {
		return ErrAppVersionUnsupported
	}

	// スコアデータの整合性検証
	errorCount := 0
	maxErrorsToReport := 10
	errorMessages := make([]string, 0, maxErrorsToReport)

	// 通常譜面のスコア検証
	for i, entry := range payload.Scores.Full {
		if errorCount >= maxErrorsToReport {
			break
		}
		if err := validateScoreEntry(&entry, "full", i); err != nil {
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
}

type calculatedOverpowerSummary struct {
	Value   *float64
	Percent *float64
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
) PlayerDataUsecase {
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
		return nil, errors.New("invalid player data")
	}

	loc, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		loc = time.FixedZone("Asia/Tokyo", 9*60*60)
	}

	var lastPlayedAt *time.Time
	if strings.TrimSpace(payload.LastPlayed) != "" {
		parsed, parseErr := time.ParseInLocation(tokyoLayout, payload.LastPlayed, loc)
		if parseErr != nil {
			return nil, errors.New("invalid player data")
		}
		lastPlayedAt = &parsed
	}

	updatedAt, err := time.Parse(time.RFC3339, payload.UpdatedAt)
	if err != nil {
		return nil, errors.New("invalid player data")
	}

	// トランザクション開始前にペイロードの事前検証を実行
	// 明らかに不正なデータがある場合はここで拒否する
	if err := validatePlayerDataPayload(payload); err != nil {
		slog.Warn("player data validation failed", "user_id", user.ID, "error", err.Error())
		return nil, fmt.Errorf("invalid player data: %w", err)
	}

	summaryInput := &PlayerDataSummaryInput{
		Name:           nameVO.String(),
		Level:          payload.Level,
		OfficialRating: payload.Rating,
		LastPlayedAt:   lastPlayedAt,
	}

	result := &api_internal.PlayerDataResult{
		AppVersion: payload.AppVersion,
		ImportedAt: time.Now().UTC(),
	}

	err = us.tm.Transactional(ctx, func(tx repository.Executor) error {
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

		playerID, ensureErr := us.ensurePlayer(ctx, tx, user, summaryInput, updatedAt)
		if ensureErr != nil {
			return ensureErr
		}
		result.PlayerID = playerID

		skippedRecords := make([]api_internal.SkippedRecord, 0, 4)

		honorSkipped, honorErr := us.applyHonors(ctx, tx, playerID, payload.Honors, masters)
		if honorErr != nil {
			return honorErr
		}
		skippedRecords = append(skippedRecords, honorSkipped...)

		counts, scoreSkipped, overpowerSummary, scoreErr := us.applyScores(ctx, tx, playerID, payload.Scores, masters, updatedAt)
		if scoreErr != nil {
			return scoreErr
		}
		skippedRecords = append(skippedRecords, scoreSkipped...)
		summaryInput.OverpowerValue = overpowerSummary.Value
		summaryInput.OverpowerPercent = overpowerSummary.Percent

		playerID, ensureErr = us.ensurePlayer(ctx, tx, user, summaryInput, updatedAt)
		if ensureErr != nil {
			return ensureErr
		}
		result.PlayerID = playerID

		// レーティングを再計算して更新
		ratingErr := us.calculateAndUpdateRatings(ctx, tx, playerID)
		if ratingErr != nil {
			return ratingErr
		}

		result.Counts = counts
		result.Counts.HonorsSkipped = len(honorSkipped)
		result.Summary = api_internal.PlayerDataSummary{
			Name:             summaryInput.Name,
			Level:            summaryInput.Level,
			Rating:           summaryInput.OfficialRating,
			LastPlayedAt:     summaryInput.LastPlayedAt,
			OverpowerValue:   summaryInput.OverpowerValue,
			OverpowerPercent: summaryInput.OverpowerPercent,
		}
		if len(skippedRecords) > 0 {
			result.SkippedRecords = skippedRecords
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	slog.Info("player data imported", "user_id", user.ID, "player_id", result.PlayerID, "hash", bodyHash)
	return result, nil
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
	}

	idxSet := make(map[string]struct{})
	for _, entry := range payload.Scores.Full {
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
// プレイヤー情報（名前、レベル、レーティング等）を更新し、プレイヤーIDを返します。
func (us *playerDataUsecase) ensurePlayer(ctx context.Context, tx repository.Executor, user *entity.User, summary *PlayerDataSummaryInput, updatedAt time.Time) (int, error) {
	// ユーザーに紐づくプレイヤーを検索
	existingPlayer, err := us.playerRepo.FindByUserID(ctx, tx, user.ID)
	if err != nil {
		return 0, err
	}

	// PlayerNameのバリデーション
	playerName, err := playername.NewPlayerName(summary.Name)
	if err != nil {
		return 0, fmt.Errorf("invalid player name: %w", err)
	}

	// エンティティを作成または更新
	player := &entity.Player{
		UserID:            user.ID,
		Name:              playerName,
		Level:             summary.Level,
		OfficialRating:    summary.OfficialRating,
		ClassEmblemID:     summary.ClassEmblemID,
		ClassEmblemBaseID: summary.ClassBaseID,
		LastPlayedAt:      summary.LastPlayedAt,
		OverpowerValue:    summary.OverpowerValue,
		OverpowerPercent:  summary.OverpowerPercent,
		UpdatedAt:         updatedAt,
	}

	if existingPlayer != nil {
		// 既存のプレイヤーを更新
		player.ID = existingPlayer.ID
		player.CreatedAt = existingPlayer.CreatedAt
		// 計算レーティング等は既存の値を維持
		player.CalculatedRating = existingPlayer.CalculatedRating
		player.NewAverageRating = existingPlayer.NewAverageRating
		player.BestAverageRating = existingPlayer.BestAverageRating
	} else {
		player.CreatedAt = time.Now()
	}

	// 保存（IDがなければINSERT、それ以外はUPDATE）
	if err := us.playerRepo.Save(ctx, tx, player); err != nil {
		return 0, err
	}

	// ユーザーとプレイヤーのリンク
	if user.PlayerID == nil || *user.PlayerID != player.ID {
		user.LinkPlayer(player.ID)
		if err := us.userRepo.Save(ctx, tx, user); err != nil {
			return 0, err
		}
	}

	return player.ID, nil
}

// applyHonors はプレイヤーの称号情報を更新します。
// 既存の称号を削除し、新しい称号をバルクインサートします。
// 称号は最大3つであるため、EnsureHonorのループ内呼び出しによるN+1問題を許容します。
func (us *playerDataUsecase) applyHonors(ctx context.Context, tx repository.Executor, playerID int, honors map[string]PlayerDataHonorPayload, masters *playerDataMaster) ([]api_internal.SkippedRecord, error) {
	skipped := make([]api_internal.SkippedRecord, 0, 4)
	if honors == nil {
		return skipped, nil
	}
	if err := us.honorRepo.DeletePlayerHonors(ctx, tx, playerID); err != nil {
		return skipped, err
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

		honorTitle := strings.TrimSpace(honor.Title)
		if honorTitle == "" && (honor.Img == nil || strings.TrimSpace(*honor.Img) == "") {
			skipped = append(skipped, api_internal.SkippedRecord{
				RecordType: "honor",
				Reason:     "image_url required when title is empty",
				Details:    fmt.Sprintf("slot=%d, class=%s", slot, honor.Class),
			})
			continue
		}

		honorID, err := us.honorRepo.EnsureHonor(ctx, tx, honorTitle, typeItem.ID, honor.Img)
		if err != nil {
			skipped = append(skipped, api_internal.SkippedRecord{
				RecordType: "honor",
				Reason:     "failed to create honor",
				Details:    fmt.Sprintf("slot=%d, title=%s, error=%s", slot, honor.Title, err.Error()),
			})
			continue
		}

		assignments = append(assignments, repository.HonorAssignment{
			PlayerID: playerID,
			HonorID:  honorID,
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

	return skipped, nil
}

// applyScores はプレイヤーのスコア情報を更新します。
// 通常譜面とWORLD'S END譜面のスコアをUPSERTします。
func (us *playerDataUsecase) applyScores(ctx context.Context, tx repository.Executor, playerID int, scores PlayerDataScorePayload, masters *playerDataMaster, updatedAt time.Time) (api_internal.PlayerDataCounts, []api_internal.SkippedRecord, calculatedOverpowerSummary, error) {
	counts, skipped, fullRecordsToUpsert := applyFullScores(playerID, scores.Full, masters, updatedAt)
	worldsendCounts, worldsendSkipped, worldsendRecordsToUpsert := applyWorldsendScores(playerID, scores.Worldsend, masters, updatedAt)
	counts.WorldsendRecordsUpserted = worldsendCounts.WorldsendRecordsUpserted
	counts.WorldsendRecordsSkipped = worldsendCounts.WorldsendRecordsSkipped
	skipped = append(skipped, worldsendSkipped...)

	if err := us.playerDataRepo.SavePlayerData(ctx, tx, repository.PlayerDataSaveInput{
		FullRecords:      fullRecordsToUpsert,
		WorldsendRecords: worldsendRecordsToUpsert,
	}); err != nil {
		return counts, skipped, calculatedOverpowerSummary{}, err
	}

	overpowerTargetStats, err := us.playerDataRepo.GetOverpowerTargetStats(ctx, repository.OverpowerTargetFilter{
		ExcludeWorldsend: true,
		ExcludeDeleted:   true,
		PlayerID:         &playerID,
	})
	if err != nil {
		return counts, skipped, calculatedOverpowerSummary{}, err
	}

	records, recErr := us.playerRecRepo.FindByPlayerID(ctx, tx, playerID)
	if recErr != nil {
		return counts, skipped, calculatedOverpowerSummary{}, fmt.Errorf("failed to fetch player records for overpower calculation: %w", recErr)
	}
	lockedSongs, lockedErr := us.listLockedSongsForOverpower(ctx, tx, playerID)
	if lockedErr != nil {
		return counts, skipped, calculatedOverpowerSummary{}, fmt.Errorf("failed to fetch locked songs for overpower calculation: %w", lockedErr)
	}
	overpowerSummary, err := calculateOverpowerSummaryFromPlayerRecords(records, lockedSongs, overpowerTargetStats.MaxOverpowerTotal)
	if err != nil {
		return counts, skipped, calculatedOverpowerSummary{}, fmt.Errorf("failed to aggregate overpower from player records: %w", err)
	}

	return counts, skipped, overpowerSummary, nil
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
				RecordType: "full",
				Reason:     "failed to resolve chart",
				Details:    fmt.Sprintf("idx=%s, diff=%s, error=%s", entry.Idx, entry.Diff, err.Error()),
			})
			continue
		}

		if skippedRecord, ok := validateScoreRange("full", entry, song); ok {
			counts.FullRecordsSkipped++
			skipped = append(skipped, skippedRecord)
			continue
		}

		lampIDs, skippedRecord := resolveCommonLampIDs("full", entry, song, masters)
		if skippedRecord != nil {
			counts.FullRecordsSkipped++
			skipped = append(skipped, *skippedRecord)
			continue
		}

		slotID, err := resolveSlotID(entry.Slot, masters)
		if err != nil {
			counts.FullRecordsSkipped++
			skipped = append(skipped, newResolveSkippedRecord("full", "slot", "slot", entry, song, optionalStringValue(entry.Slot), err))
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
	overpowerRecords, err := playerRecordsToOverpowerRecords(records, false, func(record *entity.PlayerRecord) bool {
		if len(lockedSet) == 0 {
			return true
		}
		if record.ChartDifficulty == nil {
			return false
		}
		_, exists := lockedSet[lockedSongKey(record.Song.ID, record.ChartDifficulty.Name == info.DifficultyNameUltima)]
		return !exists
	})
	if err != nil {
		return calculatedOverpowerSummary{}, err
	}
	if len(overpowerRecords) != len(records) {
		slog.Warn("skipped player records with missing related data during overpower recalculation", "total_records", len(records), "aggregated_records", len(overpowerRecords))
	}
	value, percent := service.CalcOverpowerSummary(overpowerRecords, maxOverpowerTotal)
	return calculatedOverpowerSummary{Value: &value, Percent: &percent}, nil
}

func roundFloat(value float64, scale int) float64 {
	factor := math.Pow10(scale)
	return math.Round(value*factor) / factor
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
func (us *playerDataUsecase) calculateAndUpdateRatings(ctx context.Context, tx repository.Executor, playerID int) error {
	// レーティング計算対象のレコードを取得（slot='none'のレコードは除外）
	records, err := us.playerRecRepo.FindByPlayerIDForRating(ctx, tx, playerID)
	if err != nil {
		return fmt.Errorf("failed to fetch player records: %w", err)
	}

	// レーティング計算用のレコードに変換
	ratingRecords := make([]service.RatingRecord, 0, len(records))
	for _, rec := range records {
		// スロット名が"new"または"new_candidate"の場合は新曲として扱う
		isNew := false
		if rec.Slot != nil {
			slotName := strings.ToLower(rec.Slot.Name)
			isNew = slotName == "new" || slotName == "new_candidate"
		}

		// スコアと譜面定数を取得
		score := uint32(rec.Score) // #nosec G115
		chartConst := 0.0
		if rec.Chart != nil {
			chartConst = float64(rec.Chart.Const)
		}

		ratingRecords = append(ratingRecords, service.RatingRecord{
			Score:      score,
			ChartConst: chartConst,
			IsNew:      isNew,
		})
	}

	// レーティング計算
	stats := service.CalcRatingStats(ratingRecords)

	// データベースに保存
	return us.playerRepo.UpdateCalculatedRatings(ctx, tx, playerID, stats.PlayerRating, stats.BestAverage, stats.NewAverage)
}

func (us *playerDataUsecase) Delete(ctx context.Context, user *entity.User) error {
	if user == nil {
		return errors.New("invalid request")
	}

	return us.tm.Transactional(ctx, func(tx repository.Executor) error {
		if err := us.playerRepo.DeleteByUserID(ctx, tx, user.ID); err != nil {
			return fmt.Errorf("failed to delete player data: %w", err)
		}

		if !user.HasLinkedPlayer() {
			return nil
		}

		user.UnlinkPlayer()
		if err := us.userRepo.Save(ctx, tx, user); err != nil {
			return fmt.Errorf("failed to unlink player from user: %w", err)
		}

		return nil
	})
}
