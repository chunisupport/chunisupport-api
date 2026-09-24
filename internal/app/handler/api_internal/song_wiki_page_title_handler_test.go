package api_internal

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainmasterdata "github.com/chunisupport/chunisupport-api/internal/domain/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/master"
	"github.com/chunisupport/chunisupport-api/internal/infra/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/testutil"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// wikiPageTitleUpdateCases は楽曲更新APIにおける wiki_page_title の受け付けケースです。
// 通常楽曲と WORLD'S END 楽曲で同じ契約を検証するため共通化しています。
var wikiPageTitleUpdateCases = []struct {
	name string
	// Given: リクエストに追加する wiki_page_title のJSON断片（空文字は省略）
	wikiPageTitleJSON string
	// Then: ユースケースへ渡る値、またはエラーコード
	expectedUpdate  bool
	expectedTitle   *string
	expectedErrCode string
}{
	{
		name:              "省略時は既存値維持としてユースケースへ渡す",
		wikiPageTitleJSON: "",
		expectedUpdate:    false,
		expectedTitle:     nil,
	},
	{
		name:              "nullの場合はNULLへの更新としてユースケースへ渡す",
		wikiPageTitleJSON: `,"wiki_page_title":null`,
		expectedUpdate:    true,
		expectedTitle:     nil,
	},
	{
		name:              "文字列の場合はその値への更新としてユースケースへ渡す",
		wikiPageTitleJSON: `,"wiki_page_title":"楽曲(CHUNITHM)"`,
		expectedUpdate:    true,
		expectedTitle:     new("楽曲(CHUNITHM)"),
	},
	{
		name:              "空文字はvalidation_failedを返す（空にする場合はnullを指定する）",
		wikiPageTitleJSON: `,"wiki_page_title":""`,
		expectedErrCode:   apierror.CodeValidationFailed,
	},
	{
		name:              "300文字は受け付ける",
		wikiPageTitleJSON: `,"wiki_page_title":"` + strings.Repeat("あ", 300) + `"`,
		expectedUpdate:    true,
		expectedTitle:     new(strings.Repeat("あ", 300)),
	},
	{
		name:              "301文字はvalidation_failedを返す",
		wikiPageTitleJSON: `,"wiki_page_title":"` + strings.Repeat("あ", 301) + `"`,
		expectedErrCode:   apierror.CodeValidationFailed,
	},
}

func TestSongHandler_UpdateSongs_WikiPageTitle(t *testing.T) {
	for _, tt := range wikiPageTitleUpdateCases {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			var received []*usecase.UpdateSongInput
			handler := NewSongHandler(&testutil.MockSongUsecase{
				UpdateSongsFunc: func(ctx context.Context, requests []*usecase.UpdateSongInput) error {
					received = requests
					return nil
				},
			}, &testutil.MockChartStatsUsecase{}, &masterdata.Cache{}, &masterdata.StaticCache{})
			e := echo.New()
			e.Validator = &testValidator{validator: validator.New()}
			body := `[{"id":"1234567890abcdef","title":"曲","artist":"A"` + tt.wikiPageTitleJSON + `}]`
			req := httptest.NewRequest(http.MethodPut, "/internal/songs", bytes.NewBufferString(body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

			// When
			err := handler.UpdateSongs(e.NewContext(req, httptest.NewRecorder()))

			// Then
			if tt.expectedErrCode != "" {
				var apiErr *apierror.APIError
				require.ErrorAs(t, err, &apiErr)
				assert.Equal(t, tt.expectedErrCode, apiErr.Code)
				assert.Nil(t, received)
				return
			}
			require.NoError(t, err)
			require.Len(t, received, 1)
			assert.Equal(t, tt.expectedUpdate, received[0].UpdateWikiPageTitle)
			assert.Equal(t, tt.expectedTitle, received[0].WikiPageTitle)
		})
	}
}

func TestWorldsendHandler_UpdateWorldsendSongs_WikiPageTitle(t *testing.T) {
	for _, tt := range wikiPageTitleUpdateCases {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			var received []*usecase.UpdateWorldsendSongInput
			handler := NewWorldsendHandler(&testutil.MockWorldsendUsecase{
				UpdateWorldsendSongsFunc: func(ctx context.Context, requests []*usecase.UpdateWorldsendSongInput, masters *domainmasterdata.SongMasters) error {
					received = requests
					return nil
				},
			}, &masterdata.Cache{Genres: map[string]master.Genre{"POPS & ANIME": {ID: 1, Name: "POPS & ANIME"}}})
			e := echo.New()
			e.Validator = &testValidator{validator: validator.New()}
			body := `[{"id":"1234567890abcdef","title":"WE曲","artist":"A","is_new":false` + tt.wikiPageTitleJSON + `}]`
			req := httptest.NewRequest(http.MethodPut, "/internal/worldsend-songs", bytes.NewBufferString(body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

			// When
			err := handler.UpdateWorldsendSongs(e.NewContext(req, httptest.NewRecorder()))

			// Then
			if tt.expectedErrCode != "" {
				var apiErr *apierror.APIError
				require.ErrorAs(t, err, &apiErr)
				assert.Equal(t, tt.expectedErrCode, apiErr.Code)
				assert.Nil(t, received)
				return
			}
			require.NoError(t, err)
			require.Len(t, received, 1)
			assert.Equal(t, tt.expectedUpdate, received[0].UpdateWikiPageTitle)
			assert.Equal(t, tt.expectedTitle, received[0].WikiPageTitle)
		})
	}
}

