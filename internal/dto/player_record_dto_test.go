package dto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/service"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/score"
)

func TestToPlayerRecordDTO_OverpowerPercent(t *testing.T) {
	// Given
	recordScore, err := score.NewScore(1009000)
	require.NoError(t, err)

	record := &entity.PlayerRecord{
		Score:       recordScore,
		ComboLampID: 3,
		Chart: &entity.Chart{
			Const: chartconstant.ChartConstant(14.0),
		},
	}

	// When
	actual := ToPlayerRecordDTO(record)

	// Then
	require.NotNil(t, actual)
	assert.Equal(t, service.CalcSingleOverpowerPercent(1009000, 14.0, 3), actual.OverpowerPercent)
}
