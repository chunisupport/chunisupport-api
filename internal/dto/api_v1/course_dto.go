package api_v1

import (
	"time"

	"github.com/chunisupport/chunisupport-api/internal/dto"
)

type V1CourseDTO struct {
	DisplayID string `json:"id"`
	Idx       string `json:"idx"`
	Name      string `json:"name"`
	Class     string `json:"class"`
}

type V1CourseRecordDTO struct {
	DisplayID string     `json:"id"`
	Idx       string     `json:"idx"`
	Name      string     `json:"name"`
	Class     string     `json:"class"`
	IsPlayed  bool       `json:"is_played"`
	Score     uint32     `json:"score"`
	IsClear   bool       `json:"is_clear"`
	ComboLamp *string    `json:"combo_lamp"`
	UpdatedAt *time.Time `json:"updated_at"`
}

type V1CourseListResponse struct {
	Courses []*V1CourseDTO `json:"courses"`
}
type V1CourseRecordListResponse struct {
	UpdatedAt *time.Time           `json:"updated_at"`
	Courses   []*V1CourseRecordDTO `json:"courses"`
}

func ToV1CourseDTO(course *dto.CourseDTO) *V1CourseDTO {
	if course == nil {
		return nil
	}
	return &V1CourseDTO{DisplayID: course.DisplayID, Idx: course.Idx, Name: course.Name, Class: course.Class}
}

func ToV1CourseRecordDTO(course *dto.CourseRecordDTO) *V1CourseRecordDTO {
	if course == nil {
		return nil
	}
	return &V1CourseRecordDTO{
		DisplayID: course.DisplayID,
		Idx:       course.Idx,
		Name:      course.Name,
		Class:     course.Class,
		IsPlayed:  course.IsPlayed,
		Score:     course.Score,
		IsClear:   course.IsClear,
		ComboLamp: course.ComboLamp,
		UpdatedAt: course.UpdatedAt,
	}
}