func TestSongHandler_CreateSong_WikiPageTitle(t *testing.T) {
	// Given
	var received *usecase.CreateSongInput
	handler := NewSongHandler(&testutil.MockSongUsecase{
		CreateSongFunc: func(ctx context.Context, input *usecase.CreateSongInput) (*entity.Song, error) {
			received = input
			return entity.NewSong(), nil
		},
	}, &testutil.MockChartStatsUsecase{}, &masterdata.Cache{}, &masterdata.StaticCache{})
	e := echo.New()
	e.Validator = &testValidator{validator: validator.New()}
	body := `{"official_idx":"1","title":"曲","artist":"A","genre":"POPS & ANIME","wiki_page_title":"曲(CHUNITHM)"}`
	req := httptest.NewRequest(http.MethodPost, "/internal/songs", bytes.NewBufferString(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	// When
	err := handler.CreateSong(e.NewContext(req, httptest.NewRecorder()))

	// Then
	require.NoError(t, err)
	require.NotNil(t, received)
	assert.Equal(t, new("曲(CHUNITHM)"), received.WikiPageTitle)
}

func TestWorldsendHandler_CreateWorldsendSong_WikiPageTitle(t *testing.T) {
	// Given
	var received *usecase.CreateWorldsendSongInput
	handler := NewWorldsendHandler(&testutil.MockWorldsendUsecase{
		CreateWorldsendSongFunc: func(ctx context.Context, input *usecase.CreateWorldsendSongInput, masters *domainmasterdata.SongMasters) (*entity.WorldsendSongWithChart, error) {
			received = input
			return &entity.WorldsendSongWithChart{Song: entity.NewSong()}, nil
		},
	}, &masterdata.Cache{Genres: map[string]master.Genre{"POPS & ANIME": {ID: 1, Name: "POPS & ANIME"}}})
	e := echo.New()
	e.Validator = &testValidator{validator: validator.New()}
	body := `{"official_idx":"1","title":"WE曲","artist":"A","genre":"POPS & ANIME","wiki_page_title":"WE曲(WORLD'S END)"}`
	req := httptest.NewRequest(http.MethodPost, "/internal/worldsend-songs", bytes.NewBufferString(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	// When
	err := handler.CreateWorldsendSong(e.NewContext(req, httptest.NewRecorder()))

	// Then
	require.NoError(t, err)
	require.NotNil(t, received)
	assert.Equal(t, new("WE曲(WORLD'S END)"), received.WikiPageTitle)
}

func TestCreateSongHandlers_WikiPageTitleEmptyStringIsRejected(t *testing.T) {
	tests := []struct {
		name string
		// When: 空文字の wiki_page_title を指定して作成APIを呼び出す
		create func(called *bool) error
	}{
		{
			name: "通常楽曲の作成で空文字はエラーを返す",
			create: func(called *bool) error {
				handler := NewSongHandler(&testutil.MockSongUsecase{
					CreateSongFunc: func(ctx context.Context, input *usecase.CreateSongInput) (*entity.Song, error) {
						*called = true
						return entity.NewSong(), nil
					},
				}, &testutil.MockChartStatsUsecase{}, &masterdata.Cache{}, &masterdata.StaticCache{})
				body := `{"official_idx":"1","title":"曲","artist":"A","genre":"POPS & ANIME","wiki_page_title":""}`
				return handler.CreateSong(newWikiPageTitleCreateContext(body, "/internal/songs"))
			},
		},
		{
			name: "WORLD'S END楽曲の作成で空文字はエラーを返す",
			create: func(called *bool) error {
				handler := NewWorldsendHandler(&testutil.MockWorldsendUsecase{
					CreateWorldsendSongFunc: func(ctx context.Context, input *usecase.CreateWorldsendSongInput, masters *domainmasterdata.SongMasters) (*entity.WorldsendSongWithChart, error) {
						*called = true
						return &entity.WorldsendSongWithChart{Song: entity.NewSong()}, nil
					},
				}, &masterdata.Cache{Genres: map[string]master.Genre{"POPS & ANIME": {ID: 1, Name: "POPS & ANIME"}}})
				body := `{"official_idx":"1","title":"WE曲","artist":"A","genre":"POPS & ANIME","wiki_page_title":""}`
				return handler.CreateWorldsendSong(newWikiPageTitleCreateContext(body, "/internal/worldsend-songs"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			called := false

			// When
			err := tt.create(&called)

			// Then
			assert.Error(t, err)
			assert.False(t, called)
		})
	}
}

// newWikiPageTitleCreateContext は作成APIのテスト用コンテキストを生成します。
func newWikiPageTitleCreateContext(body, path string) *echo.Context {
	e := echo.New()
	e.Validator = &testValidator{validator: validator.New()}
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	return e.NewContext(req, httptest.NewRecorder())
}
