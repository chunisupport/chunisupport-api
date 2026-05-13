package masterdata

import (
	"context"
	"errors"
	"testing"

	domainmasterdata "github.com/chunisupport/chunisupport-api/internal/domain/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/master"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/ratingband"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type masterLoaderMock struct {
	dynamic *Cache
	static  *StaticCache
	err     error
}

func (m *masterLoaderMock) Load(_ context.Context) (*Cache, *StaticCache, error) {
	if m.err != nil {
		return nil, nil, m.err
	}
	return m.dynamic, m.static, nil
}

func TestRuntimeCache_リロードに成功した場合は最新スナップショットを返す(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "初期化後に再読み込みすると最新マスタを返す"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			firstDynamic := &Cache{Genres: map[string]master.Genre{"POPS": {ID: 1, Name: "POPS"}}}
			firstStatic := &StaticCache{RatingBands: []*ratingband.RatingBand{{ID: 1, Label: "A"}}}
			loader := &masterLoaderMock{dynamic: firstDynamic, static: firstStatic}

			runtime, err := NewRuntimeCache(context.Background(), loader)
			require.NoError(t, err)

			secondDynamic := &Cache{Genres: map[string]master.Genre{"GAME": {ID: 2, Name: "GAME"}}}
			secondStatic := &StaticCache{RatingBands: []*ratingband.RatingBand{{ID: 2, Label: "S"}}}
			loader.dynamic = secondDynamic
			loader.static = secondStatic

			err = runtime.Reload(context.Background())
			require.NoError(t, err)

			assert.Equal(t, secondDynamic, runtime.snapshot())
			assert.Equal(t, secondStatic, runtime.staticSnapshot())
			assert.Equal(t, secondDynamic.Genres, runtime.MasterDataMasters().Genres)
			assert.Equal(t, secondStatic.RatingBands, runtime.RatingBands())
		})
	}
}

func TestRuntimeCache_リロードに失敗した場合は旧データを維持する(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "ロード失敗時は直前のマスタを維持する"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldDynamic := &Cache{Genres: map[string]master.Genre{"POPS": {ID: 1, Name: "POPS"}}}
			oldStatic := &StaticCache{RatingBands: []*ratingband.RatingBand{{ID: 1, Label: "A"}}}
			loader := &masterLoaderMock{dynamic: oldDynamic, static: oldStatic}

			runtime, err := NewRuntimeCache(context.Background(), loader)
			require.NoError(t, err)

			loader.err = errors.New("db error")
			err = runtime.Reload(context.Background())
			require.Error(t, err)

			assert.Equal(t, oldDynamic, runtime.snapshot())
			assert.Equal(t, oldStatic, runtime.staticSnapshot())
		})
	}
}

func TestRuntimeCache_リロードでnilキャッシュが返された場合は旧データを維持する(t *testing.T) {
	oldDynamic := &Cache{Genres: map[string]master.Genre{"POPS": {ID: 1, Name: "POPS"}}}
	oldStatic := &StaticCache{RatingBands: []*ratingband.RatingBand{{ID: 1, Label: "A"}}}
	loader := &masterLoaderMock{dynamic: oldDynamic, static: oldStatic}

	runtime, err := NewRuntimeCache(context.Background(), loader)
	require.NoError(t, err)

	loader.dynamic = nil
	err = runtime.Reload(context.Background())
	require.Error(t, err)

	assert.Equal(t, oldDynamic, runtime.snapshot())
	assert.Equal(t, oldStatic, runtime.staticSnapshot())
}

func TestRuntimeCache_初期ロードに失敗した場合はエラーを返す(t *testing.T) {
	loader := &masterLoaderMock{err: errors.New("load error")}

	runtime, err := NewRuntimeCache(context.Background(), loader)

	require.Error(t, err)
	assert.Nil(t, runtime)
}

func TestRuntimeCache_既存インターフェースを満たす(t *testing.T) {
	dynamic := &Cache{
		Genres:               map[string]master.Genre{"POPS": {ID: 1, Name: "POPS"}},
		GenreNamesByID:       map[int]string{1: "POPS"},
		Difficulties:         map[string]master.ChartDifficulty{"MASTER": {ID: 4, Name: "MASTER"}},
		DifficultyNamesByID:  map[int]string{4: "MASTER"},
		AccountTypes:         map[string]master.AccountType{"ADMIN": {ID: 3, Name: "ADMIN"}},
		ClassEmblems:         map[string]master.ClassEmblem{},
		ClassEmblemBases:     map[string]master.ClassEmblemBase{},
		ClearLamps:           map[string]master.ClearLampType{},
		ComboLamps:           map[string]master.ComboLampType{},
		FullChains:           map[string]master.FullChainType{},
		Slots:                map[string]master.Slot{},
		HonorTypes:           map[string]master.HonorType{},
		AchievementTypes:     map[string]domainmasterdata.Item{},
		AchievementTypesByID: map[int]string{},
		VersionsByID:         map[int]Version{},
	}
	static := &StaticCache{RatingBands: []*ratingband.RatingBand{{ID: 1, Label: "A"}}}
	loader := &masterLoaderMock{dynamic: dynamic, static: static}
	runtime, err := NewRuntimeCache(context.Background(), loader)
	require.NoError(t, err)

	assert.Equal(t, "ADMIN", runtime.GetAccountTypeNameByID(3))
	assert.Equal(t, dynamic.SongMasters().Genres, runtime.SongMasters().Genres)
	assert.Equal(t, dynamic.PlayerDataMasters().Difficulties, runtime.PlayerDataMasters().Difficulties)
	assert.Equal(t, dynamic.GoalMasters().GenreNamesByID, runtime.GoalMasters().GenreNamesByID)
	assert.Equal(t, dynamic.MasterDataMasters().Genres, runtime.MasterDataMasters().Genres)
	assert.Equal(t, static.RatingBands, runtime.RatingBands())
}
