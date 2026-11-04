# 現行バージョンに対する記法調査（chunisupport-api）

調査日: 2026-11-04。
調査対象コミット: `b965119eef26ffc81f61ff6fb6c9fd7a7bc005d3`。

このレポートは上記コミットのソース・設定と、調査環境にインストール済みの依存関係を基準にした静的調査の記録です。レポートを追加するブランチ先端の再調査結果ではありません。行番号は調査対象コミット基準です。コードの修正は行っていません。

## 対象バージョン

`go.mod` は Go 1.27.0、toolchain は Go 1.27.3。Echo v5.4.0 等の依存宣言とソースを確認した。

## 標準機能への更新候補

以下は既存の書き方だけで不具合を意味するものではない。

- `internal/infra/repository/user_data_transfer_import.go:108`: map のキーをループでスライスに収集し、112行で `sort.Strings(result)`。`slices.Sorted(maps.Keys(unresolved))` が候補。ただし空の場合の nil/空スライスの違いが呼び出し側に影響しないか確認する。
- `internal/app/handler/compat/reiwa/dto.go:217`: `sort.Slice`。`slices.SortFunc` と `cmp.Compare` が候補。非推奨 API ではないため優先度は低い。
- `internal/domain/entity/api_token.go:135,143`: `copyValue := *value; return &copyValue` は Go 1.26 以降の `return new(*value)` にできる。同様のポインター複製処理はほかのエンティティ等にもある。nil チェックは引き続き必要。
- `internal/infra/repository/overpower_denominator_provider_impl.go:147` 付近: map をループでコピーしている2か所は `maps.Copy` にできる。公式ツールでも検出。
- `internal/infra/repository/player_locked_song_repository_impl.go:106` 付近など: ループ内の SQL 文字列の `+=` は `strings.Builder` にできる。公式ツールでも検出。
- `errors.As` は本番コードに18か所。Go 1.26 以降の `errors.AsType[T]` にできる候補がある。出力引数の使い方を確認して置換する。既存の API は非推奨ではない。
- `internal/infra/songbatch/importer/` の5ファイルに `os.IsNotExist(err)` が各1か所。`errors.Is(err, fs.ErrNotExist)` が現在の推奨形。いずれも `os.Stat` の直後なので、この書き方だけで不具合とは言えない。

## 公式ツールによる検出

Go 1.27.3 の `go fix -diff ./...` を実行した。差分は適用していない。

| 差分対象ファイル数 | うち本番コード | 差分ハンク数 |
| ---: | ---: | ---: |
| 55 | 9 | 197 |

テストコードではポインター生成ヘルパーのインライン化・`new(値)` 化が多い。追加側の `new(...)` を含む行は268行で、これは変更件数や不具合件数ではない。本番コードでは `errors.AsType`、ポインター生成、`maps.Copy`、`strings.Builder` などが検出された。

本番コードの自動検出対象:

- `internal/app/apierror/mapping.go`
- `internal/app/handler/compat/chunirec/error_handler.go`
- `internal/app/handler/compat/reiwa/error_handler.go`
- `internal/config/config.go`
- `internal/dto/api_v1/song_dto.go`
- `internal/dto/api_internal/song_dto.go`
- `internal/infra/songbatch/datasource/http_retry.go`
- `internal/infra/repository/overpower_denominator_provider_impl.go`
- `internal/infra/repository/player_locked_song_repository_impl.go`

## 確認結果と限界

- 本番コードの `io/ioutil`、`interface{}`、`strings.Title`、`rand.Seed` は検出していない。
- 自動検出結果と手動で確認した候補は区別している。公式ツールの差分も個別レビューしてから適用する。
- 今回は静的調査であり、全テスト、実行時動作、依存パッケージ全件の外部リリースノート照合は実施していない。
