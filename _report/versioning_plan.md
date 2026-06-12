# バージョニング方針案

ローリングリリース運用に適した、実用的かつ自動化可能なバージョニング方針についてまとめます。

## 1. バージョン形式

「人間への分かりやすさ」と「開発時の追跡可能性」を両立するため、以下の併記形式を採用します。

**形式**: `vYYYY.MM.DD (Git短縮ハッシュ)`
**例**: `v2024.05.28 (a1b2c3d)`

### 各項目の役割
- **日付 (CalVer)**: アプリケーションの鮮度（いつリリースされたものか）をユーザーに伝えます。
- **Gitハッシュ**: 開発者がバグ報告を受けた際、どの時点のコードに不具合があるのかを完全に特定するために使用します。

## 2. 定義と自動化

バージョン情報はソースコードにハードコードせず、ビルド時に自動注入します。

### 定義場所 (`internal/info/info.go`)
```go
var (
    Version   = "dev"       // 日付: 2024.05.28
    Revision  = "unknown"   // ハッシュ: a1b2c3d
    BuildTime = "unknown"   // ビルド日時: RFC3339形式
)
```
※現在はconstでやっているので修正必須

### 自動注入 (Build flags)
ビルド時に以下のフラグを指定することで、変数を動的に書き換えます。
```bash
go build -ldflags "-X 'github.com/chunisupport/chunisupport-api/internal/info.Version=$(date +%Y.%m.%d)' \
                   -X 'github.com/chunisupport/chunisupport-api/internal/info.Revision=$(git rev-parse --short HEAD)' \
                   -X 'github.com/chunisupport/chunisupport-api/internal/info.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)'"
```

## 3. 配信・露出戦略

セキュリティと利便性のバランスを考え、情報の粒度をエンドポイントごとに分けます。

### A. 一般公開用 (`GET /`)
不特定多数がアクセスできる場所では、日付ベースのバージョンのみを公開し、詳細なリビジョン（ハッシュ）は伏せます。

**レスポンス例**:
```json
{
  "app_name": "chunisupport-api",
  "version": "2024.05.28"
}
```

### B. 管理者・開発用 (`GET /health`)
認証済みの管理者のみがアクセスできるエンドポイントでは、デバッグに必要な全情報を公開します。

**レスポンス例**:
```json
{
  "status": "ok",
  "version": "2024.05.28",
  "revision": "a1b2c3d",
  "build_time": "2024-05-28T12:34:56Z",
  "go_version": "go1.22.3"
}
```

## 4. プロトコル互換性の管理

`info.Version`（ビルドバージョン）とは別に、**`SupportedAppVersions`（プロトコルバージョン）**を引き続き保持します。

- **`info.Version`**: コードが更新されるたびに自動で上がる「どのバイナリか」を示す識別子。
- **`SupportedAppVersions`**: 通信プロトコルやJSON構造に破壊的変更があった時だけ手動で更新する「通信の互換性」を示すフラグ。

このように分離することで、頻繁なコード更新（ローリングリリース）を行いながら、クライアントとの互換性チェックを安定して継続できます。