package repository

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/service"
	"github.com/jmoiron/sqlx"
)

const defaultOverpowerDenominatorTTL = 10 * time.Minute

var _ domainrepo.OverpowerDenominatorProvider = (*OverpowerDenominatorProvider)(nil)

// OverpowerDenominatorProvider は最新マスタに基づくOVER POWER割合分母をプロセス内にキャッシュします。
type OverpowerDenominatorProvider struct {
	db  *sqlx.DB
	ttl time.Duration

	mu        sync.RWMutex
	snapshot  *domainrepo.OverpowerDenominatorSnapshot
	expiresAt time.Time
}

// NewOverpowerDenominatorProvider はデフォルトTTLの分母Providerを生成します。
func NewOverpowerDenominatorProvider(db *sqlx.DB) *OverpowerDenominatorProvider {
	return NewOverpowerDenominatorProviderWithTTL(db, defaultOverpowerDenominatorTTL)
}

// NewOverpowerDenominatorProviderWithTTL はテストや調整用途でTTLを指定してProviderを生成します。
func NewOverpowerDenominatorProviderWithTTL(db *sqlx.DB, ttl time.Duration) *OverpowerDenominatorProvider {
	if ttl <= 0 {
		ttl = defaultOverpowerDenominatorTTL
	}
	return &OverpowerDenominatorProvider{
		db:  db,
		ttl: ttl,
	}
}

func (p *OverpowerDenominatorProvider) Snapshot(ctx context.Context) (*domainrepo.OverpowerDenominatorSnapshot, error) {
	now := time.Now()
	p.mu.RLock()
	if p.snapshot != nil && now.Before(p.expiresAt) {
		snapshot := cloneOverpowerDenominatorSnapshot(p.snapshot)
		p.mu.RUnlock()
		return snapshot, nil
	}
	p.mu.RUnlock()

	p.mu.Lock()
	defer p.mu.Unlock()

	now = time.Now()
	if p.snapshot != nil && now.Before(p.expiresAt) {
		return cloneOverpowerDenominatorSnapshot(p.snapshot), nil
	}

	snapshot, err := p.buildSnapshot(ctx)
	if err != nil {
		return nil, err
	}
	p.snapshot = snapshot
	p.expiresAt = now.Add(p.ttl)
	slog.Info("overpower denominator snapshot rebuilt", "song_count", len(snapshot.SongMaxOP), "ttl", p.ttl.String())

	return cloneOverpowerDenominatorSnapshot(snapshot), nil
}

func (p *OverpowerDenominatorProvider) Invalidate(ctx context.Context) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.snapshot = nil
	p.expiresAt = time.Time{}
}

func (p *OverpowerDenominatorProvider) buildSnapshot(ctx context.Context) (*domainrepo.OverpowerDenominatorSnapshot, error) {
	const query = `
		SELECT
			s.id AS song_id,
			d.name AS difficulty_name,
			c.const AS chart_const
		FROM songs s
		INNER JOIN charts c ON c.song_id = s.id
		INNER JOIN difficulties d ON d.id = c.difficulty_id
		WHERE s.is_worldsend = 0
		  AND s.is_deleted = 0
	`

	var rows []struct {
		SongID         int     `db:"song_id"`
		DifficultyName string  `db:"difficulty_name"`
		ChartConst     float64 `db:"chart_const"`
	}
	if err := p.db.SelectContext(ctx, &rows, query); err != nil {
		return nil, fmt.Errorf("%w: failed to select overpower denominator snapshot: %w", domainrepo.ErrRepositoryOperationFailed, err)
	}

	maxConstBySongID := make(map[int]float64, len(rows))
	maxConstWithoutUltimaBySongID := make(map[int]float64, len(rows))
	for _, row := range rows {
		if row.ChartConst > maxConstBySongID[row.SongID] {
			maxConstBySongID[row.SongID] = row.ChartConst
		}
		if strings.ToUpper(row.DifficultyName) == "ULTIMA" {
			continue
		}
		if row.ChartConst > maxConstWithoutUltimaBySongID[row.SongID] {
			maxConstWithoutUltimaBySongID[row.SongID] = row.ChartConst
		}
	}

	snapshot := &domainrepo.OverpowerDenominatorSnapshot{
		SongMaxOP:              make(map[int]float64, len(maxConstBySongID)),
		SongMaxOPWithoutUltima: make(map[int]float64, len(maxConstBySongID)),
	}
	for songID, maxConst := range maxConstBySongID {
		maxOP := service.CalcSongMaxOP(maxConst)
		snapshot.SongMaxOP[songID] = maxOP
		snapshot.GlobalTotal += maxOP
		snapshot.SongMaxOPWithoutUltima[songID] = service.CalcSongMaxOP(maxConstWithoutUltimaBySongID[songID])
	}

	return snapshot, nil
}

func cloneOverpowerDenominatorSnapshot(src *domainrepo.OverpowerDenominatorSnapshot) *domainrepo.OverpowerDenominatorSnapshot {
	if src == nil {
		return nil
	}

	dst := &domainrepo.OverpowerDenominatorSnapshot{
		GlobalTotal:            src.GlobalTotal,
		SongMaxOP:              make(map[int]float64, len(src.SongMaxOP)),
		SongMaxOPWithoutUltima: make(map[int]float64, len(src.SongMaxOPWithoutUltima)),
	}
	for songID, maxOP := range src.SongMaxOP {
		dst.SongMaxOP[songID] = maxOP
	}
	for songID, maxOP := range src.SongMaxOPWithoutUltima {
		dst.SongMaxOPWithoutUltima[songID] = maxOP
	}
	return dst
}
