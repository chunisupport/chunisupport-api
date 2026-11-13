package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/songcatalog"
	"github.com/jmoiron/sqlx"
)

var _ domainrepo.SongCatalogProvider = (*SongCatalogProvider)(nil)

// SongCatalogProvider は楽曲・譜面のスナップショットをAPIプロセス内で共有します。
//
// 参照は atomic.Pointer から公開中の世代を読むだけで、ロックもDB問い合わせも行いません。
// 再構築は refreshMu で直列化し、song_catalog_revision の値が公開中の世代と同じなら省略します。
// 新しい世代は完成後に1回の Store で差し替えるため、構築途中のデータや、データより先に進んだ更新時刻は公開されません。
type SongCatalogProvider struct {
	db            *sqlx.DB
	songRepo      domainrepo.SongRepository
	worldsendRepo domainrepo.WorldsendChartRepository

	current   atomic.Pointer[songcatalog.Catalog]
	refreshMu sync.Mutex
}

// NewSongCatalogProvider は楽曲カタログProviderを生成します。生成時にはDBへアクセスしません。
func NewSongCatalogProvider(db *sqlx.DB) *SongCatalogProvider {
	return &SongCatalogProvider{
		db:            db,
		songRepo:      NewSongRepository(db),
		worldsendRepo: NewWorldsendChartRepository(db),
	}
}

// Snapshot は公開中のスナップショットを返します。未構築の場合だけ構築を試みます。
func (p *SongCatalogProvider) Snapshot(ctx context.Context) (*songcatalog.Catalog, error) {
	if catalog := p.current.Load(); catalog != nil {
		return catalog, nil
	}
	if err := p.Refresh(ctx); err != nil {
		return nil, err
	}
	return p.current.Load(), nil
}

// Refresh はDB上の変更世代が公開中の世代と異なる場合に再構築して差し替えます。
func (p *SongCatalogProvider) Refresh(ctx context.Context) error {
	p.refreshMu.Lock()
	defer p.refreshMu.Unlock()

	current := p.current.Load()
	if current != nil {
		revision, err := findSongCatalogRevision(ctx, p.db)
		if err != nil {
			return err
		}
		if revision == current.Revision() {
			return nil
		}
	}

	startedAt := time.Now()
	catalog, err := p.build(ctx)
	if err != nil {
		return err
	}
	p.current.Store(catalog)
	slog.InfoContext(ctx, "song catalog rebuilt",
		"revision", catalog.Revision(),
		"duration_ms", time.Since(startedAt).Milliseconds(),
	)
	return nil
}

// Run は interval ごとに Refresh を呼び、別プロセスによる変更を反映します。
// 開始直後にも1回実行して、最初のリクエストより前に構築を済ませます。
// 失敗しても公開中のスナップショットを維持し、次の周期で再試行します。ctx の終了で戻ります。
func (p *SongCatalogProvider) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if err := p.Refresh(ctx); err != nil && !errors.Is(err, context.Canceled) {
			slog.ErrorContext(ctx, "failed to refresh song catalog", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// build は読み取り専用トランザクション内で世代番号・楽曲・譜面・更新時刻を読み取ります。
// REPEATABLE READ の一貫性読み取りにより、すべて同じ時点の値になります。
func (p *SongCatalogProvider) build(ctx context.Context) (catalog *songcatalog.Catalog, err error) {
	tx, err := p.db.BeginTxx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("%w: begin song catalog transaction: %w", domainrepo.ErrRepositoryOperationFailed, err)
	}
	defer func() {
		if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
			err = errors.Join(err, rollbackErr)
		}
	}()

	revision, err := findSongCatalogRevision(ctx, tx)
	if err != nil {
		return nil, err
	}
	songs, err := p.songRepo.FindAllExcludingWorldsend(ctx, tx, true)
	if err != nil {
		return nil, fmt.Errorf("%w: load songs for song catalog: %w", domainrepo.ErrRepositoryOperationFailed, err)
	}
	worldsendSongs, err := p.worldsendRepo.FindAll(ctx, tx, true)
	if err != nil {
		return nil, fmt.Errorf("%w: load worldsend songs for song catalog: %w", domainrepo.ErrRepositoryOperationFailed, err)
	}
	updatedAt, err := p.songRepo.FindLatestUpdatedAt(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("%w: load song catalog updated_at: %w", domainrepo.ErrRepositoryOperationFailed, err)
	}
	return songcatalog.New(revision, songs, worldsendSongs, updatedAt), nil
}
