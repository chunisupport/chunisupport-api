package entity

import (
	"strings"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo/displayid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRestoreAppAnnouncement(t *testing.T) {
	startsAt := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	expiresAt := startsAt.Add(24 * time.Hour)
	body := "本文\n2行目"
	emptyBody := " \n"
	appPath := "/tools/new-tool"
	emptyURL := ""
	longTitle := strings.Repeat("あ", AppAnnouncementTitleMaxLength+1)
	longURL := "/" + strings.Repeat("a", AppAnnouncementTargetURLMaxLength)
	longBody := strings.Repeat("あ", AppAnnouncementBodyMaxBytes/3+1)
	docsURL := "https://docs.chunisupport.net/announcements/"
	schemelessURL := "example.com"
	scriptURL := "javascript:alert(1)"
	protocolRelativeURL := "//evil.example.com"
	backslashURL := "/\\evil.example.com"

	tests := []struct {
		name      string
		displayID string
		kind      string
		title     string
		body      *string
		targetURL *string
		startsAt  time.Time
		expiresAt time.Time
		revision  int
		wantErr   error
	}{
		{name: "本文型を復元できる", displayID: "0123456789abcdef", kind: "message", title: "お知らせ", body: &body, startsAt: startsAt, expiresAt: expiresAt, revision: 1},
		{name: "リンク型を復元できる", displayID: "0123456789abcdef", kind: "link", title: "新ツール", targetURL: &appPath, startsAt: startsAt, expiresAt: expiresAt, revision: 2},
		{name: "不正な表示用IDはエラー", displayID: "invalid", kind: "message", title: "お知らせ", body: &body, startsAt: startsAt, expiresAt: expiresAt, revision: 1, wantErr: displayid.ErrInvalidDisplayIDFormat},
		{name: "不正な種別はエラー", displayID: "0123456789abcdef", kind: "modal", title: "お知らせ", body: &body, startsAt: startsAt, expiresAt: expiresAt, revision: 1, wantErr: ErrAppAnnouncementKindInvalid},
		{name: "空白だけのタイトルはエラー", displayID: "0123456789abcdef", kind: "message", title: " ", body: &body, startsAt: startsAt, expiresAt: expiresAt, revision: 1, wantErr: ErrAppAnnouncementTitleRequired},
		{name: "本文型で本文がないとエラー", displayID: "0123456789abcdef", kind: "message", title: "お知らせ", startsAt: startsAt, expiresAt: expiresAt, revision: 1, wantErr: ErrAppAnnouncementBodyRequired},
		{name: "本文型で本文が空白だけだとエラー", displayID: "0123456789abcdef", kind: "message", title: "お知らせ", body: &emptyBody, startsAt: startsAt, expiresAt: expiresAt, revision: 1, wantErr: ErrAppAnnouncementBodyRequired},
		{name: "本文型にURLがあるとエラー", displayID: "0123456789abcdef", kind: "message", title: "お知らせ", body: &body, targetURL: &appPath, startsAt: startsAt, expiresAt: expiresAt, revision: 1, wantErr: ErrAppAnnouncementTargetURLMustBeNull},
		{name: "リンク型でURLがないとエラー", displayID: "0123456789abcdef", kind: "link", title: "新ツール", startsAt: startsAt, expiresAt: expiresAt, revision: 1, wantErr: ErrAppAnnouncementTargetURLRequired},
		{name: "リンク型に本文があるとエラー", displayID: "0123456789abcdef", kind: "link", title: "新ツール", body: &body, targetURL: &appPath, startsAt: startsAt, expiresAt: expiresAt, revision: 1, wantErr: ErrAppAnnouncementBodyMustBeNull},
		{name: "リンク型でURLが空だとエラー", displayID: "0123456789abcdef", kind: "link", title: "新ツール", targetURL: &emptyURL, startsAt: startsAt, expiresAt: expiresAt, revision: 1, wantErr: ErrAppAnnouncementTargetURLRequired},
		{name: "開始と終了が同じだとエラー", displayID: "0123456789abcdef", kind: "message", title: "お知らせ", body: &body, startsAt: startsAt, expiresAt: startsAt, revision: 1, wantErr: ErrAppAnnouncementPeriodInvalid},
		{name: "開始日時がないとエラー", displayID: "0123456789abcdef", kind: "message", title: "お知らせ", body: &body, expiresAt: expiresAt, revision: 1, wantErr: ErrAppAnnouncementPeriodInvalid},
		{name: "秒未満を切り捨てると開始と終了が同じになるとエラー", displayID: "0123456789abcdef", kind: "message", title: "お知らせ", body: &body, startsAt: startsAt.Add(100 * time.Millisecond), expiresAt: startsAt.Add(900 * time.Millisecond), revision: 1, wantErr: ErrAppAnnouncementPeriodInvalid},
		{name: "タイトルが上限を超えるとエラー", displayID: "0123456789abcdef", kind: "message", title: longTitle, body: &body, startsAt: startsAt, expiresAt: expiresAt, revision: 1, wantErr: ErrAppAnnouncementTitleTooLong},
		{name: "URLが上限を超えるとエラー", displayID: "0123456789abcdef", kind: "link", title: "新ツール", targetURL: &longURL, startsAt: startsAt, expiresAt: expiresAt, revision: 1, wantErr: ErrAppAnnouncementTargetURLTooLong},
		{name: "外部URLのリンク型を復元できる", displayID: "0123456789abcdef", kind: "link", title: "記事", targetURL: &docsURL, startsAt: startsAt, expiresAt: expiresAt, revision: 1},
		{name: "本文が上限を超えるとエラー", displayID: "0123456789abcdef", kind: "message", title: "お知らせ", body: &longBody, startsAt: startsAt, expiresAt: expiresAt, revision: 1, wantErr: ErrAppAnnouncementBodyTooLong},
		{name: "スキームのないURLはエラー", displayID: "0123456789abcdef", kind: "link", title: "新ツール", targetURL: &schemelessURL, startsAt: startsAt, expiresAt: expiresAt, revision: 1, wantErr: ErrAppAnnouncementTargetURLInvalid},
		{name: "javascriptのURLはエラー", displayID: "0123456789abcdef", kind: "link", title: "新ツール", targetURL: &scriptURL, startsAt: startsAt, expiresAt: expiresAt, revision: 1, wantErr: ErrAppAnnouncementTargetURLInvalid},
		{name: "プロトコル相対URLはエラー", displayID: "0123456789abcdef", kind: "link", title: "新ツール", targetURL: &protocolRelativeURL, startsAt: startsAt, expiresAt: expiresAt, revision: 1, wantErr: ErrAppAnnouncementTargetURLInvalid},
		{name: "バックスラッシュを含むパスはエラー", displayID: "0123456789abcdef", kind: "link", title: "新ツール", targetURL: &backslashURL, startsAt: startsAt, expiresAt: expiresAt, revision: 1, wantErr: ErrAppAnnouncementTargetURLInvalid},
		{name: "リビジョンが上限を超えるとエラー", displayID: "0123456789abcdef", kind: "message", title: "お知らせ", body: &body, startsAt: startsAt, expiresAt: expiresAt, revision: AppAnnouncementRevisionMax + 1, wantErr: ErrAppAnnouncementRevisionInvalid},
		{name: "リビジョンが0だとエラー", displayID: "0123456789abcdef", kind: "message", title: "お知らせ", body: &body, startsAt: startsAt, expiresAt: expiresAt, revision: 0, wantErr: ErrAppAnnouncementRevisionInvalid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			got, err := RestoreAppAnnouncement(tt.displayID, tt.kind, tt.title, tt.body, tt.targetURL, tt.startsAt, tt.expiresAt, tt.revision)

			// Then
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.displayID, got.DisplayID.String())
			assert.Equal(t, AppAnnouncementKind(tt.kind), got.Kind)
			assert.Equal(t, tt.title, got.Title)
			assert.Equal(t, tt.body, got.Body)
			assert.Equal(t, tt.targetURL, got.TargetURL)
			assert.Equal(t, tt.revision, got.Revision)
		})
	}
}

