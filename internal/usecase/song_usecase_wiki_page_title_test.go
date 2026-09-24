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

func TestUpdateSongs_WikiPageTitle(t *testing.T) {
	tests := []struct {
		name string
		// Given: 更新入力
		updateWikiPageTitle bool
		wikiPageTitle       *string
		// Then: リポジトリへ渡す更新内容
		expectedUpdate bool
		expectedTitle  *string
	}{
		{
			name:                "更新指定がない場合は既存値維持としてリポジトリへ渡す",
			updateWikiPageTitle: false,
			wikiPageTitle:       nil,
			expectedUpdate:      false,
			expectedTitle:       nil,
		},
		{
			name:                "更新指定がある場合は指定値をリポジトリへ渡す",
			updateWikiPageTitle: true,
			wikiPageTitle:       new("楽曲(CHUNITHM)"),
			expectedUpdate:      true,
			expectedTitle:       new("楽曲(CHUNITHM)"),
		},
	}

	for _, tt := range tests {
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
				DisplayID:           "1234567890abcdef",
				Title:               "楽曲",
				Artist:              "アーティスト",
				WikiPageTitle:       tt.wikiPageTitle,
				UpdateWikiPageTitle: tt.updateWikiPageTitle,
			}})

			// Then
			require.NoError(t, err)
			require.Len(t, saved, 1)
			assert.Equal(t, tt.expectedUpdate, saved[0].UpdateWikiPageTitle)
			assert.Equal(t, tt.expectedTitle, saved[0].Song.WikiPageTitle)
		})
	}
}

func TestCreateSong_WikiPageTitle(t *testing.T) {
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
		OfficialIdx:   "123",
		Title:         "楽曲",
		Artist:        "アーティスト",
		Genre:         "POPS & ANIME",
		WikiPageTitle: new("楽曲(CHUNITHM)"),
	})

	// Then
	require.NoError(t, err)
	require.NotNil(t, created)
	assert.Equal(t, new("楽曲(CHUNITHM)"), created.WikiPageTitle)
}
