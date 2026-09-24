package usecase

import (
	"context"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainmasterdata "github.com/chunisupport/chunisupport-api/internal/domain/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/master"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestUpdateWorldsendSongs_WikiPageTitle(t *testing.T) {
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
			wikiPageTitle:       new("楽曲(WORLD'S END)"),
			expectedUpdate:      true,
			expectedTitle:       new("楽曲(WORLD'S END)"),
		},
	}

	for _, tt := range tests {
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
				DisplayID:           "1234567890abcdef",
				Title:               "楽曲",
				Artist:              "アーティスト",
				WikiPageTitle:       tt.wikiPageTitle,
				UpdateWikiPageTitle: tt.updateWikiPageTitle,
			}}, &domainmasterdata.SongMasters{})

			// Then
			require.NoError(t, err)
			require.Len(t, saved, 1)
			assert.Equal(t, tt.expectedUpdate, saved[0].UpdateWikiPageTitle)
			assert.Equal(t, tt.expectedTitle, saved[0].Song.WikiPageTitle)
		})
	}
}

func TestCreateWorldsendSong_WikiPageTitle(t *testing.T) {
	// Given
	mockRepo := new(MockWorldsendChartRepository)
	mockExec := new(MockExecutor)
	uc := newWorldsendUsecaseForTest(mockRepo, &passthroughTransactionManager{tx: mockExec}, mockExec)
	masters := &domainmasterdata.SongMasters{Genres: map[string]master.Genre{"POPS & ANIME": {ID: 1, Name: "POPS & ANIME"}}}
	var created *entity.Song
	mockRepo.On("CreateSong", mock.Anything, mockExec, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) { created = args.Get(2).(*entity.Song) }).
		Return(&entity.WorldsendSongWithChart{}, nil)

	// When
	_, err := uc.CreateWorldsendSong(context.Background(), &CreateWorldsendSongInput{
		OfficialIdx:   "123",
		Title:         "楽曲",
		Artist:        "アーティスト",
		Genre:         "POPS & ANIME",
		WikiPageTitle: new("楽曲(WORLD'S END)"),
	}, masters)

	// Then
	require.NoError(t, err)
	require.NotNil(t, created)
	assert.Equal(t, new("楽曲(WORLD'S END)"), created.WikiPageTitle)
}