func TestRestoreAppAnnouncement_日時の秒未満を切り捨てる(t *testing.T) {
	// Given: DATETIME 列に保存する前の、秒未満を含む日時
	startsAt := time.Date(2026, 12, 1, 0, 0, 0, 600_000_000, time.UTC)
	expiresAt := time.Date(2026, 12, 2, 0, 0, 0, 400_000_000, time.UTC)
	body := "本文"

	// When
	got, err := RestoreAppAnnouncement("0123456789abcdef", "message", "お知らせ", &body, nil, startsAt, expiresAt, 1)

	// Then: DBの丸めに任せず、保存値とレスポンスを揃える
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC), got.StartsAt)
	assert.Equal(t, time.Date(2026, 12, 2, 0, 0, 0, 0, time.UTC), got.ExpiresAt)
}

func TestAppAnnouncement_IsActiveAt(t *testing.T) {
	startsAt := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	expiresAt := startsAt.Add(24 * time.Hour)
	body := "本文"
	announcement, err := RestoreAppAnnouncement("0123456789abcdef", "message", "お知らせ", &body, nil, startsAt, expiresAt, 1)
	require.NoError(t, err)

	tests := []struct {
		name     string
		now      time.Time
		expected bool
	}{
		{name: "開始前は表示しない", now: startsAt.Add(-time.Nanosecond), expected: false},
		{name: "開始時刻ちょうどは表示する", now: startsAt, expected: true},
		{name: "終了直前は表示する", now: expiresAt.Add(-time.Nanosecond), expected: true},
		{name: "終了時刻ちょうどは表示しない", now: expiresAt, expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, announcement.IsActiveAt(tt.now))
		})
	}
}
