# リファクタリング指摘書 (2026-08-24時点)

本ドキュメントは、現行コードベースを再確認したうえで、**未解決の改善点のみ**を整理したものです。
解消済みの項目、現行実装では成立しない項目、他項目と重複する指摘は削除または統合しています。
2026-08-24 に、前回確認後に追加されたデータ移行、プレイヤーメトリック履歴、楽曲スナップショット、システムメンテナンスの各機能を含めて再棚卸ししました。

## 優先度定義

- **Critical (緊急)**: セキュリティ上の重大な欠陥、または主要機能の停止に直結する問題。即時対応が必要。
- **High (高)**: データ破損・認可不備・アーキテクチャの根幹に関わる問題。優先して対応が必要。
- **Medium (中)**: 安定性、性能、保守性、API契約を明確に損なう問題。機能追加の前に解消することが望ましい。
- **Low (低)**: コード品質や一貫性に関わる問題。計画的に解消する。

## 対象範囲

- Goコード: `cmd`, `internal` 配下の本番コードおよび関連テスト
- データベース: `migration` 配下のマイグレーション、`migration/schema_mysql.sql`
- ドキュメント: `README.md`, `ARCHITECTURE.md`, `docs/*.md`
- レポート: `_report/*.md`

## 確認結果

- `go test ./...`: 成功
- `go vet ./...`: 成功
- コンパイルや既存テストでは検出されない、認可・並行更新・境界設計・API契約の問題を中心に記載しています。

## 作業者へ注意

解決した事項は「解決済み」と追記せず、**必ずこの文書から削除してください**。
過去の適用済みマイグレーションは変更せず、必要なスキーマ変更は新しいマイグレーションとして追加してください。

---

## 課題一覧

### セキュリティ (SEC)

| ID | 優先度 | 概要 | 詳細・対応方針 |
|---|---|---|---|
| **SEC-04** | **Medium** | HTTPタイムアウトがEchoの既定値に依存 | `internal/app/server.go:56-60` は `echo.StartConfig` へ移行済みで、Echo v5.2.1は内部で `ReadTimeout=30秒` を設定しています。`ReadHeaderTimeout` / `IdleTimeout` は標準ライブラリのフォールバックでこの値を使いますが、`WriteTimeout` は0のままです。依存ライブラリの既定値に任せず、`BeforeServeFunc` で `ReadHeaderTimeout` / `ReadTimeout` / `WriteTimeout` / `IdleTimeout` を用途に合わせて明示すべきです。 |
| **SEC-05** | **Medium** | DB接続のTLS設定がない | `internal/infra/db/connection.go:78-84` のMySQL DSNに `tls` 指定がなく、`internal/config/config.go:113-125` にもTLS設定がありません。DBが同一ホストまたは信頼できる閉域網に限定されない環境では、TLSを設定可能にして本番で必須化すべきです。 |
| **SEC-07** | **Low** | ログの機微情報・サニタイズ方針が統一されていない | `internal/usecase/firebase_auth_usecase.go:107`、`internal/usecase/user_credential_usecase.go:179-210` などがFirebase UIDを記録します。標準・chunirec・reiwaの各エラーハンドラーにはCR/LF除去が個別実装されていますが、全ログ属性へ適用される共通方針はありません。識別子の記録要否、マスキング、制御文字除去、保存期間、閲覧権限を統一すべきです。 |
| **SEC-10** | **Medium** | APIトークンに有効期限・権限スコープがない | APIトークンは32バイト乱数で生成され、SHA-256ハッシュだけを保存します。現行実装では名前付きトークンを1ユーザー最大10件まで発行でき、個別削除と `last_used_at` の記録もあります。一方、`expires_at` と権限スコープはなく、漏えいしたトークンは手動削除まで有効です。また `/v1` には EDITOR / ADMIN 向けの更新APIもあるため、トークンは所有ユーザーの現在のロール権限をそのまま行使できます。有効期限を導入し、既定を読み取り専用とした用途別スコープを定義するか、長期・全権限トークンを許容する明示的な運用方針と失効手順を定めるべきです。 |
| **SEC-15** | **Low** | 管理者によるユーザー削除の監査ログに実行者が残らない | `DeleteUser` は strict な Firebase 認証、ADMIN ロールのミドルウェア、Usecase内の再検証で認可されています。管理操作に recent sign-in を必須とする要件がない限り、その欠如だけを脆弱性とは扱いません。一方、`internal/usecase/user_usecase_impl.go:411` の成功ログは削除対象のユーザー名・IDだけを記録し、実行した管理者を記録しません。少なくとも実行者IDと削除対象IDを同じ監査イベントへ残し、`OPS-001` のリクエストIDとも関連付けるべきです。step-up認証を導入する場合は、管理操作全体の認証強度ポリシーとして別途定義します。 |

