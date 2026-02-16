package usecase

import (
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
)

func TestToWorldsendSongDTO_ReleaseDateIsFormattedAsDate(t *testing.T) {
	genreID := 1
	bpm := 180
	jacket := "jacket.png"
	releasedAt := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	input := &repository.WorldsendSongWithChart{
		Song: &entity.Song{
			DisplayID:   "0123456789abcdef",
			Title:       "test song",
			Artist:      "test artist",
			GenreID:     &genreID,
			BPM:         &bpm,
			ReleasedAt:  &releasedAt,
			OfficialIdx: "123",
			Jacket:      &jacket,
			IsDeleted:   false,
		},
	}

	result := toWorldsendSongDTO(input)
	if result.ReleasedAt == nil {
		t.Fatal("expected released_at to be set")
	}

	if *result.ReleasedAt != "2024-01-15" {
		t.Fatalf("expected released_at to be 2024-01-15, got %s", *result.ReleasedAt)
	}
}

func TestToWorldsendSongDTO_ReleaseDateCanBeNil(t *testing.T) {
	input := &repository.WorldsendSongWithChart{
		Song: &entity.Song{
			DisplayID:   "0123456789abcdef",
			Title:       "test song",
			Artist:      "test artist",
			OfficialIdx: "123",
		},
	}

	result := toWorldsendSongDTO(input)
	if result.ReleasedAt != nil {
		t.Fatalf("expected released_at to be nil, got %s", *result.ReleasedAt)
	}
}
