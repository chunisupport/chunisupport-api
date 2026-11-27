package api_internal

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type adminAppAnnouncementUsecaseStub struct {
	announcement    *entity.AppAnnouncement
	err             error
	gotID           string
	gotInput        usecase.AppAnnouncementInput
	gotBumpRevision bool
}

func (s *adminAppAnnouncementUsecaseStub) ListAll(context.Context) ([]*entity.AppAnnouncement, error) {
	return []*entity.AppAnnouncement{s.announcement}, s.err
}

func (s *adminAppAnnouncementUsecaseStub) Create(_ context.Context, input usecase.AppAnnouncementInput) (*entity.AppAnnouncement, error) {
	s.gotInput = input
	return s.announcement, s.err
}

func (s *adminAppAnnouncementUsecaseStub) Update(_ context.Context, id string, input usecase.AppAnnouncementInput, bumpRevision bool) (*entity.AppAnnouncement, error) {
	s.gotID, s.gotInput, s.gotBumpRevision = id, input, bumpRevision
	return s.announcement, s.err
}

func (s *adminAppAnnouncementUsecaseStub) Delete(_ context.Context, id string) error {
	s.gotID = id
	return s.err
}

// newAdminAppAnnouncementContext はJSONボディとパスIDを持つリクエストコンテキストを作ります。
func newAdminAppAnnouncementContext(method, body, id string) (*echo.Context, *httptest.ResponseRecorder) {
	req := httptest.NewRequestWithContext(context.Background(), method, "/internal/admin/app-announcements", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := echo.New().NewContext(req, rec)
	if id != "" {
		c.SetPathValues(echo.PathValues{{Name: "id", Value: id}})
	}
	return c, rec
}

func newAdminTestAnnouncement(t *testing.T) *entity.AppAnnouncement {
	t.Helper()
	startsAt := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	body := "本文"
	announcement, err := entity.RestoreAppAnnouncement("000000000000000a", "message", "お知らせ", &body, nil, startsAt, startsAt.Add(time.Hour), 2)
	require.NoError(t, err)
	return announcement
}

func TestAdminAppAnnouncementHandler_Create(t *testing.T) {
	// Given
	stub := &adminAppAnnouncementUsecaseStub{announcement: newAdminTestAnnouncement(t)}
	handler := NewAdminAppAnnouncementHandler(stub)
	c, rec := newAdminAppAnnouncementContext(http.MethodPost, `{
		"kind": "message", "title": "お知らせ", "body": "本文", "target_url": null,
		"starts_at": "2026-12-01T09:00:00+09:00", "expires_at": "2026-12-01T10:00:00+09:00"
	}`, "")

	// When
	err := handler.Create(c)

	// Then
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.Equal(t, "message", stub.gotInput.Kind)
	assert.Nil(t, stub.gotInput.TargetURL)
	assert.True(t, time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC).Equal(stub.gotInput.StartsAt))
	assert.JSONEq(t, `{
		"id": "000000000000000a", "kind": "message", "title": "お知らせ", "body": "本文", "target_url": null,
		"starts_at": "2026-12-01T00:00:00Z", "expires_at": "2026-12-01T01:00:00Z", "revision": 2
	}`, rec.Body.String())
}

func TestAdminAppAnnouncementHandler_Create_未知のフィールドは拒否する(t *testing.T) {
	// Given: 追加時は bump_revision を受け付けない
	handler := NewAdminAppAnnouncementHandler(&adminAppAnnouncementUsecaseStub{})
	c, _ := newAdminAppAnnouncementContext(http.MethodPost, `{"kind": "message", "bump_revision": true}`, "")

	// When
	err := handler.Create(c)

	// Then
	assert.ErrorIs(t, err, apierror.ErrValidationFailedBadRequest)
}

func TestAdminAppAnnouncementHandler_Update(t *testing.T) {
	// Given
	stub := &adminAppAnnouncementUsecaseStub{announcement: newAdminTestAnnouncement(t)}
	handler := NewAdminAppAnnouncementHandler(stub)
	c, rec := newAdminAppAnnouncementContext(http.MethodPut, `{
		"kind": "link", "title": "新ツール", "body": null, "target_url": "/tools/dashboard",
		"starts_at": "2026-12-01T00:00:00Z", "expires_at": "2026-12-02T00:00:00Z", "bump_revision": true
	}`, "000000000000000a")

	// When
	err := handler.Update(c)

	// Then
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "000000000000000a", stub.gotID)
	assert.Equal(t, "/tools/dashboard", *stub.gotInput.TargetURL)
	assert.True(t, stub.gotBumpRevision)
}

func TestAdminAppAnnouncementHandler_Delete(t *testing.T) {
	// Given
	stub := &adminAppAnnouncementUsecaseStub{}
	handler := NewAdminAppAnnouncementHandler(stub)
	c, rec := newAdminAppAnnouncementContext(http.MethodDelete, "", "000000000000000a")

	// When
	err := handler.Delete(c)

	// Then
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, "000000000000000a", stub.gotID)
}

func TestAdminAppAnnouncementHandler_ユースケースのエラーを変換する(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{name: "入力エラーは422", err: usecase.ErrInvalidAppAnnouncementInput, wantStatus: http.StatusUnprocessableEntity},
		{name: "存在しないお知らせは404", err: repository.ErrAppAnnouncementNotFound, wantStatus: http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			handler := NewAdminAppAnnouncementHandler(&adminAppAnnouncementUsecaseStub{err: tt.err})
			c, _ := newAdminAppAnnouncementContext(http.MethodDelete, "", "000000000000000a")

			// When
			err := handler.Delete(c)

			// Then
			var apiErr *apierror.APIError
			require.ErrorAs(t, err, &apiErr)
			assert.Equal(t, tt.wantStatus, apiErr.HTTPStatus)
		})
	}
}
