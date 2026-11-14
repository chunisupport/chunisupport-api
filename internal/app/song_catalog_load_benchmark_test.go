package app

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/config"
	"github.com/chunisupport/chunisupport-api/internal/infra/masterdata"
	infra "github.com/chunisupport/chunisupport-api/internal/infra/repository"
	"github.com/chunisupport/chunisupport-api/internal/testutil/mysqltest"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/require"
)

// 実データ相当の件数です。通常楽曲の約1割がULTIMAを持つ想定にしています。
// プレイヤーは通常譜面の約半数とWORLD'S ENDの一部をプレイ済みとし、残りが未プレイ補完の対象になります。
const (
	loadBenchmarkStandardSongCount  = 1500
	loadBenchmarkUltimaInterval     = 10
	loadBenchmarkWorldsendCount     = 700
	loadBenchmarkPlayedChartPercent = 50
	loadBenchmarkPlayedWorldsend    = 200
	loadBenchmarkUsername           = "loadtester"
	// 本番設定例（example.setting.json）と同じ接続数です。
	loadBenchmarkMaxOpenConns = 25
)

// seedSongCatalogLoadBenchmark は楽曲・譜面と、レコードを持つ公開ユーザー1人を投入します。
func seedSongCatalogLoadBenchmark(b *testing.B, db *sqlx.DB) {
	b.Helper()
	insertRows := func(prefix string, rows []string) {
		for start := 0; start < len(rows); start += 500 {
			end := min(start+500, len(rows))
			_, err := db.Exec(prefix + strings.Join(rows[start:end], ","))
			require.NoError(b, err)
		}
	}

	songs := make([]string, 0, loadBenchmarkStandardSongCount+loadBenchmarkWorldsendCount)
	charts := make([]string, 0, loadBenchmarkStandardSongCount*5)
	worldsendCharts := make([]string, 0, loadBenchmarkWorldsendCount)
	for id := 1; id <= loadBenchmarkStandardSongCount+loadBenchmarkWorldsendCount; id++ {
		isWorldsend := id > loadBenchmarkStandardSongCount
		songs = append(songs, fmt.Sprintf(
			"(%d, '%016x', 'ベンチマーク楽曲タイトル%d', 'べんちまーく%d', 'アーティスト名%d', 1, 180, '2025-01-01', 'IDX%05d', 'jacket%05d.png', %t, 0)",
			id, id, id, id, id, id, id, isWorldsend,
		))
		if isWorldsend {
			worldsendCharts = append(worldsendCharts, fmt.Sprintf("(%d, %d, 3, '狂', 1500, '譜面作者')", id, id))
			continue
		}
		difficulties := 4
		if id%loadBenchmarkUltimaInterval == 0 {
			difficulties = 5
		}
		for difficultyID := 1; difficultyID <= difficulties; difficultyID++ {
			charts = append(charts, fmt.Sprintf("(%d, %d, %.1f, 0, 1000, '譜面作者')", id, difficultyID, 10.0+float64(difficultyID)))
		}
	}
	insertRows("INSERT INTO songs (id, display_id, title, reading, artist, genre_id, bpm, released_at, official_idx, jacket, is_worldsend, is_deleted) VALUES ", songs)
	insertRows("INSERT INTO charts (song_id, difficulty_id, const, is_const_unknown, notes, notes_designer) VALUES ", charts)
	insertRows("INSERT INTO worldsend_charts (id, song_id, level_star, attribute, notes, notes_designer) VALUES ", worldsendCharts)

	// users と players は相互に参照するため、ユーザー作成後にプレイヤーを紐付けます。
	_, err := db.Exec(`INSERT INTO users (id, username, is_private) VALUES (1, ?, 0)`, loadBenchmarkUsername)
	require.NoError(b, err)
	_, err = db.Exec(`INSERT INTO players (id, user_id, player_name, player_level, official_player_rating) VALUES (1, 1, 'ＬＯＡＤＴＥＳＴ', 99, 17.00)`)
	require.NoError(b, err)
	_, err = db.Exec(`UPDATE users SET player_id = 1 WHERE id = 1`)
	require.NoError(b, err)

	var chartIDs []int
	require.NoError(b, db.Select(&chartIDs, `SELECT id FROM charts ORDER BY id`))
	records := make([]string, 0, len(chartIDs))
	for i, chartID := range chartIDs {
		if i%100 >= loadBenchmarkPlayedChartPercent {
			continue
		}
		records = append(records, fmt.Sprintf("(1, %d, 1005000, 2, 1, 1, 1)", chartID))
	}
	insertRows("INSERT INTO player_records (player_id, chart_id, score, clear_lamp_id, combo_lamp_id, full_chain_id, slot_id) VALUES ", records)

	worldsendRecords := make([]string, 0, loadBenchmarkPlayedWorldsend)
	for i := range loadBenchmarkPlayedWorldsend {
		worldsendRecords = append(worldsendRecords, fmt.Sprintf("(1, %d, 990000, 2, 1, 1)", loadBenchmarkStandardSongCount+1+i))
	}
	insertRows("INSERT INTO player_worldsend_records (player_id, worldsend_chart_id, score, clear_lamp_id, combo_lamp_id, full_chain_id) VALUES ", worldsendRecords)
}

