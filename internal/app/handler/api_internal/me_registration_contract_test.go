package api_internal_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/app/handler/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/chunisupport/chunisupport-api/internal/usecase/playerdataresult"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type registrationContractUsecase struct {
	usecase.PlayerDataUsecase
	register func(context.Context, *entity.User, *usecase.PlayerDataPayload, string) (*playerdataresult.Result, error)
}

func (s *registrationContractUsecase) Register(ctx context.Context, user *entity.User, payload *usecase.PlayerDataPayload, hash string) (*playerdataresult.Result, error) {
	return s.register(ctx, user, payload, hash)
}

func TestMeHandler_RegisterData_本文とハッシュを渡し受信と解凍後の上限を区別する(t *testing.T) {
	const limit = 5_242_880
	const original = `{"app_ver":"1.2.3","name":"テスト","level":42,"scores":{"standard":[{"idx":"123","diff":"MASTER","score":1009000,"cmb_lv":3}],"worldsend":[{"idx":"456","score":990000}],"course":[{"idx":"789","score":3000000,"is_clear":true,"cmb_lv":2}]}}`
	for _, tt := range []struct {
		name, format          string
		decodedSize, wireSize int
		tooLarge              bool
	}{
		{"生JSONの上限一致", "json", limit, 0, false},
		{"生JSONの上限超過", "json", limit + 1, 0, true},
		{"解凍後の上限一致", "gzip", limit, 0, false},
		{"解凍後の上限超過", "gzip", limit + 1, 0, true},
		{"圧縮形式の受信上限一致", "gzip", len(original), limit, false},
		{"圧縮形式の受信上限超過", "gzip", len(original), limit + 1, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			jsonData := original + strings.Repeat(" ", tt.decodedSize-len(original))
			body := jsonData
			path := "/internal/me/register-data?format=json"
			if tt.format == "gzip" {
				var err error
				body, err = compressAndEncodeGzipBase64([]byte(jsonData))
				require.NoError(t, err)
				path = "/internal/me/register-data"
				if tt.wireSize > 0 {
					body += strings.Repeat("\n", tt.wireSize-len(body))
				}
			}
			user := &entity.User{ID: 42}
			ctx := context.WithValue(context.Background(), registrationRequestKey{}, "request")
			called := false
			uc := &registrationContractUsecase{register: func(gotCtx context.Context, gotUser *entity.User, payload *usecase.PlayerDataPayload, hash string) (*playerdataresult.Result, error) {
				called = true
				assert.Same(t, ctx, gotCtx)
				assert.Same(t, user, gotUser)
				assert.Equal(t, &usecase.PlayerDataPayload{
					AppVersion: "1.2.3", Name: "テスト", Level: 42, Honors: map[string]usecase.PlayerDataHonorPayload{},
					Scores: usecase.PlayerDataScorePayload{
						Standard:  []usecase.PlayerDataScoreEntry{{Idx: "123", Diff: "MASTER", Score: 1009000, ComboLv: new(3)}},
						Worldsend: []usecase.PlayerDataScoreEntry{{Idx: "456", Score: 990000}},
						Course:    []usecase.PlayerDataCourseEntry{{Idx: "789", Score: 3000000, IsClear: true, ComboLv: 2}},
					},
				}, payload)
				sum := sha256.Sum256([]byte(jsonData))
				assert.Equal(t, hex.EncodeToString(sum[:]), hash)
				return &playerdataresult.Result{PlayerID: 7, AppVersion: "1.2.3"}, nil
			}}
			e := echo.New()
			rec := httptest.NewRecorder()
			c := e.NewContext(httptest.NewRequestWithContext(ctx, http.MethodPost, path, strings.NewReader(body)), rec)
			c.Set("userEntity", user)
			err := api_internal.NewMeHandler(uc).RegisterData(c)
			if tt.tooLarge {
				assert.ErrorIs(t, err, apierror.ErrPayloadTooLarge)
				assert.False(t, called)
			} else {
				require.NoError(t, err)
				assert.True(t, called)
				assert.Equal(t, http.StatusOK, rec.Code)
			}
		})
	}
}

type registrationRequestKey struct{}