### パフォーマンス (PERF)

| ID | 優先度 | 概要 | 詳細・対応方針 |
|---|---|---|---|
| **PERF-003** | **Medium** | ユーザーレコードAPIが通常・WORLD'S ENDを全件返却 | `GetUserProfileWithRecords` / `GetUserProfileRecordView` は通常譜面とWORLD'S ENDの全レコードを取得・DTO化し、リポジトリクエリにも `LIMIT` がありません。楽曲単位APIとrating viewは分離済みですが、一覧レスポンスは収録譜面数に比例して肥大化します。用途別エンドポイントまたはページネーションを導入すべきです。旧 `PERF-004` は本項目の実装原因と重複するため統合しました。 |
| **PERF-005** | **Low** | 楽曲一覧・互換マスタAPIが毎回全楽曲と全譜面を生成 | internal/v1の楽曲一覧に加え、`internal/app/handler/compat/chunirec/chunirec_handler.go:36-49` と `internal/app/handler/compat/reiwa/reiwa_handler.go:29-40` も全楽曲・全譜面を取得します。互換仕様上ページネーションできない場合は、更新時キャッシュ、ETag、Last-ModifiedなどでDB取得・変換・転送を抑制すべきです。 |
| **PERF-006** | **Medium** | rating viewがOP対象判定のため通常レコードを重複取得 | `internal/usecase/user_usecase_impl.go:273-291` は `FindByPlayerIDForRating` の直後に、OP対象フラグ算出だけのため `FindByPlayerID` で通常レコード全件を再取得します。OP対象譜面IDを含む用途別クエリ、または1回の取得で両用途を満たす読み取りモデルにすべきです。 |
| **PERF-007** | **Medium** | chunirec互換APIがレスポンスに不要な集約を全件取得 | `/users/show` は `GetUserProfileWithRecords` を呼びながらレコードを一切使用しません。`/records/showall` はRecordView経由でWORLD'S END、称号、動的OP計算まで取得して捨て、さらにジャンル解決だけのため全楽曲・全譜面を取得します（`internal/app/handler/compat/chunirec/chunirec_handler.go:90-123,145-159`）。各レスポンス専用のquery portを用意し、必要列を1回で取得すべきです。 |
| **PERF-008** | **Medium** | フレンド受信申請が無上限かつ全件返却 | 上限100件は送信者側の外向き `pending` / `accepted` だけに適用され、1ユーザーが受信できる申請数には上限がありません。`internal/infra/repository/friendship_repository_impl.go:117-132` の受信一覧にも `LIMIT` がないため、多数アカウントから集中申請されるとレスポンスとDB負荷が無界に増えます。受信上限、ページネーション、申請作成のユーザー単位レート制限を組み合わせるべきです。 |

### データ整合性・並行実行 (DATA)

| ID | 優先度 | 概要 | 詳細・対応方針 |
|---|---|---|---|
| **DATA-004** | **Medium** | 再計算バッチのマスタ「スナップショット」が原子的でない | `internal/infra/repository/player_data_batch_repository_impl.go:23-69` はversion、songs、charts、slots、Player上限を5本のautocommit SELECTで取得します。途中で編集がcommitされると、旧楽曲属性と新譜面定数が混在したrun全体スナップショットになります。`REPEATABLE READ` または明示的なconsistent snapshotを使うread-only transactionでまとめて取得し、そのトランザクションから構築した値だけをrun中に使用すべきです。 |
| **DATA-005** | **Medium** | 楽曲スナップショットが取得時・公開時とも単一世代にならない | `internal/app/songexport/exporter.go:68-127` は通常楽曲とWORLD'S END楽曲を別々のautocommit読み取りで取得し、4種類の固定キーへ順次PUTします。途中のマスタ更新で生成物の基準時点がずれるほか、後続PUTの失敗時は先に成功したオブジェクトだけが新世代となります。この部分更新は `docs/song_snapshot_export.md` に明記されていますが、利用者が4形式を同一世代として扱う契約を保証できません。DB取得をconsistent snapshotへまとめ、世代付きキーへ全成果物を保存してからmanifestまたは公開ポインタを最後に切り替えるべきです。 |

### 信頼性・運用 (OPS)

