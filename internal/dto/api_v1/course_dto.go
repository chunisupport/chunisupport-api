package api_v1

import (
	"time"

	"github.com/chunisupport/chunisupport-api/internal/dto"
)

type V1CourseDTO struct {
	DisplayID string `json:"display_id"`
	Idx       string `json:"idx"`
	Name      string `json:"name"`
	Class     string `json:"class"`
}
type V1CourseListResponse struct {
	Courses []*V1CourseDTO `json:"courses"`
}
type V1CourseRecordListResponse struct {
	UpdatedAt *time.Time             `json:"updated_at"`
	Courses   []*dto.CourseRecordDTO `json:"courses"`
}

func ToV1CourseDTO(course *dto.CourseDTO) *V1CourseDTO {
	if course == nil {
		return nil
	}
	return &V1CourseDTO{DisplayID: course.DisplayID, Idx: course.Idx, Name: course.Name, Class: course.Class}
}
