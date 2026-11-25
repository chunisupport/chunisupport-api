package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/scorepercentile"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/jmoiron/sqlx"
	"golang.org/x/sync/singleflight"
)

var _ domainrepo.ScorePercentileProvider = (*ScorePercentileProvider)(nil)

// ScorePercentileProvider は通常譜面・WORLD'S END譜面・コースの上位%スナップショットをAPIプロセス内で共有します。
//
// 参照は atomic.Pointer から公開中の世代を読むだけで、ロックもDB問い合わせも行いません。
// 再構築は refreshSem で直列化し、chart_stats_revision の値が公開中の世代と同じなら省略します。
type ScorePercentileProvider struct {
	db *sqlx.DB

	current    atomic.Pointer[scorepercentile.Snapshot]
	refreshSem chan struct{}
	initGroup  singleflight.Group
}

// NewScorePercentileProvider は上位%スナップショットProviderを生成します。生成時にはDBへアクセスしません。
func NewScorePercentileProvider(db *sqlx.DB) *ScorePercentileProvider {
	return &ScorePercentileProvider{
		db:         db,
		refreshSem: make(chan struct{}, 1),
	}
}

// Snapshot は公開中のスナップショットを返します。未構築の場合だけ構築を試みます。
func (p *ScorePercentileProvider) Snapshot(ctx context.Context) (*scorepercentile.Snapshot, error) {
	if snapshot := p.current.Load(); snapshot != nil {
		return snapshot, nil
	}
	result := p.initGroup.DoChan("initial", func() (any, error) {
		buildCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), info.ScorePercentileRefreshTimeout)
		defer cancel()
		return nil, p.Refresh(buildCtx)
	})
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("%w: wait for score percentiles: %w", domainrepo.ErrRepositoryOperationFailed, ctx.Err())
	case r := <-result:
		if r.Err != nil {
			return nil, r.Err
		}
	}
	return p.current.Load(), nil
}

// Refresh はDB上の統計変更世代が公開中の世代と異なる場合に再構築して差し替えます。
func (p *ScorePercentileProvider) Refresh(ctx context.Context) error {
	select {
	case p.refreshSem <- struct{}{}:
	case <-ctx.Done():
		return fmt.Errorf("%w: wait for score percentiles refresh: %w", domainrepo.ErrRepositoryOperationFailed, ctx.Err())
	}
	defer func() { <-p.refreshSem }()

	current := p.current.Load()
	if current != nil {
		revision, err := findChartStatsRevision(ctx, p.db)
		if err != nil {
			return err
		}
		if revision == current.Revision() {
			return nil
		}
	}

	startedAt := time.Now()
	snapshot, err := p.build(ctx)
	if err != nil {
		return err
	}
	p.current.Store(snapshot)
	slog.InfoContext(ctx, "score percentiles rebuilt",
		"revision", snapshot.Revision(),
		"duration_ms", time.Since(startedAt).Milliseconds(),
	)
	return nil
}

// Run は interval ごとに Refresh を呼び、別プロセス（CLIのchart-stats-batch）による更新を取り込みます。
func (p *ScorePercentileProvider) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		refreshCtx, cancel := context.WithTimeout(ctx, info.ScorePercentileRefreshTimeout)
		err := p.Refresh(refreshCtx)
		cancel()
		if err != nil && !errors.Is(err, context.Canceled) {
			slog.ErrorContext(ctx, "failed to refresh score percentiles", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

type scorePercentileRow struct {
	TargetID   int    `db:"target_id"`
	Thresholds []byte `db:"thresholds"`
}

// build は読み取り専用の REPEATABLE READ トランザクション内で世代番号と3つの境界テーブルを一貫して読み取ります。
func (p *ScorePercentileProvider) build(ctx context.Context) (snapshot *scorepercentile.Snapshot, err error) {
	tx, err := p.db.BeginTxx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("%w: begin score percentiles transaction: %w", domainrepo.ErrRepositoryOperationFailed, err)
	}
	defer func() {
		if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
			err = errors.Join(err, rollbackErr)
		}
	}()

	revision, err := findChartStatsRevision(ctx, tx)
	if err != nil {
		return nil, err
	}
	charts, err := loadScorePercentileEntries(ctx, tx, `SELECT chart_id AS target_id, thresholds FROM chart_score_percentiles ORDER BY chart_id`)
	if err != nil {
		return nil, fmt.Errorf("%w: load chart score percentiles: %w", domainrepo.ErrRepositoryOperationFailed, err)
	}
	worldsendCharts, err := loadScorePercentileEntries(ctx, tx, `SELECT worldsend_chart_id AS target_id, thresholds FROM worldsend_chart_score_percentiles ORDER BY worldsend_chart_id`)
	if err != nil {
		return nil, fmt.Errorf("%w: load worldsend chart score percentiles: %w", domainrepo.ErrRepositoryOperationFailed, err)
	}
	courses, err := loadScorePercentileEntries(ctx, tx, `SELECT course_id AS target_id, thresholds FROM course_score_percentiles ORDER BY course_id`)
	if err != nil {
		return nil, fmt.Errorf("%w: load course score percentiles: %w", domainrepo.ErrRepositoryOperationFailed, err)
	}
	return scorepercentile.NewSnapshot(revision, charts, worldsendCharts, courses), nil
}

func loadScorePercentileEntries(ctx context.Context, exec domainrepo.Executor, query string) ([]scorepercentile.Entry, error) {
	var rows []scorePercentileRow
	if err := exec.SelectContext(ctx, &rows, query); err != nil {
		return nil, err
	}
	entries := make([]scorepercentile.Entry, 0, len(rows))
	for _, row := range rows {
		thresholds, err := scorepercentile.DecodeThresholds(row.Thresholds)
		if err != nil {
			return nil, fmt.Errorf("target_id=%d: %w", row.TargetID, err)
		}
		entries = append(entries, scorepercentile.Entry{
			TargetID:   row.TargetID,
			Thresholds: thresholds,
		})
	}
	return entries, nil
}
