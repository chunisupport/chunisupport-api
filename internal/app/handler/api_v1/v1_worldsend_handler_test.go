package api_v1

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	api_v1_dto "github.com/chunisupport/chunisupport-api/internal/dto/api_v1"
	"github.com/chunisupport/chunisupport-api/internal/infra/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/testutil"
	"github.com/labstack/echo/v4"
)

func TestGetWorldsendSongs(t *testing.T) {
	updatedAt := time.Date(2026, 3, 22, 15, 4, 5, 0, time.UTC)
	handler := &V1WorldsendHandler{
		worldsendUsecase: &testutil.MockWorldsendUsecase{
			GetAllWorldsendSongsFunc: func(ctx context.Context, includeDeleted bool, requesterAccountTypeID *int) ([]*repository.WorldsendSongWithChart, error) {
				return []*repository.WorldsendSongWithChart{{
					Song:  &entity.Song{DisplayID: "we1234567890abcd", Title: "WE曲", Artist: "WEアーティスト", IsWorldsend: true},
					Chart: &entity.WorldsendChart{},
				}}, nil
			},
			GetWorldsendSongsLastUpdatedAtFunc: func(ctx context.Context, includeDeleted bool, requesterAccountTypeID *int) (*time.Time, error) {
				return &updatedAt, nil
			},
		},
		masterCache: &masterdata.Cache{GenreNamesByID: map[int]string{}},
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/v1/songs/worldsend", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.GetWorldsendSongs(c)
	if err != nil {
		t.Fatalf("GetWorldsendSongs returned error: %v", err)
	}

	var response api_v1_dto.V1WorldsendSongsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response.UpdatedAt == nil {
		t.Fatal("UpdatedAt should not be nil")
	}
}
