// Package mysqltest は本番マイグレーションを適用した使い捨てDBを結合テストへ提供します。
package mysqltest

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

// NewDatabase は既存DBを変更せず、新規DBへ全upを順番に適用します。
// DSNがないローカル実行だけをスキップし、CIではTEST_MYSQL_DSNを必須設定します。
func NewDatabase(t testing.TB, legacyEnv string) *sqlx.DB {
	t.Helper()
	dsn := os.Getenv("TEST_MYSQL_DSN")
	if dsn == "" && legacyEnv != "" {
		dsn = os.Getenv(legacyEnv)
	}
	if dsn == "" {
		t.Skip("TEST_MYSQL_DSN が未設定です")
	}
	cfg, err := mysql.ParseDSN(dsn)
	require.NoError(t, err)
	cfg.DBName = ""
	cfg.ParseTime = true
	cfg.Loc = time.UTC
	if cfg.Params == nil {
		cfg.Params = make(map[string]string)
	}
	cfg.Params["time_zone"] = "'+00:00'"
	cfg.MultiStatements = true
	cfg.ClientFoundRows = true
	cfg.Timeout = 5 * time.Second
	admin, err := sqlx.Connect("mysql", cfg.FormatDSN())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, admin.Close()) })
	name := fmt.Sprintf("chunisupport_test_%d", time.Now().UnixNano())
	_, err = admin.Exec("CREATE DATABASE " + name + " CHARACTER SET utf8mb4")
	require.NoError(t, err)
	t.Cleanup(func() { _, err := admin.Exec("DROP DATABASE " + name); require.NoError(t, err) })
	cfg.DBName = name
	db, err := sqlx.Connect("mysql", cfg.FormatDSN())
	require.NoError(t, err)
	db.SetMaxOpenConns(5)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	migrationDir := filepath.Join(filepath.Dir(file), "..", "..", "..", "migration", "mysql")
	// 読み込みをマイグレーションディレクトリ内に制限し、外部パスを参照させません。
	root, err := os.OpenRoot(migrationDir)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, root.Close()) })
	files, err := filepath.Glob(filepath.Join(migrationDir, "*.up.sql"))
	require.NoError(t, err)
	require.NotEmpty(t, files)
	for _, file := range files {
		data, err := root.ReadFile(filepath.Base(file))
		require.NoError(t, err)
		_, err = db.Exec(string(data))
		require.NoError(t, err, filepath.Base(file))
	}
	return db
}
