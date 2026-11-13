package repository

import (
	"context"
	"fmt"
	"runtime"
	"runtime/debug"
	"strings"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/testutil/mysqltest"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

// 実データ相当の件数です。通常楽曲の約1割がULTIMAを持つ想定にしています。
const (
	benchmarkStandardSongCount = 1500
	benchmarkUltimaInterval    = 10
	benchmarkWorldsendCount    = 700
)

// seedSongCatalogBenchmarkData は実データ相当の件数の楽曲・譜面を投入します。
func seedSongCatalogBenchmarkData(b *testing.B, db *sqlx.DB) {
	b.Helper()
	insertRows := func(prefix string, rows []string) {
		for start := 0; start < len(rows); start += 500 {
			end := min(start+500, len(rows))
			_, err := db.Exec(prefix + strings.Join(rows[start:end], ","))
			require.NoError(b, err)
		}
	}

	songs := make([]string, 0, benchmarkStandardSongCount+benchmarkWorldsendCount)
	charts := make([]string, 0, benchmarkStandardSongCount*5)
	worldsendCharts := make([]string, 0, benchmarkWorldsendCount)
	for id := 1; id <= benchmarkStandardSongCount+benchmarkWorldsendCount; id++ {
		isWorldsend := id > benchmarkStandardSongCount
		songs = append(songs, fmt.Sprintf(
			"(%d, '%016x', 'ベンチマーク楽曲タイトル%d', 'べんちまーく%d', 'アーティスト名%d', 1, 180, '2025-01-01', 'IDX%05d', 'jacket%05d.png', %t, 0)",
			id, id, id, id, id, id, id, isWorldsend,
		))
		if isWorldsend {
			worldsendCharts = append(worldsendCharts, fmt.Sprintf("(%d, 3, '狂', 1500, '譜面作者')", id))
			continue
		}
		difficulties := 4
		if id%benchmarkUltimaInterval == 0 {
			difficulties = 5
		}
		for difficultyID := 1; difficultyID <= difficulties; difficultyID++ {
			charts = append(charts, fmt.Sprintf("(%d, %d, %.1f, 0, 1000, '譜面作者')", id, difficultyID, 10.0+float64(difficultyID)))
		}
	}
	insertRows("INSERT INTO songs (id, display_id, title, reading, artist, genre_id, bpm, released_at, official_idx, jacket, is_worldsend, is_deleted) VALUES ", songs)
	insertRows("INSERT INTO charts (song_id, difficulty_id, const, is_const_unknown, notes, notes_designer) VALUES ", charts)
	insertRows("INSERT INTO worldsend_charts (song_id, level_star, attribute, notes, notes_designer) VALUES ", worldsendCharts)
}

func setupSongCatalogBenchmark(b *testing.B) (*sqlx.DB, *SongCatalogProvider) {
	b.Helper()
	db := mysqltest.NewDatabase(b, "")
	seedSongCatalogBenchmarkData(b, db)
	return db, NewSongCatalogProvider(db)
}

// countQueries は Com_select の差分で、計測対象が発行したSELECT文の数を数えます。
func countQueries(b *testing.B, db *sqlx.DB, f func()) int {
	b.Helper()
	conn, err := db.Connx(context.Background())
	require.NoError(b, err)
	defer conn.Close()
	read := func() int {
		var name string
		var value int
		require.NoError(b, conn.QueryRowxContext(context.Background(), "SHOW GLOBAL STATUS LIKE 'Com_select'").Scan(&name, &value))
		return value
	}
	before := read()
	f()
	// SHOW STATUS 自体は Com_select に数えられないため、差分がそのまま計測対象の数になります。
	return read() - before
}

// BenchmarkSongCatalog_Rebuild は全件再構築1回の時間と総確保量を計測します。
func BenchmarkSongCatalog_Rebuild(b *testing.B) {
	db, provider := setupSongCatalogBenchmark(b)
	ctx := context.Background()
	rebuild := func() {
		_, err := provider.build(ctx)
		require.NoError(b, err)
	}
	b.ReportAllocs()
	for b.Loop() {
		rebuild()
	}
	b.ReportMetric(float64(countQueries(b, db, rebuild)), "queries/op")
}

// BenchmarkSongCatalog_Memory は保持メモリ量と、旧世代を保持したまま再構築したときのピークを計測します。
func BenchmarkSongCatalog_Memory(b *testing.B) {
	_, provider := setupSongCatalogBenchmark(b)
	ctx := context.Background()
	heapAlloc := func() uint64 {
		var stats runtime.MemStats
		runtime.ReadMemStats(&stats)
		return stats.HeapAlloc
	}

	for b.Loop() {
		runtime.GC()
		base := heapAlloc()
		old, err := provider.build(ctx)
		require.NoError(b, err)
		runtime.GC()
		retained := heapAlloc() - base

		// GCを止めて再構築し、旧世代を保持したままの最大使用量（上限値）を求めます。
		previous := debug.SetGCPercent(-1)
		beforeRebuild := heapAlloc()
		next, err := provider.build(ctx)
		require.NoError(b, err)
		peak := heapAlloc() - beforeRebuild + retained
		debug.SetGCPercent(previous)

		b.ReportMetric(float64(retained)/1024/1024, "retained-MiB")
		b.ReportMetric(float64(peak)/1024/1024, "peak-upper-MiB")
		runtime.KeepAlive(old)
		runtime.KeepAlive(next)
	}
}

// BenchmarkUnplayedCompletionSource は未プレイ補完に使う楽曲・譜面の取得を改善前後で比較します。
// before は改善前のリクエストごとのDB全件取得、after はキャッシュからの複製です。
func BenchmarkUnplayedCompletionSource(b *testing.B) {
	db, provider := setupSongCatalogBenchmark(b)
	ctx := context.Background()
	songRepo := NewSongRepository(db)
	worldsendRepo := NewWorldsendChartRepository(db)
	_, err := provider.Snapshot(ctx)
	require.NoError(b, err)

	before := func() {
		_, err := songRepo.FindAllExcludingWorldsend(ctx, db, false)
		require.NoError(b, err)
		_, err = worldsendRepo.FindAll(ctx, db, false)
		require.NoError(b, err)
	}
	after := func() {
		catalog, err := provider.Snapshot(ctx)
		require.NoError(b, err)
		_ = catalog.Songs(false)
		_ = catalog.WorldsendSongs(false)
	}

	b.Run("before_db", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			before()
		}
		b.ReportMetric(float64(countQueries(b, db, before)), "queries/op")
	})
	b.Run("after_catalog", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			after()
		}
		b.ReportMetric(float64(countQueries(b, db, after)), "queries/op")
	})
}

// BenchmarkSongsUpdatedAt は updated-at の取得を改善前後で比較します。
func BenchmarkSongsUpdatedAt(b *testing.B) {
	db, provider := setupSongCatalogBenchmark(b)
	ctx := context.Background()
	songRepo := NewSongRepository(db)
	_, err := provider.Snapshot(ctx)
	require.NoError(b, err)

	before := func() {
		_, err := songRepo.FindLatestUpdatedAt(ctx, db)
		require.NoError(b, err)
	}
	after := func() {
		catalog, err := provider.Snapshot(ctx)
		require.NoError(b, err)
		_ = catalog.UpdatedAt()
	}

	b.Run("before_db", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			before()
		}
		b.ReportMetric(float64(countQueries(b, db, before)), "queries/op")
	})
	b.Run("after_catalog", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			after()
		}
		b.ReportMetric(float64(countQueries(b, db, after)), "queries/op")
	})
}
