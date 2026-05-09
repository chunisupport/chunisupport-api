package api_internal

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockPlayerLockedSongUsecase struct {
	listFunc   func(ctx context.Context, username string, requester *entity.User) ([]*usecase.PlayerLockedSongOutput, error)
	unlockFunc func(ctx context.Context, userID int, input *usecase.PlayerLockedSongInput) error
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

func TestPlayerLockedSongHandler_List(t *testing.T) {
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
	c.SetParamNames("username")
	c.SetParamValues("testuser")

	// When
	err := handler.List(c)

	// Then
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"items":[{"display_id":"1234567890abcdef","title":"テスト楽曲","is_ultima":true}]}`, rec.Body.String())
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
			c.SetParamNames("displayid")
			c.SetParamValues(tt.displayID)

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
