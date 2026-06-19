package logger

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
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
	dir := t.TempDir()
	logPath := filepath.Join(dir, "app.log")

	writer, err := NewReopenableFileWriter(logPath)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, writer.Close())
	}()
	_, err = writer.Write([]byte("before\n"))
	require.NoError(t, err)

	require.NoError(t, writer.Reopen())
	_, err = writer.Write([]byte("after\n"))
	require.NoError(t, err)

	newContent, err := os.ReadFile(logPath)
	require.NoError(t, err)

	assert.Contains(t, string(newContent), "before")
	assert.Contains(t, string(newContent), "after")
}

func TestReopenableFileWriter_CreatesLogFileWithOwnerOnlyPermission(t *testing.T) {
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

func TestMultiReopenableWriteCloser(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "access.log")
	fileWriter, err := NewReopenableFileWriter(logPath)
	require.NoError(t, err)

	writer := &multiReopenableWriteCloser{
		writer: io.MultiWriter(fileWriter),
		file:   fileWriter,
	}
	_, err = writer.Write([]byte("access\n"))
	require.NoError(t, err)
	require.NoError(t, writer.Reopen())
	require.NoError(t, writer.Close())

	content, err := os.ReadFile(logPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "access")
}
