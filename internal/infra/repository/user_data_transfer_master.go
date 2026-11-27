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
	possessionIDs         map[string]int
	honorTypeIDs          map[string]int
	honorIDsByImage       map[string]int
	honorIDsByNameAndType map[string]int
	achievementTypeIDs    map[string]int
	// goalAttributeNames は目標の絞り込み属性ごとに、保存値から移行ファイル上のマスタ名を引きます。
	goalAttributeNames map[string]map[string]string
	// goalAttributeStoredValues は目標の絞り込み属性ごとに、移行ファイル上のマスタ名から保存値を引きます。
	goalAttributeStoredValues map[string]map[string]string
}

// transferGoalAttributeQueries は目標の絞り込み属性ごとの保存値とマスタ名の対応を取得するSQLです。
// 難易度・ジャンルはマスタ名、バージョンは改名で参照が切れないよう公開IDで保存しています。
var transferGoalAttributeQueries = map[string]string{
	"diff":  `SELECT name AS stored_value, name FROM difficulties`,
	"genre": `SELECT name AS stored_value, name FROM genres`,
	"ver":   `SELECT display_id AS stored_value, name FROM versions`,
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
	if masters.possessionIDs, err = selectTransferStringIntMap(ctx, exec, `SELECT name, id FROM possessions`); err != nil {
		return nil, err
	}
	if masters.honorTypeIDs, err = selectTransferStringIntMap(ctx, exec, `SELECT name, id FROM honor_types`); err != nil {
		return nil, err
	}
	if masters.achievementTypeIDs, err = selectTransferStringIntMap(ctx, exec, `SELECT code AS name, id FROM achievement_types`); err != nil {
		return nil, err
	}
	masters.goalAttributeNames = make(map[string]map[string]string, len(transferGoalAttributeQueries))
	masters.goalAttributeStoredValues = make(map[string]map[string]string, len(transferGoalAttributeQueries))
	for key, query := range transferGoalAttributeQueries {
		var rows []struct {
			StoredValue string `db:"stored_value"`
			Name        string `db:"name"`
		}
		if err := exec.SelectContext(ctx, &rows, query); err != nil {
			return nil, err
		}
		names := make(map[string]string, len(rows))
		storedValues := make(map[string]string, len(rows))
		for _, row := range rows {
			names[row.StoredValue] = row.Name
			storedValues[row.Name] = row.StoredValue
		}
		masters.goalAttributeNames[key] = names
		masters.goalAttributeStoredValues[key] = storedValues
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

func (m *transferMasterData) rememberHonor(id int, imageURL *string, name, typeName string) {
	if imageURL != nil {
		m.honorIDsByImage[*imageURL] = id
	}
	m.honorIDsByNameAndType[name+"\x00"+typeName] = id
}

// transformTransferGoalAttributes は目標の絞り込み属性 diff / genre / ver の各値を conversions で置き換えます。
// 単一値・配列の形は維持し、対応がない値はエラーにします。
func transformTransferGoalAttributes(raw json.RawMessage, conversions map[string]map[string]string) (json.RawMessage, error) {
	var attrs map[string]json.RawMessage
	if err := json.Unmarshal(raw, &attrs); err != nil {
		return nil, err
	}
	for _, key := range []string{"diff", "genre", "ver"} {
		value, ok := attrs[key]
		if !ok {
			continue
		}
		var decoded any
		if err := json.Unmarshal(value, &decoded); err != nil {
			return nil, err
		}
		items := []any{decoded}
		array, isArray := decoded.([]any)
		if isArray {
			items = array
		}
		converted := make([]any, 0, len(items))
		for _, item := range items {
			text, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("goal attribute %s contains a non-string value", key)
			}
			convertedValue, ok := conversions[key][text]
			if !ok {
				return nil, fmt.Errorf("goal attribute %s contains an unknown value", key)
			}
			converted = append(converted, convertedValue)
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

// externalizeGoalAttributes は保存済みの目標属性を移行ファイル形式（マスタ名）へ変換します。
func (m *transferMasterData) externalizeGoalAttributes(raw json.RawMessage) (json.RawMessage, error) {
	return transformTransferGoalAttributes(raw, m.goalAttributeNames)
}

// internalizeGoalAttributes は移行ファイル形式（マスタ名）の目標属性を保存形式へ変換します。
func (m *transferMasterData) internalizeGoalAttributes(raw json.RawMessage) (json.RawMessage, error) {
	return transformTransferGoalAttributes(raw, m.goalAttributeStoredValues)
}

func parseTransferOfficialIndex(value string) uint64 {
	parsed, _ := strconv.ParseUint(value, 10, 64)
	return parsed
}
