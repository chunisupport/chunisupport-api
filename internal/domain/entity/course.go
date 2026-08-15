package entity

import (
	"errors"
	"strings"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo/displayid"
)

// CourseClass はコース固有のクラスマスタです。
type CourseClass struct {
	ID        int
	Name      string
	SortOrder int
}

// Course はコースマスタを表します。
type Course struct {
	ID            int
	DisplayID     displayid.DisplayID
	OfficialIdx   string
	Name          string
	CourseClassID int
	IsDeleted     bool
	UpdatedAt     time.Time
	CourseClass   *CourseClass
}

// Validate はコースが常に有効な識別子と名称を持つことを保証します。
func (c *Course) Validate() error {
	if !c.DisplayID.IsValid() {
		return errors.New("display_idは16文字の小文字16進数である必要があります")
	}
	if strings.TrimSpace(c.OfficialIdx) == "" {
		return errors.New("official_idxは必須です")
	}
	if strings.TrimSpace(c.Name) == "" {
		return errors.New("nameは必須です")
	}
	if c.CourseClassID <= 0 {
		return errors.New("course_class_idは正の整数である必要があります")
	}
	return nil
}

func (c *Course) Delete()  { c.IsDeleted = true }
func (c *Course) Restore() { c.IsDeleted = false }
