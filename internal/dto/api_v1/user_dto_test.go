package api_v1

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/playername/playernametest"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/score"
	"github.com/chunisupport/chunisupport-api/internal/dto"
	"github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
)

func TestToV1PlayerDTO_Ratingには計算値を設定する(t *testing.T) {
	// Given
	officialRating := 17.25
	calculatedRating := 17.1234
	officialOPPercent := 98.7654
	player := &entity.Player{
		Name:                     playernametest.New(t, "テストプレイヤー"),
		OfficialRating:           officialRating,
		OfficialOverpower:        12345.67,
		OfficialOverpowerPercent: &officialOPPercent,
		CalculatedRating:         &calculatedRating,
	}

	// When
	actual := ToV1PlayerDTO(player)

	// Then
	require.NotNil(t, actual)
	assert.Equal(t, &calculatedRating, actual.Rating)
	assert.Equal(t, 12345.67, actual.OfficialOverpower)
	assert.Equal(t, &officialOPPercent, actual.OfficialOverpowerPercent)
}

func TestToV1UserProfileDTO_Ratingには計算値を設定する(t *testing.T) {
	// Given
	officialRating := 17.25
	calculatedRating := 17.1234
	officialOPPercent := 98.7654
	profile := &api_internal.UserProfileWithRecordsDTO{
		Player: &dto.PlayerDTO{
			Rating:                   &officialRating,
			CalculatedRating:         &calculatedRating,
			OfficialOverpower:        12345.67,
			OfficialOverpowerPercent: &officialOPPercent,
		},
	}

	// When
	actual := ToV1UserProfileDTO(profile)

	// Then
	require.NotNil(t, actual)
	require.NotNil(t, actual.Player)
	assert.Equal(t, &calculatedRating, actual.Player.Rating)
	assert.Equal(t, 12345.67, actual.Player.OfficialOverpower)
	assert.Equal(t, &officialOPPercent, actual.Player.OfficialOverpowerPercent)
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

func TestToV1PlayerRecordDTO_IsOPTarget(t *testing.T) {
	// Given
	record := &dto.PlayerRecordDTO{
		IsOPTarget: true,
	}

	// When
	actual := ToV1PlayerRecordDTO(record)

	// Then
	require.NotNil(t, actual)
	assert.True(t, actual.IsOPTarget)
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

func TestToV1UserRecordResponseDTO_コースIDをidとして返す(t *testing.T) {
	// Given
	records := &dto.UserRecordResponseDTO{
		Courses: []*dto.CourseRecordDTO{{DisplayID: "0123456789abcdef"}},
	}

	// When
	actual, err := json.Marshal(ToV1UserRecordResponseDTO(records))

	// Then
	require.NoError(t, err)
	assert.Contains(t, string(actual), `"course":[{"id":"0123456789abcdef"`)
	assert.NotContains(t, string(actual), "display_id")
}

func TestToV1UserRatingDTO_プレイヤー未連携時は空配列とnullを返す(t *testing.T) {
	// Given
	input := &usecase.UserProfileRatingViewOutput{}

	// When
	actual, err := json.Marshal(ToV1UserRatingDTO(input))

	// Then
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"rating": null,
		"best_average": null,
		"new_average": null,
		"best": [],
		"best_candidate": [],
		"new": [],
		"new_candidate": [],
		"meta": {"updated_at": null}
	}`, string(actual))
	assert.NotContains(t, string(actual), `"standard"`)
	assert.NotContains(t, string(actual), `"worldsend"`)
	assert.NotContains(t, string(actual), `"course"`)
}

// newV1RatingTestRecord はレーティング枠テスト用の記録を生成します。
// 内側のエンティティは譜面なし（定数なし・ランプなし）とし、難易度・譜面定数・ランプ・
// スロット・プレイ済み・OP対象の上書き側だけに枠ごとの非既定値を設定します。
// 上書きの代入が消えると既定値に戻るため、退行を検知できます。
func newV1RatingTestRecord(displayID, title, artist, jacket string, scoreValue uint32, difficulty string, chartConst chartconstant.ChartConstant, isOPTarget bool, updatedAt time.Time, slot string) *usecase.PlayerRecordOutput {
	clearLamp := "CLEAR"
	comboLamp := "FULL COMBO"
	fullChain := "FULL CHAIN"
	return &usecase.PlayerRecordOutput{
		PlayerRecord: &entity.PlayerRecord{
			Score: score.Score(scoreValue),
			Song: &entity.Song{
				DisplayID: displayID,
				Title:     title,
				Artist:    artist,
				Jacket:    &jacket,
			},
		},
		UpdatedAt:  &updatedAt,
		IsPlayed:   true,
		IsOPTarget: isOPTarget,
		Difficulty: difficulty,
		Const:      chartConst,
		ClearLamp:  &clearLamp,
		ComboLamp:  &comboLamp,
		FullChain:  &fullChain,
		Slot:       &slot,
	}
}

func TestToV1UserRatingDTO_4枠の記録と項目をJSON全体で固定する(t *testing.T) {
	// Given
	rating := 17.1234
	bestAverage := 17.2345
	newAverage := 16.9567
	recordsUpdatedAt := time.Date(2026, time.August, 27, 12, 34, 56, 0, time.UTC)
	input := &usecase.UserProfileRatingViewOutput{
		Username: "testuser",
		Player: &usecase.UserPlayerOutput{Player: &entity.Player{
			CalculatedRating:  &rating,
			BestAverageRating: &bestAverage,
			NewAverageRating:  &newAverage,
		}},
		Records: &usecase.UserRatingRecordOutput{
			UpdatedAt: recordsUpdatedAt,
			Best: []*usecase.PlayerRecordOutput{
				newV1RatingTestRecord("0000000000000001", "楽曲1", "アーティスト1", "https://example.com/jacket1.png", 1010000, "MASTER", chartconstant.ChartConstant(15), true, time.Date(2026, time.August, 20, 10, 0, 0, 0, time.UTC), "best"),
			},
			BestCandidate: []*usecase.PlayerRecordOutput{
				newV1RatingTestRecord("0000000000000002", "楽曲2", "アーティスト2", "https://example.com/jacket2.png", 1009500, "EXPERT", chartconstant.ChartConstant(14.5), false, time.Date(2026, time.August, 21, 10, 0, 0, 0, time.UTC), "best_candidate"),
			},
			New: []*usecase.PlayerRecordOutput{
				newV1RatingTestRecord("0000000000000003", "楽曲3", "アーティスト3", "https://example.com/jacket3.png", 1008000, "ADVANCED", chartconstant.ChartConstant(14), true, time.Date(2026, time.August, 22, 10, 0, 0, 0, time.UTC), "new"),
			},
			NewCandidate: []*usecase.PlayerRecordOutput{
				newV1RatingTestRecord("0000000000000004", "楽曲4", "アーティスト4", "https://example.com/jacket4.png", 1005000, "ULTIMA", chartconstant.ChartConstant(13.5), false, time.Date(2026, time.August, 23, 10, 0, 0, 0, time.UTC), "new_candidate"),
			},
		},
	}

	// When
	actual, err := json.Marshal(ToV1UserRatingDTO(input))

	// Then
	require.NoError(t, err)
	// 4つの枠それぞれに互いに異なる楽曲IDとスコアが入り、難易度・プレイ済み・OP対象・
	// 譜面定数・各ランプが上書き値で出ることを全体で固定する（枠の接続先取り違えや代入削除を検知する）。
	// 内側のエンティティに譜面を持たせていないため、単曲レーティング系は 0 になる。
	// 理論値（1010000点）の best のみ justice_count が 0 になり、残りは null になる。
	assert.JSONEq(t, `{
		"rating": 17.1234,
		"best_average": 17.2345,
		"new_average": 16.9567,
		"best": [
			{
				"updated_at": "2026-08-20T10:00:00Z",
				"is_played": true,
				"is_op_target": true,
				"difficulty": "MASTER",
				"id": "0000000000000001",
				"title": "楽曲1",
				"artist": "アーティスト1",
				"const": 15,
				"is_const_unknown": false,
				"score": 1010000,
				"justice_count": 0,
				"rating": 0,
				"overpower": 0,
				"overpower_percent": 0,
				"img": "https://example.com/jacket1.png",
				"clear_lamp": "CLEAR",
				"combo_lamp": "FULL COMBO",
				"full_chain": "FULL CHAIN",
				"slot": "best"
			}
		],
		"best_candidate": [
			{
				"updated_at": "2026-08-21T10:00:00Z",
				"is_played": true,
				"is_op_target": false,
				"difficulty": "EXPERT",
				"id": "0000000000000002",
				"title": "楽曲2",
				"artist": "アーティスト2",
				"const": 14.5,
				"is_const_unknown": false,
				"score": 1009500,
				"justice_count": null,
				"rating": 0,
				"overpower": 0,
				"overpower_percent": 0,
				"img": "https://example.com/jacket2.png",
				"clear_lamp": "CLEAR",
				"combo_lamp": "FULL COMBO",
				"full_chain": "FULL CHAIN",
				"slot": "best_candidate"
			}
		],
		"new": [
			{
				"updated_at": "2026-08-22T10:00:00Z",
				"is_played": true,
				"is_op_target": true,
				"difficulty": "ADVANCED",
				"id": "0000000000000003",
				"title": "楽曲3",
				"artist": "アーティスト3",
				"const": 14,
				"is_const_unknown": false,
				"score": 1008000,
				"justice_count": null,
				"rating": 0,
				"overpower": 0,
				"overpower_percent": 0,
				"img": "https://example.com/jacket3.png",
				"clear_lamp": "CLEAR",
				"combo_lamp": "FULL COMBO",
				"full_chain": "FULL CHAIN",
				"slot": "new"
			}
		],
		"new_candidate": [
			{
				"updated_at": "2026-08-23T10:00:00Z",
				"is_played": true,
				"is_op_target": false,
				"difficulty": "ULTIMA",
				"id": "0000000000000004",
				"title": "楽曲4",
				"artist": "アーティスト4",
				"const": 13.5,
				"is_const_unknown": false,
				"score": 1005000,
				"justice_count": null,
				"rating": 0,
				"overpower": 0,
				"overpower_percent": 0,
				"img": "https://example.com/jacket4.png",
				"clear_lamp": "CLEAR",
				"combo_lamp": "FULL COMBO",
				"full_chain": "FULL CHAIN",
				"slot": "new_candidate"
			}
		],
		"meta": {"updated_at": "2026-08-27T12:34:56Z"}
	}`, string(actual))
	assert.NotContains(t, string(actual), `"standard"`)
	assert.NotContains(t, string(actual), `"worldsend"`)
	assert.NotContains(t, string(actual), `"course"`)
}

func TestToV1UserRatingDTO_記録が無いときは更新日時のみ返す(t *testing.T) {
	// Given
	rating := 17.1234
	bestAverage := 17.2345
	newAverage := 16.9567
	playerUpdatedAt := time.Date(2026, time.August, 20, 10, 0, 0, 0, time.UTC)
	input := &usecase.UserProfileRatingViewOutput{
		Username: "testuser",
		Player: &usecase.UserPlayerOutput{Player: &entity.Player{
			CalculatedRating:  &rating,
			BestAverageRating: &bestAverage,
			NewAverageRating:  &newAverage,
		}},
		Records: &usecase.UserRatingRecordOutput{UpdatedAt: playerUpdatedAt},
	}

	// When
	actual, err := json.Marshal(ToV1UserRatingDTO(input))

	// Then
	require.NoError(t, err)
	// 記録が無い場合は各枠が空配列になり、meta.updated_at にレコード更新日時
	//（対象レコードがなければプレイヤー更新日時）が入ることを固定する。
	assert.JSONEq(t, `{
		"rating": 17.1234,
		"best_average": 17.2345,
		"new_average": 16.9567,
		"best": [],
		"best_candidate": [],
		"new": [],
		"new_candidate": [],
		"meta": {"updated_at": "2026-08-20T10:00:00Z"}
	}`, string(actual))
}
