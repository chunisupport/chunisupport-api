package api_internal

import (
	"encoding/json"
	"reflect"
	"slices"
	"strings"

	"github.com/chunisupport/chunisupport-api/internal/usecase"
)

type playerDataRequest struct {
	AppVersion  string                            `json:"app_ver"`
	Name        string                            `json:"name"`
	Level       int                               `json:"level"`
	Rating      *float64                          `json:"rating"`
	LastPlayed  string                            `json:"last_played"`
	Overpower   playerDataOverpowerRequest        `json:"overpower"`
	ClassEmblem playerDataClassRequest            `json:"class_emblem"`
	Possession  string                            `json:"possession"`
	Team        playerDataTeamRequest             `json:"team"`
	Honors      map[string]playerDataHonorRequest `json:"honors"`
	Scores      playerDataScoreRequest            `json:"scores"`
	UpdatedAt   string                            `json:"updated_at"`
}
type playerDataOverpowerRequest struct {
	Value      *float64 `json:"value"`
	Percentage *float64 `json:"percentage"`
}
type playerDataClassRequest struct {
	MedalClass string `json:"medal_class"`
	BaseClass  string `json:"base_class"`
}
type playerDataTeamRequest struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}
type playerDataHonorRequest struct {
	Title string  `json:"title"`
	Class string  `json:"class"`
	Img   *string `json:"img_url"`
}
type playerDataScoreRequest struct {
	Standard  []playerDataScoreEntryRequest  `json:"standard"`
	Worldsend []playerDataScoreEntryRequest  `json:"worldsend"`
	Course    []playerDataCourseEntryRequest `json:"course"`
}
type playerDataCourseEntryRequest struct {
	Score   int    `json:"score"`
	IsClear bool   `json:"is_clear"`
	ComboLv int    `json:"cmb_lv"`
	Idx     string `json:"idx"`
}
type playerDataScoreEntryRequest struct {
	Diff      string  `json:"diff"`
	Idx       string  `json:"idx"`
	Score     int     `json:"score"`
	ClearLamp *string `json:"clear_lamp"`
	ComboLv   *int    `json:"cmb_lv"`
	FullChain *int    `json:"fch_lv"`
	Slot      *string `json:"slot"`
	Order     *int    `json:"order"`
}

func (r playerDataRequest) toUsecase() usecase.PlayerDataPayload {
	honors := make(map[string]usecase.PlayerDataHonorPayload, len(r.Honors))
	for key, value := range r.Honors {
		honors[key] = usecase.PlayerDataHonorPayload{Title: value.Title, Class: value.Class, Img: value.Img}
	}
	standard := make([]usecase.PlayerDataScoreEntry, len(r.Scores.Standard))
	for i, value := range r.Scores.Standard {
		standard[i] = value.toUsecase()
	}
	worldsend := make([]usecase.PlayerDataScoreEntry, len(r.Scores.Worldsend))
	for i, value := range r.Scores.Worldsend {
		worldsend[i] = value.toUsecase()
	}
	courses := make([]usecase.PlayerDataCourseEntry, len(r.Scores.Course))
	for i, value := range r.Scores.Course {
		courses[i] = usecase.PlayerDataCourseEntry{Score: value.Score, IsClear: value.IsClear, ComboLv: value.ComboLv, Idx: value.Idx}
	}
	return usecase.PlayerDataPayload{AppVersion: r.AppVersion, Name: r.Name, Level: r.Level, Rating: r.Rating, LastPlayed: r.LastPlayed, Overpower: usecase.PlayerDataOverpowerPayload{Value: r.Overpower.Value, Percentage: r.Overpower.Percentage}, ClassEmblem: usecase.PlayerDataClassPayload{MedalClass: r.ClassEmblem.MedalClass, BaseClass: r.ClassEmblem.BaseClass}, Possession: r.Possession, Team: usecase.PlayerDataTeamPayload{Name: r.Team.Name, Color: r.Team.Color}, Honors: honors, Scores: usecase.PlayerDataScorePayload{Standard: standard, Worldsend: worldsend, Course: courses}, UpdatedAt: r.UpdatedAt}
}
func (r playerDataScoreEntryRequest) toUsecase() usecase.PlayerDataScoreEntry {
	return usecase.PlayerDataScoreEntry{Diff: r.Diff, Idx: r.Idx, Score: r.Score, ClearLamp: r.ClearLamp, ComboLv: r.ComboLv, FullChain: r.FullChain, Slot: r.Slot, Order: r.Order}
}

// playerDataRequestFields は playerDataRequest のトップレベルのJSONフィールド名です。
// 入力型のJSONタグから導出し、フィールド追加時に未知フィールド判定との二重管理を不要にします。
var playerDataRequestFields = jsonFieldNames(reflect.TypeFor[playerDataRequest]())

// jsonFieldNames は構造体型のJSONフィールド名の集合を返します。
// 埋め込み構造体のフィールド昇格は扱わないため、埋め込みを持たない入力型にだけ使います。
func jsonFieldNames(t reflect.Type) map[string]struct{} {
	names := make(map[string]struct{}, t.NumField())
	for field := range t.Fields() {
		if !field.IsExported() {
			continue
		}
		name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
		if name == "-" {
			continue
		}
		if name == "" {
			name = field.Name
		}
		names[name] = struct{}{}
	}
	return names
}

// unknownPlayerDataFields はプレイヤーデータJSONのトップレベルにある未知フィールド名を名前順で返します。
// 公式エクスポートJSONの前方互換性を保つため、未知フィールドは登録を拒否せず警告ログの対象にします。
func unknownPlayerDataFields(data []byte) ([]string, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	var unknown []string
	for key := range raw {
		if _, ok := playerDataRequestFields[key]; !ok {
			unknown = append(unknown, key)
		}
	}
	slices.Sort(unknown)
	return unknown, nil
}