// newSongCatalogLoadBenchmarkRouter は本番と同じ組み立てのルーターを、計測用のDBで作ります。
func newSongCatalogLoadBenchmarkRouter(b *testing.B) (*echo.Echo, *sqlx.DB) {
	b.Helper()
	db := mysqltest.NewDatabase(b, "")
	db.SetMaxOpenConns(loadBenchmarkMaxOpenConns)
	db.SetMaxIdleConns(loadBenchmarkMaxOpenConns)
	seedSongCatalogLoadBenchmark(b, db)

	ctx := context.Background()
	masterCache, err := masterdata.Preload(ctx, db)
	require.NoError(b, err)
	staticMasterCache, err := masterdata.PreloadStatic(ctx, db)
	require.NoError(b, err)
	router, err := NewRouter(ctx, db, config.Config{
		DataTransferHMACSecret: testDataTransferHMACSecret,
		Location:               time.FixedZone("JST", 9*60*60),
		CORS:                   config.CORS{AllowOrigins: []string{"https://example.com"}},
	}, masterCache, staticMasterCache, tokenVerifierWithRecentSignIn{}, nil, nil, nil)
	require.NoError(b, err)
	return router, db
}

// loadResult は同時アクセス計測の集計値です。
type loadResult struct {
	latencies []time.Duration
	mallocs   uint64
	bytes     uint64
	numGC     uint32
	pauseNs   uint64
	maxPause  time.Duration
}

// runConcurrentLoad は workers 個の goroutine から requests 回 f を実行し、応答時間とGCの統計を集めます。
func runConcurrentLoad(b *testing.B, workers, requests int, f func(i int)) loadResult {
	b.Helper()
	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)

	latencies := make([]time.Duration, requests)
	var next atomic.Int64
	var wg sync.WaitGroup
	for range workers {
		wg.Go(func() {
			for {
				i := int(next.Add(1)) - 1
				if i >= requests {
					return
				}
				start := time.Now()
				f(i)
				latencies[i] = time.Since(start)
			}
		})
	}
	wg.Wait()

	var after runtime.MemStats
	runtime.ReadMemStats(&after)
	result := loadResult{
		latencies: latencies,
		mallocs:   after.Mallocs - before.Mallocs,
		bytes:     after.TotalAlloc - before.TotalAlloc,
		numGC:     after.NumGC - before.NumGC,
		pauseNs:   after.PauseTotalNs - before.PauseTotalNs,
	}
	// PauseNs は直近256回分の循環バッファです。計測中のGCのうち、残っている直近256回から最大値を求めます。
	first := before.NumGC + 1
	if after.NumGC >= uint32(len(after.PauseNs)) {
		first = max(first, after.NumGC-uint32(len(after.PauseNs))+1)
	}
	for n := first; n <= after.NumGC; n++ {
		result.maxPause = max(result.maxPause, time.Duration(after.PauseNs[(n-1)%uint32(len(after.PauseNs))]))
	}
	return result
}

