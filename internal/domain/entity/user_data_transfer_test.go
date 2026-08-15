package entity

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo/coursescore"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/goalgroupname"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/playername"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/score"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserDataTransferSnapshotValidate(t *testing.T) {
	t.Run("全セクションが妥当なら検証に成功する", func(t *testing.T) {
		snapshot := validUserDataTransferSnapshot(t)

		err := snapshot.Validate()

		assert.NoError(t, err)
	})

	t.Run("トップレベル配列がnilなら拒否する", func(t *testing.T) {
		snapshot := validUserDataTransferSnapshot(t)
		snapshot.Records = nil

		err := snapshot.Validate()

		assert.ErrorIs(t, err, ErrInvalidUserDataTransfer)
		assert.Contains(t, err.Error(), "records")
	})

	t.Run("通常譜面キーの重複を拒否する", func(t *testing.T) {
		snapshot := validUserDataTransferSnapshot(t)
		snapshot.Records = append(snapshot.Records, snapshot.Records[0])

		err := snapshot.Validate()

		assert.ErrorIs(t, err, ErrInvalidUserDataTransfer)
		assert.Contains(t, err.Error(), "records[1]")
	})

	t.Run("難易度の小文字表記を拒否する", func(t *testing.T) {
		snapshot := validUserDataTransferSnapshot(t)
		snapshot.Records[0].Difficulty = "master"

		err := snapshot.Validate()

		assert.ErrorIs(t, err, ErrInvalidUserDataTransfer)
		assert.Contains(t, err.Error(), "difficulty")
	})

	t.Run("DB時刻精度で重複する通常履歴を拒否する", func(t *testing.T) {
		snapshot := validUserDataTransferSnapshot(t)
		duplicate := snapshot.RecordHistories[0]
		duplicate.UpdatedAt = duplicate.UpdatedAt.Add(500 * time.Millisecond)
		snapshot.RecordHistories = append(snapshot.RecordHistories, duplicate)

		err := snapshot.Validate()

		assert.ErrorIs(t, err, ErrInvalidUserDataTransfer)
		assert.Contains(t, err.Error(), "record_histories[1]")
	})

	t.Run("1譜面の履歴が上限を超えたら拒否する", func(t *testing.T) {
		snapshot := validUserDataTransferSnapshot(t)
		snapshot.RecordHistories = make([]UserDataTransferRecordHistory, info.MaxScoreHistoryEntriesPerChart+1)
		for i := range snapshot.RecordHistories {
			snapshot.RecordHistories[i] = validRecordHistory(time.Date(2026, 8, 1, 0, 0, i, 0, time.UTC))
		}

		err := snapshot.Validate()

		assert.ErrorIs(t, err, ErrInvalidUserDataTransfer)
		assert.Contains(t, err.Error(), "record_histories")
	})

	t.Run("スロットの上限を超えるslot_orderを拒否する", func(t *testing.T) {
		snapshot := validUserDataTransferSnapshot(t)
		snapshot.Records[0].SlotOrder = intPointer(31)

		err := snapshot.Validate()

		assert.ErrorIs(t, err, ErrInvalidUserDataTransfer)
		assert.Contains(t, err.Error(), "slot_order")
	})

	t.Run("同一スロット内のslot_order重複を拒否する", func(t *testing.T) {
		snapshot := validUserDataTransferSnapshot(t)
		duplicateOrder := snapshot.Records[0]
		duplicateOrder.SongOfficialIdx = "1002"
		snapshot.Records = append(snapshot.Records, duplicateOrder)

		err := snapshot.Validate()

		assert.ErrorIs(t, err, ErrInvalidUserDataTransfer)
		assert.Contains(t, err.Error(), "slot_order")
	})

	t.Run("公式指標履歴が時系列順でなければ拒否する", func(t *testing.T) {
		snapshot := validUserDataTransferSnapshot(t)
		snapshot.MetricHistories = append(snapshot.MetricHistories, UserDataTransferMetricHistory{
			OfficialRating:    17.01,
			OfficialOverpower: 12001.01,
			DataCollectedAt:   snapshot.MetricHistories[0].DataCollectedAt.Add(-time.Second),
		})

		err := snapshot.Validate()

		assert.ErrorIs(t, err, ErrInvalidUserDataTransfer)
		assert.Contains(t, err.Error(), "metric_histories[1]")
	})

	t.Run("現在値と同時刻以降の公式指標履歴を拒否する", func(t *testing.T) {
		snapshot := validUserDataTransferSnapshot(t)
		snapshot.MetricHistories[0].DataCollectedAt = *snapshot.Player.DataCollectedAt

		err := snapshot.Validate()

		assert.ErrorIs(t, err, ErrInvalidUserDataTransfer)
		assert.Contains(t, err.Error(), "metric_histories[0]")
	})

	t.Run("称号スロットの重複を拒否する", func(t *testing.T) {
		snapshot := validUserDataTransferSnapshot(t)
		duplicate := snapshot.Honors[0]
		duplicate.Name = "別称号"
		snapshot.Honors = append(snapshot.Honors, duplicate)

		err := snapshot.Validate()

		assert.ErrorIs(t, err, ErrInvalidUserDataTransfer)
		assert.Contains(t, err.Error(), "honors[1].slot")
	})

	t.Run("お気に入り上限超過を拒否する", func(t *testing.T) {
		snapshot := validUserDataTransferSnapshot(t)
		snapshot.FavoriteSongs = make([]UserDataTransferFavoriteSong, info.PlayerFavoriteSongMaxCount+1)
		for i := range snapshot.FavoriteSongs {
			snapshot.FavoriteSongs[i] = UserDataTransferFavoriteSong{
				SongOfficialIdx: strings.Repeat("0", 9) + string(rune('A'+i)),
				FavoritedAt:     time.Date(2026, 8, 1, 0, 0, i, 0, time.UTC),
			}
		}

		err := snapshot.Validate()

		assert.ErrorIs(t, err, ErrInvalidUserDataTransfer)
		assert.Contains(t, err.Error(), "favorite_songs")
	})

	t.Run("未解禁楽曲キーの重複を拒否する", func(t *testing.T) {
		snapshot := validUserDataTransferSnapshot(t)
		snapshot.LockedSongs = append(snapshot.LockedSongs, snapshot.LockedSongs[0])

		err := snapshot.Validate()

		assert.ErrorIs(t, err, ErrInvalidUserDataTransfer)
		assert.Contains(t, err.Error(), "locked_songs[1]")
	})

	t.Run("目標グループ名の重複を拒否する", func(t *testing.T) {
		snapshot := validUserDataTransferSnapshot(t)
		duplicate := snapshot.Goals.Groups[0]
		duplicate.SortOrder = 2
		duplicate.Goals = []UserDataTransferGoal{}
		snapshot.Goals.Groups = append(snapshot.Goals.Groups, duplicate)

		err := snapshot.Validate()

		assert.ErrorIs(t, err, ErrInvalidUserDataTransfer)
		assert.Contains(t, err.Error(), "goals.groups[1].name")
	})

	t.Run("グループ内目標の表示順が連続しなければ拒否する", func(t *testing.T) {
		snapshot := validUserDataTransferSnapshot(t)
		snapshot.Goals.Groups[0].Goals[0].SortOrder = 2

		err := snapshot.Validate()

		assert.ErrorIs(t, err, ErrInvalidUserDataTransfer)
		assert.Contains(t, err.Error(), "sort_order")
	})

	t.Run("保存済みフィルタ本文がJSONオブジェクトでなければ拒否する", func(t *testing.T) {
		snapshot := validUserDataTransferSnapshot(t)
		snapshot.RecordFilters[0].Filter = json.RawMessage(`[]`)

		err := snapshot.Validate()

		assert.ErrorIs(t, err, ErrInvalidUserDataTransfer)
		assert.Contains(t, err.Error(), "record_filters[0].filter")
	})

	t.Run("保存済みフィルタが既存上限を超えたら拒否する", func(t *testing.T) {
		snapshot := validUserDataTransferSnapshot(t)
		snapshot.RecordFilters[0].Filter = json.RawMessage(`{"value":"` + strings.Repeat("a", info.RecordFilterMaxPayloadBytes) + `"}`)

		err := snapshot.Validate()

		assert.ErrorIs(t, err, ErrInvalidUserDataTransfer)
		assert.Contains(t, err.Error(), "record_filters[0]")
	})
}

