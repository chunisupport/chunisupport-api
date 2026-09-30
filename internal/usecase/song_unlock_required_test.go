package usecase

import (
	"context"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/master"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// unlockRequiredUpdateCases は楽曲更新における要解禁フラグの受け渡しケースです。
// 通常楽曲と WORLD'S END 楽曲で同じ契約を検証するため共通化しています。
var unlockRequiredUpdateCases = []struct {
	name string
	// Given: 更新入力。Then: 同じ値をリポジトリへ渡す
	unlockRequired *bool
}{
	{name: "指定がない場合は既存値維持としてリポジトリへ渡す", unlockRequired: nil},
	{name: "trueの場合は要解禁への更新としてリポジトリへ渡す", unlockRequired: new(true)},
	{name: "falseの場合は解禁不要への更新としてリポジトリへ渡す", unlockRequired: new(false)},
}

func TestUpdateSongs_UnlockRequired(t *testing.T) {
	for _, tt := range unlockRequiredUpdateCases {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			mockRepo := new(MockSongRepository)
			mockMasterCache := new(MockSongMasterProvider)
			mockExec := new(MockExecutor)
			mockMasterCache.On("SongMasters").Return(&masterdata.SongMasters{})
			var saved []*repository.SongUpdate
			mockRepo.On("UpdateSongs", mock.Anything, mockExec, mock.Anything).
				Run(func(args mock.Arguments) { saved = args.Get(2).([]*repository.SongUpdate) }).
				Return(nil)
			uc := NewSongUsecase(mockRepo, mockMasterCache, &passthroughTransactionManager{tx: mockExec}, mockExec)

			// When
			err := uc.UpdateSongs(context.Background(), []*UpdateSongInput{{
				DisplayID:      "1234567890abcdef",
				Title:          "楽曲",
				Artist:         "アーティスト",
				UnlockRequired: tt.unlockRequired,
			}})

			// Then
			require.NoError(t, err)
			require.Len(t, saved, 1)
			assert.Equal(t, tt.unlockRequired, saved[0].UnlockRequired)
		})
	}
}

func TestUpdateWorldsendSongs_UnlockRequired(t *testing.T) {
	for _, tt := range unlockRequiredUpdateCases {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			mockRepo := new(MockWorldsendChartRepository)
			mockExec := new(MockExecutor)
			uc := newWorldsendUsecaseForTest(mockRepo, &passthroughTransactionManager{tx: mockExec}, mockExec)
			var saved []*repository.WorldsendUpdate
			mockRepo.On("UpdateSongs", mock.Anything, mockExec, mock.Anything).
				Run(func(args mock.Arguments) { saved = args.Get(2).([]*repository.WorldsendUpdate) }).
				Return(nil)

			// When
			err := uc.UpdateWorldsendSongs(context.Background(), []*UpdateWorldsendSongInput{{
				DisplayID:      "1234567890abcdef",
				Title:          "楽曲",
				Artist:         "アーティスト",
				UnlockRequired: tt.unlockRequired,
			}}, &masterdata.SongMasters{})

			// Then
			require.NoError(t, err)
			require.Len(t, saved, 1)
			assert.Equal(t, tt.unlockRequired, saved[0].UnlockRequired)
		})
	}
}

func TestCreateSong_UnlockRequired(t *testing.T) {
	// Given
	mockRepo := new(MockSongRepository)
	mockMasterCache := new(MockSongMasterProvider)
	mockExec := new(MockExecutor)
	mockMasterCache.On("SongMasters").Return(&masterdata.SongMasters{
		Genres: map[string]master.Genre{"POPS & ANIME": {ID: 1, Name: "POPS & ANIME"}},
	})
	var created *entity.Song
	mockRepo.On("Create", mock.Anything, mockExec, mock.Anything).
		Run(func(args mock.Arguments) { created = args.Get(2).(*entity.Song) }).
		Return(&entity.Song{}, nil)
	uc := NewSongUsecase(mockRepo, mockMasterCache, &passthroughTransactionManager{tx: mockExec}, mockExec)

	// When
	_, err := uc.CreateSong(context.Background(), &CreateSongInput{
		OfficialIdx:    "123",
		Title:          "楽曲",
		Artist:         "アーティスト",
		Genre:          "POPS & ANIME",
		UnlockRequired: true,
	})

	// Then
	require.NoError(t, err)
	require.NotNil(t, created)
	assert.True(t, created.UnlockRequired)
}

func TestCreateWorldsendSong_UnlockRequired(t *testing.T) {
	// Given
	mockRepo := new(MockWorldsendChartRepository)
	mockExec := new(MockExecutor)
	uc := newWorldsendUsecaseForTest(mockRepo, &passthroughTransactionManager{tx: mockExec}, mockExec)
	masters := &masterdata.SongMasters{Genres: map[string]master.Genre{"POPS & ANIME": {ID: 1, Name: "POPS & ANIME"}}}
	var created *entity.Song
	mockRepo.On("CreateSong", mock.Anything, mockExec, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) { created = args.Get(2).(*entity.Song) }).
		Return(&entity.WorldsendSongWithChart{}, nil)

	// When
	_, err := uc.CreateWorldsendSong(context.Background(), &CreateWorldsendSongInput{
		OfficialIdx:    "123",
		Title:          "楽曲",
		Artist:         "アーティスト",
		Genre:          "POPS & ANIME",
		UnlockRequired: true,
	}, masters)

	// Then
	require.NoError(t, err)
	require.NotNil(t, created)
	assert.True(t, created.UnlockRequired)
}
