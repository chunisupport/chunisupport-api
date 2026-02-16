package usecase

import (
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
)

func TestValidateUpdateSongRequests(t *testing.T) {
	testCases := []struct {
		name      string
		requests  []*api_internal.UpdateSongRequest
		wantError bool
		wantField string
	}{
		{
			name:      "トップレベルnullはエラー",
			requests:  nil,
			wantError: true,
			wantField: "requests",
		},
		{
			name: "要素nullはエラー",
			requests: []*api_internal.UpdateSongRequest{
				nil,
			},
			wantError: true,
			wantField: "requests[0]",
		},
		{
			name: "chartsのnull要素はエラー",
			requests: []*api_internal.UpdateSongRequest{
				{
					DisplayID: "1234567890123456",
					Title:     "テスト楽曲",
					Artist:    "テストアーティスト",
					Charts: []*api_internal.UpdateChartRequest{
						nil,
					},
				},
			},
			wantError: true,
			wantField: "requests[0].charts[0]",
		},
		{
			name:     "空配列は許可",
			requests: []*api_internal.UpdateSongRequest{},
		},
		{
			name: "正常な配列は許可",
			requests: []*api_internal.UpdateSongRequest{
				{
					DisplayID: "1234567890123456",
					Title:     "テスト楽曲",
					Artist:    "テストアーティスト",
					Charts: []*api_internal.UpdateChartRequest{
						{DifficultyID: 1, Const: 5.0},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateUpdateSongRequests(tc.requests)
			if tc.wantError {
				if err == nil {
					t.Fatal("validateUpdateSongRequests should return error")
				}

				validationErr, ok := err.(*SongValidationError)
				if !ok {
					t.Fatalf("error type = %T, want *SongValidationError", err)
				}
				if validationErr.Field != tc.wantField {
					t.Fatalf("validationErr.Field = %s, want %s", validationErr.Field, tc.wantField)
				}
				return
			}

			if err != nil {
				t.Fatalf("validateUpdateSongRequests returned error: %v", err)
			}
		})
	}
}
