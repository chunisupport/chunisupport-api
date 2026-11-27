package api_internal

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type appAnnouncementUsecaseStub struct {
	announcements []*entity.AppAnnouncement
	err           error
}

func (s *appAnnouncementUsecaseStub) ListActive(context.Context) ([]*entity.AppAnnouncement, error) {
	return s.announcements, s.err
}

func TestAppAnnouncementHandler_List(t *testing.T) {
	// Given
	startsAt := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	expiresAt := startsAt.Add(24 * time.Hour)
	body := "本文\n2行目"
	docsURL := "https://docs.chunisupport.net/announcements/"
	link, err := entity.RestoreAppAnnouncement("000000000000000b", "link", "大型アップデート", nil, &docsURL, startsAt, expiresAt, 2)
	require.NoError(t, err)
	message, err := entity.RestoreAppAnnouncement("000000000000000a", "message", "お知らせ", &body, nil, startsAt, expiresAt, 1)
	require.NoError(t, err)
	handler := NewAppAnnouncementHandler(&appAnnouncementUsecaseStub{announcements: []*entity.AppAnnouncement{link, message}})
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/internal/app-announcements", nil)
	rec := httptest.NewRecorder()
	c := echo.New().NewContext(req, rec)

	// When
	err = handler.List(c)

	// Then
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "no-store", rec.Header().Get(echo.HeaderCacheControl))
	assert.JSONEq(t, `{
		"announcements": [
			{
				"id": "000000000000000b",
				"kind": "link",
				"title": "大型アップデート",
				"body": null,
				"target_url": "https://docs.chunisupport.net/announcements/",
				"starts_at": "2026-12-01T00:00:00Z",
				"expires_at": "2026-12-02T00:00:00Z",
				"revision": 2
			},
			{
				"id": "000000000000000a",
				"kind": "message",
				"title": "お知らせ",
				"body": "本文\n2行目",
				"target_url": null,
				"starts_at": "2026-12-01T00:00:00Z",
				"expires_at": "2026-12-02T00:00:00Z",
				"revision": 1
			}
		]
	}`, rec.Body.String())
}

func TestAppAnnouncementHandler_List_配信中がなければ空配列を返す(t *testing.T) {
	// Given
	handler := NewAppAnnouncementHandler(&appAnnouncementUsecaseStub{})
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/internal/app-announcements", nil)
	rec := httptest.NewRecorder()
	c := echo.New().NewContext(req, rec)

	// When
	err := handler.List(c)

	// Then
	require.NoError(t, err)
	assert.JSONEq(t, `{"announcements": []}`, rec.Body.String())
}

func TestAppAnnouncementHandler_List_ユースケースのエラーを返す(t *testing.T) {
	// Given
	handler := NewAppAnnouncementHandler(&appAnnouncementUsecaseStub{err: errors.New("db error")})
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/internal/app-announcements", nil)
	c := echo.New().NewContext(req, httptest.NewRecorder())

	// When
	err := handler.List(c)

	// Then
	assert.Error(t, err)
}
