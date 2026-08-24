package api_internal

import "github.com/chunisupport/chunisupport-api/internal/usecase"

type playerDataRequest struct {
	AppVersion  string                            `json:"app_ver"`
	Name        string                            `json:"name"`
	Level       int                               `json:"level"`
	Rating      *float64                          `json:"rating"`
	LastPlayed  string                            `json:"last_played"`
	Overpower   playerDataOverpowerRequest        `json:"overpower"`
	ClassEmblem playerDataClassRequest            `json:"class_emblem"`
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
	return usecase.PlayerDataPayload{AppVersion: r.AppVersion, Name: r.Name, Level: r.Level, Rating: r.Rating, LastPlayed: r.LastPlayed, Overpower: usecase.PlayerDataOverpowerPayload{Value: r.Overpower.Value, Percentage: r.Overpower.Percentage}, ClassEmblem: usecase.PlayerDataClassPayload{MedalClass: r.ClassEmblem.MedalClass, BaseClass: r.ClassEmblem.BaseClass}, Team: usecase.PlayerDataTeamPayload{Name: r.Team.Name, Color: r.Team.Color}, Honors: honors, Scores: usecase.PlayerDataScorePayload{Standard: standard, Worldsend: worldsend, Course: courses}, UpdatedAt: r.UpdatedAt}
}
func (r playerDataScoreEntryRequest) toUsecase() usecase.PlayerDataScoreEntry {
	return usecase.PlayerDataScoreEntry{Diff: r.Diff, Idx: r.Idx, Score: r.Score, ClearLamp: r.ClearLamp, ComboLv: r.ComboLv, FullChain: r.FullChain, Slot: r.Slot, Order: r.Order}
}
