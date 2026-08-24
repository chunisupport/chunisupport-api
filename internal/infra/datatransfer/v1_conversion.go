package datatransfer

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/coursescore"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/goalgroupname"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/playername"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/score"
)

func newPayloadV1(snapshot *entity.UserDataTransferSnapshot) payloadV1 {
	payload := payloadV1{
		Player: playerV1{
			Name:                     snapshot.Player.Name.String(),
			Level:                    snapshot.Player.Level,
			OfficialRating:           snapshot.Player.OfficialRating,
			OfficialOverpower:        snapshot.Player.OfficialOverpower,
			OfficialOverpowerPercent: cloneFloat64Pointer(snapshot.Player.OfficialOverpowerPercent),
			ClassEmblemName:          cloneStringPointer(snapshot.Player.ClassEmblemName),
			ClassEmblemBaseName:      cloneStringPointer(snapshot.Player.ClassEmblemBaseName),
			LastPlayedAt:             newOptionalUTCDateTime(snapshot.Player.LastPlayedAt),
			DataCollectedAt:          newOptionalUTCDateTime(snapshot.Player.DataCollectedAt),
			CreatedAt:                newUTCDateTime(snapshot.Player.CreatedAt),
		},
		Records:                  make([]recordV1, len(snapshot.Records)),
		RecordHistories:          make([]recordHistoryV1, len(snapshot.RecordHistories)),
		WorldsendRecords:         make([]worldsendRecordV1, len(snapshot.WorldsendRecords)),
		WorldsendRecordHistories: make([]worldsendRecordHistoryV1, len(snapshot.WorldsendRecordHistories)),
		MetricHistories:          make([]metricHistoryV1, len(snapshot.MetricHistories)),
		CourseRecords:            make([]courseRecordV1, len(snapshot.CourseRecords)),
		Honors:                   make([]honorV1, len(snapshot.Honors)),
		FavoriteSongs:            make([]favoriteSongV1, len(snapshot.FavoriteSongs)),
		LockedSongs:              make([]lockedSongV1, len(snapshot.LockedSongs)),
		Goals: goalsV1{
			Groups:    make([]goalGroupV1, len(snapshot.Goals.Groups)),
			Ungrouped: make([]goalV1, len(snapshot.Goals.Ungrouped)),
		},
		RecordFilters: make([]recordFilterV1, len(snapshot.RecordFilters)),
	}

	for i, record := range snapshot.Records {
		payload.Records[i] = recordV1{
			SongOfficialIdx: record.SongOfficialIdx,
			Difficulty:      record.Difficulty,
			Score:           uint32(record.Score),
			ClearLampName:   record.ClearLampName,
			ComboLampName:   record.ComboLampName,
			FullChainName:   record.FullChainName,
			SlotName:        record.SlotName,
			SlotOrder:       cloneIntPointer(record.SlotOrder),
			UpdatedAt:       newUTCDateTime(record.UpdatedAt),
		}
	}
	for i, history := range snapshot.RecordHistories {
		payload.RecordHistories[i] = recordHistoryV1{
			SongOfficialIdx: history.SongOfficialIdx,
			Difficulty:      history.Difficulty,
			Score:           uint32(history.Score),
			ClearLampName:   history.ClearLampName,
			ComboLampName:   history.ComboLampName,
			FullChainName:   history.FullChainName,
			UpdatedAt:       newUTCDateTime(history.UpdatedAt),
		}
	}
	for i, record := range snapshot.WorldsendRecords {
		payload.WorldsendRecords[i] = worldsendRecordV1{
			SongOfficialIdx: record.SongOfficialIdx,
			Score:           uint32(record.Score),
			ClearLampName:   record.ClearLampName,
			ComboLampName:   record.ComboLampName,
			FullChainName:   record.FullChainName,
			UpdatedAt:       newUTCDateTime(record.UpdatedAt),
		}
	}
	for i, history := range snapshot.WorldsendRecordHistories {
		payload.WorldsendRecordHistories[i] = worldsendRecordHistoryV1{
			SongOfficialIdx: history.SongOfficialIdx,
			Score:           uint32(history.Score),
			ClearLampName:   history.ClearLampName,
			ComboLampName:   history.ComboLampName,
			FullChainName:   history.FullChainName,
			UpdatedAt:       newUTCDateTime(history.UpdatedAt),
		}
	}
	for i, history := range snapshot.MetricHistories {
		payload.MetricHistories[i] = metricHistoryV1{
			OfficialRating:           history.OfficialRating,
			OfficialOverpower:        history.OfficialOverpower,
			OfficialOverpowerPercent: cloneFloat64Pointer(history.OfficialOverpowerPercent),
			DataCollectedAt:          newUTCDateTime(history.DataCollectedAt),
		}
	}
	for i, record := range snapshot.CourseRecords {
		payload.CourseRecords[i] = courseRecordV1{
			CourseOfficialIdx: record.CourseOfficialIdx,
			Score:             record.Score.Uint32(),
			IsClear:           record.IsClear,
			ComboLampName:     record.ComboLampName,
			UpdatedAt:         newUTCDateTime(record.UpdatedAt),
		}
	}
	for i, honor := range snapshot.Honors {
		payload.Honors[i] = honorV1{
			Slot:       honor.Slot,
			ImageURL:   cloneStringPointer(honor.ImageURL),
			Name:       honor.Name,
			TypeName:   honor.TypeName,
			EquippedAt: newUTCDateTime(honor.EquippedAt),
		}
	}
	for i, favorite := range snapshot.FavoriteSongs {
		payload.FavoriteSongs[i] = favoriteSongV1{
			SongOfficialIdx: favorite.SongOfficialIdx,
			FavoritedAt:     newUTCDateTime(favorite.FavoritedAt),
		}
	}
	for i, locked := range snapshot.LockedSongs {
		payload.LockedSongs[i] = lockedSongV1{
			SongOfficialIdx: locked.SongOfficialIdx,
			IsUltima:        locked.IsUltima,
		}
	}
	for i, group := range snapshot.Goals.Groups {
		goals := make([]goalV1, len(group.Goals))
		for goalIndex, goal := range group.Goals {
			goals[goalIndex] = newGoalV1(goal)
		}
		payload.Goals.Groups[i] = goalGroupV1{
			Name:      group.Name.String(),
			SortOrder: group.SortOrder,
			CreatedAt: newUTCDateTime(group.CreatedAt),
			Goals:     goals,
		}
	}
	for i, goal := range snapshot.Goals.Ungrouped {
		payload.Goals.Ungrouped[i] = newGoalV1(goal)
	}
	for i, filter := range snapshot.RecordFilters {
		payload.RecordFilters[i] = recordFilterV1{
			Name:          filter.Name,
			FilterType:    filter.FilterType,
			SchemaVersion: filter.SchemaVersion,
			Filter:        cloneRawMessage(filter.Filter),
			CreatedAt:     newUTCDateTime(filter.CreatedAt),
			UpdatedAt:     newUTCDateTime(filter.UpdatedAt),
		}
	}
	return payload
}