func (r loadResult) report(b *testing.B) {
	b.Helper()
	sorted := slices.Clone(r.latencies)
	slices.Sort(sorted)
	percentile := func(p float64) float64 {
		return float64(sorted[int(float64(len(sorted)-1)*p)].Microseconds()) / 1000
	}
	n := float64(len(r.latencies))
	b.ReportMetric(percentile(0.50), "p50-ms")
	b.ReportMetric(percentile(0.95), "p95-ms")
	b.ReportMetric(percentile(0.99), "p99-ms")
	b.ReportMetric(float64(r.mallocs)/n, "allocs/req")
	b.ReportMetric(float64(r.bytes)/n/1024/1024, "MiB/req")
	b.ReportMetric(float64(r.numGC)/n*1000, "gc/1000req")
	b.ReportMetric(float64(r.pauseNs)/n/1000, "gc-pause-µs/req")
	b.ReportMetric(float64(r.maxPause.Microseconds()), "gc-max-pause-µs")
}

// BenchmarkSongCatalogLoad は未プレイ補完を含むAPIへの同時アクセスで、
// 応答時間・確保量・GCを計測し、そのうち楽曲カタログの複製が占める割合を比較します。
// 1回の実行で workers ごとに loadBenchmarkRequests 件を処理します（b.N には依存しません）。
//
//	TEST_MYSQL_DSN='root:<password>@tcp(127.0.0.1:3306)/' go test -run '^$' -bench SongCatalogLoad -benchtime=1x ./internal/app/
func BenchmarkSongCatalogLoad(b *testing.B) {
	const loadBenchmarkRequests = 500
	router, db := newSongCatalogLoadBenchmarkRouter(b)
	catalogProvider := infra.NewSongCatalogProvider(db)
	ctx := context.Background()
	_, err := catalogProvider.Snapshot(ctx)
	require.NoError(b, err)

	request := func(i int) {
		req := httptest.NewRequest(http.MethodGet, "/internal/users/"+loadBenchmarkUsername, nil)
		// 匿名アクセスは送信元IPごとに回数制限されるため、リクエストごとに別のIPにします。
		req.RemoteAddr = fmt.Sprintf("10.%d.%d.%d:1234", (i>>16)&0xff, (i>>8)&0xff, i&0xff)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			b.Errorf("unexpected status: %d %s", rec.Code, rec.Body.String())
		}
	}
	// 未プレイ補完でリクエストごとに行う複製（通常譜面と WORLD'S END）だけを、同じ条件で実行します。
	cloneOnly := func(int) {
		catalog, err := catalogProvider.Snapshot(ctx)
		if err != nil {
			b.Error(err)
			return
		}
		_ = catalog.Songs(false)
		_ = catalog.WorldsendSongs(false)
	}

	// 計測前にキャッシュ・接続プールを温めます。
	for i := range 20 {
		request(i)
	}

	for _, workers := range []int{1, 8, 32} {
		b.Run(fmt.Sprintf("api/workers=%d", workers), func(b *testing.B) {
			for b.Loop() {
				runConcurrentLoad(b, workers, loadBenchmarkRequests, request).report(b)
			}
		})
		b.Run(fmt.Sprintf("clone_only/workers=%d", workers), func(b *testing.B) {
			for b.Loop() {
				runConcurrentLoad(b, workers, loadBenchmarkRequests, cloneOnly).report(b)
			}
		})
	}
}
