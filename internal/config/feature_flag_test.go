package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/featureflag"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testFlag featureflag.Name = "test_feature"
const otherFlag featureflag.Name = "other_feature"

func TestParseFeatureFlags(t *testing.T) {
	tests := []struct {
		name        string
		data        string
		env         string
		registered  []featureflag.Name
		wantEnabled bool
		wantErr     string
	}{
		{name: "空ファイル"},
		{name: "コメントのみ", data: "# 書式例\n"},
		{name: "空マッピング", data: "{}"},
		{name: "ルートが配列", data: "[]", wantErr: "root"},
		{name: "ルートがスカラー", data: "true", wantErr: "root"},
		{name: "明示的なnullルート", data: "null", wantErr: "root"},
		{name: "複数ドキュメント", data: "{}\n---\n{}", wantErr: "multiple"},
		{name: "空の第2ドキュメント", data: "{}\n---\n", wantErr: "multiple"},
		{name: "不正なYAML", data: "[", wantErr: "YAML"},
		{name: "機能名重複", data: "test_feature: {}\ntest_feature: {}", registered: []featureflag.Name{testFlag}, wantErr: "duplicate"},
		{name: "機能の空値", data: "test_feature:", registered: []featureflag.Name{testFlag}, wantErr: "mapping"},
		{name: "機能のスカラー", data: "test_feature: true", registered: []featureflag.Name{testFlag}, wantErr: "mapping"},
		{name: "機能の配列", data: "test_feature: []", registered: []featureflag.Name{testFlag}, wantErr: "mapping"},
		{name: "機能の空マッピング", data: "test_feature: {}", registered: []featureflag.Name{testFlag}},
		{name: "未知の環境キー", data: "test_feature: {test: true}", registered: []featureflag.Name{testFlag}, wantErr: "environment"},
		{name: "環境キー重複", data: "test_feature: {develop: true, develop: false}", registered: []featureflag.Name{testFlag}, wantErr: "duplicate"},
		{name: "文字列のtrue", data: "test_feature: {develop: \"true\"}", registered: []featureflag.Name{testFlag}, wantErr: "bool"},
		{name: "yesは不可", data: "test_feature: {develop: yes}", registered: []featureflag.Name{testFlag}, wantErr: "bool"},
		{name: "数値は不可", data: "test_feature: {develop: 1}", registered: []featureflag.Name{testFlag}, wantErr: "bool"},
		{name: "nullは不可", data: "test_feature: {develop: null}", registered: []featureflag.Name{testFlag}, wantErr: "bool"},
		{name: "空値は不可", data: "test_feature: {develop: }", registered: []featureflag.Name{testFlag}, wantErr: "bool"},
		{name: "配列の値は不可", data: "test_feature: {develop: [true]}", registered: []featureflag.Name{testFlag}, wantErr: "bool"},
		{name: "マッピングの値は不可", data: "test_feature: {develop: {}}", registered: []featureflag.Name{testFlag}, wantErr: "bool"},
		{name: "別タグの値は不可", data: "test_feature: {develop: !custom true}", registered: []featureflag.Name{testFlag}, wantErr: "bool"},
		{name: "boolタグでも不正な値は不可", data: "test_feature: {develop: !!bool invalid}", registered: []featureflag.Name{testFlag}, wantErr: "bool"},
		{name: "YAMLだけに存在", data: "test_feature: {}", wantErr: "registered names"},
		{name: "登録一覧だけに存在", data: "{}", registered: []featureflag.Name{testFlag}, wantErr: "missing from YAML"},
		{name: "空ファイルでも突き合わせる", registered: []featureflag.Name{testFlag}, wantErr: "missing from YAML"},
		{name: "開発環境で有効", data: "test_feature: {develop: true}", env: info.AppEnvDevelop, registered: []featureflag.Name{testFlag}, wantEnabled: true},
		{name: "ステージング環境で有効", data: "test_feature: {staging: true}", env: info.AppEnvStaging, registered: []featureflag.Name{testFlag}, wantEnabled: true},
		{name: "本番環境で有効", data: "test_feature: {production: true}", env: info.AppEnvProduction, registered: []featureflag.Name{testFlag}, wantEnabled: true},
		{name: "明示したfalse", data: "test_feature: {develop: false}", env: info.AppEnvDevelop, registered: []featureflag.Name{testFlag}},
		{name: "省略した環境は無効", data: "test_feature: {develop: true}", env: info.AppEnvProduction, registered: []featureflag.Name{testFlag}},
		{name: "未知のAPP_ENVは無効", data: "test_feature: {develop: true}", env: "test", registered: []featureflag.Name{testFlag}},
		{name: "未知のAPP_ENVでも全体を検証", data: "test_feature: {production: yes}", env: "test", registered: []featureflag.Name{testFlag}, wantErr: "bool"},
		{name: "未知のAPP_ENVでも登録を検証", data: "test_feature: {}", env: "test", wantErr: "registered names"},
		{name: "選択していない環境も検証", data: "test_feature: {develop: true, production: yes}", env: info.AppEnvDevelop, registered: []featureflag.Name{testFlag}, wantErr: "bool"},
		{name: "選択していない機能も検証", data: "test_feature: {develop: true}\nother_feature: {production: yes}", env: info.AppEnvDevelop, registered: []featureflag.Name{testFlag, otherFlag}, wantErr: "bool"},
		{name: "機能名は文字列のみ", data: "1: {}", wantErr: "string"},
		{name: "環境キーは文字列のみ", data: "test_feature: {true: false}", registered: []featureflag.Name{testFlag}, wantErr: "string"},
		{name: "タグ付きでもルートがマッピングなら許可", data: "!custom {}"},
		{name: "タグ付きでも機能の値がマッピングなら許可", data: "test_feature: !custom {}", registered: []featureflag.Name{testFlag}},
		{name: "別名参照のboolは不可", data: "test_feature: {develop: &value true, staging: *value}", registered: []featureflag.Name{testFlag}, wantErr: "bool"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			data := []byte(tt.data)
			// When
			flags, err := parseFeatureFlags(data, tt.env, tt.registered)
			// Then
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				assert.False(t, flags.Enabled(testFlag))
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantEnabled, flags.Enabled(testFlag))
			assert.False(t, flags.Enabled(otherFlag))
		})
	}
}

