package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/app/handler/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/app/handler/api_v1"
	"github.com/chunisupport/chunisupport-api/internal/app/middleware"
	"github.com/chunisupport/chunisupport-api/internal/config"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type courseTracksUsecaseStub struct {
	usecase.CourseUsecase
	tracks    []entity.CourseTrack
	resultErr error
}

func (s *courseTracksUsecaseStub) ReplaceTracks(_ context.Context, _ string, tracks []entity.CourseTrack) error {
	s.tracks = tracks
	return s.resultErr
}

func (s *courseTracksUsecaseStub) List(_ context.Context, _ bool) ([]*usecase.CourseOutput, error) {
	return []*usecase.CourseOutput{{
		DisplayID: "0123456789abcdef", Idx: "50020", Name: "コース", Class: "1",
		Tracks: []usecase.CourseTrackOutput{{
			CourseTrack: entity.CourseTrack{Number: 1, SelectionType: entity.CourseTrackFixed, Charts: []entity.CourseChartRef{{SongID: "fedcba9876543210", Difficulty: "MASTER"}}},
		}},
	}}, nil
}

func (s *courseTracksUsecaseStub) Get(ctx context.Context, _ string, editor bool) (*usecase.CourseOutput, error) {
	courses, _ := s.List(ctx, editor)
	item := courses[0]
	item.Tracks[0].Candidates = []entity.CourseChartCandidate{}
	return item, nil
}

func newCourseTrackTestRouter(uc usecase.CourseUsecase) *echo.Echo {
	e := echo.New()
	e.Validator = NewCustomValidator()
	e.HTTPErrorHandler = middleware.CustomHTTPErrorHandler
	handlers := newAuthorizationTestHandlers()
	handlers.Course = api_internal.NewCourseHandler(uc)
	handlers.V1Course = api_v1.NewV1CourseHandler(uc)
	registerRoutes(e, handlers, stubFirebaseAuthenticator{}, stubFirebaseAuthenticator{}, stubAPITokenUsecase{}, stubMaintenanceUsecase{}, config.Config{})
	return e
}

func TestRegisterRoutes_コース構成の一括更新(t *testing.T) {
	for _, tt := range []struct {
		name, token, body string
		status, tracks    int
		usecaseError      error
	}{
		{name: "編集権限なし", token: "player-token", body: `{"tracks":[]}`, status: 403},
		{name: "構成を空に戻す", token: "editor-token", body: `{"tracks":[]}`, status: 204},
		{name: "3枠を登録", token: "editor-token", body: `{"tracks":[
			{"track_no":1,"selection_type":"FIXED","charts":[{"id":"fedcba9876543210","difficulty":"MASTER"}]},
			{"track_no":2,"selection_type":"RANDOM_FILTER","filter":{"min_level":"14+","genres":["ゲキマイ","東方Project"]}},
			{"track_no":3,"selection_type":"RANDOM_POOL","charts":[{"id":"fedcba9876543210","difficulty":"ULTIMA"}]}
		]}`, status: 204, tracks: 3},
		{name: "tracks欠落", token: "editor-token", body: `{}`, status: 422},
		{name: "tracksにnull", token: "editor-token", body: `{"tracks":null}`, status: 422},
		{name: "レベルの書式が不正", token: "editor-token", body: `{"tracks":[{"track_no":1,"selection_type":"RANDOM_FILTER","filter":{"min_level":"14.5"}}]}`, status: 422},
		{name: "フィルタの未知項目", token: "editor-token", body: `{"tracks":[{"track_no":1,"selection_type":"RANDOM_FILTER","filter":{"op_target":true}}]}`, status: 400},
		{name: "プール名は受け付けない", token: "editor-token", body: `{"tracks":[{"track_no":3,"selection_type":"RANDOM_POOL","pool_label":"任意名","charts":[{"id":"fedcba9876543210","difficulty":"ULTIMA"}]}]}`, status: 400},
		{name: "不正な構成", token: "editor-token", body: `{"tracks":[]}`, status: 422, usecaseError: usecase.ErrInvalidCourseInput},
		{name: "コースなし", token: "editor-token", body: `{"tracks":[]}`, status: 404, usecaseError: usecase.ErrCourseNotFound},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc := &courseTracksUsecaseStub{resultErr: tt.usecaseError}
			e := newCourseTrackTestRouter(uc)
			req := httptest.NewRequest(http.MethodPut, "/internal/courses/0123456789abcdef/tracks", strings.NewReader(tt.body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			req.Header.Set(echo.HeaderAuthorization, "Bearer "+tt.token)
			rec := httptest.NewRecorder()

			e.ServeHTTP(rec, req)

			require.Equal(t, tt.status, rec.Code, rec.Body.String())
			assert.Len(t, uc.tracks, tt.tracks)
			if tt.status == http.StatusNoContent {
				require.NotNil(t, uc.tracks)
			}
			if tt.tracks == 3 {
				filter := uc.tracks[1].Filter
				require.NotNil(t, filter)
				require.NotNil(t, filter.MinLevel)
				assert.Equal(t, "14+", filter.MinLevel.String())
				assert.Nil(t, filter.MaxLevel)
				assert.Equal(t, []string{"ゲキマイ", "東方Project"}, filter.Genres)
			}
		})
	}
}

func TestRegisterRoutes_コース詳細だけが解決済み候補を返す(t *testing.T) {
	for _, prefix := range []string{"/internal/courses", "/v1/courses"} {
		t.Run(prefix, func(t *testing.T) {
			e := newCourseTrackTestRouter(&courseTracksUsecaseStub{})
			for _, detail := range []bool{false, true} {
				path := prefix
				if detail {
					path += "/0123456789abcdef"
				}
				req := httptest.NewRequest(http.MethodGet, path, nil)
				req.Header.Set(echo.HeaderAuthorization, "Bearer player-token")
				rec := httptest.NewRecorder()

				e.ServeHTTP(rec, req)

				require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
				assert.Contains(t, rec.Body.String(), `"charts":[{"id":"fedcba9876543210","difficulty":"MASTER"}]`)
				if detail {
					assert.Contains(t, rec.Body.String(), `"candidates":[]`)
				} else {
					assert.NotContains(t, rec.Body.String(), `"candidates"`)
				}
			}
		})
	}
}
