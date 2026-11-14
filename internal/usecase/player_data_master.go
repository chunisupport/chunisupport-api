package usecase

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/masterdata"
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

// playerDataMaster はプレイヤーデータ登録時に使用するマスターデータのキャッシュを保持します。
type playerDataMaster struct {
	*masterdata.PlayerDataMasters
	songs             map[string]entity.PlayerDataSong
	chartsByKey       map[string]entity.PlayerDataChart
	chartsByID        map[int]entity.PlayerDataChart
	worldsendBySongID map[int]entity.PlayerDataWorldsendChart
	courses           map[string]*entity.Course
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
		// クラスエンブレム未登録で他の情報の登録を失敗させないよう、未解決時は nil のままにします。
	}

	baseKey := normalizeClassEmblemKey(payload.BaseClass)
	if baseKey != "" {
		if item, ok := masters.ClassEmblemBases[baseKey]; ok {
			v := item.ID
			baseID = &v
		}
		// クラスエンブレムベース未登録で他の情報の登録を失敗させないよう、未解決時は nil のままにします。
	}

	return classID, baseID, nil
}

func resolvePossessionID(raw string, masters *playerDataMaster) int {
	key := strings.ToLower(strings.TrimSpace(raw))
	if key == "" {
		key = entity.PossessionNameNormal
	}
	if masters != nil {
		if item, ok := masters.Possessions[key]; ok {
			return item.ID
		}
		if item, ok := masters.Possessions[entity.PossessionNameNormal]; ok {
			return item.ID
		}
	}
	return entity.DefaultPossessionID
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
	// fch_lv の 2/3 は一般的な GOLD/PLATINUM の順序と逆に、2 を FULL CHAIN PLATINUM、3 を FULL CHAIN GOLD として扱います。
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