func TestEmbeddedFeatureFlags(t *testing.T) {
	for _, env := range []string{info.AppEnvDevelop, info.AppEnvStaging, info.AppEnvProduction} {
		t.Run(env, func(t *testing.T) {
			// Given
			names := featureflag.RegisteredNames()
			// When
			_, err := parseFeatureFlags(featureFlagYAML, env, names)
			// Then
			require.NoError(t, err)
		})
	}
}

func TestLoadBatchConfigAggregatesFeatureFlagErrors(t *testing.T) {
	// Given: 未知のAPP_ENVでもYAMLを検証し、既存設定のエラーと一緒に返す。
	t.Chdir(t.TempDir())
	require.NoError(t, os.MkdirAll(info.ConfigDir, 0o750))
	settings := []byte(`{
		"timezone": "UTC",
		"logging": {"level": "info", "stdout": true},
		"shutdown_timeout_seconds": 0,
		"database": {"pool": {
			"max_open_conns": 2, "max_idle_conns": 2,
			"conn_max_lifetime_sec": 300, "conn_max_idle_time_sec": 60
		}}
	}`)
	require.NoError(t, os.WriteFile(filepath.Join(info.ConfigDir, "test.settings.json"), settings, 0o600))
	t.Setenv("APP_ENV", "test")
	t.Setenv("DB_NAME", "test")
	t.Setenv("DB_HOST", "localhost")
	t.Setenv("DB_PORT", "3306")
	t.Setenv("DB_USER", "test")
	t.Setenv("DB_PASS", "test")
	t.Setenv("DATA_TRANSFER_HMAC_SECRET", "")
	original := featureFlagYAML
	featureFlagYAML = []byte("[")
	t.Cleanup(func() { featureFlagYAML = original })

	// When
	_, err := LoadBatchConfig()

	// Then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "shutdown_timeout_seconds must be greater than 0")
	assert.Contains(t, err.Error(), "failed to decode feature-flag YAML")
}