func validUserDataTransferSnapshot(t *testing.T) *UserDataTransferSnapshot {
	t.Helper()

	name, err := playername.NewPlayerName("テスト")
	require.NoError(t, err)
	groupName, err := goalgroupname.NewGoalGroupName("今月")
	require.NoError(t, err)
	standardScore, err := score.NewScore(1_009_000)
	require.NoError(t, err)
	courseScore, err := coursescore.New(3_000_000)
	require.NoError(t, err)

	dataCollectedAt := time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC)
	lastPlayedAt := dataCollectedAt.Add(-time.Hour)
	imageURL := "https://example.com/honor.png"

	goal := UserDataTransferGoal{
		Title:             "SSSを取る",
		AchievementType:   "rank_count",
		AchievementParams: json.RawMessage(`{"count":1}`),
		Attributes:        json.RawMessage(`{"diff":["MASTER"]}`),
		SortOrder:         1,
		CreatedAt:         time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
	}

	return &UserDataTransferSnapshot{
		Player: UserDataTransferPlayer{
			Name:                name,
			Level:               100,
			OfficialRating:      17.25,
			OfficialOverpower:   12_345.67,
			ClassEmblemName:     stringPointer("inf"),
			ClassEmblemBaseName: stringPointer("5"),
			LastPlayedAt:        &lastPlayedAt,
			DataCollectedAt:     &dataCollectedAt,
			CreatedAt:           time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		},
		Records: []UserDataTransferRecord{{
			SongOfficialIdx: "1001",
			Difficulty:      "MASTER",
			Score:           standardScore,
			ClearLampName:   "CLEAR",
			ComboLampName:   "FULL COMBO",
			FullChainName:   "NONE",
			SlotName:        "best",
			SlotOrder:       intPointer(1),
			UpdatedAt:       time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC),
		}},
		RecordHistories: []UserDataTransferRecordHistory{
			validRecordHistory(time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)),
		},
		WorldsendRecords: []UserDataTransferWorldsendRecord{{
			SongOfficialIdx: "2001",
			Score:           standardScore,
			ClearLampName:   "CLEAR",
			ComboLampName:   "FULL COMBO",
			FullChainName:   "NONE",
			UpdatedAt:       time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC),
		}},
		WorldsendRecordHistories: []UserDataTransferWorldsendRecordHistory{{
			SongOfficialIdx: "2001",
			Score:           standardScore,
			ClearLampName:   "FAILED",
			ComboLampName:   "NONE",
			FullChainName:   "NONE",
			UpdatedAt:       time.Date(2026, 8, 9, 0, 0, 0, 0, time.UTC),
		}},
		MetricHistories: []UserDataTransferMetricHistory{{
			OfficialRating:    17.00,
			OfficialOverpower: 12_000.00,
			DataCollectedAt:   time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC),
		}},
		CourseRecords: []UserDataTransferCourseRecord{{
			CourseOfficialIdx: "course-1",
			Score:             courseScore,
			IsClear:           true,
			ComboLampName:     "ALL JUSTICE",
			UpdatedAt:         time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC),
		}},
		Honors: []UserDataTransferHonor{{
			Slot:       1,
			ImageURL:   &imageURL,
			Name:       "虹色称号",
			TypeName:   "rainbow",
			EquippedAt: time.Date(2026, 8, 11, 0, 0, 0, 0, time.UTC),
		}},
		FavoriteSongs: []UserDataTransferFavoriteSong{{
			SongOfficialIdx: "1001",
			FavoritedAt:     time.Date(2026, 8, 11, 0, 0, 0, 0, time.UTC),
		}},
		LockedSongs: []UserDataTransferLockedSong{{
			SongOfficialIdx: "1002",
			IsUltima:        false,
		}},
		Goals: UserDataTransferGoals{
			Groups: []UserDataTransferGoalGroup{{
				Name:      groupName,
				SortOrder: 1,
				CreatedAt: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
				Goals:     []UserDataTransferGoal{goal},
			}},
			Ungrouped: []UserDataTransferGoal{},
		},
		RecordFilters: []UserDataTransferRecordFilter{{
			Name:          "高難度",
			FilterType:    "standard",
			SchemaVersion: 1,
			Filter:        json.RawMessage(`{"difficulties":["MASTER"]}`),
			CreatedAt:     time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
			UpdatedAt:     time.Date(2026, 8, 2, 0, 0, 0, 0, time.UTC),
		}},
	}
}

func validRecordHistory(updatedAt time.Time) UserDataTransferRecordHistory {
	standardScore, _ := score.NewScore(1_000_000)
	return UserDataTransferRecordHistory{
		SongOfficialIdx: "1001",
		Difficulty:      "MASTER",
		Score:           standardScore,
		ClearLampName:   "CLEAR",
		ComboLampName:   "FULL COMBO",
		FullChainName:   "NONE",
		UpdatedAt:       updatedAt,
	}
}

func intPointer(value int) *int          { return &value }
func stringPointer(value string) *string { return &value }
