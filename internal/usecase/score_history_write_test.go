package usecase

import (
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/master"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildStandardHistories(t *testing.T) {
	now := time.Date(2026, 6, 28, 0, 0, 0, 0, time.UTC)
	masters := &playerDataMaster{
		PlayerDataMasters: &masterdata.PlayerDataMasters{
			CommonMasters: masterdata.CommonMasters{
				DifficultyNamesByID: map[int]string{1: "BASIC", 3: "EXPERT"},
			},
			Difficulties: map[string]master.ChartDifficulty{},
		},
		chartsByID: map[int]entity.PlayerDataChart{
			10: {ID: 10, DifficultyID: 3},
			11: {ID: 11, DifficultyID: 1},
		},
	}
	before := map[int]repository.PlayerRecordState{
		10: {Score: 900000, UpdatedAt: now},
		11: {Score: 900000, UpdatedAt: now},
	}
	after := []repository.PlayerRecordForUpsert{
		{ChartID: 10, State: repository.PlayerRecordState{Score: 950000}},
		{ChartID: 11, State: repository.PlayerRecordState{Score: 950000}},
		{ChartID: 12, State: repository.PlayerRecordState{Score: 950000}},
	}

	rows, chartIDs := buildStandardHistories(1, before, after, masters)

	require.Len(t, rows, 1)
	assert.Equal(t, 10, rows[0].ChartID)
	assert.Equal(t, before[10], rows[0].State)
	assert.Equal(t, []int{10}, chartIDs)
}

func TestBuildWorldsendHistories_意味のある更新だけ退避する(t *testing.T) {
	now := time.Date(2026, 6, 28, 0, 0, 0, 0, time.UTC)
	before := map[int]repository.WorldsendRecordState{
		20: {Score: 900000, UpdatedAt: now},
		21: {Score: 900000, UpdatedAt: now},
	}
	after := []repository.WorldsendRecordForUpsert{
		{ChartID: 20, State: repository.WorldsendRecordState{Score: 950000}},
		{ChartID: 21, State: repository.WorldsendRecordState{Score: 900000, UpdatedAt: now.Add(time.Hour)}},
		{ChartID: 22, State: repository.WorldsendRecordState{Score: 950000}},
	}

	rows, chartIDs := buildWorldsendHistories(1, before, after)

	require.Len(t, rows, 1)
	assert.Equal(t, 20, rows[0].WorldsendChartID)
	assert.Equal(t, before[20], rows[0].State)
	assert.Equal(t, []int{20}, chartIDs)
}

type stubScoreHistoryMasterProvider struct {
	masters *masterdata.PlayerDataMasters
}

func (s *stubScoreHistoryMasterProvider) PlayerDataMasters() *masterdata.PlayerDataMasters {
	return s.masters
}

func TestScoreHistoryUsecase_ランプ名のnone表記を大文字小文字によらずnullにする(t *testing.T) {
	us := &scoreHistoryUsecase{
		masterProvider: &stubScoreHistoryMasterProvider{
			masters: &masterdata.PlayerDataMasters{
				ClearLampNamesByID: map[int]string{1: "none"},
				ComboLampNamesByID: map[int]string{1: "NoNe"},
				FullChainNamesByID: map[int]string{1: "NONE"},
			},
		},
	}

	entries, err := us.convertEntries([]entity.ScoreHistoryEntry{{
		Score: 900000, ClearLampID: 1, ComboLampID: 1, FullChainID: 1,
	}})

	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Nil(t, entries[0].ClearLamp)
	assert.Nil(t, entries[0].ComboLamp)
	assert.Nil(t, entries[0].FullChain)
}
