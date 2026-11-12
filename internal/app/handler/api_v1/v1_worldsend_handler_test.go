package api_v1

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/infra/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/testutil"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
)

func TestV1WorldsendHandler_GetWorldsendSongRejectsInvalidDisplayID(t *testing.T) {
	e := echo.New()
	called := false
	handler := NewV1WorldsendHandler(&testutil.MockWorldsendUsecase{
		GetWorldsendSongByDisplayIDFunc: func(ctx context.Context, displayID string, requesterAccountTypeID *int) (*entity.WorldsendSongWithChart, error) {
			called = true
			return nil, nil
		},
	}, &masterdata.Cache{})

	req := httptest.NewRequest(http.MethodGet, "/v1/worldsend-songs/invalid", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPathValues(echo.PathValues{{Name: "id", Value: "invalid"}})

	err := handler.GetWorldsendSong(c)

	var apiErr *apierror.APIError
	if assert.ErrorAs(t, err, &apiErr) {
		assert.Equal(t, apierror.CodeValidationFailed, apiErr.Code)
	}
	assert.False(t, called)
}

func TestV1WorldsendHandler_GetWorldsendSongはidパスパラメータを使用する(t *testing.T) {
	// Given: usecaseが変換対象のエンティティを返す
	const id = "0123456789abcdef"
	var actualID string
	handler := NewV1WorldsendHandler(&testutil.MockWorldsendUsecase{
		GetWorldsendSongByDisplayIDFunc: func(ctx context.Context, displayID string, requesterAccountTypeID *int) (*entity.WorldsendSongWithChart, error) {
			actualID = displayID
			return &entity.WorldsendSongWithChart{
				Song: &entity.Song{
					DisplayID:      displayID,
					Title:          "ハンドラー変換テスト",
					NameFolderCode: "NUMBER",
					Artist:         "テストアーティスト",
					OfficialIdx:    "123",
				},
				Chart: nil,
			}, nil
		},
	}, &masterdata.Cache{GenreNamesByID: map[int]string{}})
	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(httptest.NewRequest(http.MethodGet, "/v1/worldsend-songs/"+id, nil), rec)
	c.SetPathValues(echo.PathValues{{Name: "id", Value: id}})

	// When: ハンドラーを実行する
	err := handler.GetWorldsendSong(c)

	// Then: usecaseへIDを渡し、変換結果をそのまま返す
	assert.NoError(t, err)
	assert.Equal(t, id, actualID)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"id":"0123456789abcdef"`)
	assert.Contains(t, rec.Body.String(), `"title":"ハンドラー変換テスト"`)
	assert.Contains(t, rec.Body.String(), `"WORLDSEND"`)
}
