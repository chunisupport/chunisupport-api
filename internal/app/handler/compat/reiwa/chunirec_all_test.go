package reiwa

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/notes"
	"github.com/chunisupport/chunisupport-api/internal/infra/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/testutil"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToChunirecAllResponse_契約(t *testing.T) {
	constant, err := chartconstant.NewChartConstant(14.7)
	require.NoError(t, err)
	n, err := notes.NewNotes(1000)
	require.NoError(t, err)
	songs := []*entity.Song{{DisplayID: "normal", Title: "同名曲", Artist: "作者", GenreID: new(1), OfficialIdx: "003", Jacket: new("jacket"), BPM: new(180), ReleasedAt: new(time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)), Charts: []*entity.Chart{{DifficultyID: 4, Const: constant, Notes: &n, IsConstUnknown: true}}}}
	we := []*entity.WorldsendSongWithChart{{Song: &entity.Song{DisplayID: "we", Title: "同名曲", OfficialIdx: "999", Jacket: new("omit")}, Chart: &entity.WorldsendChart{Attribute: new("狂"), Notes: &n}}}
	response := ToChunirecAllResponse(songs, we, &masterdata.Cache{GenreNamesByID: map[int]string{1: "POPS & ANIME"}})
	body, err := json.Marshal(response)
	require.NoError(t, err)
	assert.JSONEq(t, `[{"meta":{"id":"normal","title":"同名曲","genre":"POPS&ANIME","artist":"作者","release":"2026-10-01","bpm":180,"img":"jacket","idx":"003"},"data":{"MAS":{"level":14.5,"const":14.7,"maxcombo":1000,"is_const_unknown":1}}},{"meta":{"id":"we","title":"同名曲【狂】","genre":"WORLD'S END","artist":"","release":"","bpm":0},"data":{"WE":{"level":0,"const":0,"maxcombo":1000,"is_const_unknown":0}}}]`, string(body))
}

func TestToChunirecAllResponse_欠損と削除済み(t *testing.T) {
	c, err := chartconstant.NewChartConstant(4)
	require.NoError(t, err)
	response := ToChunirecAllResponse([]*entity.Song{{DisplayID: "n", Charts: []*entity.Chart{{DifficultyID: 1, Const: c}}}, {DisplayID: "deleted", IsDeleted: true}}, []*entity.WorldsendSongWithChart{{Song: &entity.Song{DisplayID: "w"}, Chart: &entity.WorldsendChart{}}, {Song: &entity.Song{DisplayID: "deleted-we", IsDeleted: true}, Chart: &entity.WorldsendChart{}}}, nil)
	body, err := json.Marshal(response)
	require.NoError(t, err)
	assert.JSONEq(t, `[{"meta":{"id":"n","title":"","artist":"","genre":"","release":"","bpm":0,"img":"","idx":""},"data":{"BAS":{"level":4,"const":4,"maxcombo":0,"is_const_unknown":0}}},{"meta":{"id":"w","title":"","artist":"","genre":"WORLD'S END","release":"","bpm":0},"data":{"WE":{"level":0,"const":0,"maxcombo":0,"is_const_unknown":0}}}]`, string(body))
	empty, err := json.Marshal(ToChunirecAllResponse(nil, nil, nil))
	require.NoError(t, err)
	assert.JSONEq(t, `[]`, string(empty))
}

func TestToChunirecAllResponse_並び順(t *testing.T) {
	songs := []*entity.Song{{DisplayID: "invalid-b", OfficialIdx: "x"}, {DisplayID: "ten", OfficialIdx: "10"}, {DisplayID: "two-b", OfficialIdx: "2"}, {DisplayID: "two-a", OfficialIdx: "02"}, {DisplayID: "invalid-a"}}
	we := []*entity.WorldsendSongWithChart{{Song: &entity.Song{DisplayID: "we-b"}, Chart: &entity.WorldsendChart{}}, {Song: &entity.Song{DisplayID: "we-a"}, Chart: &entity.WorldsendChart{}}}
	response := ToChunirecAllResponse(songs, we, nil)
	ids := make([]string, 0, len(response))
	for _, item := range response {
		ids = append(ids, item.Meta.ID)
	}
	assert.Equal(t, []string{"two-a", "two-b", "ten", "invalid-a", "invalid-b", "we-a", "we-b"}, ids)
	assert.Equal(t, "invalid-b", songs[0].DisplayID)
}

