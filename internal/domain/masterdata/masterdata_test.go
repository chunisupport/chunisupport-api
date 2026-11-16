package masterdata_test

import (
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/master"
	"github.com/stretchr/testify/assert"
)

func TestSongMasters_DifficultySortOrderByID(t *testing.T) {
	tests := []struct {
		name string
		// Given
		masters *masterdata.SongMasters
		// Then
		expected map[int]int
	}{
		{
			name: "難易度が存在する場合、ID→SortOrderのマップが返される",
			masters: &masterdata.SongMasters{
				Difficulties: map[string]master.ChartDifficulty{
					"BASIC":    {ID: 1, Name: "BASIC", SortOrder: 0},
					"ADVANCED": {ID: 2, Name: "ADVANCED", SortOrder: 1},
					"EXPERT":   {ID: 3, Name: "EXPERT", SortOrder: 2},
					"MASTER":   {ID: 4, Name: "MASTER", SortOrder: 3},
					"ULTIMA":   {ID: 5, Name: "ULTIMA", SortOrder: 4},
				},
			},
			expected: map[int]int{1: 0, 2: 1, 3: 2, 4: 3, 5: 4},
		},
		{
			name:     "Difficultiesが空の場合はnilが返される",
			masters:  &masterdata.SongMasters{},
			expected: nil,
		},
		{
			name:     "SongMasters自体がnilの場合はnilが返される",
			masters:  nil,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			result := tt.masters.DifficultySortOrderByID()

			// Then
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPlayerDataMasters_ProfileNames(t *testing.T) {
	masters := &masterdata.PlayerDataMasters{
		ClassEmblems:     map[string]master.ClassEmblem{"MEDAL_V": {ID: 5, Name: "MEDAL_V"}},
		ClassEmblemBases: map[string]master.ClassEmblemBase{"BASE_IV": {ID: 4, Name: "BASE_IV"}},
		Possessions: map[string]master.Possession{
			"normal": {ID: 1, Name: "normal"},
			"gold":   {ID: 3, Name: "gold"},
		},
	}
	classEmblemID := 5
	classEmblemBaseID := 4
	unknownID := 99
	classEmblem := "MEDAL_V"
	classEmblemBase := "BASE_IV"

	tests := []struct {
		name string
		// Given
		classEmblemID     *int
		classEmblemBaseID *int
		possessionID      int
		// Then
		expected masterdata.ProfileMasterNames
	}{
		{
			name:              "各マスタIDが名称へ解決される",
			classEmblemID:     &classEmblemID,
			classEmblemBaseID: &classEmblemBaseID,
			possessionID:      3,
			expected:          masterdata.ProfileMasterNames{ClassEmblem: &classEmblem, ClassEmblemBase: &classEmblemBase, Possession: "gold"},
		},
		{
			name:              "未設定・未知のIDはエンブレム名を持たず、ポゼッションはnormalになる",
			classEmblemID:     nil,
			classEmblemBaseID: &unknownID,
			possessionID:      99,
			expected:          masterdata.ProfileMasterNames{Possession: "normal"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			got := masters.ProfileNames(tt.classEmblemID, tt.classEmblemBaseID, tt.possessionID)

			// Then
			assert.Equal(t, tt.expected, got)
		})
	}
}
