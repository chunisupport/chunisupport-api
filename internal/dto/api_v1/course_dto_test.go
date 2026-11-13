package api_v1

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestV1CourseRecordDTO_MarshalsExternalIDAsID(t *testing.T) {
	// Given
	dto := &V1CourseRecordDTO{DisplayID: "0123456789abcdef", Idx: "50020", Name: "course", Class: "1"}

	// When
	actual, err := json.Marshal(dto)

	// Then
	require.NoError(t, err)
	assert.Contains(t, string(actual), `"id":"0123456789abcdef"`)
	assert.NotContains(t, string(actual), "display_id")
}
