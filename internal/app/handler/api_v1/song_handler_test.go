package api_v1

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/notes"
	"github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/infra/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/testutil"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testValidator struct {
	validator *validator.Validate
}

func (tv *testValidator) Validate(i any) error {
	return tv.validator.Struct(i)
}

// TestConvertToV1SongDTO はV1SongHandlerのconvertToV1SongDTOメソッドをテストします。
func TestConvertToV1SongDTO(t *testing.T) {
	// マスタデータキャッシュの準備
	masterCache := &masterdata.Cache{
		GenreNamesByID: map[int]string{
			1: "POPS & ANIME",
			2: "niconico",
		},
		DifficultyNamesByID: map[int]string{
			1: "BASIC",
			2: "ADVANCED",
			3: "EXPERT",
			4: "MASTER",
			5: "ULTIMA",
		},
	}

	handler := &V1SongHandler{
		songUsecase: &testutil.MockSongUsecase{},
		masterCache: masterCache,
	}

	// テストデータの準備
	genreID := 2
	bpm := 200
	imgURL := "https://example.com/v1jacket.jpg"

	song := &entity.Song{
		DisplayID:            "v1test1234567890",
		Title:                "V1テスト楽曲",
		Artist:               "V1アーティスト",
		GenreID:              &genreID,
		BPM:                  &bpm,
		Jacket:               &imgURL,
		IsMaxOPUnknown:       true,
		OpTargetDifficultyID: 4,
	}

	notes1Value := 600
	notes2Value := 1200
	notes1, err := notes.NewNotes(notes1Value)
	if err != nil {
		t.Fatalf("notes.NewNotes failed for notes1Value: %v", err)
	}
	notes2, err := notes.NewNotes(notes2Value)
	if err != nil {
		t.Fatalf("notes.NewNotes failed for notes2Value: %v", err)
	}

	charts := []*entity.Chart{
		{
			DifficultyID:   2, // advanced
			Const:          9.0,
			IsConstUnknown: false,
			Notes:          &notes1,
			NotesDesigner:  stringPtr("譜面作者A"),
		},
		{
			DifficultyID:   4, // master
			Const:          13.7,
			IsConstUnknown: false,
			Notes:          &notes2,
			NotesDesigner:  stringPtr("譜面作者B"),
		},
	}

	song.Charts = charts

	// 変換実行
	dto := handler.convertToV1SongDTO(song)

	// アサーション
	if dto == nil {
		t.Fatal("convertToV1SongDTO returned nil")
	}

	if dto.DisplayID != "v1test1234567890" {
		t.Errorf("DisplayID = %v, want %v", dto.DisplayID, "v1test1234567890")
	}

	if dto.MaxOP != 90 {
		t.Errorf("MaxOP = %v, want %v", dto.MaxOP, 90)
	}

	// IsMaxOPUnknown が反映されていることを確認
	if !dto.IsMaxOPUnknown {
		t.Errorf("IsMaxOPUnknown = %v, want %v", dto.IsMaxOPUnknown, true)
	}

	if dto.OpTargetDifficulty == nil || *dto.OpTargetDifficulty != "MASTER" {
		t.Errorf("OpTargetDifficulty = %v, want %v", dto.OpTargetDifficulty, "MASTER")
	}

	// Charts マップのキーが存在するか確認
	if dto.Charts == nil {
		t.Fatal("Charts is nil")
	}

	// advanced 譜面が存在することを確認
	if advancedChart, ok := dto.Charts["ADVANCED"]; !ok || advancedChart == nil {
		t.Error("ADVANCED chart not found")
	} else {
		if advancedChart.Const != 9.0 {
			t.Errorf("ADVANCED chart Const = %v, want %v", advancedChart.Const, 9.0)
		}
		if advancedChart.NotesDesigner == nil || *advancedChart.NotesDesigner != "譜面作者A" {
			t.Errorf("ADVANCED chart NotesDesigner = %v, want %v", advancedChart.NotesDesigner, "譜面作者A")
		}
	}

	// master 譜面が存在することを確認
	if masterChart, ok := dto.Charts["MASTER"]; !ok || masterChart == nil {
		t.Error("MASTER chart not found")
	} else {
		if masterChart.Const != 13.7 {
			t.Errorf("MASTER chart Const = %v, want %v", masterChart.Const, 13.7)
		}
		if masterChart.NotesDesigner == nil || *masterChart.NotesDesigner != "譜面作者B" {
			t.Errorf("MASTER chart NotesDesigner = %v, want %v", masterChart.NotesDesigner, "譜面作者B")
		}
	}

	// basic 譜面は存在しないので nil であることを確認
	if basicChart, ok := dto.Charts["BASIC"]; !ok {
		t.Error("BASIC key not found in map")
	} else if basicChart != nil {
		t.Error("BASIC chart should be nil")
	}

	// expert 譜面は存在しないので nil であることを確認
	if expertChart, ok := dto.Charts["EXPERT"]; !ok {
		t.Error("EXPERT key not found in map")
	} else if expertChart != nil {
		t.Error("EXPERT chart should be nil")
	}

	// ultima 譜面は存在しないので nil であることを確認
	if ultimaChart, ok := dto.Charts["ULTIMA"]; !ok {
		t.Error("ULTIMA key not found in map")
	} else if ultimaChart != nil {
		t.Error("ULTIMA chart should be nil")
	}
}

