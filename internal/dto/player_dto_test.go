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
	player := &entity.Player{
		Name:             playername.MustNewPlayerName("テストプレイヤー"),
		OfficialRating:   &officialRating,
		CalculatedRating: &calculatedRating,
	}

	// When
	actual := ToPlayerDTO(player)

	// Then
	require.NotNil(t, actual)
	assert.Equal(t, &calculatedRating, actual.Rating)
}
