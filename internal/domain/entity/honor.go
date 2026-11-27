package entity

import (
	"strings"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo/displayid"
)

// Honor は称号マスタの1件を表します。
type Honor struct {
	// ID はDB内部の参照専用で、APIには公開しません。
	ID int
	// DisplayID はAPI上の id として公開する識別子です。
	DisplayID   displayid.DisplayID
	Name        string
	HonorTypeID int
	TypeName    string
	ImageURL    string
	CreatedAt   *time.Time
}

// NewHonor は保存可能な称号エンティティを生成します。
func NewHonor(displayID displayid.DisplayID, name string, honorTypeID int, typeName string, imageURL string) *Honor {
	return &Honor{
		DisplayID:   displayID,
		Name:        strings.TrimSpace(name),
		HonorTypeID: honorTypeID,
		TypeName:    strings.TrimSpace(typeName),
		ImageURL:    strings.TrimSpace(imageURL),
	}
}

// Rename は称号名を変更します。
func (h *Honor) Rename(name string) {
	h.Name = strings.TrimSpace(name)
}

// ChangeType は称号タイプを変更します。
func (h *Honor) ChangeType(honorTypeID int, typeName string) {
	h.HonorTypeID = honorTypeID
	h.TypeName = strings.TrimSpace(typeName)
}

// ChangeImageURL は称号画像URLを変更します。
func (h *Honor) ChangeImageURL(imageURL string) {
	h.ImageURL = strings.TrimSpace(imageURL)
}
