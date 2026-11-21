package handler

import (
	"encoding/json"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToCourseTracksDTO_Worldsend候補は専用メタデータをトップレベルでnull出力する(t *testing.T) {
	tracks := []usecase.CourseTrackOutput{{
		CourseTrack: entity.CourseTrack{Number: 1, SelectionType: entity.CourseTrackFixed},
		Candidates: []entity.CourseChartCandidate{{
			CourseChartRef: entity.CourseChartRef{SongID: "0123456789abcdef", Difficulty: entity.CourseTrackDifficultyWorldsend},
			Title:          "WORLD'S END譜面",
		}},
	}}

	dtoTracks := ToCourseTracksDTO(tracks)
	payload, err := json.Marshal(dtoTracks[0].Candidates)
	require.NoError(t, err)

	assert.JSONEq(t, `[{
		"id":"0123456789abcdef",
		"difficulty":"WORLDSEND",
		"title":"WORLD'S END譜面",
		"jacket":null,
		"attribute":null,
		"level_star":null,
		"notes":null
	}]`, string(payload))
}

func TestToCourseTracksDTO_通常候補のJSON契約を維持する(t *testing.T) {
	jacket := "jacket"
	tracks := []usecase.CourseTrackOutput{{
		CourseTrack: entity.CourseTrack{Number: 1, SelectionType: entity.CourseTrackFixed},
		Candidates: []entity.CourseChartCandidate{{
			CourseChartRef: entity.CourseChartRef{SongID: "0123456789abcdef", Difficulty: "MASTER"},
			Title:          "通常譜面",
			Const:          14.5,
			IsConstUnknown: true,
			Jacket:         &jacket,
		}},
	}}

	dtoTracks := ToCourseTracksDTO(tracks)
	payload, err := json.Marshal(dtoTracks[0].Candidates)
	require.NoError(t, err)

	assert.JSONEq(t, `[{
		"id":"0123456789abcdef",
		"difficulty":"MASTER",
		"title":"通常譜面",
		"const":14.5,
		"is_const_unknown":true,
		"jacket":"jacket"
	}]`, string(payload))
}

func TestToCourseTracksDTO_Worldsend候補のメタデータをマッピングする(t *testing.T) {
	attribute := "狂"
	levelStar := 5
	notes := 1234
	tracks := []usecase.CourseTrackOutput{{
		CourseTrack: entity.CourseTrack{Number: 1, SelectionType: entity.CourseTrackRandomPool},
		Candidates: []entity.CourseChartCandidate{{
			CourseChartRef: entity.CourseChartRef{SongID: "fedcba9876543210", Difficulty: entity.CourseTrackDifficultyWorldsend},
			Title:          "候補",
			Jacket:         new("jacket"),
			Attribute:      &attribute,
			LevelStar:      &levelStar,
			Notes:          &notes,
		}},
	}}

	dtoTracks := ToCourseTracksDTO(tracks)
	payload, err := json.Marshal(dtoTracks[0].Candidates)
	require.NoError(t, err)

	assert.JSONEq(t, `[{
		"id":"fedcba9876543210",
		"difficulty":"WORLDSEND",
		"title":"候補",
		"jacket":"jacket",
		"attribute":"狂",
		"level_star":5,
		"notes":1234
	}]`, string(payload))
}
