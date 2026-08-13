package entity

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo/coursescore"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/goalgroupname"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/playername"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/score"
)

var ErrInvalidUserDataTransfer = errors.New("invalid user data transfer")

// UserDataTransferSnapshot は、サーバー固有IDを含めずに一括検証・保存するための移行集約です。
type UserDataTransferSnapshot struct {
	Player                   UserDataTransferPlayer
	Records                  []UserDataTransferRecord
	RecordHistories          []UserDataTransferRecordHistory
	WorldsendRecords         []UserDataTransferWorldsendRecord
	WorldsendRecordHistories []UserDataTransferWorldsendRecordHistory
	MetricHistories          []UserDataTransferMetricHistory
	CourseRecords            []UserDataTransferCourseRecord
	Honors                   []UserDataTransferHonor
	FavoriteSongs            []UserDataTransferFavoriteSong
	LockedSongs              []UserDataTransferLockedSong
	Goals                    UserDataTransferGoals
	RecordFilters            []UserDataTransferRecordFilter
}

type UserDataTransferPlayer struct {
	Name                playername.PlayerName
	Level               int
	OfficialRating      float64
	OfficialOverpower   float64
	ClassEmblemName     *string
	ClassEmblemBaseName *string
	LastPlayedAt        *time.Time
	DataCollectedAt     *time.Time
	CreatedAt           time.Time
}

type UserDataTransferRecord struct {
	SongOfficialIdx string
	Difficulty      string
	Score           score.Score
	ClearLampName   string
	ComboLampName   string
	FullChainName   string
	SlotName        string
	SlotOrder       *int
	UpdatedAt       time.Time
}

type UserDataTransferRecordHistory struct {
	SongOfficialIdx string
	Difficulty      string
	Score           score.Score
	ClearLampName   string
	ComboLampName   string
	FullChainName   string
	UpdatedAt       time.Time
}

type UserDataTransferWorldsendRecord struct {
	SongOfficialIdx string
	Score           score.Score
	ClearLampName   string
	ComboLampName   string
	FullChainName   string
	UpdatedAt       time.Time
}

type UserDataTransferWorldsendRecordHistory struct {
	SongOfficialIdx string
	Score           score.Score
	ClearLampName   string
	ComboLampName   string
	FullChainName   string
	UpdatedAt       time.Time
}

type UserDataTransferMetricHistory struct {
	OfficialRating    float64
	OfficialOverpower float64
	DataCollectedAt   time.Time
}

type UserDataTransferCourseRecord struct {
	CourseOfficialIdx string
	Score             coursescore.CourseScore
	IsClear           bool
	ComboLampName     string
	UpdatedAt         time.Time
}

type UserDataTransferHonor struct {
	Slot       int
	ImageURL   *string
	Name       string
	TypeName   string
	EquippedAt time.Time
}

type UserDataTransferFavoriteSong struct {
	SongOfficialIdx string
	FavoritedAt     time.Time
}

type UserDataTransferLockedSong struct {
	SongOfficialIdx string
	IsUltima        bool
}

type UserDataTransferGoals struct {
	Groups    []UserDataTransferGoalGroup
	Ungrouped []UserDataTransferGoal
}

type UserDataTransferGoalGroup struct {
	Name      goalgroupname.GoalGroupName
	SortOrder uint16
	CreatedAt time.Time
	Goals     []UserDataTransferGoal
}

type UserDataTransferGoal struct {
	Title             string
	AchievementType   string
	AchievementParams json.RawMessage
	Attributes        json.RawMessage
	InvertValue       bool
	InvertPercentage  bool
	SortOrder         uint16
	CreatedAt         time.Time
}

type UserDataTransferRecordFilter struct {
	Name          string
	FilterType    string
	SchemaVersion int
	Filter        json.RawMessage
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// Validate は、署名済みファイルでも不正状態を永続化しないために集約全体の不変条件を検証します。
func (s *UserDataTransferSnapshot) Validate() error {
	if s == nil {
		return invalidUserDataTransfer("snapshot is required")
	}
	if err := s.validateNonNilCollections(); err != nil {
		return err
	}
	if err := validateUserDataTransferPlayer(s.Player, s.MetricHistories); err != nil {
		return err
	}
	if err := validateUserDataTransferRecords(s.Records); err != nil {
		return err
	}
	if err := validateUserDataTransferRecordHistories(s.RecordHistories); err != nil {
		return err
	}
	if err := validateUserDataTransferWorldsendRecords(s.WorldsendRecords); err != nil {
		return err
	}
	if err := validateUserDataTransferWorldsendRecordHistories(s.WorldsendRecordHistories); err != nil {
		return err
	}
	if err := validateUserDataTransferMetricHistories(s.MetricHistories, s.Player.DataCollectedAt); err != nil {
		return err
	}
	if err := validateUserDataTransferCourseRecords(s.CourseRecords); err != nil {
		return err
	}
	if err := validateUserDataTransferHonors(s.Honors); err != nil {
		return err
	}
	if err := validateUserDataTransferFavoriteSongs(s.FavoriteSongs); err != nil {
		return err
	}
	if err := validateUserDataTransferLockedSongs(s.LockedSongs); err != nil {
		return err
	}
	if err := validateUserDataTransferGoals(s.Goals); err != nil {
		return err
	}
	return validateUserDataTransferRecordFilters(s.RecordFilters)
}

func (s *UserDataTransferSnapshot) validateNonNilCollections() error {
	collections := []struct {
		name  string
		isNil bool
	}{
		{name: "records", isNil: s.Records == nil},
		{name: "record_histories", isNil: s.RecordHistories == nil},
		{name: "worldsend_records", isNil: s.WorldsendRecords == nil},
		{name: "worldsend_record_histories", isNil: s.WorldsendRecordHistories == nil},
		{name: "metric_histories", isNil: s.MetricHistories == nil},
		{name: "course_records", isNil: s.CourseRecords == nil},
		{name: "honors", isNil: s.Honors == nil},
		{name: "favorite_songs", isNil: s.FavoriteSongs == nil},
		{name: "locked_songs", isNil: s.LockedSongs == nil},
		{name: "goals.groups", isNil: s.Goals.Groups == nil},
		{name: "goals.ungrouped", isNil: s.Goals.Ungrouped == nil},
		{name: "record_filters", isNil: s.RecordFilters == nil},
	}
	for _, collection := range collections {
		if collection.isNil {
			return invalidUserDataTransfer(collection.name + " must be an array")
		}
	}
	for i, group := range s.Goals.Groups {
		if group.Goals == nil {
			return invalidUserDataTransfer(fmt.Sprintf("goals.groups[%d].goals must be an array", i))
		}
	}
	return nil
}
