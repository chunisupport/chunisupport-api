# バージョニング方針案

ローリングリリース運用に適した、実用的かつ自動化可能なバージョニング方針についてまとめます。

## 1. バージョン情報の形式

「人間への分かりやすさ」と「開発時の追跡可能性」を両立するため、バージョン情報は以下の2項目に分けて扱います。

- **コミット日時**: `YYYYMMDD`
- **Git短縮ハッシュ**: `a1b2c3d`

固定の表示用フォーマットは定義しません。
コミットハッシュだけを表示したい場面や、日付とハッシュを両方表示したい場面があるため、表示側が必要な粒度を選べるようにします。
`v` プレフィックスは付けません。

### 各項目の役割
- **日付 (CalVer)**: アプリケーションの鮮度（どのコミット日時のものか）をユーザーに伝えます。
- **Gitハッシュ**: 開発者がバグ報告を受けた際、どの時点のコードに不具合があるのかを完全に特定するために使用します。

## 2. 定義と自動化

バージョン情報はソースコードにハードコードせず、GitHub Actions 上のビルド時にのみ自動注入します。
ローカルビルドでは `dev` を返す前提とし、ローカル環境向けのバージョン埋め込みは行いません。

### 定義場所 (`internal/info/info.go`)
```go
var (
    CommitDate = "dev" // コミット日時: YYYYMMDD
    Revision   = "dev" // Git短縮ハッシュ: a1b2c3d
)
```
※現在はconstでやっているので修正必須

固定の `Version` 文字列は持たず、レスポンスやログでは `CommitDate` と `Revision` を必要に応じて個別に使用します。
これにより、公開用の日付表記、開発者向けのハッシュ表示、両方を並べた表示を同じ情報源から安全に生成できます。

### 自動注入 (GitHub Actions / Build flags)
GitHub Actions のビルド時に以下のフラグを指定することで、変数を動的に書き換えます。
日時は「ビルド日時」ではなく「コミット日時」を使用します。

```bash
COMMIT_DATE=$(git show -s --format=%cd --date=format:%Y%m%d HEAD)
REVISION=$(git rev-parse --short HEAD)

go build -ldflags "-X 'github.com/chunisupport/chunisupport-api/internal/info.CommitDate=${COMMIT_DATE}' \
                   -X 'github.com/chunisupport/chunisupport-api/internal/info.Revision=${REVISION}'"
```

既存のリリースビルドでは `-trimpath` と `-ldflags="-s -w"` を使用しているため、実装時は既存の最適化フラグを維持したまま `-X` を追加します。

```bash
COMMIT_DATE=$(git show -s --format=%cd --date=format:%Y%m%d HEAD)
REVISION=$(git rev-parse --short HEAD)

go build -trimpath \
  -ldflags="-s -w -X github.com/chunisupport/chunisupport-api/internal/info.CommitDate=${COMMIT_DATE} -X github.com/chunisupport/chunisupport-api/internal/info.Revision=${REVISION}" \
  -o "${BINARY_NAME}" .
```

## 3. 配信・露出戦略

セキュリティと利便性のバランスを考え、情報の粒度をエンドポイントごとに分けます。

### A. 一般公開用 (`GET /`)
不特定多数がアクセスできる場所では、`CommitDate` のみを公開し、リビジョン（ハッシュ）は伏せます。

**レスポンス例**:
```json
{
  "app_name": "chunisupport-api",
  "commit_date": "20240528"
}
```

### B. 管理者・開発用 (`GET /health`)
認証済みの管理者のみがアクセスできるエンドポイントでは、デバッグに必要な全情報を公開します。
現状の `/health` は空レスポンスですが、現在は利用していないため後方互換性を破壊する仕様変更として扱います。
実装時は `docs/API.md` と関連テストも新仕様へ更新します。

**レスポンス例**:
```json
{
  "status": "ok",
  "commit_date": "20240528",
  "revision": "a1b2c3d",
  "go_version": "go1.26.4"
}
```

`go_version` はビルド時注入ではなく、実行時に `runtime.Version()` から取得します。

起動ログなど人間が読む表示では、`CommitDate` と `Revision` を必要な粒度で出力します。
`v` プレフィックスは付けません。現在の起動ログにある `v` プレフィックスは削除します。

**表示例**: `chunisupport-api commit_date=20240528 revision=a1b2c3d`

## 4. プロトコル互換性の管理

`CommitDate` / `Revision`（ビルド識別子）とは別に、**`SupportedAppVersions`（プロトコルバージョン）**を引き続き保持します。

- **`CommitDate` / `Revision`**: コードが更新されるたびに自動で上がる「どのバイナリか」を示す識別子。
- **`SupportedAppVersions`**: 通信プロトコルやJSON構造に破壊的変更があった時だけ手動で更新する「通信の互換性」を示すフラグ。

このように分離することで、頻繁なコード更新（ローリングリリース）を行いながら、クライアントとの互換性チェックを安定して継続できます。
