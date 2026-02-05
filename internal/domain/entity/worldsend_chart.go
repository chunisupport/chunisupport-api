package entity

import (
	"fmt"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo/notes"
)

// WorldsendChart は WORLD'S END 譜面エンティティを表します。
// WORLD'S END は1曲1譜面が保証されています。
type WorldsendChart struct {
	ID      int
	SongID  int
	WeStar  *int    // 星の数（1～5）
	WeKanji *string // カテゴリ漢字（光、蔵、改、狂、etc.）
	Notes   *notes.Notes
}

// Validate は WorldsendChart のバリデーションを行います。
func (w *WorldsendChart) Validate() error {
	if w.WeStar != nil {
		if *w.WeStar < 1 || *w.WeStar > 5 {
			return fmt.Errorf("we_star: 星の数は1～5の範囲で指定してください")
		}
	}

	if w.Notes != nil {
		if _, err := notes.NewNotes(int(*w.Notes)); err != nil {
			return fmt.Errorf("notes: %w", err)
		}
	}

	return nil
}
