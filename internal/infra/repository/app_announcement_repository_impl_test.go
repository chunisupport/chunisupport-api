package repository

import (
	"context"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/displayid"
	"github.com/chunisupport/chunisupport-api/internal/testutil/mysqltest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const insertAppAnnouncementSQL = `
INSERT INTO app_announcements (id, kind, title, body, target_url, starts_at, expires_at, revision)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

func TestAppAnnouncementRepository_FindUnexpired(t *testing.T) {
	// Given
	db := mysqltest.NewDatabase(t, "")
	now := time.Date(2026, 12, 1, 12, 0, 0, 0, time.UTC)
	insert := func(displayID, kind string, body, targetURL *string, startsAt, expiresAt time.Time) {
		t.Helper()
		_, err := db.Exec(insertAppAnnouncementSQL, displayID, kind, "タイトル", body, targetURL, startsAt, expiresAt, 1)
		require.NoError(t, err)
	}
	body := "本文"
	docsURL := "https://docs.chunisupport.net/announcements/"
	insert("000000000000000a", "message", &body, nil, now.Add(-2*time.Hour), now.Add(time.Hour))
	insert("000000000000000b", "link", nil, &docsURL, now.Add(-time.Hour), now.Add(time.Hour))
	insert("000000000000000c", "message", &body, nil, now.Add(time.Hour), now.Add(2*time.Hour))
	insert("000000000000000d", "message", &body, nil, now.Add(-2*time.Hour), now)
	repo := NewAppAnnouncementRepository(db)

	// When
	got, err := repo.FindUnexpired(context.Background(), now)

	// Then: 終了済みは除き、開始前を含めて開始日時の新しい順に返す
	require.NoError(t, err)
	ids := make([]string, 0, len(got))
	for _, announcement := range got {
		ids = append(ids, announcement.DisplayID.String())
	}
	assert.Equal(t, []string{"000000000000000c", "000000000000000b", "000000000000000a"}, ids)
	assert.Equal(t, &docsURL, got[1].TargetURL)
	assert.Nil(t, got[1].Body)
	assert.Equal(t, now.Add(-time.Hour), got[1].StartsAt)
}

func TestAppAnnouncementRepository_FindUnexpired_不正な行はエラーにする(t *testing.T) {
	// Given: 開始前の行も検証し、登録直後に誤りへ気付けるようにする
	db := mysqltest.NewDatabase(t, "")
	now := time.Date(2026, 12, 1, 12, 0, 0, 0, time.UTC)
	_, err := db.Exec(insertAppAnnouncementSQL, "000000000000000a", "modal", "タイトル", "本文", nil, now.Add(time.Hour), now.Add(2*time.Hour), 1)
	require.NoError(t, err)
	repo := NewAppAnnouncementRepository(db)

	// When
	_, err = repo.FindUnexpired(context.Background(), now)

	// Then
	assert.ErrorIs(t, err, entity.ErrAppAnnouncementKindInvalid)
}

func TestAppAnnouncementsMigration_不正な期間とリビジョンを拒否する(t *testing.T) {
	db := mysqltest.NewDatabase(t, "")
	startsAt := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	body := "本文"

	tests := []struct {
		name      string
		expiresAt time.Time
		revision  int
	}{
		{name: "開始と終了が同じ", expiresAt: startsAt, revision: 1},
		{name: "リビジョンが0", expiresAt: startsAt.Add(time.Hour), revision: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			_, err := db.Exec(insertAppAnnouncementSQL, "000000000000000a", "message", "タイトル", &body, nil, startsAt, tt.expiresAt, tt.revision)

			// Then
			assert.ErrorContains(t, err, "chk_app_announcements_")
		})
	}
}

func TestAppAnnouncementRepository_管理操作(t *testing.T) {
	// Given
	db := mysqltest.NewDatabase(t, "")
	repo := NewAppAnnouncementRepository(db)
	ctx := context.Background()
	jst := time.FixedZone("JST", 9*60*60)
	startsAt := time.Date(2026, 12, 1, 9, 0, 0, 0, jst)
	body := "本文"
	docsURL := "https://docs.chunisupport.net/announcements/"
	older, err := entity.RestoreAppAnnouncement("000000000000000a", "message", "古い", &body, nil, startsAt.Add(-48*time.Hour), startsAt.Add(-24*time.Hour), 1)
	require.NoError(t, err)
	newer, err := entity.RestoreAppAnnouncement("000000000000000b", "link", "新しい", nil, &docsURL, startsAt, startsAt.Add(time.Hour), 1)
	require.NoError(t, err)

	// When: 追加すると、終了済みを含めて開始日時の新しい順に取得できる
	require.NoError(t, repo.Create(ctx, older))
	require.NoError(t, repo.Create(ctx, newer))
	all, err := repo.FindAll(ctx)

	// Then: 日時はUTCで保存される
	require.NoError(t, err)
	require.Len(t, all, 2)
	assert.Equal(t, "000000000000000b", all[0].DisplayID.String())
	assert.True(t, startsAt.Equal(all[0].StartsAt))
	assert.Equal(t, "000000000000000a", all[1].DisplayID.String())

	// When: 種別を変えて上書きする
	updated, err := entity.RestoreAppAnnouncement("000000000000000b", "message", "改訂", &body, nil, startsAt, startsAt.Add(2*time.Hour), 2)
	require.NoError(t, err)
	require.NoError(t, repo.Update(ctx, updated))
	require.NoError(t, repo.Update(ctx, updated))
	got, err := repo.FindByDisplayID(ctx, updated.DisplayID)

	// Then: 値が変わらない2回目の上書きも成功し、内容が置き換わる
	require.NoError(t, err)
	assert.Equal(t, entity.AppAnnouncementKindMessage, got.Kind)
	assert.Equal(t, "改訂", got.Title)
	assert.Nil(t, got.TargetURL)
	assert.Equal(t, 2, got.Revision)

	// When & Then: 削除後は見つからない
	require.NoError(t, repo.Delete(ctx, updated.DisplayID))
	_, err = repo.FindByDisplayID(ctx, updated.DisplayID)
	assert.ErrorIs(t, err, domainrepo.ErrAppAnnouncementNotFound)
	assert.ErrorIs(t, repo.Delete(ctx, updated.DisplayID), domainrepo.ErrAppAnnouncementNotFound)
	missing := displayid.DisplayID("00000000000000ff")
	missingAnnouncement, err := entity.RestoreAppAnnouncement(missing.String(), "message", "なし", &body, nil, startsAt, startsAt.Add(time.Hour), 1)
	require.NoError(t, err)
	assert.ErrorIs(t, repo.Update(ctx, missingAnnouncement), domainrepo.ErrAppAnnouncementNotFound)
}
