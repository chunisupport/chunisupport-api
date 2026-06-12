package report_test

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const versioningPlanPath = "versioning_plan.md"

func readVersioningPlan(t *testing.T) string {
	t.Helper()
	content, err := os.ReadFile(versioningPlanPath)
	require.NoError(t, err, "versioning_plan.md が読み込めること")
	return string(content)
}

// TestVersioningPlanFileExists はバージョニング方針ファイルが存在し、内容があることを確認します。
func TestVersioningPlanFileExists(t *testing.T) {
	tests := []struct {
		name     string
		wantErr  bool
		minBytes int
	}{
		{name: "ファイルが存在する", wantErr: false, minBytes: 1},
		{name: "ファイルが空でない", wantErr: false, minBytes: 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := os.Stat(versioningPlanPath)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err, "ファイルが存在すること")
				assert.GreaterOrEqual(t, int(info.Size()), tt.minBytes, "ファイルサイズが最低バイト数以上であること")
			}
		})
	}
}

// TestVersioningPlanRequiredSections はドキュメントに必須セクションがすべて含まれていることを確認します。
func TestVersioningPlanRequiredSections(t *testing.T) {
	content := readVersioningPlan(t)

	tests := []struct {
		name    string
		section string
	}{
		{name: "セクション1: バージョン形式", section: "## 1."},
		{name: "セクション2: 定義と自動化", section: "## 2."},
		{name: "セクション3: 配信・露出戦略", section: "## 3."},
		{name: "セクション4: プロトコル互換性の管理", section: "## 4."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.True(t, strings.Contains(content, tt.section), "セクション %q がドキュメントに存在すること", tt.section)
		})
	}
}

// TestVersioningPlanVersionFormat はCalVer形式が正しくドキュメントに記載されていることを確認します。
func TestVersioningPlanVersionFormat(t *testing.T) {
	content := readVersioningPlan(t)

	tests := []struct {
		name    string
		pattern string
	}{
		{name: "CalVer形式 vYYYY.MM.DD が記載されている", pattern: `vYYYY\.MM\.DD`},
		{name: "Gitハッシュの併記形式が記載されている", pattern: `vYYYY\.MM\.DD.*Git`},
		{name: "バージョン例 v2024.05.28 が記載されている", pattern: `v2024\.05\.28`},
		{name: "ハッシュ例 a1b2c3d が記載されている", pattern: `a1b2c3d`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			re := regexp.MustCompile(tt.pattern)
			assert.True(t, re.MatchString(content), "パターン %q がドキュメントに含まれること", tt.pattern)
		})
	}
}

