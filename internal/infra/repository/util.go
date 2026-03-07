package repository

import (
	"fmt"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
)

// collectUniqueDisplayIDs は楽曲配列からdisplay_idを抽出し、重複がないことを検証します。
func collectUniqueDisplayIDs(songs []*entity.Song) ([]string, error) {
	displayIDs := make([]string, 0, len(songs))
	seen := make(map[string]struct{}, len(songs))
	for _, song := range songs {
		if _, exists := seen[song.DisplayID]; exists {
			return nil, fmt.Errorf("%w: display_id=%s", repository.ErrDuplicateDisplayID, song.DisplayID)
		}
		seen[song.DisplayID] = struct{}{}
		displayIDs = append(displayIDs, song.DisplayID)
	}
	return displayIDs, nil
}
