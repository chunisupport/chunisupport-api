package api_v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/playername"
	"github.com/chunisupport/chunisupport-api/internal/dto"
	"github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
)

func TestToV1PlayerDTO_Ratingには計算値を設定する(t *testing.T) {
	// Given
	officialRating := 17.25
	calculatedRating := 17.1234
	player := &entity.Player{
		Name:             playername.MustNewPlayerName("テストプレイヤー"),
		OfficialRating:   &officialRating,
		CalculatedRating: &calculatedRating,
	}

	// When
	actual := ToV1PlayerDTO(player)

	// Then
	require.NotNil(t, actual)
	assert.Equal(t, &calculatedRating, actual.Rating)
}

func TestToV1UserProfileDTO_Ratingには計算値を設定する(t *testing.T) {
	// Given
	officialRating := 17.25
	calculatedRating := 17.1234
	profile := &api_internal.UserProfileWithRecordsDTO{
		Player: &dto.PlayerDTO{
			Rating:           &officialRating,
			CalculatedRating: &calculatedRating,
		},
	}

	// When
	actual := ToV1UserProfileDTO(profile)

	// Then
	require.NotNil(t, actual)
	require.NotNil(t, actual.Player)
	assert.Equal(t, &calculatedRating, actual.Player.Rating)
}

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
