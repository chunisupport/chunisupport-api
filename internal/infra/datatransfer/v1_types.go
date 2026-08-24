package datatransfer

import (
	"encoding/json"
	"errors"
	"time"
)

type envelope struct {
	Protected string `json:"protected"`
	Payload   string `json:"payload"`
	Signature string `json:"signature"`
}

type protectedHeaderV1 struct {
	Format        string `json:"format"`
	SchemaVersion int    `json:"schema_version"`
}

type payloadV1 struct {
	Player                   playerV1                   `json:"player"`
	Records                  []recordV1                 `json:"records"`
	RecordHistories          []recordHistoryV1          `json:"record_histories"`
	WorldsendRecords         []worldsendRecordV1        `json:"worldsend_records"`
	WorldsendRecordHistories []worldsendRecordHistoryV1 `json:"worldsend_record_histories"`
	MetricHistories          []metricHistoryV1          `json:"metric_histories"`
	CourseRecords            []courseRecordV1           `json:"course_records"`
	Honors                   []honorV1                  `json:"honors"`
	FavoriteSongs            []favoriteSongV1           `json:"favorite_songs"`
	LockedSongs              []lockedSongV1             `json:"locked_songs"`
	Goals                    goalsV1                    `json:"goals"`
	RecordFilters            []recordFilterV1           `json:"record_filters"`
}

type playerV1 struct {
	Name                     string       `json:"name"`
	Level                    int          `json:"level"`
	OfficialRating           float64      `json:"official_rating"`
	OfficialOverpower        float64      `json:"official_overpower"`
	OfficialOverpowerPercent *float64     `json:"official_overpower_percent"`
	ClassEmblemName          *string      `json:"class_emblem_name"`
	ClassEmblemBaseName      *string      `json:"class_emblem_base_name"`
	LastPlayedAt             *utcDateTime `json:"last_played_at"`
	DataCollectedAt          *utcDateTime `json:"data_collected_at"`
	CreatedAt                utcDateTime  `json:"created_at"`
}

type recordV1 struct {
	SongOfficialIdx string      `json:"song_official_idx"`
	Difficulty      string      `json:"difficulty"`
	Score           uint32      `json:"score"`
	ClearLampName   string      `json:"clear_lamp_name"`
	ComboLampName   string      `json:"combo_lamp_name"`
	FullChainName   string      `json:"full_chain_name"`
	SlotName        string      `json:"slot_name"`
	SlotOrder       *int        `json:"slot_order"`
	UpdatedAt       utcDateTime `json:"updated_at"`
}

type recordHistoryV1 struct {
	SongOfficialIdx string      `json:"song_official_idx"`
	Difficulty      string      `json:"difficulty"`
	Score           uint32      `json:"score"`
	ClearLampName   string      `json:"clear_lamp_name"`
	ComboLampName   string      `json:"combo_lamp_name"`
	FullChainName   string      `json:"full_chain_name"`
	UpdatedAt       utcDateTime `json:"updated_at"`
}

type worldsendRecordV1 struct {
	SongOfficialIdx string      `json:"song_official_idx"`
	Score           uint32      `json:"score"`
	ClearLampName   string      `json:"clear_lamp_name"`
	ComboLampName   string      `json:"combo_lamp_name"`
	FullChainName   string      `json:"full_chain_name"`
	UpdatedAt       utcDateTime `json:"updated_at"`
}

type worldsendRecordHistoryV1 struct {
	SongOfficialIdx string      `json:"song_official_idx"`
	Score           uint32      `json:"score"`
	ClearLampName   string      `json:"clear_lamp_name"`
	ComboLampName   string      `json:"combo_lamp_name"`
	FullChainName   string      `json:"full_chain_name"`
	UpdatedAt       utcDateTime `json:"updated_at"`
}

type metricHistoryV1 struct {
	OfficialRating           float64     `json:"official_rating"`
	OfficialOverpower        float64     `json:"official_overpower"`
	OfficialOverpowerPercent *float64    `json:"official_overpower_percent"`
	DataCollectedAt          utcDateTime `json:"data_collected_at"`
}

type courseRecordV1 struct {
	CourseOfficialIdx string      `json:"course_official_idx"`
	Score             uint32      `json:"score"`
	IsClear           bool        `json:"is_clear"`
	ComboLampName     string      `json:"combo_lamp_name"`
	UpdatedAt         utcDateTime `json:"updated_at"`
}

type honorV1 struct {
	Slot       int         `json:"slot"`
	ImageURL   *string     `json:"image_url"`
	Name       string      `json:"name"`
	TypeName   string      `json:"type_name"`
	EquippedAt utcDateTime `json:"equipped_at"`
}

type favoriteSongV1 struct {
	SongOfficialIdx string      `json:"song_official_idx"`
	FavoritedAt     utcDateTime `json:"favorited_at"`
}

type lockedSongV1 struct {
	SongOfficialIdx string `json:"song_official_idx"`
	IsUltima        bool   `json:"is_ultima"`
}

type goalsV1 struct {
	Groups    []goalGroupV1 `json:"groups"`
	Ungrouped []goalV1      `json:"ungrouped"`
}

type goalGroupV1 struct {
	Name      string      `json:"name"`
	SortOrder uint16      `json:"sort_order"`
	CreatedAt utcDateTime `json:"created_at"`
	Goals     []goalV1    `json:"goals"`
}

type goalV1 struct {
	Title             string          `json:"title"`
	AchievementType   string          `json:"achievement_type"`
	AchievementParams json.RawMessage `json:"achievement_params"`
	Attributes        json.RawMessage `json:"attributes"`
	InvertValue       bool            `json:"invert_value"`
	InvertPercentage  bool            `json:"invert_percentage"`
	SortOrder         uint16          `json:"sort_order"`
	CreatedAt         utcDateTime     `json:"created_at"`
}

type recordFilterV1 struct {
	Name          string          `json:"name"`
	FilterType    string          `json:"filter_type"`
	SchemaVersion int             `json:"schema_version"`
	Filter        json.RawMessage `json:"filter"`
	CreatedAt     utcDateTime     `json:"created_at"`
	UpdatedAt     utcDateTime     `json:"updated_at"`
}

type utcDateTime struct {
	time.Time
}

func newUTCDateTime(value time.Time) utcDateTime {
	return utcDateTime{Time: value.UTC().Truncate(time.Second)}
}

func newOptionalUTCDateTime(value *time.Time) *utcDateTime {
	if value == nil {
		return nil
	}
	result := newUTCDateTime(*value)
	return &result
}

func (value utcDateTime) MarshalJSON() ([]byte, error) {
	if value.IsZero() {
		return nil, errors.New("UTC date-time is required")
	}
	return json.Marshal(value.UTC().Truncate(time.Second).Format(time.RFC3339))
}

func (value *utcDateTime) UnmarshalJSON(data []byte) error {
	var encoded string
	if err := json.Unmarshal(data, &encoded); err != nil {
		return err
	}
	parsed, err := time.Parse(time.RFC3339, encoded)
	if err != nil {
		return err
	}
	canonical := parsed.UTC().Truncate(time.Second).Format(time.RFC3339)
	if encoded != canonical {
		return errors.New("UTC date-time must use canonical RFC 3339 format")
	}
	value.Time = parsed.UTC()
	return nil
}