| ID | 優先度 | 概要 | 詳細・対応方針 |
|---|---|---|---|
| **OPS-001** | **Low** | リクエストIDがない | `X-Request-ID` の受理・生成とログへの相関ID付与がなく、複数ログを1リクエスト単位で追跡できません。共通ミドルウェアで設定し、アプリケーションログとアクセスログへ同じ値を渡すべきです。 |
| **OPS-002** | **Low** | 通常DBクエリにアプリケーション側タイムアウトがない | `context.Context` は伝播されていますが、`context.WithTimeout` はDB起動再試行などに限られ、通常リクエストのDBアクセス上限はありません。クライアント接続が維持された長時間クエリを抑止できるよう、ユースケースまたはトランザクション境界で設定可能な期限を設けるべきです。 |
| **OPS-003** | **Medium** | メンテナンス状態がAPIプロセス間で同期されない | `internal/usecase/system_maintenance_usecase.go:29-105` は起動時にDB状態を1回読み込み、以後はプロセス内の `atomic.Pointer` だけを参照・更新します。状態更新を受けたプロセス以外はDB変更を再読込しないため、複数プロセスまたは複数インスタンス構成では、再起動まで通常運用とメンテナンス中の判定が分裂します。単一プロセス運用を必須条件として文書化するか、DBのversionを短周期で確認する共有キャッシュ、通知、または各リクエストで一貫した共有状態を参照する方式へ変更すべきです。 |

### 実装品質・保守性 (QUAL)

| ID | 優先度 | 概要 | 詳細・対応方針 |
|---|---|---|---|
| **QUAL-002** | **Medium** | セキュリティヘッダーの付与元がコード上で保証されない | EchoのSecure相当ミドルウェアがなく、HSTS、`X-Content-Type-Options`、`X-Frame-Options` 等をアプリケーションでは設定していません。リバースプロキシで付与する方針ならインフラ設定と検証手順を文書化し、そうでなければ共通ミドルウェアで設定すべきです。 |
| **QUAL-010** | **Medium** | Domain層の `Executor` が `sqlx` に依存 | `internal/domain/repository/executor.go` が `*sqlx.Rows` / `*sqlx.Row` とSQL実行APIを公開します。新設のFriendship・ランキング・お気に入り系portにも `Executor` 引数が広がり、Domainがインフラ実装詳細を定義する範囲が拡大しています。トランザクション境界をUsecase向け抽象へ切り出し、repository portから `sqlx` 型を排除すべきです。 |
| **QUAL-011** | **Low** | 難易度文字列の大文字統一方針と実装・API仕様が不一致 | `internal/info/statistics.go:15-54` は難易度パスを小文字定数で定義し、検索に `strings.ToLower()` を使用します。AGENTS.mdの「DB・コード・API入出力で大文字統一」「比較に `ToLower` を使わない」方針と衝突しています。公開済みパスとの互換性を確認したうえで、正規化・定数・API.mdを一つの規約へ揃えるべきです。 |

### アーキテクチャ・ドメイン (ARCH / DOM)

| ID | 優先度 | 概要 | 詳細・対応方針 |
|---|---|---|---|
| **ARCH-004** | **Medium** | フレンド認可依存を匿名interfaceで後注入 | User、ScoreHistory、LockedSong、FavoriteSongの各Usecaseはコンストラクタで `FriendshipRepository` を受け取らず、`internal/app/router.go:151-192` が匿名interfaceへの型アサーションで `SetFriendshipRepository` を呼びます。アサーション失敗時も起動を継続し、非公開フレンド閲覧だけが静かに404へ退行します。認可に必要な依存はコンストラクタ契約へ明示し、欠落時は生成エラーにすべきです。 |
| **ARCH-005** | **Medium** | Player集約の永続化経路が部分更新と直接SQLへ分裂 | `internal/domain/repository/player_repository.go:25` に禁止方針と逆行する `UpdateCalculatedRatings` が残り、通常登録の `Save`、未解禁曲更新の `Save`、再計算バッチの直接UPDATEと更新経路が分散しています。並行更新時の所有列とロック規約が揃わない根因になっているため、Playerの変更メソッドとロック済み集約の `Save` へ統一し、バッチ固有の競合条件も同じ規約上で定義すべきです。 |
| **DOM-006** | **Medium** | GoalのJSON表現がDomainからAPIまで型安全でない | `internal/domain/entity/goal.go` は `AchievementParams` / `Attributes` を `[]byte` で保持し、`internal/dto/api_internal/goal_dto.go` は対応値を `map[string]any` で公開します。インフラ表現がDomainへ入り、コンパイル時にスキーマを保証できません。achievement typeごとの値オブジェクトまたは明示的なunion型へ移し、永続化モデルだけでJSONへ変換すべきです。旧 `DTO-001` は本項目へ統合しました。 |
| **DOM-007** | **Low** | 本番パッケージにpanicする `Must` コンストラクタが残存 | `username.MustNewUserName`、`reauthtoken.MustNew`、`playername.MustNewPlayerName` は本番パッケージの公開APIとして残っています。現行の非テストコードからは呼ばれていませんが、誤用余地をなくすためテストヘルパーへ移すか非公開化すべきです。 |

### インフラ層 (INFRA)

