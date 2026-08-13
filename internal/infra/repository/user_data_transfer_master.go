package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
)

type transferChartMaster struct {
	ID          int     `db:"id"`
	SongID      int     `db:"song_id"`
	OfficialIdx string  `db:"official_idx"`
	Difficulty  string  `db:"difficulty"`
	ChartConst  float64 `db:"chart_const"`
	IsDeleted   bool    `db:"is_deleted"`
}

type transferMasterData struct {
	songIDs               map[string]int
	charts                map[string]transferChartMaster
	worldsendChartIDs     map[string]int
	courseIDs             map[string]int
	clearLampIDs          map[string]int
	comboLampIDs          map[string]int
	fullChainIDs          map[string]int
	slotIDs               map[string]int
	classEmblemIDs        map[string]int
	classEmblemBaseIDs    map[string]int
	honorIDsByImage       map[string]int
	honorIDsByNameAndType map[string]int
	achievementTypeIDs    map[string]int
	difficultyIDs         map[string]int
	difficultyNames       map[int]string
	genreIDs              map[string]int
	genreNames            map[int]string
	versionIDs            map[string]int
	versionNames          map[int]string
}

func loadTransferMasterData(ctx context.Context, exec domainrepo.Executor) (*transferMasterData, error) {
	masters := &transferMasterData{}
	var err error
	if masters.songIDs, err = selectTransferStringIntMap(ctx, exec, `SELECT official_idx AS name, id FROM songs`); err != nil {
		return nil, err
	}
	var charts []transferChartMaster
	if err := exec.SelectContext(ctx, &charts, `SELECT c.id, c.song_id, s.official_idx, d.name AS difficulty, c.const AS chart_const, s.is_deleted FROM charts c INNER JOIN songs s ON s.id = c.song_id INNER JOIN difficulties d ON d.id = c.difficulty_id`); err != nil {
		return nil, err
	}
	masters.charts = make(map[string]transferChartMaster, len(charts))
	for _, chart := range charts {
		masters.charts[transferChartKey(chart.OfficialIdx, chart.Difficulty)] = chart
	}
	if masters.worldsendChartIDs, err = selectTransferStringIntMap(ctx, exec, `SELECT s.official_idx AS name, wc.id FROM worldsend_charts wc INNER JOIN songs s ON s.id = wc.song_id`); err != nil {
		return nil, err
	}
	if masters.courseIDs, err = selectTransferStringIntMap(ctx, exec, `SELECT official_idx AS name, id FROM courses`); err != nil {
		return nil, err
	}
	if masters.clearLampIDs, err = selectTransferStringIntMap(ctx, exec, `SELECT name, id FROM clear_lamp_types`); err != nil {
		return nil, err
	}
	if masters.comboLampIDs, err = selectTransferStringIntMap(ctx, exec, `SELECT name, id FROM combo_lamp_types`); err != nil {
		return nil, err
	}
	if masters.fullChainIDs, err = selectTransferStringIntMap(ctx, exec, `SELECT name, id FROM full_chain_types`); err != nil {
		return nil, err
	}
	if masters.slotIDs, err = selectTransferStringIntMap(ctx, exec, `SELECT name, id FROM slots`); err != nil {
		return nil, err
	}
	if masters.classEmblemIDs, err = selectTransferStringIntMap(ctx, exec, `SELECT name, id FROM class_emblems`); err != nil {
		return nil, err
	}
	if masters.classEmblemBaseIDs, err = selectTransferStringIntMap(ctx, exec, `SELECT name, id FROM class_emblem_bases`); err != nil {
		return nil, err
	}
	if masters.achievementTypeIDs, err = selectTransferStringIntMap(ctx, exec, `SELECT code AS name, id FROM achievement_types`); err != nil {
		return nil, err
	}
	if masters.difficultyIDs, masters.difficultyNames, err = selectTransferBidirectionalMap(ctx, exec, `SELECT id, name FROM difficulties`); err != nil {
		return nil, err
	}
	if masters.genreIDs, masters.genreNames, err = selectTransferBidirectionalMap(ctx, exec, `SELECT id, name FROM genres`); err != nil {
		return nil, err
	}
	if masters.versionIDs, masters.versionNames, err = selectTransferBidirectionalMap(ctx, exec, `SELECT id, name FROM versions`); err != nil {
		return nil, err
	}
	var honors []struct {
		ID       int     `db:"id"`
		Name     string  `db:"name"`
		TypeName string  `db:"type_name"`
		ImageURL *string `db:"image_url"`
	}
	if err := exec.SelectContext(ctx, &honors, `SELECT h.id, h.name, ht.name AS type_name, h.image_url FROM honors h INNER JOIN honor_types ht ON ht.id = h.honor_type_id`); err != nil {
		return nil, err
	}
	masters.honorIDsByImage = make(map[string]int, len(honors))
	masters.honorIDsByNameAndType = make(map[string]int, len(honors))
	for _, honor := range honors {
		if honor.ImageURL != nil {
			masters.honorIDsByImage[*honor.ImageURL] = honor.ID
		}
		masters.honorIDsByNameAndType[honor.Name+"\x00"+honor.TypeName] = honor.ID
	}
	return masters, nil
}

