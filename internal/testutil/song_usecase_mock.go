package testutil

import (
	"context"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
)

// MockSongUsecase は楽曲ユースケースのテスト用モックです。
type MockSongUsecase struct {
	GetAllSongsExcludingWorldsendFunc func(ctx context.Context, includeDeleted bool, requesterAccountTypeID *int) ([]*entity.Song, error)
	GetSongByDisplayIDFunc            func(ctx context.Context, displayID string, requesterAccountTypeID *int) (*entity.Song, error)
	GetSongsUpdatedAtFunc             func(ctx context.Context) (*time.Time, error)
	DeleteSongFunc                    func(ctx context.Context, displayID string) error
	RestoreSongFunc                   func(ctx context.Context, displayID string) error
	UpdateSongsFunc                   func(ctx context.Context, requests []*api_internal.UpdateSongRequest) error
	CalcSongMaxOPFunc                 func(song *entity.Song) float64
}

func (m *MockSongUsecase) GetAllSongsExcludingWorldsend(ctx context.Context, includeDeleted bool, requesterAccountTypeID *int) ([]*entity.Song, error) {
	if m.GetAllSongsExcludingWorldsendFunc != nil {
		return m.GetAllSongsExcludingWorldsendFunc(ctx, includeDeleted, requesterAccountTypeID)
	}
	return nil, nil
}

func (m *MockSongUsecase) GetSongByDisplayID(ctx context.Context, displayID string, requesterAccountTypeID *int) (*entity.Song, error) {
	if m.GetSongByDisplayIDFunc != nil {
		return m.GetSongByDisplayIDFunc(ctx, displayID, requesterAccountTypeID)
	}
	return nil, nil
}

func (m *MockSongUsecase) GetSongsUpdatedAt(ctx context.Context) (*time.Time, error) {
	if m.GetSongsUpdatedAtFunc != nil {
		return m.GetSongsUpdatedAtFunc(ctx)
	}
	return nil, nil
}

func (m *MockSongUsecase) DeleteSong(ctx context.Context, displayID string) error {
	if m.DeleteSongFunc != nil {
		return m.DeleteSongFunc(ctx, displayID)
	}
	return nil
}

func (m *MockSongUsecase) RestoreSong(ctx context.Context, displayID string) error {
	if m.RestoreSongFunc != nil {
		return m.RestoreSongFunc(ctx, displayID)
	}
	return nil
}

func (m *MockSongUsecase) UpdateSongs(ctx context.Context, requests []*api_internal.UpdateSongRequest) error {
	if m.UpdateSongsFunc != nil {
		return m.UpdateSongsFunc(ctx, requests)
	}
	return nil
}

func (m *MockSongUsecase) CalcSongMaxOP(song *entity.Song) float64 {
	if m.CalcSongMaxOPFunc != nil {
		return m.CalcSongMaxOPFunc(song)
	}
	if song == nil {
		return 0
	}
	return 90
}
