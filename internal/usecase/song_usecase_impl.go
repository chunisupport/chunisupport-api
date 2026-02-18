package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainmasterdata "github.com/chunisupport/chunisupport-api/internal/domain/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/service"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/notes"
	"github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/info"
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
// includeDeleted が true かつ requesterAccountTypeID が EDITOR 未満の場合、削除済み楽曲は除外されます。
func (s *songUsecaseImpl) GetAllSongsExcludingWorldsend(ctx context.Context, includeDeleted bool, requesterAccountTypeID *int) ([]*entity.Song, error) {
	// 削除済み楽曲を含める場合はEDITOR権限が必要
	if includeDeleted {
		if requesterAccountTypeID == nil || *requesterAccountTypeID < info.AccountTypeEditor {
			includeDeleted = false
		}
	}

	return s.songRepo.FindAllExcludingWorldsend(ctx, s.defaultExecutor, includeDeleted)
}

// GetSongByDisplayID は指定されたDisplayIDの楽曲を取得します。
// requesterAccountTypeIDがnilまたはEDITOR(2)未満の場合、削除済み楽曲はErrSongNotFoundを返します。
func (s *songUsecaseImpl) GetSongByDisplayID(ctx context.Context, displayID string, requesterAccountTypeID *int) (*entity.Song, error) {
	song, err := s.songRepo.FindByDisplayID(ctx, s.defaultExecutor, displayID)
	if err != nil {
		return nil, err
	}

	// 削除済み楽曲の権限チェック
	if song.IsDeleted {
		// EDITOR以上の権限を持たない場合は404を返す
		if requesterAccountTypeID == nil || *requesterAccountTypeID < info.AccountTypeEditor {
			return nil, repository.ErrSongNotFound
		}
	}

	return song, nil
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

	// DTOからエンティティへ変換
	songsWithCharts, err := s.convertRequestsToEntities(requests, masters)
	if err != nil {
		return fmt.Errorf("failed to convert requests to entities: %w", err)
	}

	// トランザクション内でリポジトリに委譲
	return s.tm.Transactional(ctx, func(tx repository.Executor) error {
		return s.songRepo.UpdateSongs(ctx, tx, songsWithCharts)
	})
}

// CalcSongMaxOP は楽曲の最大譜面定数から理論値の最大OPを計算します。
// MaxChartConst はドメインサービスにより譜面集約で設定済みです。
func (s *songUsecaseImpl) CalcSongMaxOP(song *entity.Song) float64 {
	if song == nil {
		return 0
	}

	return service.CalcSongMaxOP(song.MaxChartConst)
}

// convertRequestsToEntities はDTOリストからエンティティリストに変換します。
// IDフィールドは既存データの参照に使用されないため、0のままです。
func (s *songUsecaseImpl) convertRequestsToEntities(requests []*api_internal.UpdateSongRequest, masters *domainmasterdata.SongMasters) ([]*entity.Song, error) {
	result := make([]*entity.Song, 0, len(requests))

	for _, req := range requests {
		var genreID *int
		if req.Genre != nil {
			// ジャンル名の検証とID変換
			if item, ok := masters.Genres[*req.Genre]; ok {
				genreID = &item.ID
			} else {
				return nil, fmt.Errorf("invalid genre: %s (song: %s)", *req.Genre, req.DisplayID)
			}
		}

		song := &entity.Song{
			DisplayID:  req.DisplayID,
			Title:      req.Title,
			Artist:     req.Artist,
			GenreID:    genreID,
			BPM:        req.BPM,
			ReleasedAt: req.ReleasedAt.TimePtr(),
			Jacket:     req.Jacket,
		}

		charts := make([]*entity.Chart, 0, len(req.Charts))
		for diffName, chartReq := range req.Charts {
			// 難易度名の検証とID変換（大文字に変換してチェック）
			diffKey := strings.ToUpper(diffName)
			item, ok := masters.Difficulties[diffKey]
			if !ok {
				return nil, fmt.Errorf("invalid difficulty: %s (song: %s)", diffName, req.DisplayID)
			}
			difficultyID := item.ID

			cc, err := chartconstant.NewChartConstant(chartReq.Const)
			if err != nil {
				return nil, fmt.Errorf("invalid chart constant (song: %s, difficulty: %s): %w", req.DisplayID, diffName, err)
			}

			var notesVO *notes.Notes
			if chartReq.Notes != nil {
				n, err := notes.NewNotes(*chartReq.Notes)
				if err != nil {
					return nil, fmt.Errorf("invalid notes (song: %s, difficulty: %s): %w", req.DisplayID, diffName, err)
				}
				notesVO = &n
			}

			chart := &entity.Chart{
				DifficultyID:   difficultyID,
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
