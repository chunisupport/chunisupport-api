package datasource

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDownloader_DownloadAll_ErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/success" {
			fmt.Fprint(w, `{"status":"ok"}`)
		} else {
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer server.Close()

	outputDir, err := os.MkdirTemp("", "downloader-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(outputDir)

	datasources := []Datasource{
		{Type: "success", URL: server.URL + "/success"},
		{Type: "fail", URL: server.URL + "/fail"},
	}

	downloader := NewDownloader(outputDir)
	results, err := downloader.DownloadAll(context.Background(), datasources)
	if err != nil {
		t.Fatalf("DownloadAll returned unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	byType := map[string]DownloadResult{}
	for _, result := range results {
		byType[result.Type] = result
	}

	success := byType["success"]
	if !success.Success {
		t.Errorf("expected success result to be successful, error=%s", success.Error)
	}
	if success.Path == "" {
		t.Error("expected successful download to have a path")
	}
	if success.Bytes == 0 {
		t.Error("expected successful download to report bytes")
	}
	if success.FetchedAt.IsZero() {
		t.Error("expected successful download to report fetched time")
	}

	fail := byType["fail"]
	if fail.Success {
		t.Error("expected fail result to be unsuccessful")
	}
	if fail.Error == "" {
		t.Error("expected fail result to have an error")
	}

	successPath := filepath.Join(outputDir, "success.json")
	if _, err := os.Stat(successPath); os.IsNotExist(err) {
		t.Errorf("Expected successful download file to exist")
	}

	failPath := filepath.Join(outputDir, "fail.json")
	if _, err := os.Stat(failPath); err == nil {
		t.Errorf("Expected failed download file to not exist")
	}
}

func TestDownloader_DownloadAll_CanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	downloader := NewDownloader(t.TempDir())
	results, err := downloader.DownloadAll(ctx, []Datasource{
		{Type: "official", URL: "https://example.invalid"},
	})
	if err != nil {
		t.Fatalf("DownloadAll returned unexpected error: %v", err)
	}
	if len(results) != 1 || results[0].Success {
		t.Fatalf("results = %v, want one failed result", results)
	}
	if !strings.Contains(results[0].Error, context.Canceled.Error()) {
		t.Fatalf("result error = %q, want context cancellation", results[0].Error)
	}
}

func TestDownloader_DownloadAll_RetriesTransientThenSucceeds(t *testing.T) {
	original := retryBackoff
	retryBackoff = func(int) time.Duration { return 0 }
	t.Cleanup(func() { retryBackoff = original })

	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			http.Error(w, "unavailable", http.StatusServiceUnavailable)
			return
		}
		fmt.Fprint(w, `{"status":"ok"}`)
	}))
	defer server.Close()

	outputDir, err := os.MkdirTemp("", "downloader-retry-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(outputDir)

	downloader := NewDownloader(outputDir)
	results, err := downloader.DownloadAll(context.Background(), []Datasource{
		{Type: "official", URL: server.URL},
	})
	if err != nil {
		t.Fatalf("DownloadAll returned unexpected error: %v", err)
	}
	if len(results) != 1 || !results[0].Success {
		t.Fatalf("expected retry success, results=%v", results)
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d, want 2", attempts)
	}
}

