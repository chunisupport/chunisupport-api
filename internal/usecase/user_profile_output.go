package usecase

import (
	"strings"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/master"
)

type UserPlayerOutput struct {
	*entity.Player
	Honors []*entity.PlayerHonor
}

type UserRecordOutput struct {
	UpdatedAt                              time.Time
	Best, BestCandidate, New, NewCandidate []*PlayerRecordOutput
	All                                    []*PlayerRecordOutput
	WorldsEnd                              []*WorldsendRecordOutput
	Courses                                []*CourseRecordOutput
}

type UserProfileOutput struct {
	Username string
	Player   *UserPlayerOutput
}
type UserUpdatedAtOutput struct{ UpdatedAt *time.Time }
type UserProfileWithRecordsOutput struct {
	UserID    int
	Username  string
	Player    *UserPlayerOutput
	Records   *UserRecordOutput
	UpdatedAt *time.Time
}
type UserRatingRecordOutput struct {
	UpdatedAt                              time.Time
	Best, BestCandidate, New, NewCandidate []*PlayerRecordOutput
}
type UserProfileRatingViewOutput struct {
	Username  string
	Player    *UserPlayerOutput
	Records   *UserRatingRecordOutput
	UpdatedAt *time.Time
}
type UserRecordViewOutput struct {
	UpdatedAt time.Time
	All       []*PlayerRecordOutput
	Worldsend []*WorldsendRecordOutput
	Courses   []*CourseRecordOutput
}
type UserProfileRecordViewOutput struct {
	Username  string
	Player    *UserPlayerOutput
	Records   *UserRecordViewOutput
	UpdatedAt *time.Time
}
type UserSongRecordOutput struct {
	Standard  []*PlayerRecordOutput
	UpdatedAt *time.Time
	Meta      *UserSongRecordMetaOutput
}
type UserWorldsendSongRecordOutput struct {
	Worldsend *WorldsendRecordOutput
	UpdatedAt *time.Time
	Meta      *UserSongRecordMetaOutput
}
type UserSongRecordMetaOutput struct{ UpdatedAt *time.Time }
type PlayerRecordOutput struct {
	*entity.PlayerRecord
	UpdatedAt                             *time.Time
	IsPlayed, IsOPTarget                  bool
	Difficulty                            string
	Const                                 chartconstant.ChartConstant
	ClearLamp, ComboLamp, FullChain, Slot *string
}
type WorldsendRecordOutput struct {
	*entity.PlayerWorldsendRecord
	UpdatedAt *time.Time
	IsPlayed  bool
	ID        string
}

func toPlayerRecordOutput(record *entity.PlayerRecord) *PlayerRecordOutput {
	if record == nil {
		return nil
	}
	result := &PlayerRecordOutput{PlayerRecord: record, IsOPTarget: record.IsOPTarget, ClearLamp: outputMasterName(record.ClearLamp), ComboLamp: outputMasterName(record.ComboLamp), FullChain: outputMasterName(record.FullChain), Slot: outputMasterName(record.Slot)}
	if !record.UpdatedAt.IsZero() {
		value := record.UpdatedAt
		result.UpdatedAt, result.IsPlayed = &value, true
	}
	if record.Chart != nil {
		result.Const = record.Chart.Const
	}
	if record.ChartDifficulty != nil {
		result.Difficulty = strings.ToUpper(record.ChartDifficulty.Name)
	}
	return result
}
func toPlayerRecordOutputs(records []*entity.PlayerRecord) []*PlayerRecordOutput {
	result := make([]*PlayerRecordOutput, 0, len(records))
	for _, record := range records {
		result = append(result, toPlayerRecordOutput(record))
	}
	return result
}
func toWorldsendRecordOutput(record *entity.PlayerWorldsendRecord) *WorldsendRecordOutput {
	if record == nil {
		return nil
	}
	result := &WorldsendRecordOutput{PlayerWorldsendRecord: record}
	if !record.UpdatedAt.IsZero() {
		value := record.UpdatedAt
		result.UpdatedAt, result.IsPlayed = &value, true
	}
	if record.Song != nil {
		result.ID = record.Song.DisplayID
	}
	return result
}
func toWorldsendRecordOutputs(records []*entity.PlayerWorldsendRecord) []*WorldsendRecordOutput {
	result := make([]*WorldsendRecordOutput, 0, len(records))
	for _, record := range records {
		result = append(result, toWorldsendRecordOutput(record))
	}
	return result
}
func outputMasterName(value any) *string {
	var name string
	switch v := value.(type) {
	case *master.ClearLampType:
		if v != nil {
			name = v.Name
		}
	case *master.ComboLampType:
		if v != nil {
			name = v.Name
		}
	case *master.FullChainType:
		if v != nil {
			name = v.Name
		}
	case *master.Slot:
		if v != nil {
			name = v.Name
		}
	}
	if name == "" || strings.EqualFold(name, "none") {
		return nil
	}
	return &name
}

type AdminUserOutput struct {
	UserName       string
	AccountType    string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	PlayerName     *string
	Rating         *float64
	OverPowerValue *float64
	IsSuspicious   bool
	IsPrivate      bool
	FirebaseUID    *string
}
