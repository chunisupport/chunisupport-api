package api_internal_test

import (
	"encoding/json"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/app/handler/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/playername/playernametest"
	v1 "github.com/chunisupport/chunisupport-api/internal/dto/api_v1"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlayerProfile_エンブレムのJSON契約(t *testing.T) {
	id, baseID := 42, 17
	emblem, base := "inf", "3"
	tests := []struct {
		name         string
		id, baseID   *int
		emblem, base *string
	}{
		{name: "マスタ名を返す", id: &id, baseID: &baseID, emblem: &emblem, base: &base},
		{name: "未設定はnullを返す"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := &usecase.UserProfileWithRecordsOutput{Player: &usecase.UserPlayerOutput{
				Player:      &entity.Player{Name: playernametest.New(t, "テスト"), ClassEmblemID: tt.id, ClassEmblemBaseID: tt.baseID},
				ClassEmblem: tt.emblem, ClassEmblemBase: tt.base,
			}}
			internal := api_internal.ToUserProfileWithRecordsDTO(output)
			for _, api := range []struct {
				name    string
				value   any
				keepIDs bool
			}{
				{name: "internal", value: internal},
				{name: "v1", value: v1.ToV1UserProfileDTO(internal), keepIDs: true},
			} {
				t.Run(api.name, func(t *testing.T) {
					data, err := json.Marshal(api.value)
					require.NoError(t, err)
					var body struct {
						Player map[string]json.RawMessage `json:"player"`
					}
					require.NoError(t, json.Unmarshal(data, &body))
					for key, expected := range map[string]any{"class_emblem": tt.emblem, "class_emblem_base": tt.base} {
						want, err := json.Marshal(expected)
						require.NoError(t, err)
						assert.JSONEq(t, string(want), string(body.Player[key]))
					}
					for key, expected := range map[string]*int{"class_emblem_id": tt.id, "class_emblem_base_id": tt.baseID} {
						if api.keepIDs {
							want, err := json.Marshal(expected)
							require.NoError(t, err)
							assert.JSONEq(t, string(want), string(body.Player[key]))
						} else {
							assert.NotContains(t, body.Player, key)
						}
					}
				})
			}
		})
	}
}
