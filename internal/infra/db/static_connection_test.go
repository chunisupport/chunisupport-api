package db

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConnectStatic(t *testing.T) {
	t.Run("静的DBへ接続してPing済みのDBを返す", func(t *testing.T) {
		// Given
		dbPath := filepath.Join(t.TempDir(), "static.sqlite")

		// When
		database, err := ConnectStatic(dbPath)

		// Then
		require.NoError(t, err)
		require.NotNil(t, database)
		require.NoError(t, database.Ping())
		require.NoError(t, database.Close())
	})
}