func TestReiwaHandler_GetChunirecAll(t *testing.T) {
	tests := []struct {
		name           string
		songErr, weErr error
		status         int
	}{
		{name: "成功", status: 200},
		{name: "通常曲取得失敗", songErr: errors.New("db"), status: 503},
		{name: "WE取得失敗", weErr: errors.New("db"), status: 503},
		{name: "通常曲取得中の切断", songErr: context.Canceled, status: 499},
		{name: "WE取得中の切断", weErr: context.Canceled, status: 499},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			songCalls, weCalls := 0, 0
			songs := &testutil.MockSongUsecase{GetAllSongsExcludingWorldsendFunc: func(_ context.Context, deleted bool, role *int) ([]*entity.Song, error) {
				songCalls++
				assert.False(t, deleted)
				assert.Nil(t, role)
				return []*entity.Song{{DisplayID: "n"}}, tt.songErr
			}}
			we := &testutil.MockWorldsendUsecase{GetAllWorldsendSongsFunc: func(_ context.Context, deleted bool, role *int) ([]*entity.WorldsendSongWithChart, error) {
				weCalls++
				assert.False(t, deleted)
				assert.Nil(t, role)
				return nil, tt.weErr
			}}
			h := NewReiwaHandler(songs, we, nil, nil)
			e := echo.New()
			e.GET("/chunirec_all.json", h.GetChunirecAll, ReiwaErrorHandlerMiddleware())
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/chunirec_all.json", nil))
			assert.Equal(t, tt.status, rec.Code)
			assert.Equal(t, 1, songCalls)
			if tt.songErr == nil {
				assert.Equal(t, 1, weCalls)
			} else {
				assert.Zero(t, weCalls)
			}
			if tt.status == 200 {
				assert.Contains(t, rec.Body.String(), `"id":"n"`)
			}
			if tt.status == 499 {
				assert.Empty(t, rec.Body.String())
			}
		})
	}
}

func TestToChunirecAllResponse_全難易度と空属性(t *testing.T) {
	c, err := chartconstant.NewChartConstant(4)
	require.NoError(t, err)
	charts := make([]*entity.Chart, 0, 5)
	for id := 1; id <= 5; id++ {
		charts = append(charts, &entity.Chart{DifficultyID: id, Const: c})
	}
	response := ToChunirecAllResponse([]*entity.Song{{Charts: charts}}, []*entity.WorldsendSongWithChart{{Song: &entity.Song{Title: "楽曲"}, Chart: &entity.WorldsendChart{Attribute: new("")}}}, nil)
	assert.Len(t, response[0].Data, 5)
	for _, diff := range []string{"BAS", "ADV", "EXP", "MAS", "ULT"} {
		assert.Contains(t, response[0].Data, diff)
	}
	assert.Equal(t, "楽曲", response[1].Meta.Title)
}

func TestToChunirecAllResponse_ジャケットの拡張子を除く(t *testing.T) {
	for _, tt := range []struct{ jacket, want string }{{"image.png", "image"}, {"image.JPG", "image"}, {"identifier", "identifier"}, {"", ""}} {
		t.Run(tt.jacket, func(t *testing.T) {
			response := ToChunirecAllResponse([]*entity.Song{{Jacket: &tt.jacket}}, nil, nil)
			require.NotNil(t, response[0].Meta.Img)
			assert.Equal(t, tt.want, *response[0].Meta.Img)
		})
	}
}
