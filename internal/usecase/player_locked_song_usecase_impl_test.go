package usecase

import (
	"context"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/displayid"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/master"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type stubPlayerLockedSongMasterProvider struct {
	masters *masterdata.SongMasters
}

func (s *stubPlayerLockedSongMasterProvider) SongMasters() *masterdata.SongMasters {
	return s.masters
}

type stubPlayerLockedSongPlayerRepository struct {
	player *entity.Player
}

func (s *stubPlayerLockedSongPlayerRepository) FindByID(ctx context.Context, exec repository.Executor, id int) (*entity.Player, error) {
	return nil, nil
}

func (s *stubPlayerLockedSongPlayerRepository) FindByIDWithHonors(ctx context.Context, exec repository.Executor, id int) (*repository.PlayerWithHonors, error) {
	return nil, nil
}

func (s *stubPlayerLockedSongPlayerRepository) FindByUserID(ctx context.Context, exec repository.Executor, userID int) (*entity.Player, error) {
	return s.player, nil
}

func (s *stubPlayerLockedSongPlayerRepository) FindHonorsByPlayerID(ctx context.Context, exec repository.Executor, playerID int) ([]*entity.PlayerHonor, error) {
	return nil, nil
}

func (s *stubPlayerLockedSongPlayerRepository) UpdateCalculatedRatings(ctx context.Context, exec repository.Executor, playerID int, calculatedRating, bestAverage, newAverage float64) error {
	return nil
}

func (s *stubPlayerLockedSongPlayerRepository) Save(ctx context.Context, exec repository.Executor, player *entity.Player) error {
	return nil
}

func (s *stubPlayerLockedSongPlayerRepository) DeleteByUserID(ctx context.Context, exec repository.Executor, userID int) error {
	return nil
}

type spyPlayerLockedSongRepository struct {
	createCalled bool
}

func (s *spyPlayerLockedSongRepository) ListByPlayerID(ctx context.Context, exec repository.Executor, playerID int) ([]*entity.PlayerLockedSong, error) {
	return nil, nil
}

func (s *spyPlayerLockedSongRepository) Create(ctx context.Context, exec repository.Executor, lockedSong *entity.PlayerLockedSong) error {
	s.createCalled = true
	return nil
}

func (s *spyPlayerLockedSongRepository) Delete(ctx context.Context, exec repository.Executor, playerID int, songID int, isUltima bool) error {
	return nil
}

func TestPlayerLockedSongUltimaDifficulty(t *testing.T) {
	// Given
	expected := master.ChartDifficulty{ID: 99, Name: difficultyNameUltima, SortOrder: 4}
	u := &playerLockedSongUsecase{
		masterProvider: &stubPlayerLockedSongMasterProvider{
			masters: &masterdata.SongMasters{
				Difficulties: map[string]master.ChartDifficulty{
					difficultyNameUltima: expected,
				},
			},
		},
	}

	// When
	actual, err := u.ultimaDifficulty()

	// Then
	require.NoError(t, err)
	require.NotNil(t, actual)
	assert.Equal(t, expected, *actual)
}

func TestPlayerLockedSongLock(t *testing.T) {
	tests := []struct {
		name    string
		song    *entity.Song
		wantErr error
	}{
		{
			name:    "WORLD'S END楽曲は見つからない楽曲として扱う",
			song:    &entity.Song{ID: 1, DisplayID: "0123456789abcdef", IsWorldsend: true, Charts: []*entity.Chart{}},
			wantErr: repository.ErrSongNotFound,
		},
		{
			name:    "削除済み楽曲は見つからない楽曲として扱う",
			song:    &entity.Song{ID: 1, DisplayID: "0123456789abcdef", IsDeleted: true, Charts: []*entity.Chart{}},
			wantErr: repository.ErrSongNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			displayID, err := displayid.NewDisplayID("0123456789abcdef")
			require.NoError(t, err)
			songRepo := new(MockSongRepository)
			songRepo.On("FindByDisplayID", mock.Anything, mock.Anything, "0123456789abcdef").Return(tt.song, nil).Once()
			lockedRepo := &spyPlayerLockedSongRepository{}
			u := &playerLockedSongUsecase{
				playerRepo: &stubPlayerLockedSongPlayerRepository{player: &entity.Player{ID: 10}},
				songRepo:   songRepo,
				lockedRepo: lockedRepo,
			}

			// When
			err = u.Lock(context.Background(), 100, &PlayerLockedSongInput{DisplayID: displayID})

			// Then
			assert.ErrorIs(t, err, tt.wantErr)
			assert.False(t, lockedRepo.createCalled)
			songRepo.AssertExpectations(t)
		})
	}
}

func TestPlayerLockedSongDifficultyChart(t *testing.T) {
	tests := []struct {
		name       string
		chartIDs   []int
		difficulty master.ChartDifficulty
		expected   bool
	}{
		{
			name:       "対象難易度の譜面が存在する場合true",
			chartIDs:   []int{10, 20, 30},
			difficulty: master.ChartDifficulty{ID: 20, Name: difficultyNameUltima, SortOrder: 4},
			expected:   true,
		},
		{
			name:       "対象難易度の譜面が存在しない場合false",
			chartIDs:   []int{10, 20, 30},
			difficulty: master.ChartDifficulty{ID: 40, Name: difficultyNameUltima, SortOrder: 4},
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			song := &entity.Song{Charts: make([]*entity.Chart, 0, len(tt.chartIDs))}
			for _, chartID := range tt.chartIDs {
				song.Charts = append(song.Charts, &entity.Chart{DifficultyID: chartID})
			}

			// When
			actual := hasDifficultyChart(song, tt.difficulty)

			// Then
			assert.Equal(t, tt.expected, actual)
		})
	}
}
