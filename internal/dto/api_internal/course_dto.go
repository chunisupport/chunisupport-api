package api_internal

import "github.com/chunisupport/chunisupport-api/internal/dto"

type CourseListResponse struct {
	Courses []*dto.CourseDTO `json:"courses"`
}
type CourseRecordListResponse struct {
	Courses []*dto.CourseRecordDTO `json:"courses"`
	Meta    *UserRecordMetaDTO     `json:"meta"`
}

type CreateCourseRequest struct {
	Idx   string `json:"idx" validate:"required,max=32"`
	Name  string `json:"name" validate:"required,max=255"`
	Class string `json:"class" validate:"required,max=16"`
}

type UpdateCourseRequest struct {
	Name  string `json:"name" validate:"required,max=255"`
	Class string `json:"class" validate:"required,max=16"`
}
