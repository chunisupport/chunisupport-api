package dto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/playername"
)

func TestToPlayerDTO_Ratingには計算値を設定する(t *testing.T) {
	// Given
	officialRating := 17.25
	calculatedRating := 17.1234
	officialOPPercent := 98.7654
	player := &entity.Player{
		Name:                     playername.MustNewPlayerName("テストプレイヤー"),
		OfficialRating:           officialRating,
		OfficialOverpower:        12345.67,
		OfficialOverpowerPercent: &officialOPPercent,
		CalculatedRating:         &calculatedRating,
	}

	// When
	actual := ToPlayerDTO(player)

	// Then
	require.NotNil(t, actual)
	assert.Equal(t, &calculatedRating, actual.Rating)
	assert.Equal(t, 12345.67, actual.OfficialOverpower)
	assert.Equal(t, &officialOPPercent, actual.OfficialOverpowerPercent)
}