func TestV1SongHandler_UpdateSongs(t *testing.T) {
	e := echo.New()
	e.Validator = &testValidator{validator: validator.New()}

	newContext := func(body string) echo.Context {
		req := httptest.NewRequest(http.MethodPut, "/v1/songs", bytes.NewBufferString(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		return e.NewContext(req, rec)
	}

	tests := []struct {
		name             string
		body             string
		wantStatus       int
		wantErrCode      string
		wantUsecaseCall  bool
		assertUsecaseReq func(t *testing.T, requests []*api_internal.UpdateSongRequest)
	}{
		{
			name:            "正常な配列で204を返す",
			body:            `[{"id":"1234567890abcdef","title":"テスト楽曲","artist":"テストアーティスト","charts":{"MASTER":{"const":14.5,"is_const_unknown":false,"notes":1234,"notes_designer":"譜面作者A"}}}]`,
			wantStatus:      http.StatusNoContent,
			wantUsecaseCall: true,
			assertUsecaseReq: func(t *testing.T, requests []*api_internal.UpdateSongRequest) {
				t.Helper()
				require.Len(t, requests, 1)
				assert.Equal(t, "1234567890abcdef", requests[0].DisplayID)
				require.Contains(t, requests[0].Charts, "MASTER")
				assert.InDelta(t, 14.5, requests[0].Charts["MASTER"].Const, 0.0001)
			},
		},
		{
			name:        "トップレベルnullはvalidation_failedを返す",
			body:        `null`,
			wantErrCode: apierror.CodeValidationFailed,
		},
		{
			name:        "不正なdisplay_idはvalidation_failedを返す",
			body:        `[{"id":"short","title":"テスト楽曲","artist":"テストアーティスト"}]`,
			wantErrCode: apierror.CodeValidationFailed,
		},
		{
			name:        "chartsのnull要素はvalidation_failedを返す",
			body:        `[{"id":"1234567890abcdef","title":"テスト楽曲","artist":"テストアーティスト","charts":{"MASTER":null}}]`,
			wantErrCode: apierror.CodeValidationFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			usecaseCalled := false
			handler := NewV1SongHandler(&testutil.MockSongUsecase{
				UpdateSongsFunc: func(ctx context.Context, requests []*api_internal.UpdateSongRequest) error {
					usecaseCalled = true
					if tt.assertUsecaseReq != nil {
						tt.assertUsecaseReq(t, requests)
					}
					return nil
				},
			}, &testutil.MockChartStatsUsecase{}, &masterdata.Cache{}, &masterdata.StaticCache{})

			c := newContext(tt.body)
			rec := c.Response().Writer.(*httptest.ResponseRecorder)

			err := handler.UpdateSongs(c)

			if tt.wantErrCode == "" {
				require.NoError(t, err)
				assert.Equal(t, tt.wantStatus, rec.Code)
			} else {
				var apiErr *apierror.APIError
				require.ErrorAs(t, err, &apiErr)
				assert.Equal(t, tt.wantErrCode, apiErr.Code)
			}
			assert.Equal(t, tt.wantUsecaseCall, usecaseCalled)
		})
	}
}

func stringPtr(value string) *string {
	return &value
}
