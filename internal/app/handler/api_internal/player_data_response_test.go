package api_internal

import (
	"encoding/json"
	"testing"

	playerdataresult "github.com/chunisupport/chunisupport-api/internal/usecase/playerdataresult"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToPlayerDataResponse_JSON契約を維持する(t *testing.T) {
	response := toPlayerDataResponse(&playerdataresult.Result{
		PlayerID: 42,
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
	assert.Contains(t, string(encoded), `"changes":[{"record_type":"standard","change_type":"new","idx":"1","before":null,"after":{"score":1000000,"clear_lamp":null,"combo_lamp":null,"full_chain":null}}]`)
	assert.Contains(t, string(encoded), `"skipped_records":[]`)
}