func newGoalV1(goal entity.UserDataTransferGoal) goalV1 {
	return goalV1{
		Title:             goal.Title,
		AchievementType:   goal.AchievementType,
		AchievementParams: cloneRawMessage(goal.AchievementParams),
		Attributes:        cloneRawMessage(goal.Attributes),
		InvertValue:       goal.InvertValue,
		InvertPercentage:  goal.InvertPercentage,
		SortOrder:         goal.SortOrder,
		CreatedAt:         newUTCDateTime(goal.CreatedAt),
	}
}

func (payload payloadV1) toSnapshot() (*entity.UserDataTransferSnapshot, error) {
	name, err := playername.NewPlayerName(payload.Player.Name)
	if err != nil {
		return nil, fmt.Errorf("player name: %w", err)
	}
	records, err := convertSlice(payload.Records, recordV1.toEntity)
	if err != nil {
		return nil, err
	}
	recordHistories, err := convertSlice(payload.RecordHistories, recordHistoryV1.toEntity)
	if err != nil {
		return nil, err
	}
	worldsendRecords, err := convertSlice(payload.WorldsendRecords, worldsendRecordV1.toEntity)
	if err != nil {
		return nil, err
	}
	worldsendHistories, err := convertSlice(payload.WorldsendRecordHistories, worldsendRecordHistoryV1.toEntity)
	if err != nil {
		return nil, err
	}
	courseRecords, err := convertSlice(payload.CourseRecords, courseRecordV1.toEntity)
	if err != nil {
		return nil, err
	}
	groups, err := convertSlice(payload.Goals.Groups, goalGroupV1.toEntity)
	if err != nil {
		return nil, err
	}

	return &entity.UserDataTransferSnapshot{
		Player: entity.UserDataTransferPlayer{
			Name:                     name,
			Level:                    payload.Player.Level,
			OfficialRating:           payload.Player.OfficialRating,
			OfficialOverpower:        payload.Player.OfficialOverpower,
			OfficialOverpowerPercent: cloneFloat64Pointer(payload.Player.OfficialOverpowerPercent),
			ClassEmblemName:          cloneStringPointer(payload.Player.ClassEmblemName),
			ClassEmblemBaseName:      cloneStringPointer(payload.Player.ClassEmblemBaseName),
			LastPlayedAt:             optionalTime(payload.Player.LastPlayedAt),
			DataCollectedAt:          optionalTime(payload.Player.DataCollectedAt),
			CreatedAt:                payload.Player.CreatedAt.Time,
		},
		Records:                  records,
		RecordHistories:          recordHistories,
		WorldsendRecords:         worldsendRecords,
		WorldsendRecordHistories: worldsendHistories,
		MetricHistories: mapSlice(payload.MetricHistories, func(history metricHistoryV1) entity.UserDataTransferMetricHistory {
			return entity.UserDataTransferMetricHistory{
				OfficialRating:           history.OfficialRating,
				OfficialOverpower:        history.OfficialOverpower,
				OfficialOverpowerPercent: cloneFloat64Pointer(history.OfficialOverpowerPercent),
				DataCollectedAt:          history.DataCollectedAt.Time,
			}
		}),
		CourseRecords: courseRecords,
		Honors: mapSlice(payload.Honors, func(honor honorV1) entity.UserDataTransferHonor {
			return entity.UserDataTransferHonor{
				Slot:       honor.Slot,
				ImageURL:   cloneStringPointer(honor.ImageURL),
				Name:       honor.Name,
				TypeName:   honor.TypeName,
				EquippedAt: honor.EquippedAt.Time,
			}
		}),
		FavoriteSongs: mapSlice(payload.FavoriteSongs, func(favorite favoriteSongV1) entity.UserDataTransferFavoriteSong {
			return entity.UserDataTransferFavoriteSong{
				SongOfficialIdx: favorite.SongOfficialIdx,
				FavoritedAt:     favorite.FavoritedAt.Time,
			}
		}),
		LockedSongs: mapSlice(payload.LockedSongs, func(locked lockedSongV1) entity.UserDataTransferLockedSong {
			return entity.UserDataTransferLockedSong{
				SongOfficialIdx: locked.SongOfficialIdx,
				IsUltima:        locked.IsUltima,
			}
		}),
		Goals: entity.UserDataTransferGoals{
			Groups:    groups,
			Ungrouped: mapSlice(payload.Goals.Ungrouped, goalV1.toEntity),
		},
		RecordFilters: mapSlice(payload.RecordFilters, func(filter recordFilterV1) entity.UserDataTransferRecordFilter {
			return entity.UserDataTransferRecordFilter{
				Name:          filter.Name,
				FilterType:    filter.FilterType,
				SchemaVersion: filter.SchemaVersion,
				Filter:        cloneRawMessage(filter.Filter),
				CreatedAt:     filter.CreatedAt.Time,
				UpdatedAt:     filter.UpdatedAt.Time,
			}
		}),
	}, nil
}

