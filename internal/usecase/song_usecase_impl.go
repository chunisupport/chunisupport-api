package usecase

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

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
// includeDeleted が true かつ requesterAccountTypeID が EDITOR 権限を満たさない場合、削除済み楽曲は除外されます。
func (s *songUsecaseImpl) GetAllSongsExcludingWorldsend(ctx context.Context, includeDeleted bool, requesterAccountTypeID *int) ([]*entity.Song, error) {
	// 削除済み楽曲を含める場合はEDITOR権限が必要
	if includeDeleted {
		if requesterAccountTypeID == nil || !info.HasRole(*requesterAccountTypeID, info.AccountTypeEditor) {
			includeDeleted = false
		}
	}

	return s.songRepo.FindAllExcludingWorldsend(ctx, s.defaultExecutor, includeDeleted)
}

// GetSongByDisplayID は指定されたDisplayIDの楽曲を取得します。
// requesterAccountTypeIDがnilまたはEDITOR権限を満たさない場合、削除済み楽曲はErrSongNotFoundを返します。
func (s *songUsecaseImpl) GetSongByDisplayID(ctx context.Context, displayID string, requesterAccountTypeID *int) (*entity.Song, error) {
	song, err := s.songRepo.FindByDisplayID(ctx, s.defaultExecutor, displayID)
	if err != nil {
		return nil, err
	}

	// 削除済み楽曲の権限チェック
	if !song.IsActive() {
		// EDITOR以上の権限を持たない場合は404を返す
		if requesterAccountTypeID == nil || !info.HasRole(*requesterAccountTypeID, info.AccountTypeEditor) {
			return nil, repository.ErrSongNotFound
		}
	}

	return song, nil
}

// GetSongsUpdatedAt は楽曲関連データの updated_at の最大値を取得します。
func (s *songUsecaseImpl) GetSongsUpdatedAt(ctx context.Context) (*time.Time, error) {
	return s.songRepo.FindLatestUpdatedAt(ctx, s.defaultExecutor)
}

// DeleteSong は指定されたDisplayIDの楽曲を論理削除します。
func (s *songUsecaseImpl) DeleteSong(ctx context.Context, displayID string) error {
	return s.tm.Transactional(ctx, func(tx repository.Executor) error {
		song, err := s.songRepo.FindByDisplayID(ctx, tx, displayID)
		if err != nil {
			return err
		}

		song.Delete()
		return s.songRepo.Save(ctx, tx, song)
	})
}

// RestoreSong は指定されたDisplayIDの楽曲を復活させます。
func (s *songUsecaseImpl) RestoreSong(ctx context.Context, displayID string) error {
	return s.tm.Transactional(ctx, func(tx repository.Executor) error {
		song, err := s.songRepo.FindByDisplayID(ctx, tx, displayID)
		if err != nil {
			return err
		}

		song.Restore()
		return s.songRepo.Save(ctx, tx, song)
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

		song := entity.NewSong()
		song.DisplayID = req.DisplayID
		song.Title = req.Title
		song.Artist = req.Artist
		song.GenreID = genreID
		song.BPM = req.BPM
		song.ReleasedAt = req.ReleasedAt.TimePtr()
		song.Jacket = req.Jacket

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
				NotesDesigner:  chartReq.NotesDesigner,
			}
			charts = append(charts, chart)
		}

		song.Charts = charts
		result = append(result, song)
	}

	return result, nil
}

// generateDisplayID は crypto/rand で 8 バイトの乱数を生成し、16 文字の小文字 16 進文字列を返します。
func generateDisplayID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate display_id: %w", err)
	}

	return hex.EncodeToString(b), nil
}

// CreateSong は新規楽曲を追加します。
// display_id はサーバー側で生成します。
func (s *songUsecaseImpl) CreateSong(ctx context.Context, input *CreateSongInput) (*entity.Song, error) {
	masters := s.masterCache.SongMasters()
	if masters == nil {
		return nil, fmt.Errorf("master cache is not initialized")
	}

	// ジャンル名の検証とID変換
	genreItem, ok := masters.Genres[input.Genre]
	if !ok {
		return nil, fmt.Errorf("%w: genre=%s", ErrInvalidDifficulty, input.Genre)
	}
	genreID := genreItem.ID

	// 譜面の変換
	charts := make([]*entity.Chart, 0, len(input.Charts))
	for _, chartInput := range input.Charts {
		diffKey := strings.ToUpper(chartInput.Difficulty)
		item, ok := masters.Difficulties[diffKey]
		if !ok {
			return nil, fmt.Errorf("%w: difficulty=%s", ErrInvalidDifficulty, chartInput.Difficulty)
		}

		cc, err := chartconstant.NewChartConstant(chartInput.Const)
		if err != nil {
			return nil, fmt.Errorf("invalid chart constant (difficulty: %s): %w", chartInput.Difficulty, err)
		}

		var notesVO *notes.Notes
		if chartInput.Notes != nil {
			n, err := notes.NewNotes(*chartInput.Notes)
			if err != nil {
				return nil, fmt.Errorf("invalid notes (difficulty: %s): %w", chartInput.Difficulty, err)
			}
			notesVO = &n
		}

		charts = append(charts, &entity.Chart{
			DifficultyID:   item.ID,
			Const:          cc,
			IsConstUnknown: chartInput.IsConstUnknown,
			Notes:          notesVO,
			NotesDesigner:  chartInput.NotesDesigner,
		})
	}

	displayID, err := generateDisplayID()
	if err != nil {
		return nil, err
	}

	song := entity.NewSong()
	song.DisplayID = displayID
	song.OfficialIdx = input.OfficialIdx
	song.Title = input.Title
	song.Artist = input.Artist
	song.GenreID = &genreID
	song.BPM = input.BPM
	song.ReleasedAt = input.ReleasedAt
	song.Jacket = input.Jacket
	song.Charts = charts

	var created *entity.Song
	if err := s.tm.Transactional(ctx, func(tx repository.Executor) error {
		result, err := s.songRepo.Create(ctx, tx, song)
		if err != nil {
			return err
		}
		created = result
		return nil
	}); err != nil {
		return nil, err
	}

	return created, nil
}
