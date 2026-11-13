package usecase

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegister_AJと理論値の整合性(t *testing.T) {
	for _, recordType := range []string{"standard", "worldsend"} {
		t.Run(recordType, func(t *testing.T) {
			for _, tt := range []struct {
				name    string
				score   int
				combo   *int
				invalid bool
			}{
				{"AJの下限直前", 999_999, new(3), true},
				{"AJの下限一致", 1_000_000, new(3), false},
				{"FCでは理論値を取れない", 1_010_000, new(2), true},
				{"ランプ未設定では理論値を取れない", 1_010_000, nil, true},
				{"AJなら理論値を受理する", 1_010_000, new(3), false},
				{"理論値直前ならFCも受理する", 1_009_999, new(2), false},
			} {
				t.Run(tt.name, func(t *testing.T) {
					fixture := newRegistrationFixture(t)
					entry := PlayerDataScoreEntry{Idx: "full-song", Diff: "MAS", Score: tt.score, ComboLv: tt.combo, Slot: new("best"), Order: new(1)}
					if recordType == "standard" {
						fixture.payload.Scores.Standard = []PlayerDataScoreEntry{entry}
					} else {
						entry.Idx = "world-song"
						entry.Slot = nil
						entry.Order = nil
						fixture.payload.Scores.Worldsend = []PlayerDataScoreEntry{entry}
					}
					result, err := fixture.uc.Register(context.Background(), fixture.user, fixture.payload, "score-integrity-hash")
					if tt.invalid {
						var validationErr *PlayerDataValidationError
						require.ErrorAs(t, err, &validationErr)
						assert.Equal(t, "scores", validationErr.Field)
						assert.Nil(t, result)
						assert.False(t, fixture.player.saved, "不整合なスコアは永続化前に拒否する")
					} else {
						require.NoError(t, err)
						require.NotNil(t, result)
						assert.Equal(t, 77, result.PlayerID)
						if recordType == "standard" {
							require.Len(t, fixture.data.savedInput.FullRecordsToInsert, 1)
							assert.Equal(t, tt.score, fixture.data.savedInput.FullRecordsToInsert[0].State.Score)
						} else {
							require.Len(t, fixture.data.savedInput.WorldsendRecordsToInsert, 1)
							assert.Equal(t, tt.score, fixture.data.savedInput.WorldsendRecordsToInsert[0].State.Score)
						}
					}
				})
			}
		})
	}
}