func (record recordV1) toEntity() (entity.UserDataTransferRecord, error) {
	parsedScore, err := score.NewScore(record.Score)
	if err != nil {
		return entity.UserDataTransferRecord{}, fmt.Errorf("record score: %w", err)
	}
	return entity.UserDataTransferRecord{
		SongOfficialIdx: record.SongOfficialIdx,
		Difficulty:      record.Difficulty,
		Score:           parsedScore,
		ClearLampName:   record.ClearLampName,
		ComboLampName:   record.ComboLampName,
		FullChainName:   record.FullChainName,
		SlotName:        record.SlotName,
		SlotOrder:       cloneIntPointer(record.SlotOrder),
		UpdatedAt:       record.UpdatedAt.Time,
	}, nil
}

func (history recordHistoryV1) toEntity() (entity.UserDataTransferRecordHistory, error) {
	parsedScore, err := score.NewScore(history.Score)
	if err != nil {
		return entity.UserDataTransferRecordHistory{}, fmt.Errorf("record history score: %w", err)
	}
	return entity.UserDataTransferRecordHistory{
		SongOfficialIdx: history.SongOfficialIdx,
		Difficulty:      history.Difficulty,
		Score:           parsedScore,
		ClearLampName:   history.ClearLampName,
		ComboLampName:   history.ComboLampName,
		FullChainName:   history.FullChainName,
		UpdatedAt:       history.UpdatedAt.Time,
	}, nil
}

