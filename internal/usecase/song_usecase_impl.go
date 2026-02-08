package usecase

import (
	"context"
	"fmt"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/notes"
	"github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
)

// songUsecaseImpl は SongUsecase の実装です。
type songUsecaseImpl struct {
	songRepo        repository.SongRepository
	masterCache     repository.SongMasterProvider
	tm              TransactionManager
	defaultExecutor repository.Executor
}

// NewSongService は新しい SongUsecase を生成します。
func NewSongService(
	songRepo repository.SongRepository,
	masterCache repository.SongMasterProvider,
	tm TransactionManager,
	defaultExecutor repository.Executor,
) SongUsecase {
	return &songUsecaseImpl{
		songRepo:        songRepo,
		masterCache:     masterCache,
		tm:              tm,
		defaultExecutor: defaultExecutor,
	}
}

// GetAllSongsExcludingWorldsend はWORLD'S END以外の全楽曲を取得します。
// includeDeletedがfalseの場合、削除済み楽曲は除外されます。
func (s *songUsecaseImpl) GetAllSongsExcludingWorldsend(ctx context.Context, includeDeleted bool) ([]*entity.Song, error) {
	return s.songRepo.FindAllExcludingWorldsend(ctx, s.defaultExecutor, includeDeleted)
}

// GetSongByDisplayID は指定されたDisplayIDの楽曲を取得します。
func (s *songUsecaseImpl) GetSongByDisplayID(ctx context.Context, displayID string) (*entity.Song, error) {
	// リポジトリ側で既にErrSongNotFoundに変換済み
	return s.songRepo.FindByDisplayID(ctx, s.defaultExecutor, displayID)
}

// DeleteSong は指定されたDisplayIDの楽曲を論理削除します。
func (s *songUsecaseImpl) DeleteSong(ctx context.Context, displayID string) error {
	return s.tm.Transactional(ctx, func(tx repository.Executor) error {
		// 楽曲の存在確認
		_, err := s.songRepo.FindByDisplayID(ctx, tx, displayID)
		if err != nil {
			return err
		}

		return s.songRepo.DeleteSong(ctx, tx, displayID)
	})
}

// RestoreSong は指定されたDisplayIDの楽曲を復活させます。
func (s *songUsecaseImpl) RestoreSong(ctx context.Context, displayID string) error {
	return s.tm.Transactional(ctx, func(tx repository.Executor) error {
		// 楽曲の存在確認
		_, err := s.songRepo.FindByDisplayID(ctx, tx, displayID)
		if err != nil {
			return err
		}

		return s.songRepo.RestoreSong(ctx, tx, displayID)
	})
}

// UpdateSongs は楽曲および譜面情報を一括更新します。
func (s *songUsecaseImpl) UpdateSongs(ctx context.Context, requests []*api_internal.UpdateSongRequest) error {
	if len(requests) == 0 {
		return nil
	}

	// マスターデータ検証
	masters := s.masterCache.SongMasters()
	if masters == nil {
		return fmt.Errorf("master cache is not initialized")
	}

	for _, req := range requests {
		// GenreID の検証
		if req.GenreID != nil {
			if _, ok := masters.GenreNamesByID[*req.GenreID]; !ok {
				return fmt.Errorf("invalid genre_id: %d (song: %s)", *req.GenreID, req.DisplayID)
			}
		}

		// DifficultyID の検証
		for _, chart := range req.Charts {
			if _, ok := masters.DifficultyNamesByID[chart.DifficultyID]; !ok {
				return fmt.Errorf("invalid difficulty_id: %d (song: %s)", chart.DifficultyID, req.DisplayID)
			}
		}
	}

	// DTOからエンティティへ変換
	songsWithCharts, err := s.convertRequestsToEntities(requests)
	if err != nil {
		return fmt.Errorf("failed to convert requests to entities: %w", err)
	}

	// トランザクション内でリポジトリに委譲
	return s.tm.Transactional(ctx, func(tx repository.Executor) error {
		return s.songRepo.UpdateSongs(ctx, tx, songsWithCharts)
	})
}

// convertRequestsToEntities はDTOリストからエンティティリストに変換します。
// IDフィールドは既存データの参照に使用されないため、0のままです。
func (s *songUsecaseImpl) convertRequestsToEntities(requests []*api_internal.UpdateSongRequest) ([]*entity.Song, error) {
	result := make([]*entity.Song, 0, len(requests))

	for _, req := range requests {
		song := &entity.Song{
			DisplayID:  req.DisplayID,
			Title:      req.Title,
			Artist:     req.Artist,
			GenreID:    req.GenreID,
			BPM:        req.BPM,
			ReleasedAt: req.ReleasedAt,
			Jacket:     req.Jacket,
		}

		charts := make([]*entity.Chart, 0, len(req.Charts))
		for _, chartReq := range req.Charts {
			cc, err := chartconstant.NewChartConstant(chartReq.Const)
			if err != nil {
				return nil, fmt.Errorf("invalid chart constant (song: %s, difficulty: %d): %w", req.DisplayID, chartReq.DifficultyID, err)
			}

			var notesVO *notes.Notes
			if chartReq.Notes != nil {
				n, err := notes.NewNotes(*chartReq.Notes)
				if err != nil {
					return nil, fmt.Errorf("invalid notes (song: %s, difficulty: %d): %w", req.DisplayID, chartReq.DifficultyID, err)
				}
				notesVO = &n
			}

			chart := &entity.Chart{
				DifficultyID:   chartReq.DifficultyID,
				Const:          cc,
				IsConstUnknown: chartReq.IsConstUnknown,
				Notes:          notesVO,
			}
			charts = append(charts, chart)
		}

		song.Charts = charts
		result = append(result, song)
	}

	return result, nil
}