func TestDownloader_DownloadAll_AllFail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer server.Close()

	outputDir, err := os.MkdirTemp("", "downloader-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(outputDir)

	datasources := []Datasource{
		{Type: "fail1", URL: server.URL + "/fail1"},
		{Type: "fail2", URL: server.URL + "/fail2"},
	}

	downloader := NewDownloader(outputDir)
	results, err := downloader.DownloadAll(context.Background(), datasources)
	if err != nil {
		t.Fatalf("DownloadAll should not return overall error on per-source failure, got: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	for _, result := range results {
		if result.Success {
			t.Errorf("expected %s to fail", result.Type)
		}
		if result.Error == "" {
			t.Errorf("expected %s to have an error", result.Type)
		}
		path := filepath.Join(outputDir, result.Type+".json")
		if _, err := os.Stat(path); err == nil {
			t.Errorf("expected failed download file not to exist: %s", path)
		}
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

type countedBody interface {
	io.Reader
	bytesRead() int
}

type bytesBody struct {
	data []byte
	off  int
}

func (b *bytesBody) Read(p []byte) (int, error) {
	if b.off >= len(b.data) {
		return 0, io.EOF
	}
	n := copy(p, b.data[b.off:])
	b.off += n
	return n, nil
}

func (b *bytesBody) bytesRead() int { return b.off }

// fillBody は n バイトを fill で生成します。上限を超える本文をスライスに展開せずに渡せます。
type fillBody struct {
	n    int
	fill byte
	off  int
}

func (b *fillBody) Read(p []byte) (int, error) {
	if b.off >= b.n {
		return 0, io.EOF
	}
	k := min(len(p), b.n-b.off)
	for i := 0; i < k; i++ {
		p[i] = b.fill
	}
	b.off += k
	return k, nil
}

func (b *fillBody) bytesRead() int { return b.off }

func jsonObjectOfSize(t *testing.T, size int) []byte {
	t.Helper()
	const prefix = `{"v":"`
	const suffix = `"}`
	require.GreaterOrEqual(t, size, len(prefix)+len(suffix))
	body := make([]byte, 0, size)
	body = append(body, prefix...)
	body = append(body, bytes.Repeat([]byte{'a'}, size-len(prefix)-len(suffix))...)
	body = append(body, suffix...)
	require.Len(t, body, size)
	require.True(t, json.Valid(body))
	return body
}

func TestDownloader_DownloadAll_ResponseSizeLimit(t *testing.T) {
	limit := info.SongBatchDatasourceResponseMaxBytes
	exact := jsonObjectOfSize(t, limit)
	pretty := []byte("{\n  \"status\": \"ok\"\n}\n")
	prettySaved := []byte(`{"status":"ok"}`)
	const overshoot = 1024 * 1024

	tests := []struct {
		name          string
		payload       []byte
		total         int
		contentLength int64
		wantSaved     []byte
		wantRead      int
	}{
		{
			name:          "上限ちょうどのJSONはContent-Lengthが一致しても全量を保存する",
			payload:       exact,
			contentLength: int64(len(exact)),
			wantSaved:     exact,
			wantRead:      len(exact),
		},
		{
			name:          "上限ちょうどのJSONはContent-Length未設定でも全量を保存する",
			payload:       exact,
			contentLength: -1,
			wantSaved:     exact,
			wantRead:      len(exact),
		},
		{
			name:          "上限内のJSONはContent-Lengthが過大でも従来どおりminifyして保存する",
			payload:       pretty,
			contentLength: math.MaxInt64,
			wantSaved:     prettySaved,
			wantRead:      len(pretty),
		},
		{
			name:          "上限を超える本文はContent-Lengthが実サイズでも上限+1バイトで失敗する",
			total:         limit + 1 + overshoot,
			contentLength: int64(limit + 1 + overshoot),
			wantRead:      limit + 1,
		},
		{
			name:          "上限を超える本文はContent-Length未設定でも上限+1バイトで失敗する",
			total:         limit + 1 + overshoot,
			contentLength: -1,
			wantRead:      limit + 1,
		},
		{
			name:          "上限を超える本文はContent-Lengthが過少でも上限+1バイトで失敗する",
			total:         limit + 1 + overshoot,
			contentLength: 1,
			wantRead:      limit + 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var mu sync.Mutex
			var bodies []countedBody
			calls := 0
			downloader := NewDownloader(t.TempDir())
			downloader.httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
				var body countedBody
				if tt.payload != nil {
					body = &bytesBody{data: tt.payload}
				} else {
					body = &fillBody{n: tt.total, fill: 'a'}
				}
				mu.Lock()
				calls++
				bodies = append(bodies, body)
				mu.Unlock()
				return &http.Response{
					StatusCode:    http.StatusOK,
					Status:        "200 OK",
					Header:        make(http.Header),
					Body:          io.NopCloser(body),
					ContentLength: tt.contentLength,
					Request:       r,
				}, nil
			})

			results, err := downloader.DownloadAll(context.Background(), []Datasource{
				{Type: "official", URL: "http://example.test/official"},
			})

			require.NoError(t, err)
			require.Len(t, results, 1)
			mu.Lock()
			gotCalls := calls
			gotBodies := append([]countedBody(nil), bodies...)
			mu.Unlock()
			assert.Equal(t, 1, gotCalls)
			require.Len(t, gotBodies, 1)
			assert.Equal(t, tt.wantRead, gotBodies[0].bytesRead())

			path := filepath.Join(downloader.outputDir, "official.json")
			if tt.wantSaved == nil {
				assert.False(t, results[0].Success)
				assert.Contains(t, results[0].Error, fmt.Sprintf("exceeds %d bytes", limit))
				assert.Empty(t, results[0].Path)
				assert.Zero(t, results[0].Bytes)
				assert.True(t, results[0].FetchedAt.IsZero())
				_, statErr := os.Stat(path)
				assert.ErrorIs(t, statErr, os.ErrNotExist)
				return
			}

			assert.True(t, results[0].Success)
			assert.Empty(t, results[0].Error)
			assert.Equal(t, path, results[0].Path)
			assert.Equal(t, int64(len(tt.wantSaved)), results[0].Bytes)
			assert.False(t, results[0].FetchedAt.IsZero())
			saved, readErr := os.ReadFile(path)
			require.NoError(t, readErr)
			assert.Truef(t, bytes.Equal(tt.wantSaved, saved),
				"saved mismatch gotLen=%d wantLen=%d", len(saved), len(tt.wantSaved))
		})
	}
}

