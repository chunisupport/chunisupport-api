package logger

import (
	"bytes"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		name     string
		level    string
		expected slog.Level
	}{
		{name: "debug", level: "debug", expected: slog.LevelDebug},
		{name: "info", level: "info", expected: slog.LevelInfo},
		{name: "warn", level: "warn", expected: slog.LevelWarn},
		{name: "error", level: "error", expected: slog.LevelError},
		{name: "不正値はinfo", level: "warning", expected: slog.LevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, ParseLogLevel(tt.level))
		})
	}
}

func TestNewHandler_FileOutput(t *testing.T) {
	dir := t.TempDir()
	appLog := filepath.Join(dir, "app.log")

	handler, err := NewHandler(config.Logging{
		Level:   "info",
		AppFile: appLog,
		Stdout:  false,
	})
	require.NoError(t, err)
	defer func() {
		require.NoError(t, handler.Close())
	}()

	logger := slog.New(handler)
	logger.Info("アプリログ")

	content, err := os.ReadFile(appLog)
	require.NoError(t, err)
	assert.Contains(t, string(content), "アプリログ")
}

func TestNewAccessLogWriter_StdoutOnly(t *testing.T) {
	writer, err := NewAccessLogWriter(config.Logging{
		Stdout: true,
	})
	require.NoError(t, err)
	assert.NoError(t, writer.Reopen())
	assert.NoError(t, writer.Close())
}

func TestNewLogWriter_NoOutput(t *testing.T) {
	_, err := newLogWriter(false, "")
	assert.Error(t, err)
}

func TestReopenableFileWriter_Reopen(t *testing.T) {
	// Given: ローテーション前のログファイルへ前半を書き込んだ状態
	dir := t.TempDir()
	logPath := filepath.Join(dir, "app.log")
	rotatedPath := logPath + ".1"

	writer, err := NewReopenableFileWriter(logPath)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, writer.Close())
	}()
	_, err = writer.Write([]byte("before\n"))
	require.NoError(t, err)
	require.NoError(t, writer.file.Sync())
	// Windowsでは開いているファイルの改名が拒否されるため、下位ハンドルを
	// 閉じてから退避する。閉じる前にSyncしているので、退避結果はLinuxの
	// 改名と同じになる。
	require.NoError(t, writer.file.Close())
	require.NoError(t, os.Rename(logPath, rotatedPath))

	// When: 再オープンの合図を受けて後半を書き込む
	require.NoError(t, writer.Reopen())
	_, err = writer.Write([]byte("after\n"))
	require.NoError(t, err)

	// Then: 退避したファイルには前半だけ、元のパスの新しいファイルには後半だけが残る
	rotatedContent, err := os.ReadFile(rotatedPath)
	require.NoError(t, err)
	assert.Equal(t, "before\n", string(rotatedContent))
	newContent, err := os.ReadFile(logPath)
	require.NoError(t, err)
	assert.Equal(t, "after\n", string(newContent))
}

func TestReopenableFileWriter_CreatesLogFileWithOwnerOnlyPermission(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("WindowsではPOSIX形式のファイル権限を取得できないためスキップします")
	}

	dir := t.TempDir()
	logPath := filepath.Join(dir, "app.log")

	writer, err := NewReopenableFileWriter(logPath)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, writer.Close())
	}()

	info, err := os.Stat(logPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
}

func TestReopenableFileWriter_ReopenFailureKeepsOldFile(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "app.log")

	writer, err := NewReopenableFileWriter(logPath)
	require.NoError(t, err)
	_, err = writer.Write([]byte("before\n"))
	require.NoError(t, err)

	blockingPath := filepath.Join(dir, "blocked")
	require.NoError(t, os.WriteFile(blockingPath, []byte("not a directory"), 0640))
	writer.path = filepath.Join(blockingPath, "app.log")

	assert.Error(t, writer.Reopen())
	_, err = writer.Write([]byte("after\n"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	content, err := os.ReadFile(logPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "before")
	assert.Contains(t, string(content), "after")
}

func TestReopenableFileWriter_WriteAfterCloseReturnsError(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "app.log")

	writer, err := NewReopenableFileWriter(logPath)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	n, err := writer.Write([]byte("after close\n"))

	assert.Zero(t, n)
	assert.ErrorIs(t, err, os.ErrClosed)
}

func TestReopenableFileWriter_ReopenAfterCloseReturnsError(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "app.log")

	writer, err := NewReopenableFileWriter(logPath)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	err = writer.Reopen()

	assert.ErrorIs(t, err, os.ErrClosed)
}

// reopenSpyWriter は再オープンの合図の到達を数えながら配下へ転送するテスト用の書き込み先です。
type reopenSpyWriter struct {
	inner       Reopenable
	reopenCalls int
}

func (s *reopenSpyWriter) Write(p []byte) (int, error) {
	return s.inner.Write(p)
}

func (s *reopenSpyWriter) Close() error {
	return s.inner.Close()
}

func (s *reopenSpyWriter) Reopen() error {
	s.reopenCalls++
	return s.inner.Reopen()
}

func TestMultiReopenableWriteCloser(t *testing.T) {
	// Given: 標準出力側とファイル側へ転送する書き込み先
	dir := t.TempDir()
	logPath := filepath.Join(dir, "access.log")
	rotatedPath := logPath + ".1"

	fileWriter, err := NewReopenableFileWriter(logPath)
	require.NoError(t, err)
	// 本番と同じ配線（標準出力とファイルへの転送 + ファイルへの再オープン転送）を再現する。
	spy := &reopenSpyWriter{inner: fileWriter}
	var stdoutBuf bytes.Buffer
	writer := &multiReopenableWriteCloser{
		writer: io.MultiWriter(&stdoutBuf, spy),
		file:   spy,
	}
	_, err = writer.Write([]byte("before\n"))
	require.NoError(t, err)
	require.NoError(t, fileWriter.file.Sync())
	// Windowsでは開いているファイルの改名が拒否されるため、下位ハンドルを
	// 閉じてから退避する。閉じる前にSyncしているので、退避結果はLinuxの
	// 改名と同じになる。
	require.NoError(t, fileWriter.file.Close())
	require.NoError(t, os.Rename(logPath, rotatedPath))

	// When: 再オープンの合図を受けて後半を書き込む
	require.NoError(t, writer.Reopen())
	_, err = writer.Write([]byte("after\n"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	// Then: 合図はファイル側へ届き、退避ファイルには前半だけ、新しいファイルには後半だけが残る
	assert.Equal(t, 1, spy.reopenCalls)
	rotatedContent, err := os.ReadFile(rotatedPath)
	require.NoError(t, err)
	assert.Equal(t, "before\n", string(rotatedContent))
	newContent, err := os.ReadFile(logPath)
	require.NoError(t, err)
	assert.Equal(t, "after\n", string(newContent))
	// Then: 書き込みは標準出力側とファイル側の両方へ転送される
	assert.Equal(t, "before\nafter\n", stdoutBuf.String())
}