func selectTransferStringIntMap(ctx context.Context, exec domainrepo.Executor, query string) (map[string]int, error) {
	var rows []struct {
		Name string `db:"name"`
		ID   int    `db:"id"`
	}
	if err := exec.SelectContext(ctx, &rows, query); err != nil {
		return nil, err
	}
	result := make(map[string]int, len(rows))
	for _, row := range rows {
		result[row.Name] = row.ID
	}
	return result, nil
}

func selectTransferBidirectionalMap(ctx context.Context, exec domainrepo.Executor, query string) (map[string]int, map[int]string, error) {
	byName, err := selectTransferStringIntMap(ctx, exec, query)
	if err != nil {
		return nil, nil, err
	}
	byID := make(map[int]string, len(byName))
	for name, id := range byName {
		byID[id] = name
	}
	return byName, byID, nil
}

func transferChartKey(officialIdx, difficulty string) string {
	return officialIdx + "\x00" + difficulty
}

func (m *transferMasterData) resolveHonor(imageURL *string, name, typeName string) (int, bool) {
	if imageURL != nil {
		id, ok := m.honorIDsByImage[*imageURL]
		return id, ok
	}
	id, ok := m.honorIDsByNameAndType[name+"\x00"+typeName]
	return id, ok
}

func transformTransferGoalAttributes(raw json.RawMessage, maps map[string]map[string]int, reverseMaps map[string]map[int]string, toExternal bool) (json.RawMessage, error) {
	var attrs map[string]json.RawMessage
	if err := json.Unmarshal(raw, &attrs); err != nil {
		return nil, err
	}
	for _, key := range []string{"diff", "genre", "ver"} {
		value, ok := attrs[key]
		if !ok {
			continue
		}
		var items []any
		var decoded any
		if err := json.Unmarshal(value, &decoded); err != nil {
			return nil, err
		}
		isArray := false
		switch typed := decoded.(type) {
		case []any:
			items = typed
			isArray = true
		default:
			items = []any{typed}
		}
		converted := make([]any, 0, len(items))
		for _, item := range items {
			if toExternal {
				number, ok := item.(float64)
				if !ok || number != float64(int(number)) {
					return nil, fmt.Errorf("goal attribute %s contains a non-integer ID", key)
				}
				name, ok := reverseMaps[key][int(number)]
				if !ok {
					return nil, fmt.Errorf("goal attribute %s contains an unknown ID", key)
				}
				converted = append(converted, name)
			} else {
				name, ok := item.(string)
				if !ok {
					return nil, fmt.Errorf("goal attribute %s contains a non-string name", key)
				}
				id, ok := maps[key][name]
				if !ok {
					return nil, fmt.Errorf("goal attribute %s contains an unknown name", key)
				}
				converted = append(converted, id)
			}
		}
		var normalized any = converted
		if !isArray && len(converted) == 1 {
			normalized = converted[0]
		}
		encoded, err := json.Marshal(normalized)
		if err != nil {
			return nil, err
		}
		attrs[key] = encoded
	}
	encoded, err := json.Marshal(attrs)
	if err != nil {
		return nil, err
	}
	return encoded, nil
}

func (m *transferMasterData) externalizeGoalAttributes(raw json.RawMessage) (json.RawMessage, error) {
	return transformTransferGoalAttributes(raw, nil, map[string]map[int]string{"diff": m.difficultyNames, "genre": m.genreNames, "ver": m.versionNames}, true)
}

func (m *transferMasterData) internalizeGoalAttributes(raw json.RawMessage) (json.RawMessage, error) {
	return transformTransferGoalAttributes(raw, map[string]map[string]int{"diff": m.difficultyIDs, "genre": m.genreIDs, "ver": m.versionIDs}, nil, false)
}

func parseTransferOfficialIndex(value string) uint64 {
	parsed, _ := strconv.ParseUint(value, 10, 64)
	return parsed
}
