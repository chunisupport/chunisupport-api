package service

import (
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

func TestRecordCompletionService_CompletePlayerRecords(t *testing.T) {
	svc := NewRecordCompletionService()

	songs := []*entity.Song{
		{
			ID: 1,
			Charts: []*entity.Chart{
				{ID: 11, SongID: 1, DifficultyID: 3},
				{ID: 12, SongID: 1, DifficultyID: 4},
			},
		},
		{
			ID: 2,
			Charts: []*entity.Chart{
				{ID: 21, SongID: 2, DifficultyID: 2},
			},
		},
		{
			ID:        3,
			IsDeleted: true,
			Charts: []*entity.Chart{
				{ID: 31, SongID: 3, DifficultyID: 4},
			},
		},
	}

	records := []*entity.PlayerRecord{
		{
			ChartID: 11,
			Song:    songs[0],
			Chart:   songs[0].Charts[0],
			ChartDifficulty: &entity.ChartDifficulty{
				ID:   3,
				Name: "expert",
			},
		},
	}

	completed := svc.CompletePlayerRecords(records, songs, map[int]string{2: "ADVANCED", 4: "MASTER"})

	if len(completed) != 3 {
		t.Fatalf("expected 3 records, got %d", len(completed))
	}
	if completed[0].ChartID != 11 || completed[1].ChartID != 12 || completed[2].ChartID != 21 {
		t.Fatalf("unexpected sorted chart ids: %d, %d, %d", completed[0].ChartID, completed[1].ChartID, completed[2].ChartID)
	}
	if completed[1].ChartDifficulty == nil || completed[1].ChartDifficulty.Name != "MASTER" {
		t.Fatalf("expected completed difficulty MASTER, got %+v", completed[1].ChartDifficulty)
	}
	if completed[2].ChartDifficulty == nil || completed[2].ChartDifficulty.Name != "ADVANCED" {
		t.Fatalf("expected completed difficulty ADVANCED, got %+v", completed[2].ChartDifficulty)
	}
}

func TestRecordCompletionService_CompleteWorldsendRecords(t *testing.T) {
	svc := NewRecordCompletionService()

	song1 := &entity.Song{ID: 1}
	song2 := &entity.Song{ID: 2}
	song3 := &entity.Song{ID: 3, IsDeleted: true}
	chart1 := &entity.WorldsendChart{ID: 101, SongID: 1}
	chart2 := &entity.WorldsendChart{ID: 102, SongID: 2}
	chart3 := &entity.WorldsendChart{ID: 103, SongID: 3}

	records := []*entity.PlayerWorldsendRecord{
		{WorldsendChartID: 101, Song: song1, WorldsendChart: chart1},
	}
	songCharts := []*WorldsendSongChartPair{
		{Song: song1, Chart: chart1},
		{Song: song2, Chart: chart2},
		{Song: song3, Chart: chart3},
	}

	completed := svc.CompleteWorldsendRecords(records, songCharts)

	if len(completed) != 2 {
		t.Fatalf("expected 2 records, got %d", len(completed))
	}
	if completed[0].WorldsendChartID != 101 || completed[1].WorldsendChartID != 102 {
		t.Fatalf("unexpected sorted worldsend chart ids: %d, %d", completed[0].WorldsendChartID, completed[1].WorldsendChartID)
	}
}