// TestVersioningPlanGoVariableDefaults はGoコードスニペットのデフォルト値が正しいことを確認します。
func TestVersioningPlanGoVariableDefaults(t *testing.T) {
	content := readVersioningPlan(t)

	tests := []struct {
		name     string
		variable string
		defValue string
	}{
		{name: "Version のデフォルト値は dev", variable: "Version", defValue: `"dev"`},
		{name: "Revision のデフォルト値は unknown", variable: "Revision", defValue: `"unknown"`},
		{name: "BuildTime のデフォルト値は unknown", variable: "BuildTime", defValue: `"unknown"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pattern := regexp.MustCompile(regexp.QuoteMeta(tt.variable) + `\s*=\s*` + regexp.QuoteMeta(tt.defValue))
			assert.True(t, pattern.MatchString(content), "変数 %s のデフォルト値 %s がコードスニペットに存在すること", tt.variable, tt.defValue)
		})
	}
}

// TestVersioningPlanBuildFlags はldflags構文に必要な変数パスが含まれていることを確認します。
func TestVersioningPlanBuildFlags(t *testing.T) {
	content := readVersioningPlan(t)

	tests := []struct {
		name     string
		fragment string
	}{
		{name: "ldflags オプションが記載されている", fragment: "-ldflags"},
		{name: "-X フラグが記載されている", fragment: "-X"},
		{name: "モジュールパスが記載されている", fragment: "github.com/chunisupport/chunisupport-api/internal/info"},
		{name: "Version の注入パスが記載されている", fragment: "internal/info.Version"},
		{name: "Revision の注入パスが記載されている", fragment: "internal/info.Revision"},
		{name: "BuildTime の注入パスが記載されている", fragment: "internal/info.BuildTime"},
		{name: "日付コマンド date が記載されている", fragment: "date +%Y.%m.%d"},
		{name: "git rev-parse が記載されている", fragment: "git rev-parse --short HEAD"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.True(t, strings.Contains(content, tt.fragment), "ビルドフラグ内に %q が含まれること", tt.fragment)
		})
	}
}

// TestVersioningPlanPublicEndpointJSON は一般公開エンドポイントのJSONレスポンス例が有効であることを確認します。
func TestVersioningPlanPublicEndpointJSON(t *testing.T) {
	// 一般公開エンドポイントのJSONを抽出（GET /セクションのJSONブロック）
	content := readVersioningPlan(t)

	// セクション3Aの一般公開用JSONを探す
	publicJSON := extractFirstJSONBlock(content, `"app_name"`)
	require.NotEmpty(t, publicJSON, "一般公開用JSONブロックが見つかること")

	t.Run("一般公開用JSONが有効なJSON形式である", func(t *testing.T) {
		var result map[string]interface{}
		err := json.Unmarshal([]byte(publicJSON), &result)
		assert.NoError(t, err, "JSONが有効な形式であること")
	})

	t.Run("一般公開用JSONにapp_nameフィールドがある", func(t *testing.T) {
		var result map[string]interface{}
		err := json.Unmarshal([]byte(publicJSON), &result)
		require.NoError(t, err)
		_, ok := result["app_name"]
		assert.True(t, ok, "app_name フィールドが存在すること")
	})

	t.Run("一般公開用JSONにversionフィールドがある", func(t *testing.T) {
		var result map[string]interface{}
		err := json.Unmarshal([]byte(publicJSON), &result)
		require.NoError(t, err)
		_, ok := result["version"]
		assert.True(t, ok, "version フィールドが存在すること")
	})

	t.Run("一般公開用JSONにrevisionフィールドが含まれない（セキュリティ上の理由）", func(t *testing.T) {
		var result map[string]interface{}
		err := json.Unmarshal([]byte(publicJSON), &result)
		require.NoError(t, err)
		_, ok := result["revision"]
		assert.False(t, ok, "revision フィールドが一般公開レスポンスに含まれないこと")
	})

	t.Run("一般公開用JSONにbuild_timeフィールドが含まれない", func(t *testing.T) {
		var result map[string]interface{}
		err := json.Unmarshal([]byte(publicJSON), &result)
		require.NoError(t, err)
		_, ok := result["build_time"]
		assert.False(t, ok, "build_time フィールドが一般公開レスポンスに含まれないこと")
	})
}

// TestVersioningPlanAdminEndpointJSON は管理者エンドポイントのJSONレスポンス例が有効であることを確認します。
func TestVersioningPlanAdminEndpointJSON(t *testing.T) {
	content := readVersioningPlan(t)

	adminJSON := extractFirstJSONBlock(content, `"status"`)
	require.NotEmpty(t, adminJSON, "管理者用JSONブロックが見つかること")

	requiredFields := []struct {
		name  string
		field string
	}{
		{name: "statusフィールドがある", field: "status"},
		{name: "versionフィールドがある", field: "version"},
		{name: "revisionフィールドがある", field: "revision"},
		{name: "build_timeフィールドがある", field: "build_time"},
		{name: "go_versionフィールドがある", field: "go_version"},
	}

	for _, tt := range requiredFields {
		t.Run(tt.name, func(t *testing.T) {
			var result map[string]interface{}
			err := json.Unmarshal([]byte(adminJSON), &result)
			require.NoError(t, err, "管理者用JSONが有効な形式であること")
			_, ok := result[tt.field]
			assert.True(t, ok, "フィールド %q が管理者用レスポンスに存在すること", tt.field)
		})
	}

	t.Run("管理者用JSONのstatusがokである", func(t *testing.T) {
		var result map[string]interface{}
		err := json.Unmarshal([]byte(adminJSON), &result)
		require.NoError(t, err)
		assert.Equal(t, "ok", result["status"], "status フィールドの値が 'ok' であること")
	})

	t.Run("管理者用JSONのbuild_timeがRFC3339形式に準拠している", func(t *testing.T) {
		var result map[string]interface{}
		err := json.Unmarshal([]byte(adminJSON), &result)
		require.NoError(t, err)
		buildTime, ok := result["build_time"].(string)
		require.True(t, ok, "build_time が文字列であること")
		re := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$`)
		assert.True(t, re.MatchString(buildTime), "build_time %q がRFC3339形式（UTC）に準拠していること", buildTime)
	})
}

