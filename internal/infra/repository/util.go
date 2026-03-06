package repository

import (
	"fmt"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
)

// collectUniqueDisplayIDs は楽曲配列からdisplay_idを抽出し、重複がないことを検証します。
func collectUniqueDisplayIDs(songs []*entity.Song) ([]string, error) {
	displayIDs := make([]string, len(songs))
	for i, song := range songs {
		displayIDs[i] = song.DisplayID
	}

	if err := validateUniqueDisplayIDs(displayIDs); err != nil {
		return nil, err
	}

	return displayIDs, nil
}

func validateUniqueDisplayIDs(displayIDs []string) error {
	seen := make(map[string]struct{}, len(displayIDs))
	for _, displayID := range displayIDs {
		if _, exists := seen[displayID]; exists {
			return fmt.Errorf("%w: display_id=%s", repository.ErrDuplicateDisplayID, displayID)
		}
		seen[displayID] = struct{}{}
	}

	return nil
}
