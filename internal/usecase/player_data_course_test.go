package usecase

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateCourseScoreEntry(t *testing.T) {
	tests := []struct {
		name    string
		entry   PlayerDataCourseEntry
		wantErr bool
	}{
		{name: "通常のクリア記録", entry: PlayerDataCourseEntry{Idx: "50020", Score: 3023238, IsClear: true, ComboLv: 1}},
		{name: "AJでも強制終了は許容", entry: PlayerDataCourseEntry{Idx: "50020", Score: 3023238, IsClear: false, ComboLv: 3}},
		{name: "0点未クリアは許容", entry: PlayerDataCourseEntry{Idx: "50020", Score: 0, IsClear: false, ComboLv: 1}},
		{name: "0点クリアは不正", entry: PlayerDataCourseEntry{Idx: "50020", Score: 0, IsClear: true, ComboLv: 1}, wantErr: true},
		{name: "理論値はAJ必須", entry: PlayerDataCourseEntry{Idx: "50020", Score: 3030000, IsClear: true, ComboLv: 2}, wantErr: true},
		{name: "AJは300万点以上", entry: PlayerDataCourseEntry{Idx: "50020", Score: 2999999, IsClear: false, ComboLv: 3}, wantErr: true},
		{name: "上限超過", entry: PlayerDataCourseEntry{Idx: "50020", Score: 3030001, ComboLv: 1}, wantErr: true},
		{name: "idx必須", entry: PlayerDataCourseEntry{Score: 100, ComboLv: 1}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCourseScoreEntry(tt.entry, 0)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
