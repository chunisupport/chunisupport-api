package api_internal

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockPlayerLockedSongUsecase struct {
	listFunc   func(ctx context.Context, username string, requester *entity.User) ([]*usecase.PlayerLockedSongOutput, error)
	unlockFunc func(ctx context.Context, userID int, input *usecase.PlayerLockedSongInput) error
	batchFunc  func(ctx context.Context, userID int, input *usecase.PlayerLockedSongBatchInput) error
}

func (m *mockPlayerLockedSongUsecase) List(ctx context.Context, username string, requester *entity.User) ([]*usecase.PlayerLockedSongOutput, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, username, requester)
	}
	return nil, nil
}

func (m *mockPlayerLockedSongUsecase) Lock(ctx context.Context, userID int, input *usecase.PlayerLockedSongInput) error {
	return nil
}

func (m *mockPlayerLockedSongUsecase) Unlock(ctx context.Context, userID int, input *usecase.PlayerLockedSongInput) error {
	if m.unlockFunc != nil {
		return m.unlockFunc(ctx, userID, input)
	}
	return nil
}

func (m *mockPlayerLockedSongUsecase) Batch(ctx context.Context, userID int, input *usecase.PlayerLockedSongBatchInput) error {
	if m.batchFunc != nil {
		return m.batchFunc(ctx, userID, input)
	}
	return nil
}