func TestDownloader_DownloadAll_OversizedSourceStaysIsolated(t *testing.T) {
	limit := info.SongBatchDatasourceResponseMaxBytes
	okBody := []byte(`{"status":"ok"}`)

	var mu sync.Mutex
	calls := map[string]int{}
	bodies := map[string]countedBody{}
	downloader := NewDownloader(t.TempDir())
	downloader.httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		var body countedBody
		var contentLength int64
		switch r.URL.Path {
		case "/ok":
			body = &bytesBody{data: okBody}
			contentLength = int64(len(okBody))
		case "/over":
			body = &fillBody{n: limit + 1 + 1024*1024, fill: 'b'}
			contentLength = -1
		default:
			return nil, fmt.Errorf("unexpected path %s", r.URL.Path)
		}
		mu.Lock()
		calls[r.URL.Path]++
		bodies[r.URL.Path] = body
		mu.Unlock()
		return &http.Response{
			StatusCode:    http.StatusOK,
			Status:        "200 OK",
			Header:        make(http.Header),
			Body:          io.NopCloser(body),
			ContentLength: contentLength,
			Request:       r,
		}, nil
	})

	results, err := downloader.DownloadAll(context.Background(), []Datasource{
		{Type: "ok", URL: "http://example.test/ok"},
		{Type: "over", URL: "http://example.test/over"},
	})

	require.NoError(t, err)
	require.Len(t, results, 2)
	mu.Lock()
	assert.Equal(t, 1, calls["/ok"])
	assert.Equal(t, 1, calls["/over"])
	assert.Equal(t, limit+1, bodies["/over"].bytesRead())
	mu.Unlock()

	assert.True(t, results[0].Success)
	assert.Equal(t, int64(len(okBody)), results[0].Bytes)
	saved, readErr := os.ReadFile(filepath.Join(downloader.outputDir, "ok.json"))
	require.NoError(t, readErr)
	assert.Equal(t, okBody, saved)

	assert.False(t, results[1].Success)
	assert.Contains(t, results[1].Error, fmt.Sprintf("exceeds %d bytes", limit))
	_, statErr := os.Stat(filepath.Join(downloader.outputDir, "over.json"))
	assert.ErrorIs(t, statErr, os.ErrNotExist)
}
