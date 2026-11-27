package models

import (
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// AppAnnouncementModel は期間限定のお知らせのデータベースモデルです。
type AppAnnouncementModel struct {
	ID        string    `db:"id"`
	Kind      string    `db:"kind"`
	Title     string    `db:"title"`
	Body      *string   `db:"body"`
	TargetURL *string   `db:"target_url"`
	StartsAt  time.Time `db:"starts_at"`
	ExpiresAt time.Time `db:"expires_at"`
	Revision  int       `db:"revision"`
}

// ToEntity は永続化モデルを検証済みのお知らせへ変換します。
func (m *AppAnnouncementModel) ToEntity() (*entity.AppAnnouncement, error) {
	return entity.RestoreAppAnnouncement(m.ID, m.Kind, m.Title, m.Body, m.TargetURL, m.StartsAt, m.ExpiresAt, m.Revision)
}

// FromAppAnnouncementEntity はお知らせを永続化モデルへ変換します。日時はUTCで保存します。
func FromAppAnnouncementEntity(announcement *entity.AppAnnouncement) *AppAnnouncementModel {
	return &AppAnnouncementModel{
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
