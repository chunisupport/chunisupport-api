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