// TestVersioningPlanEndpointPaths はAPI エンドポイントパスが正しく記載されていることを確認します。
func TestVersioningPlanEndpointPaths(t *testing.T) {
	content := readVersioningPlan(t)

	tests := []struct {
		name     string
		endpoint string
	}{
		{name: "一般公開エンドポイント GET / が記載されている", endpoint: "GET /"},
		{name: "管理者エンドポイント GET /health が記載されている", endpoint: "GET /health"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.True(t, strings.Contains(content, tt.endpoint), "エンドポイント %q がドキュメントに含まれること", tt.endpoint)
		})
	}
}

// TestVersioningPlanProtocolVersionSeparation はinfo.VersionとSupportedAppVersionsの分離方針が記載されていることを確認します。
func TestVersioningPlanProtocolVersionSeparation(t *testing.T) {
	content := readVersioningPlan(t)

	tests := []struct {
		name     string
		fragment string
	}{
		{name: "info.Version への言及がある", fragment: "info.Version"},
		{name: "SupportedAppVersions への言及がある", fragment: "SupportedAppVersions"},
		{name: "2種類のバージョンの役割分担が説明されている", fragment: "プロトコル"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.True(t, strings.Contains(content, tt.fragment), "%q がドキュメントに含まれること", tt.fragment)
		})
	}
}

// TestVersioningPlanGoFileLocation はGoファイルの定義場所が正しく記載されていることを確認します。
func TestVersioningPlanGoFileLocation(t *testing.T) {
	content := readVersioningPlan(t)

	tests := []struct {
		name     string
		fragment string
	}{
		{name: "Goファイルのパスが記載されている", fragment: "internal/info/info.go"},
		{name: "go buildコマンドが記載されている", fragment: "go build"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.True(t, strings.Contains(content, tt.fragment), "%q がドキュメントに含まれること", tt.fragment)
		})
	}
}

// TestVersioningPlanVersionExampleFormat はバージョン例の形式が正規表現パターンに合致することを確認します。
func TestVersioningPlanVersionExampleFormat(t *testing.T) {
	content := readVersioningPlan(t)

	// ドキュメント内のバージョン例が実際のCalVer形式に合致するか検証
	calVerPattern := regexp.MustCompile(`v\d{4}\.\d{2}\.\d{2}`)
	matches := calVerPattern.FindAllString(content, -1)

	t.Run("CalVer形式の具体例がドキュメントに含まれる", func(t *testing.T) {
		assert.NotEmpty(t, matches, "CalVer形式 vYYYY.MM.DD の具体例がドキュメントに存在すること")
	})

	t.Run("バージョン例が有効な日付範囲である（月が01-12）", func(t *testing.T) {
		monthPattern := regexp.MustCompile(`v\d{4}\.(\d{2})\.\d{2}`)
		for _, match := range matches {
			submatches := monthPattern.FindStringSubmatch(match)
			if len(submatches) > 1 {
				month := submatches[1]
				assert.GreaterOrEqual(t, month, "01", "月が01以上であること: %s", match)
				assert.LessOrEqual(t, month, "12", "月が12以下であること: %s", match)
			}
		}
	})
}

// extractFirstJSONBlock はmarkdownコンテンツから、指定フラグメントを含む最初のJSONコードブロックを抽出します。
func extractFirstJSONBlock(content, mustContain string) string {
	// ```json ... ``` ブロックを探す
	jsonBlockPattern := regexp.MustCompile("(?s)```json\n(.*?)```")
	matches := jsonBlockPattern.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) > 1 && strings.Contains(match[1], mustContain) {
			return strings.TrimSpace(match[1])
		}
	}
	return ""
}
