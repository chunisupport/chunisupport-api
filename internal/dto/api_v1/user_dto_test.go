package api_v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chunisupport/chunisupport-api/internal/dto"
)

func TestToV1PlayerRecordDTO_OverpowerPercent(t *testing.T) {
	// Given
	record := &dto.PlayerRecordDTO{
		OverpowerPercent: 97.9412,
	}

	// When
	actual := ToV1PlayerRecordDTO(record)

	// Then
	require.NotNil(t, actual)
	assert.Equal(t, 97.9412, actual.OverpowerPercent)
}

func TestToV1PlayerRecordDTO_JusticeCount(t *testing.T) {
	// Given
	justiceCount := 1
	record := &dto.PlayerRecordDTO{
		JusticeCount: &justiceCount,
	}

	// When
	actual := ToV1PlayerRecordDTO(record)

	// Then
	require.NotNil(t, actual)
	assert.Equal(t, &justiceCount, actual.JusticeCount)
}

func TestToV1WorldsendRecordDTO_JusticeCount(t *testing.T) {
	// Given
	justiceCount := 1
	record := &dto.WorldsendRecordDTO{
		JusticeCount: &justiceCount,
	}

	// When
	actual := ToV1WorldsendRecordDTO(record)

	// Then
	require.NotNil(t, actual)
	assert.Equal(t, &justiceCount, actual.JusticeCount)
}
