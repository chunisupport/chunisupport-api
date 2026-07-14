package models

import (
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/coursescore"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/displayid"
)

type CourseModel struct {
	ID              int       `db:"id"`
	DisplayID       string    `db:"display_id"`
	OfficialIdx     string    `db:"official_idx"`
	Name            string    `db:"name"`
	CourseClassID   int       `db:"course_class_id"`
	IsDeleted       bool      `db:"is_deleted"`
	UpdatedAt       time.Time `db:"updated_at"`
	CourseClassName string    `db:"course_class_name"`
	ClassSortOrder  int       `db:"course_class_sort_order"`
}

func (m CourseModel) ToEntity() (*entity.Course, error) {
	displayID, err := displayid.NewDisplayID(m.DisplayID)
	if err != nil {
		return nil, err
	}
	return &entity.Course{ID: m.ID, DisplayID: displayID, OfficialIdx: m.OfficialIdx, Name: m.Name, CourseClassID: m.CourseClassID,
		IsDeleted: m.IsDeleted, UpdatedAt: m.UpdatedAt,
		CourseClass: &entity.CourseClass{ID: m.CourseClassID, Name: m.CourseClassName, SortOrder: m.ClassSortOrder}}, nil
}

type PlayerCourseRecordModel struct {
	PlayerID        int                     `db:"player_id"`
	CourseID        int                     `db:"course_id"`
	Score           coursescore.CourseScore `db:"score"`
	IsClear         bool                    `db:"is_clear"`
	ComboLampID     int                     `db:"combo_lamp_id"`
	UpdatedAt       *time.Time              `db:"updated_at"`
	OfficialIdx     string                  `db:"official_idx"`
	DisplayID       string                  `db:"display_id"`
	CourseName      string                  `db:"course_name"`
	CourseClassID   int                     `db:"course_class_id"`
	CourseClassName string                  `db:"course_class_name"`
	ClassSortOrder  int                     `db:"course_class_sort_order"`
	ComboLampName   *string                 `db:"combo_lamp_name"`
}

func (m PlayerCourseRecordModel) ToEntity() (*entity.PlayerCourseRecord, error) {
	displayID, err := displayid.NewDisplayID(m.DisplayID)
	if err != nil {
		return nil, err
	}
	var updated time.Time
	if m.UpdatedAt != nil {
		updated = *m.UpdatedAt
	}
	record := &entity.PlayerCourseRecord{PlayerID: m.PlayerID, CourseID: m.CourseID, Score: m.Score,
		IsClear: m.IsClear, ComboLampID: m.ComboLampID, UpdatedAt: updated,
		Course: &entity.Course{ID: m.CourseID, DisplayID: displayID, OfficialIdx: m.OfficialIdx, Name: m.CourseName, CourseClassID: m.CourseClassID,
			CourseClass: &entity.CourseClass{ID: m.CourseClassID, Name: m.CourseClassName, SortOrder: m.ClassSortOrder}}}
	if m.ComboLampName != nil {
		record.ComboLamp = &entity.ComboLampType{ID: m.ComboLampID, Name: *m.ComboLampName}
	}
	return record, nil
}
