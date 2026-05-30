package testutil

import (
	"context"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainmasterdata "github.com/chunisupport/chunisupport-api/internal/domain/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
)

// MockWorldsendUsecase は WORLD'S END 楽曲ユースケースのテスト用モックです。
type MockWorldsendUsecase struct {
	GetAllWorldsendSongsFunc        func(ctx context.Context, includeDeleted bool, requesterAccountTypeID *int) ([]*entity.WorldsendSongWithChart, error)
	GetWorldsendSongByDisplayIDFunc func(ctx context.Context, displayID string, requesterAccountTypeID *int) (*entity.WorldsendSongWithChart, error)
	DeleteWorldsendSongFunc         func(ctx context.Context, displayID string) error
	RestoreWorldsendSongFunc        func(ctx context.Context, displayID string) error
	UpdateWorldsendSongsFunc        func(ctx context.Context, requests []*usecase.UpdateWorldsendSongInput, masters *domainmasterdata.SongMasters) error
	CreateWorldsendSongFunc         func(ctx context.Context, input *usecase.CreateWorldsendSongInput, masters *domainmasterdata.SongMasters) (*entity.WorldsendSongWithChart, error)
}

func (m *MockWorldsendUsecase) GetAllWorldsendSongs(ctx context.Context, includeDeleted bool, requesterAccountTypeID *int) ([]*entity.WorldsendSongWithChart, error) {
	if m.GetAllWorldsendSongsFunc != nil {
		return m.GetAllWorldsendSongsFunc(ctx, includeDeleted, requesterAccountTypeID)
	}
	return nil, nil
}

func (m *MockWorldsendUsecase) GetWorldsendSongByDisplayID(ctx context.Context, displayID string, requesterAccountTypeID *int) (*entity.WorldsendSongWithChart, error) {
	if m.GetWorldsendSongByDisplayIDFunc != nil {
		return m.GetWorldsendSongByDisplayIDFunc(ctx, displayID, requesterAccountTypeID)
	}
	return nil, nil
}

func (m *MockWorldsendUsecase) DeleteWorldsendSong(ctx context.Context, displayID string) error {
	if m.DeleteWorldsendSongFunc != nil {
		return m.DeleteWorldsendSongFunc(ctx, displayID)
	}
	return nil
}

func (m *MockWorldsendUsecase) RestoreWorldsendSong(ctx context.Context, displayID string) error {
	if m.RestoreWorldsendSongFunc != nil {
		return m.RestoreWorldsendSongFunc(ctx, displayID)
	}
	return nil
}

func (m *MockWorldsendUsecase) UpdateWorldsendSongs(ctx context.Context, requests []*usecase.UpdateWorldsendSongInput, masters *domainmasterdata.SongMasters) error {
	if m.UpdateWorldsendSongsFunc != nil {
		return m.UpdateWorldsendSongsFunc(ctx, requests, masters)
	}
	return nil
}

func (m *MockWorldsendUsecase) CreateWorldsendSong(ctx context.Context, input *usecase.CreateWorldsendSongInput, masters *domainmasterdata.SongMasters) (*entity.WorldsendSongWithChart, error) {
	if m.CreateWorldsendSongFunc != nil {
		return m.CreateWorldsendSongFunc(ctx, input, masters)
	}
	return nil, nil
}
