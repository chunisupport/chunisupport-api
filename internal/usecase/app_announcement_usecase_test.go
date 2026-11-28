package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/displayid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubAppAnnouncementRepository struct {
	announcements []*entity.AppAnnouncement
	err           error
	gotNow        time.Time
	found         *entity.AppAnnouncement
	saved         *entity.AppAnnouncement
	deletedID     displayid.DisplayID
}

func (r *stubAppAnnouncementRepository) FindUnexpired(_ context.Context, now time.Time) ([]*entity.AppAnnouncement, error) {
	r.gotNow = now
	return r.announcements, r.err
}

func (r *stubAppAnnouncementRepository) FindAll(context.Context) ([]*entity.AppAnnouncement, error) {
	return r.announcements, r.err
}

func (r *stubAppAnnouncementRepository) FindByDisplayIDForUpdate(context.Context, repository.Executor, displayid.DisplayID) (*entity.AppAnnouncement, error) {
	if r.found == nil {
		return nil, repository.ErrAppAnnouncementNotFound
	}
	return r.found, nil
}

func (r *stubAppAnnouncementRepository) Create(_ context.Context, announcement *entity.AppAnnouncement) error {
	r.saved = announcement
	return r.err
}

func (r *stubAppAnnouncementRepository) Update(_ context.Context, _ repository.Executor, announcement *entity.AppAnnouncement) error {
	r.saved = announcement
	return r.err
}

func (r *stubAppAnnouncementRepository) Delete(_ context.Context, displayID displayid.DisplayID) error {
	r.deletedID = displayID
	return r.err
}

func TestAppAnnouncementUsecase_ListActive(t *testing.T) {
	// Given
	now := time.Date(2026, 12, 1, 12, 0, 0, 0, time.UTC)
	body := "本文"
	newAnnouncement := func(displayID string, startsAt time.Time) *entity.AppAnnouncement {
		announcement, err := entity.RestoreAppAnnouncement(displayID, "message", "お知らせ", &body, nil, startsAt, now.Add(time.Hour), 1)
		require.NoError(t, err)
		return announcement
	}
	notStarted := newAnnouncement("000000000000000c", now.Add(time.Second))
	startsNow := newAnnouncement("000000000000000b", now)
	started := newAnnouncement("000000000000000a", now.Add(-time.Hour))
	repo := &stubAppAnnouncementRepository{announcements: []*entity.AppAnnouncement{notStarted, startsNow, started}}
	uc := &appAnnouncementUsecase{repo: repo, clock: fixedClock{now: now}}

	// When
	got, err := uc.ListActive(context.Background())

	// Then: 開始前を除き、リポジトリの順序を保つ
	require.NoError(t, err)
	assert.Equal(t, now, repo.gotNow)
	assert.Equal(t, []*entity.AppAnnouncement{startsNow, started}, got)
}

func TestAppAnnouncementUsecase_ListActive_リポジトリのエラーを返す(t *testing.T) {
	// Given
	repoErr := errors.New("db error")
	uc := &appAnnouncementUsecase{repo: &stubAppAnnouncementRepository{err: repoErr}, clock: fixedClock{}}

	// When
	_, err := uc.ListActive(context.Background())

	// Then
	assert.ErrorIs(t, err, repoErr)
}

func TestAppAnnouncementUsecase_Create(t *testing.T) {
	// Given
	startsAt := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	body := "本文"
	repo := &stubAppAnnouncementRepository{}
	uc := &appAnnouncementUsecase{repo: repo, clock: fixedClock{}}

	// When
	got, err := uc.Create(context.Background(), AppAnnouncementInput{Kind: "message", Title: "お知らせ", Body: &body, StartsAt: startsAt, ExpiresAt: startsAt.Add(time.Hour)})

	// Then: IDを採番し、revision 1 で保存する
	require.NoError(t, err)
	assert.True(t, got.DisplayID.IsValid())
	assert.Equal(t, 1, got.Revision)
	assert.Same(t, got, repo.saved)
}

func TestAppAnnouncementUsecase_Create_不正な内容は入力エラー(t *testing.T) {
	// Given: リンク型でURLがない
	startsAt := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	repo := &stubAppAnnouncementRepository{}
	uc := &appAnnouncementUsecase{repo: repo, clock: fixedClock{}}

	// When
	_, err := uc.Create(context.Background(), AppAnnouncementInput{Kind: "link", Title: "新ツール", StartsAt: startsAt, ExpiresAt: startsAt.Add(time.Hour)})

	// Then
	assert.ErrorIs(t, err, ErrInvalidAppAnnouncementInput)
	assert.ErrorIs(t, err, entity.ErrAppAnnouncementTargetURLRequired)
	assert.Nil(t, repo.saved)
}

func TestAppAnnouncementUsecase_Update(t *testing.T) {
	startsAt := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	body := "本文"
	newBody := "改訂後の本文"
	input := AppAnnouncementInput{Kind: "message", Title: "改訂", Body: &newBody, StartsAt: startsAt, ExpiresAt: startsAt.Add(2 * time.Hour)}

	tests := []struct {
		name         string
		bumpRevision bool
		wantRevision int
	}{
		{name: "revisionを維持して上書きする", bumpRevision: false, wantRevision: 3},
		{name: "revisionを1つ増やして上書きする", bumpRevision: true, wantRevision: 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			current, err := entity.RestoreAppAnnouncement("000000000000000a", "message", "お知らせ", &body, nil, startsAt, startsAt.Add(time.Hour), 3)
			require.NoError(t, err)
			repo := &stubAppAnnouncementRepository{found: current}
			uc := NewAdminAppAnnouncementUsecase(repo, &passthroughTransactionManager{})

			// When
			got, err := uc.Update(context.Background(), "000000000000000a", input, tt.bumpRevision)

			// Then
			require.NoError(t, err)
			assert.Equal(t, tt.wantRevision, got.Revision)
			assert.Equal(t, "改訂", got.Title)
			assert.Equal(t, &newBody, got.Body)
			assert.Same(t, got, repo.saved)
		})
	}
}

func TestAppAnnouncementUsecase_Update_存在しないお知らせはエラー(t *testing.T) {
	// Given
	startsAt := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	body := "本文"
	input := AppAnnouncementInput{Kind: "message", Title: "お知らせ", Body: &body, StartsAt: startsAt, ExpiresAt: startsAt.Add(time.Hour)}
	uc := NewAdminAppAnnouncementUsecase(&stubAppAnnouncementRepository{}, &passthroughTransactionManager{})

	for _, id := range []string{"000000000000000a", "invalid"} {
		// When
		_, err := uc.Update(context.Background(), id, input, false)

		// Then
		assert.ErrorIs(t, err, repository.ErrAppAnnouncementNotFound, id)
	}
}

func TestAppAnnouncementUsecase_Delete(t *testing.T) {
	// Given
	repo := &stubAppAnnouncementRepository{}
	uc := &appAnnouncementUsecase{repo: repo, clock: fixedClock{}}

	// When
	err := uc.Delete(context.Background(), "000000000000000a")
	invalidErr := uc.Delete(context.Background(), "invalid")

	// Then
	require.NoError(t, err)
	assert.Equal(t, displayid.DisplayID("000000000000000a"), repo.deletedID)
	assert.ErrorIs(t, invalidErr, repository.ErrAppAnnouncementNotFound)
}