func (record worldsendRecordV1) toEntity() (entity.UserDataTransferWorldsendRecord, error) {
	parsedScore, err := score.NewScore(record.Score)
	if err != nil {
		return entity.UserDataTransferWorldsendRecord{}, fmt.Errorf("worldsend record score: %w", err)
	}
	return entity.UserDataTransferWorldsendRecord{
		SongOfficialIdx: record.SongOfficialIdx,
		Score:           parsedScore,
		ClearLampName:   record.ClearLampName,
		ComboLampName:   record.ComboLampName,
		FullChainName:   record.FullChainName,
		UpdatedAt:       record.UpdatedAt.Time,
	}, nil
}

func (history worldsendRecordHistoryV1) toEntity() (entity.UserDataTransferWorldsendRecordHistory, error) {
	parsedScore, err := score.NewScore(history.Score)
	if err != nil {
		return entity.UserDataTransferWorldsendRecordHistory{}, fmt.Errorf("worldsend history score: %w", err)
	}
	return entity.UserDataTransferWorldsendRecordHistory{
		SongOfficialIdx: history.SongOfficialIdx,
		Score:           parsedScore,
		ClearLampName:   history.ClearLampName,
		ComboLampName:   history.ComboLampName,
		FullChainName:   history.FullChainName,
		UpdatedAt:       history.UpdatedAt.Time,
	}, nil
}

func (record courseRecordV1) toEntity() (entity.UserDataTransferCourseRecord, error) {
	parsedScore, err := coursescore.New(record.Score)
	if err != nil {
		return entity.UserDataTransferCourseRecord{}, fmt.Errorf("course score: %w", err)
	}
	return entity.UserDataTransferCourseRecord{
		CourseOfficialIdx: record.CourseOfficialIdx,
		Score:             parsedScore,
		IsClear:           record.IsClear,
		ComboLampName:     record.ComboLampName,
		UpdatedAt:         record.UpdatedAt.Time,
	}, nil
}

func (group goalGroupV1) toEntity() (entity.UserDataTransferGoalGroup, error) {
	name, err := goalgroupname.NewGoalGroupName(group.Name)
	if err != nil {
		return entity.UserDataTransferGoalGroup{}, fmt.Errorf("goal group name: %w", err)
	}
	return entity.UserDataTransferGoalGroup{
		Name:      name,
		SortOrder: group.SortOrder,
		CreatedAt: group.CreatedAt.Time,
		Goals:     mapSlice(group.Goals, goalV1.toEntity),
	}, nil
}

func (goal goalV1) toEntity() entity.UserDataTransferGoal {
	return entity.UserDataTransferGoal{
		Title:             goal.Title,
		AchievementType:   goal.AchievementType,
		AchievementParams: cloneRawMessage(goal.AchievementParams),
		Attributes:        cloneRawMessage(goal.Attributes),
		InvertValue:       goal.InvertValue,
		InvertPercentage:  goal.InvertPercentage,
		SortOrder:         goal.SortOrder,
		CreatedAt:         goal.CreatedAt.Time,
	}
}

func convertSlice[S, D any](source []S, convert func(S) (D, error)) ([]D, error) {
	if source == nil {
		return nil, nil
	}
	result := make([]D, len(source))
	for i, value := range source {
		converted, err := convert(value)
		if err != nil {
			return nil, err
		}
		result[i] = converted
	}
	return result, nil
}

func mapSlice[S, D any](source []S, convert func(S) D) []D {
	if source == nil {
		return nil
	}
	result := make([]D, len(source))
	for i, value := range source {
		result[i] = convert(value)
	}
	return result
}

func optionalTime(value *utcDateTime) *time.Time {
	if value == nil {
		return nil
	}
	result := value.Time
	return &result
}

func cloneStringPointer(value *string) *string {
	if value == nil {
		return nil
	}
	result := *value
	return &result
}

func cloneIntPointer(value *int) *int {
	if value == nil {
		return nil
	}
	result := *value
	return &result
}

func cloneFloat64Pointer(value *float64) *float64 {
	if value == nil {
		return nil
	}
	result := *value
	return &result
}

func cloneRawMessage(value json.RawMessage) json.RawMessage {
	return append(json.RawMessage(nil), value...)
}
