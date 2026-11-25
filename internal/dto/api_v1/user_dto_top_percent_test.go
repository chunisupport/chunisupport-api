package api_v1

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestV1RecordDTOs_topPercentを公開しない(t *testing.T) {
	// Given: 内部DTOに top_percent が設定されている
	now := time.Date(2026, 11, 26, 0, 0, 0, 0, time.UTC)
	top := 1.23
	internalStd := &dto.PlayerRecordDTO{
		UpdatedAt:  &now,
		IsPlayed:   true,
		Difficulty: "MASTER",
		ID:         "0123456789abcdef",
		Score:      1009500,
		TopPercent: &top,
	}
	internalWE := &dto.WorldsendRecordDTO{
		UpdatedAt:  &now,
		IsPlayed:   true,
		ID:         "fedcba9876543210",
		Score:      1009500,
		TopPercent: &top,
	}
	internalCourse := &dto.CourseRecordDTO{
		DisplayID:  "0011223344556677",
		IsPlayed:   true,
		Score:      3020000,
		TopPercent: &top,
		UpdatedAt:  &now,
	}

	// When: 内部DTOとv1 DTOをそれぞれJSON化する
	internalStdJSON, err := json.Marshal(internalStd)
	require.NoError(t, err)
	v1StdJSON, err := json.Marshal(ToV1PlayerRecordDTO(internalStd))
	require.NoError(t, err)

	internalWEJSON, err := json.Marshal(internalWE)
	require.NoError(t, err)
	v1WEJSON, err := json.Marshal(ToV1WorldsendRecordDTO(internalWE))
	require.NoError(t, err)

	internalCourseJSON, err := json.Marshal(internalCourse)
	require.NoError(t, err)
	v1CourseJSON, err := json.Marshal(ToV1CourseRecordDTO(internalCourse))
	require.NoError(t, err)

	// Then: 内部DTOには top_percent が含まれ、v1 DTOには一切含まれない
	assert.Contains(t, string(internalStdJSON), `"top_percent":1.23`)
	assert.NotContains(t, string(v1StdJSON), `"top_percent"`)

	assert.Contains(t, string(internalWEJSON), `"top_percent":1.23`)
	assert.NotContains(t, string(v1WEJSON), `"top_percent"`)

	assert.Contains(t, string(internalCourseJSON), `"top_percent":1.23`)
	assert.NotContains(t, string(v1CourseJSON), `"top_percent"`)
}
