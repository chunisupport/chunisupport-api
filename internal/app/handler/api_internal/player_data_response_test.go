package api_internal

import (
	"encoding/json"
	"testing"

	playerdataresult "github.com/chunisupport/chunisupport-api/internal/usecase/playerdataresult"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToPlayerDataResponse_JSON契約を維持する(t *testing.T) {
	ratingBefore := 16.42
	ratingAfter := 16.45
	ratingDelta := 0.03
	response := toPlayerDataResponse(&playerdataresult.Result{
		PlayerID: 42,
		MetricDiffs: playerdataresult.MetricDiffs{
			Rating: playerdataresult.Float64Diff{
				Before: &ratingBefore,
				After:  &ratingAfter,
				Delta:  &ratingDelta,
			},
		},
		Changes: []playerdataresult.RecordChange{{
			RecordType: "standard",
			ChangeType: "new",
			Idx:        "1",
			After:      playerdataresult.RecordState{Score: 1_000_000},
		}},
		SkippedRecords: []playerdataresult.SkippedRecord{},
	})

	encoded, err := json.Marshal(response)
	require.NoError(t, err)
	assert.Contains(t, string(encoded), `"player_id":42`)
	assert.Contains(t, string(encoded), `"metric_diffs":{"rating":{"before":16.42,"after":16.45,"delta":0.03},"overpower_value":{"before":null,"after":null,"delta":null}}`)
	assert.Contains(t, string(encoded), `"changes":[{"record_type":"standard","change_type":"new","idx":"1","before":null,"after":{"score":1000000,"clear_lamp":null,"combo_lamp":null,"full_chain":null}}]`)
	assert.Contains(t, string(encoded), `"skipped_records":[]`)
}
