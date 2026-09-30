package api_internal

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

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

// unlockRequiredUpdateCases は楽曲更新APIにおける unlock_required の受け付けケースです。
// 通常楽曲と WORLD'S END 楽曲で同じ契約を検証するため共通化しています。
var unlockRequiredUpdateCases = []struct {
	name string
	// Given: リクエストに追加する unlock_required のJSON断片（空文字は省略）
	unlockRequiredJSON string
	// Then: ユースケースへ渡る値
	expected *bool
}{
	{name: "省略時は既存値維持としてユースケースへ渡す", unlockRequiredJSON: "", expected: nil},
	{name: "nullの場合は既存値維持としてユースケースへ渡す", unlockRequiredJSON: `,"unlock_required":null`, expected: nil},
	{name: "trueの場合は要解禁への更新としてユースケースへ渡す", unlockRequiredJSON: `,"unlock_required":true`, expected: new(true)},
	{name: "falseの場合は解禁不要への更新としてユースケースへ渡す", unlockRequiredJSON: `,"unlock_required":false`, expected: new(false)},
}

// newUnlockRequiredTestContext は要解禁フラグ検証用のリクエストコンテキストを生成します。
func newUnlockRequiredTestContext(method, path, body string) *echo.Context {
	e := echo.New()
	e.Validator = &testValidator{validator: validator.New()}
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	return e.NewContext(req, httptest.NewRecorder())
}

func TestSongHandler_UpdateSongs_UnlockRequired(t *testing.T) {
	for _, tt := range unlockRequiredUpdateCases {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			var received []*usecase.UpdateSongInput
			handler := NewSongHandler(&testutil.MockSongUsecase{
				UpdateSongsFunc: func(ctx context.Context, requests []*usecase.UpdateSongInput) error {
					received = requests
					return nil
				},
			}, &testutil.MockChartStatsUsecase{}, &masterdata.Cache{}, &masterdata.StaticCache{})
			body := `[{"id":"1234567890abcdef","title":"曲","artist":"A"` + tt.unlockRequiredJSON + `}]`

			// When
			err := handler.UpdateSongs(newUnlockRequiredTestContext(http.MethodPut, "/internal/songs", body))

			// Then
			require.NoError(t, err)
			require.Len(t, received, 1)
			assert.Equal(t, tt.expected, received[0].UnlockRequired)
		})
	}
}

func TestWorldsendHandler_UpdateWorldsendSongs_UnlockRequired(t *testing.T) {
	for _, tt := range unlockRequiredUpdateCases {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			var received []*usecase.UpdateWorldsendSongInput
			handler := NewWorldsendHandler(&testutil.MockWorldsendUsecase{
				UpdateWorldsendSongsFunc: func(ctx context.Context, requests []*usecase.UpdateWorldsendSongInput, masters *domainmasterdata.SongMasters) error {
					received = requests
					return nil
				},
			}, &masterdata.Cache{Genres: map[string]master.Genre{"POPS & ANIME": {ID: 1, Name: "POPS & ANIME"}}})
			body := `[{"id":"1234567890abcdef","title":"WE曲","artist":"A","is_new":false` + tt.unlockRequiredJSON + `}]`

			// When
			err := handler.UpdateWorldsendSongs(newUnlockRequiredTestContext(http.MethodPut, "/internal/worldsend-songs", body))

			// Then
			require.NoError(t, err)
			require.Len(t, received, 1)
			assert.Equal(t, tt.expected, received[0].UnlockRequired)
		})
	}
}

func TestSongHandler_CreateSong_UnlockRequired(t *testing.T) {
	// Given
	var received *usecase.CreateSongInput
	handler := NewSongHandler(&testutil.MockSongUsecase{
		CreateSongFunc: func(ctx context.Context, input *usecase.CreateSongInput) (*entity.Song, error) {
			received = input
			return entity.NewSong(), nil
		},
	}, &testutil.MockChartStatsUsecase{}, &masterdata.Cache{}, &masterdata.StaticCache{})
	body := `{"official_idx":"1","title":"曲","artist":"A","genre":"POPS & ANIME","unlock_required":true}`

	// When
	err := handler.CreateSong(newUnlockRequiredTestContext(http.MethodPost, "/internal/songs", body))

	// Then
	require.NoError(t, err)
	require.NotNil(t, received)
	assert.True(t, received.UnlockRequired)
}

func TestWorldsendHandler_CreateWorldsendSong_UnlockRequired(t *testing.T) {
	// Given
	var received *usecase.CreateWorldsendSongInput
	handler := NewWorldsendHandler(&testutil.MockWorldsendUsecase{
		CreateWorldsendSongFunc: func(ctx context.Context, input *usecase.CreateWorldsendSongInput, masters *domainmasterdata.SongMasters) (*entity.WorldsendSongWithChart, error) {
			received = input
			return &entity.WorldsendSongWithChart{Song: entity.NewSong()}, nil
		},
	}, &masterdata.Cache{Genres: map[string]master.Genre{"POPS & ANIME": {ID: 1, Name: "POPS & ANIME"}}})
	body := `{"official_idx":"1","title":"WE曲","artist":"A","genre":"POPS & ANIME","unlock_required":true}`

	// When
	err := handler.CreateWorldsendSong(newUnlockRequiredTestContext(http.MethodPost, "/internal/worldsend-songs", body))

	// Then
	require.NoError(t, err)
	require.NotNil(t, received)
	assert.True(t, received.UnlockRequired)
}
