package api_internal

import (
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// AppAnnouncementResponse は配信中のお知らせ1件のレスポンスです。
// 本文型は body、リンク型は target_url だけが値を持ち、もう一方は null です。
type AppAnnouncementResponse struct {
	ID        string    `json:"id"`
	Kind      string    `json:"kind"`
	Title     string    `json:"title"`
	Body      *string   `json:"body"`
	TargetURL *string   `json:"target_url"`
	StartsAt  time.Time `json:"starts_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Revision  int       `json:"revision"`
}

// AppAnnouncementsResponse は配信中のお知らせ一覧のレスポンスです。
type AppAnnouncementsResponse struct {
	Announcements []*AppAnnouncementResponse `json:"announcements"`
}

// AppAnnouncementRequest は管理画面からお知らせを追加するリクエストです。
// 本文型は target_url、リンク型は body を null にします。
type AppAnnouncementRequest struct {
	Kind      string    `json:"kind"`
	Title     string    `json:"title"`
	Body      *string   `json:"body"`
	TargetURL *string   `json:"target_url"`
	StartsAt  time.Time `json:"starts_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// UpdateAppAnnouncementRequest は管理画面からお知らせを更新するリクエストです。
type UpdateAppAnnouncementRequest struct {
	AppAnnouncementRequest
	// BumpRevision が true の場合は revision を1つ増やし、閉じた利用者にも再表示させます。
	BumpRevision bool `json:"bump_revision"`
}

// ToAppAnnouncementResponse はお知らせをレスポンスへ変換します。
// 取得時と登録・更新時で表記を揃えるため、日時はUTCで返します。
func ToAppAnnouncementResponse(announcement *entity.AppAnnouncement) *AppAnnouncementResponse {
	return &AppAnnouncementResponse{
		ID:        announcement.DisplayID.String(),
		Kind:      string(announcement.Kind),
		Title:     announcement.Title,
		Body:      announcement.Body,
		TargetURL: announcement.TargetURL,
		StartsAt:  announcement.StartsAt.UTC(),
		ExpiresAt: announcement.ExpiresAt.UTC(),
		Revision:  announcement.Revision,
	}
}

// ToAppAnnouncementsResponse はお知らせ一覧をレスポンスへ変換します。
func ToAppAnnouncementsResponse(announcements []*entity.AppAnnouncement) *AppAnnouncementsResponse {
	items := make([]*AppAnnouncementResponse, 0, len(announcements))
	for _, announcement := range announcements {
		items = append(items, ToAppAnnouncementResponse(announcement))
	}
	return &AppAnnouncementsResponse{Announcements: items}
}