| ID | 優先度 | 概要 | 詳細・対応方針 |
|---|---|---|---|
| **INFRA-007** | **Medium** | `FindAllWithPlayer` と管理者版の実装が重複 | `internal/infra/repository/user_repository_impl.go:102-281` で、SELECT、LIKE条件、rows走査、VO変換、Player組み立てが二重実装されています。公開範囲の条件だけを引数・query objectで分離し、共通の読み取り処理へまとめるべきです。 |
| **INFRA-017** | **Medium** | repositoryのエラーラップが原エラーを切断 | `friendship_repository_impl.go:197-201`、`friend_chart_ranking_query_service_impl.go:244-248`、お気に入り・未解禁曲系のwrapperは、sentinelだけを `%w`、原エラーを `%v` で整形します。その結果 `errors.Is(context.Canceled)` や `errors.As(*mysql.MySQLError)` が失敗し、キャンセル・制約違反・再試行可否を上位で分類できません。sentinelとcauseの両方をラップできる形へ統一すべきです。 |

### ハンドラー / ルーター層 (HDL)

| ID | 優先度 | 概要 | 詳細・対応方針 |
|---|---|---|---|
| **HDL-010** | **Low** | `knownFields` が入力構造体と手動同期 | `internal/app/handler/api_internal/me_handler.go:89-114` の未知フィールド検出は `PlayerDataPayload` のJSONタグ一覧を手書きしています。現在は一致していますが、項目追加時に警告だけがずれるため、厳格decoderまたはリフレクションで単一の定義から導出すべきです。 |
| **HDL-011** | **Low** | クエリパラメータの不正値を既定値へ黙ってフォールバック | `include_noplay` は5箇所で `strconv.ParseBool` のエラーを破棄し、楽曲一覧の `include_deleted` は文字列が厳密に `"true"` の場合だけtrue、Userの未知 `view` は全件表示へフォールバックします。不正値を400系で返す共通parserへ統一すべきです。 |
| **HDL-012** | **Low** | 厳格JSONデコードの適用が不統一 | `BindStrictJSON` 導入後も本番コードに `c.Bind` が12箇所残り、`player_locked_song_handler.go` 内でも両方式が混在します。未知フィールド、Content-Type、複数JSON値の扱いを全JSONエンドポイントで統一すべきです。 |
| **HDL-013** | **Medium** | 互換APIの専用エラー変換が認証・入力エラーを503化 | chunirec/reiwaの専用switchは400/404/405/429/503だけを扱うため、APIトークンの401、通常バリデーションの422、内部エラーの500を503へ変換します。さらにGroup middlewareはルート不一致の404/405では実行されず、同じprefix内でレスポンス形式が変わります。`docs/API.md` の401/500および標準エラーコード記載とも不一致です。互換APIのstatus・body変換を共通化し、認証ミドルウェアを含むcontract testを追加すべきです。 |
| **HDL-014** | **Medium** | スコア履歴APIが楽曲IDと削除済み楽曲を検証しない | internal/v1の両ScoreHistoryHandlerはパスの `id` を、internal版は `displayid` を `ValidateDisplayID` に通さずUsecaseへ渡します。Usecaseは削除済みも返す `SongRepository.FindByDisplayID` / WORLD'S END queryを使いながら `IsDeleted` を確認しないため、他の公開レコードAPIでは隠れる削除済み楽曲の履歴を取得できます。境界で形式検証し、Usecaseで楽曲種別と公開状態を404へ正規化すべきです。 |
| **HDL-015** | **Medium** | chunirec互換の `is_clear` がFAILEDでもtrue | `internal/app/handler/compat/chunirec/dto.go:121` は `record.ClearLamp != nil` だけでクリア判定します。`FAILED` は有効な非nilマスタ値なので、FAILEDレコードも `is_clear: true` になります。ランプ名またはドメインのクリア判定メソッドを使い、FAILEDケースのテストを追加すべきです。 |

## まとめ

- 2026-08-24 の再レビューでは、既存の未解決項目に加えて、**楽曲スナップショットの世代混在 (`DATA-005`)**と**メンテナンス状態のプロセス間不整合 (`OPS-003`)**を追加しました。データ移行機能はサイズ制限、署名、全体検証、参照再検証、トランザクションを備えており、今回の確認範囲では独立した追加指摘はありません。
- 直近追加の互換APIでは、不要な全件取得、401/422/500の503化、FAILEDのクリア誤判定が見つかりました。機能単位の正常系テストだけでなく、認証・入力不正・非公開・削除済み・並行更新を含む境界テストが必要です。
- プレイヤーデータの真正性（クライアント申告スコア）はプロダクト前提として本リストの脆弱性対象外とします。管理用の `is_suspicious` は別途運用で扱う想定です。
