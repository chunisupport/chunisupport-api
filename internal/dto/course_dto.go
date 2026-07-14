package dto

import (
	"strings"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

type CourseDTO struct {
	ID        int        `json:"id,omitempty"`
	DisplayID string     `json:"display_id"`
	Idx       string     `json:"idx"`
	Name      string     `json:"name"`
	Class     string     `json:"class"`
	IsDeleted bool       `json:"is_deleted,omitempty"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
}

type CourseRecordDTO struct {
	DisplayID string     `json:"display_id"`
	Idx       string     `json:"idx"`
	Name      string     `json:"name"`
	Class     string     `json:"class"`
	IsPlayed  bool       `json:"is_played"`
	Score     uint32     `json:"score"`
	IsClear   bool       `json:"is_clear"`
	ComboLamp *string    `json:"combo_lamp"`
	UpdatedAt *time.Time `json:"updated_at"`
}

func ToCourseDTO(course *entity.Course, editor bool) *CourseDTO {
	if course == nil {
		return nil
	}
	class := ""
	if course.CourseClass != nil {
		class = course.CourseClass.Name
	}
	result := &CourseDTO{DisplayID: course.DisplayID.String(), Idx: course.OfficialIdx, Name: course.Name, Class: class}
	if editor {
		result.ID = course.ID
		result.IsDeleted = course.IsDeleted
		result.UpdatedAt = &course.UpdatedAt
	}
	return result
}

func ToCourseRecordDTO(record *entity.PlayerCourseRecord) *CourseRecordDTO {
	if record == nil || record.Course == nil {
		return nil
	}
	class := ""
	if record.Course.CourseClass != nil {
		class = record.Course.CourseClass.Name
	}
	played := !record.UpdatedAt.IsZero()
	var updated *time.Time
	if played {
		value := record.UpdatedAt
		updated = &value
	}
	var lamp *string
	if record.ComboLamp != nil && !strings.EqualFold(record.ComboLamp.Name, "none") {
		value := record.ComboLamp.Name
		lamp = &value
	}
	return &CourseRecordDTO{DisplayID: record.Course.DisplayID.String(), Idx: record.Course.OfficialIdx, Name: record.Course.Name, Class: class, IsPlayed: played, Score: record.Score.Uint32(), IsClear: record.IsClear, ComboLamp: lamp, UpdatedAt: updated}
}