func TestPlayerLockedSongHandler_List(t *testing.T) {
	t.Run("有効なusernameで一覧を返す", func(t *testing.T) {
		// Given
		e := echo.New()
		requester := &entity.User{ID: 1}
		handler := NewPlayerLockedSongHandler(&mockPlayerLockedSongUsecase{
			listFunc: func(ctx context.Context, username string, gotRequester *entity.User) ([]*usecase.PlayerLockedSongOutput, error) {
				assert.Equal(t, "testuser", username)
				assert.Same(t, requester, gotRequester)
				return []*usecase.PlayerLockedSongOutput{
					{DisplayID: "1234567890abcdef", Title: "テスト楽曲", IsUltima: true},
				}, nil
			},
		})
		req := httptest.NewRequest(http.MethodGet, "/internal/users/testuser/locked-songs", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("userEntity", requester)
		c.SetPathValues(echo.PathValues{{Name: "username", Value: "testuser"}})

		// When
		err := handler.List(c)

		// Then
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.JSONEq(t, `{"items":[{"id":"1234567890abcdef","title":"テスト楽曲","is_ultima":true}]}`, rec.Body.String())
	})

	t.Run("不正なusernameは境界で拒否する", func(t *testing.T) {
		// Given
		e := echo.New()
		called := false
		handler := NewPlayerLockedSongHandler(&mockPlayerLockedSongUsecase{
			listFunc: func(ctx context.Context, username string, gotRequester *entity.User) ([]*usecase.PlayerLockedSongOutput, error) {
				called = true
				return nil, nil
			},
		})
		req := httptest.NewRequest(http.MethodGet, "/internal/users/InvalidUser/locked-songs", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPathValues(echo.PathValues{{Name: "username", Value: "InvalidUser"}})

		// When
		err := handler.List(c)

		// Then
		var apiErr *apierror.APIError
		if assert.ErrorAs(t, err, &apiErr) {
			assert.Equal(t, apierror.CodeUsernameInvalidChar, apiErr.Code)
		}
		assert.False(t, called)
	})
}

func TestPlayerLockedSongHandler_Unlock(t *testing.T) {
	tests := []struct {
		name             string
		displayID        string
		query            string
		expectedErrCode  string
		expectedIsUltima bool
		expectUsecaseHit bool
	}{
		{
			name:             "有効なDisplayIDとis_ultima=trueで解除できる",
			displayID:        "1234567890abcdef",
			query:            "?is_ultima=true",
			expectedIsUltima: true,
			expectUsecaseHit: true,
		},
		{
			name:             "is_ultima未指定の場合はfalseで解除できる",
			displayID:        "1234567890abcdef",
			expectedIsUltima: false,
			expectUsecaseHit: true,
		},
		{
			name:            "DisplayIDの形式が不正な場合はvalidation_failed",
			displayID:       "1234567890ABCDEF",
			expectedErrCode: apierror.CodeValidationFailed,
		},
		{
			name:            "is_ultimaがboolでない場合はbad_request",
			displayID:       "1234567890abcdef",
			query:           "?is_ultima=invalid",
			expectedErrCode: apierror.CodeBadRequest,
		},
		{
			name:            "is_ultimaが空の場合はbad_request",
			displayID:       "1234567890abcdef",
			query:           "?is_ultima=",
			expectedErrCode: apierror.CodeBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			e.Validator = &testValidator{validator: validator.New()}

			called := false
			handler := NewPlayerLockedSongHandler(&mockPlayerLockedSongUsecase{
				unlockFunc: func(ctx context.Context, userID int, input *usecase.PlayerLockedSongInput) error {
					called = true
					assert.Equal(t, 1, userID)
					require.NotNil(t, input)
					assert.Equal(t, tt.displayID, input.DisplayID.String())
					assert.Equal(t, tt.expectedIsUltima, input.IsUltima)
					return nil
				},
			})

			req := httptest.NewRequest(http.MethodDelete, "/internal/player/locked-songs/"+tt.displayID+tt.query, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.Set("userEntity", &entity.User{ID: 1})
			c.SetPathValues(echo.PathValues{{Name: "id", Value: tt.displayID}})

			err := handler.Unlock(c)

			if tt.expectedErrCode == "" {
				require.NoError(t, err)
				assert.Equal(t, http.StatusNoContent, rec.Code)
			} else {
				require.Error(t, err)
				apiErr, ok := err.(*apierror.APIError)
				require.True(t, ok)
				assert.Equal(t, tt.expectedErrCode, apiErr.Code)
			}
			assert.Equal(t, tt.expectUsecaseHit, called)
		})
	}
}

func TestPlayerLockedSongHandler_Batch(t *testing.T) {
	e := echo.New()
	e.Validator = &testValidator{validator: validator.New()}
	body := `{"add":[{"id":"1234567890abcdef","is_ultima":true}],"delete":[{"id":"fedcba0987654321","is_ultima":false}]}`
	req := httptest.NewRequest(http.MethodPost, "/internal/me/locked-songs/batch", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("userEntity", &entity.User{ID: 1})

	called := false
	handler := NewPlayerLockedSongHandler(&mockPlayerLockedSongUsecase{
		batchFunc: func(ctx context.Context, userID int, input *usecase.PlayerLockedSongBatchInput) error {
			called = true
			assert.Equal(t, 1, userID)
			require.Len(t, input.Add, 1)
			assert.Equal(t, "1234567890abcdef", input.Add[0].DisplayID.String())
			assert.True(t, input.Add[0].IsUltima)
			require.Len(t, input.Delete, 1)
			assert.Equal(t, "fedcba0987654321", input.Delete[0].DisplayID.String())
			assert.False(t, input.Delete[0].IsUltima)
			return nil
		},
	})

	err := handler.Batch(c)

	require.NoError(t, err)
	assert.True(t, called)
	assert.Equal(t, http.StatusNoContent, rec.Code)
}

// buildPlayerLockedSongBatchBody は指定したID一覧から一括更新のリクエストボディを組み立てる。
// 既存の各1件のケースと同じく、生JSON文字列を application/json で送信する形に揃えている。
func buildPlayerLockedSongBatchBody(addIDs []string, deleteIDs []string) string {
	var sb strings.Builder
	sb.WriteString(`{"add":[`)
	for i, id := range addIDs {
		if i > 0 {
			sb.WriteString(",")
		}
		fmt.Fprintf(&sb, `{"id":%q,"is_ultima":false}`, id)
	}
	sb.WriteString(`],"delete":[`)
	for i, id := range deleteIDs {
		if i > 0 {
			sb.WriteString(",")
		}
		fmt.Fprintf(&sb, `{"id":%q,"is_ultima":false}`, id)
	}
	sb.WriteString(`]}`)
	return sb.String()
}

// newPlayerLockedSongDisplayIDs は形式検証を通る有効なIDを指定件数だけ生成する。
// addとdeleteで値域を分けて重複しないようにしている。値は小文字16進16文字の固定値である。
func newPlayerLockedSongDisplayIDs(count int, offset int) []string {
	ids := make([]string, 0, count)
	for i := 0; i < count; i++ {
		ids = append(ids, fmt.Sprintf("%016x", offset+i+1))
	}
	return ids
}

func TestPlayerLockedSongHandler_Batch_TotalLimit(t *testing.T) {
	// addとdeleteの合計件数の上限（100件）の境界を検証する。
	// 内訳は片方だけでは上限に届かない件数にしており、addだけ・deleteだけを数える誤りでは
	// 合計101件のケースが受理されてしまうため検知できる。また上限ちょうどの受理により
	// 比較の1ずれ（>=で拒否する誤り）も検知できる。
	tests := []struct {
		name             string
		addCount         int
		deleteCount      int
		expectedErrCode  string
		expectUsecaseHit bool
	}{
		{
			name:             "合計100件（add60件+delete40件）は受理されusecaseにそのまま渡る",
			addCount:         60,
			deleteCount:      40,
			expectUsecaseHit: true,
		},
		{
			name:            "合計101件（add60件+delete41件）はvalidation_failedでusecaseを呼ばない",
			addCount:        60,
			deleteCount:     41,
			expectedErrCode: apierror.CodeValidationFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			e := echo.New()
			e.Validator = &testValidator{validator: validator.New()}
			addIDs := newPlayerLockedSongDisplayIDs(tt.addCount, 0)
			deleteIDs := newPlayerLockedSongDisplayIDs(tt.deleteCount, 10000)
			body := buildPlayerLockedSongBatchBody(addIDs, deleteIDs)
			req := httptest.NewRequest(http.MethodPost, "/internal/me/locked-songs/batch", strings.NewReader(body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.Set("userEntity", &entity.User{ID: 1})

			called := false
			handler := NewPlayerLockedSongHandler(&mockPlayerLockedSongUsecase{
				batchFunc: func(ctx context.Context, userID int, input *usecase.PlayerLockedSongBatchInput) error {
					called = true
					assert.Equal(t, 1, userID)
					require.Len(t, input.Add, tt.addCount)
					for i, in := range input.Add {
						require.NotNil(t, in)
						assert.Equal(t, addIDs[i], in.DisplayID.String())
						assert.False(t, in.IsUltima)
					}
					require.Len(t, input.Delete, tt.deleteCount)
					for i, in := range input.Delete {
						require.NotNil(t, in)
						assert.Equal(t, deleteIDs[i], in.DisplayID.String())
						assert.False(t, in.IsUltima)
					}
					return nil
				},
			})

			// When
			err := handler.Batch(c)

			// Then
			if tt.expectedErrCode == "" {
				require.NoError(t, err)
				assert.Equal(t, http.StatusNoContent, rec.Code)
			} else {
				require.Error(t, err)
				apiErr, ok := err.(*apierror.APIError)
				require.True(t, ok)
				assert.Equal(t, tt.expectedErrCode, apiErr.Code)
				assert.Equal(t, http.StatusUnprocessableEntity, apiErr.HTTPStatus)
			}
			assert.Equal(t, tt.expectUsecaseHit, called)
		})
	}
}

func TestPlayerLockedSongHandler_Batch_NullItem(t *testing.T) {
	// add・deleteの配列要素にnullが含まれる場合の拒否を検証する。
	tests := []struct {
		name string
		body string
	}{
		{
			name: "addにnullを含む場合はvalidation_failedでusecaseを呼ばない",
			body: `{"add":[null],"delete":[]}`,
		},
		{
			name: "deleteにnullを含む場合はvalidation_failedでusecaseを呼ばない",
			body: `{"add":[],"delete":[null]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			e := echo.New()
			e.Validator = &testValidator{validator: validator.New()}
			req := httptest.NewRequest(http.MethodPost, "/internal/me/locked-songs/batch", strings.NewReader(tt.body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.Set("userEntity", &entity.User{ID: 1})

			called := false
			handler := NewPlayerLockedSongHandler(&mockPlayerLockedSongUsecase{
				batchFunc: func(ctx context.Context, userID int, input *usecase.PlayerLockedSongBatchInput) error {
					called = true
					return nil
				},
			})

			// When
			err := handler.Batch(c)

			// Then
			require.Error(t, err)
			apiErr, ok := err.(*apierror.APIError)
			require.True(t, ok)
			assert.Equal(t, apierror.CodeValidationFailed, apiErr.Code)
			assert.Equal(t, http.StatusUnprocessableEntity, apiErr.HTTPStatus)
			assert.False(t, called)
		})
	}
}
