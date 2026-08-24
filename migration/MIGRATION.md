# データベースマイグレーションとスキーマ

## マイグレーションツール

このプロジェクトでは、データベースのスキーマ管理とマイグレーションのために [**golang-migrate**](https://github.com/golang-migrate/migrate) を使用しています。インストールにはバイナリのダウンロードではなく、以下のコマンドを利用してください。

```plaintext
go install -tags mysql github.com/golang-migrate/migrate/v4/cmd/migrate@latest
``` 

マイグレーションファイルは `migration/mysql` ディレクトリに格納されており、`*.up.sql` ファイルがスキーマの追加・変更、`*.down.sql` ファイルが変更のロールバックを定義します。

`migration/sqlite`はMySQL統計移行前の履歴として残しており、新規環境では適用しません。

現在の本番対象データベースはMySQL 8.4です。

## 主要テーブルの概要

以下は、アプリケーションのコア機能に関連する主要なテーブルの概要です。

### ユーザー・認証関連

#### `users`
- **役割**: このシステムのユーザーアカウント情報を格納します。
- **主なカラム**:
    - `id`: ユーザーのユニークID。
    - `username`: アプリ内で一意なユーザー名（ユニーク制約）。
    - `firebase_uid`: Firebase Authentication の UID（ユニーク制約、NULL可）。
    - `account_type_id`: `account_types`マスタへの外部キー（PLAYER/EDITOR/ADMIN/EXTDEV）。
    - `player_id`: `players`テーブルへの外部キー（ユニーク制約、NULL可）。
    - `is_private`: プライバシー設定（0=公開, 1=非公開）。
    - `is_suspicious`: 不審アカウントフラグ（0=正常, 1=不審）。
    - `created_at`, `updated_at`: 作成日時、更新日時。

#### `api_tokens`
- **役割**: API認証用のトークンを管理します。
- **主なカラム**:
    - `id`: トークンのユニークID。
    - `user_id`: `users`テーブルへの外部キー。
    - `name`: ユーザー内で一意の管理用名。
    - `hashed_token`: トークンのハッシュ値。
    - `token_prefix`: 新規発行トークンの表示用先頭5文字。旧仕様から移行したトークンはNULL。
    - `last_used_at`: 認証に最後に使用した日時。
    - `created_at`: 作成日時。

### システム運用関連

#### `system_maintenance`
- **役割**: API全体のメンテナンス状態と公開用コメントを単一行で永続化します。
- **主なカラム**:
    - `id`: singletonを表す固定ID。`CHECK (id = 1)` により1だけを許可します。
    - `enabled`: メンテナンス状態（0=通常、1=メンテナンス中）。
    - `comment`: 公開用コメント（最大1,000文字）。無効時は空文字です。
    - `updated_by_user_id`: 最後に状態を変更したユーザー。`users.id` を参照し、ユーザー削除時はNULLになります。
    - `updated_at`: 状態の最終更新日時（マイクロ秒精度）。
- **初期行**: `id = 1`、`enabled = false`、空コメントで作成します。行が欠落している場合、APIは設定不備として起動に失敗します。

### プレイヤー・ゲームデータ関連

#### `players`
- **役割**: CHUNITHMのプレイヤーとしてのプロフィール情報を格納します。
- **主なカラム**:
    - `id`: プレイヤーのユニークID。
    - `user_id`: `users`テーブルへの外部キー（ユニーク制約）。
    - `player_name`: プレイヤー名（20文字まで）。
    - `player_level`: プレイヤーレベル。
    - `official_player_rating`: 公式データから取得したレーティング（DECIMAL(4,2)）。
    - `calculated_player_rating`: スコアから計算したレーティング（DECIMAL(6,4)）。
    - `new_average_rating`: 新曲枠の平均レーティング（DECIMAL(6,4)）。
    - `best_average_rating`: ベスト枠の平均レーティング（DECIMAL(6,4)）。
    - `class_emblem_id`, `class_emblem_base_id`: クラスエンブレム情報への外部キー。
    - `last_played_at`: 最終プレイ日時。
    - `overpower_value`: 保存済みのOVER POWER値。割合はAPI返却時に最新マスタから随時計算。
    - `data_collected_at`: CHUNITHM-NETからのデータ取得完了日時。取得前の既存データはNULL。
    - `created_at`, `updated_at`: 作成日時、更新日時。

#### `player_records`
- **役割**: プレイヤーの通常譜面に対するスコア記録を格納します。
- **主なカラム**:
    - `player_id`, `chart_id`: プレイヤーと譜面の複合主キー。
    - `score`: スコア（0～1,010,000）。
    - `clear_lamp_id`: クリアランプID（`clear_lamp_types`マスタ参照、デフォルト1=FAILED）。
    - `combo_lamp_id`: コンボランプID（`combo_lamp_types`マスタ参照、デフォルト1=NONE）。
    - `full_chain_id`: フルチェインID（`full_chain_types`マスタ参照、デフォルト1=NONE）。
    - `slot_id`: スロット種別（`slots`マスタ参照）。
    - `slot_order`: スロット内の順序（1～255、NULL可）。
    - `updated_at`: 更新日時。

#### `player_worldsend_records`
- **役割**: プレイヤーのWORLD'S END譜面に対するスコア記録を格納します。
- **主なカラム**:
    - `player_id`, `worldsend_chart_id`: プレイヤーとWORLD'S END譜面の複合主キー。
    - `score`: スコア（0～1,010,000）。
    - `clear_lamp_id`, `combo_lamp_id`, `full_chain_id`: クリア状況を示すマスタへの外部キー。
    - `updated_at`: 更新日時。

#### `player_honors`
- **役割**: プレイヤーに装着されている称号を管理します。
- **主なカラム**:
    - `player_id`, `slot`: プレイヤーIDとスロット番号（1=上段, 2=中段, 3=下段）の複合主キー。
    - `honor_id`: `honors`テーブルへの外部キー。
    - `created_at`: 作成日時。

### 楽曲・譜面関連

#### `songs`
- **役割**: 楽曲の基本情報を格納します。
- **主なカラム**:
    - `id`: 楽曲のユニークID。
    - `display_id`: 16進数16文字の表示用ID（ユニーク制約）。
    - `title`, `reading`, `artist`: 楽曲のタイトル（300文字まで）、読み（300文字まで、NULL可）、アーティスト名（300文字まで）。
    - `genre_id`: `genres`マスタへの外部キー。
    - `bpm`: BPM（NULL可）。
    - `released_at`: リリース日（DATE型、NULL可）。
    - `official_idx`: 公式サイトのJSONから取得できるユニークID（ユニーク制約）。
    - `jacket`: ジャケット画像ファイル名（20文字まで）。
    - `is_worldsend`: WORLD'S END楽曲フラグ（0=通常, 1=WORLD'S END）。
    - `is_deleted`: 論理削除フラグ（0=有効, 1=削除済み）。
    - `updated_at`: 更新日時。

#### `charts`
- **役割**: 通常楽曲の譜面情報を格納します。一つの楽曲に対して複数の難易度（BASIC, ADVANCED, EXPERT, MASTER, ULTIMA）の譜面が存在します。
- **主なカラム**:
    - `id`: 譜面のユニークID。
    - `song_id`: `songs`テーブルへの外部キー（`ON DELETE CASCADE`設定）。
    - `difficulty_id`: `difficulties`マスタへの外部キー。
    - `const`: 譜面定数（DECIMAL(3,1)）。レーティング計算の基礎となります。
    - `is_const_unknown`: 譜面定数が未確定かどうかのフラグ（デフォルト1=未確定）。
    - `notes`: ノーツ数（NULL可）。
    - `notes_designer`: 譜面製作者名（100文字まで、NULL可）。
    - `updated_at`: 更新日時。
    - ユニーク制約: `(song_id, difficulty_id)`の組み合わせ。

#### `worldsend_charts`
- **役割**: WORLD'S END楽曲の譜面情報を格納します。WORLD'S ENDは1曲1譜面です。
- **主なカラム**:
    - `id`: 譜面のユニークID。
    - `song_id`: `songs`テーブルへの外部キー（`ON DELETE CASCADE`設定、ユニーク制約）。
    - `level_star`: WORLD'S END レベル（1～5、NULL可）。
    - `attribute`: WORLD'S END 属性（光、蔵、改、狂など、CHAR(1)）。
    - `notes`: ノーツ数（NULL可）。
    - `notes_designer`: 譜面製作者名（100文字まで、NULL可）。
    - `updated_at`: 更新日時。

### マスタテーブル

#### ゲームデータマスタ
- `genres`: ジャンルマスタ（POPS & ANIME、niconico、東方Project、VARIETY、イロドリミドリ、ゲキマイ、ORIGINAL）。表示順は `sort_order` で管理。
- `difficulties`: 譜面難易度マスタ（BASIC、ADVANCED、EXPERT、MASTER、ULTIMA）。
- `clear_lamp_types`: クリアランプ種別マスタ。
- `combo_lamp_types`: コンボランプ種別マスタ。
- `full_chain_types`: フルチェイン種別マスタ（NONE、FULL CHAIN GOLD、FULL CHAIN PLATINUM）。
- `class_emblems`: クラスエンブレムマスタ（1、2、3、4、5、inf）。
- `class_emblem_bases`: クラスエンブレムベースマスタ（1、2、3、4、5、inf）。
- `genres` / `difficulties` / `class_emblems` / `class_emblem_bases` / `clear_lamp_types` / `combo_lamp_types` / `full_chain_types`: `sort_order` カラムで0始まりの表示順を保持。
- `slots`: スロット種別マスタ（none、best、best_candidate、new、new_candidate）。
- `honor_types`: 称号種類マスタ（normal、copper、silver、gold、platina、rainbow、staff、ongeki、maimai、ultima、sp、phoenix_g、phoenix_p、phoenix_r、expert、master）。
- `account_types`: アカウント種別マスタ（PLAYER、EDITOR、ADMIN、EXTDEV）。
- `versions`: バージョンマスタ。CHUNITHMの各バージョン（無印からMateまで）の情報とリリース日を格納。

#### ゲームコンテンツマスタ
- `honors`: 称号マスタ。称号名、称号種別、画像URL等を格納。

---

## データ管理について

### 外部データソースとの連携
楽曲データの構築・更新は、このリポジトリとは別のバッチ処理用リポジトリで行われています。バッチ処理により、外部データソース（公式サイト、Chunirecなど）から取得した情報が、このAPIサーバーが使用する主要テーブル群（`songs`, `charts`など）に反映されます。

### マイグレーション履歴
- **000001**: 初期スキーマ。全マスタテーブル（genres, difficulties, class_emblems, clear_lamp_types, combo_lamp_types, slots, full_chain_types, honor_types, account_types, versions等）、楽曲・譜面関連テーブル（songs, charts, worldsend_charts）、ユーザー・認証関連テーブル（users, sessions, api_tokens, user_recovery_codes）、プレイヤー関連テーブル（players, player_records, player_worldsend_records, player_honors）、および各種インデックスを含む。
- **000002**: セッション自動クリーンアップイベントの追加。1時間ごとに期限切れのセッション（`expires_at < NOW()`）を削除するMySQLイベントスケジューラー（`cleanup_expired_sessions`）を設定。運用時は `event_scheduler = ON` の設定が必要。
- **000003**: `players.user_id` と `users.player_id` に外部キー制約を追加。
- **000004**: `worldsend_charts` の WORLD'S END 関連カラムを `we_kanji` / `we_star` から `attribute` / `level_star` へリネームし、CHECK制約を再作成。
- **000005**: `achievement_types` と `goals` テーブルを追加。
- **000006**: `users` テーブルに `firebase_uid` カラムとユニークインデックスを追加。
- **000007**: 順序を持つマスタテーブル（`difficulties`, `class_emblems`, `class_emblem_bases`, `clear_lamp_types`, `combo_lamp_types`, `full_chain_types`）に `sort_order` カラムを追加し、既存データへ明示的に表示順を投入。
- **000008**: `users` テーブルから `is_deleted` カラムを削除し、関連インデックスを整理。
- **000009**: `charts` テーブルと `worldsend_charts` テーブルに譜面製作者を保持する `notes_designer` カラムを追加。
- **000010**: `songs`、`charts`、`worldsend_charts` テーブルに `updated_at` カラムを追加し、重複・非効率なインデックスを整理（`idx_worldsend_charts_song_id` / `idx_charts_song_id` / `idx_sessions_user_id` を削除、`player_worldsend_records(player_id, updated_at DESC)` と `goals(user_id, created_at, id)` を追加）。
- **000011**: `players.overpower_value` の型を `DECIMAL(8,2)` → `DECIMAL(9,3)` へ、`players.overpower_percentage` の型を `DECIMAL(5,2)` → `DECIMAL(7,4)` へ変更。精度向上のため。
- **000012**: Firebase 認証への一本化に伴い、`cleanup_expired_sessions` イベント、`sessions` テーブル、`user_recovery_codes` テーブル、および `users.password_hash` カラムを削除。破棄された旧認証データは down でも復元されず、ロールバックではスキーマのみ復元される。
- **000013**: `player_records` の最新更新取得をプレイヤー単位で高速化するため、`idx_player_records_updated_at` を削除し、`player_records(player_id, updated_at DESC)` を追加。あわせて `player_worldsend_records` の単独 `updated_at` インデックス、`idx_goals_user_created_id` に包含される `idx_goals_user_id`、および不要になった `idx_songs_title` を削除した。
- **000014**: プレイヤーごとの解禁済み楽曲を保持する `player_locked_songs` テーブルを追加。
- **000015**: `genres` テーブルに `sort_order` カラムを追加し、ジャンルの表示順を投入。
- **000016**: `songs` テーブルに楽曲の読みを保持する `reading` カラムを追加。
- **000017**: `honors` テーブルの `image_url` を空文字デフォルトの非NULLに変更し、称号のユニーク制約を `(name, honor_type_id, image_url)` へ変更。`sp` 称号は空文字の `name` と画像URLの組み合わせで一意に扱えるようにする。
- **000018**: `players.overpower_percentage` カラムを削除。OVER POWER割合は保存値ではなく、レスポンス時点の最新マスタと未解禁設定から随時計算する。
- **000019**: バージョンマスタに「CHUNITHM Mate」（2026年7月2日稼働）を追加。
- **000024**: `players` テーブルにCHUNITHM-NETからのデータ取得完了日時を保持する `data_collected_at` カラムを追加。
- **000028**: `goals` テーブルにユーザー内の表示順を保持する `sort_order` カラムを追加。既存データは作成順で採番し、一覧用インデックスを `(user_id, sort_order, id)` へ変更。MySQLのDDLは暗黙コミットされるため、適用開始から完了までGoalの作成・更新・削除・並び替えを停止する。
- **000036**: レーティング帯マスタと譜面統計3表をMySQLへ追加。統計は別リポジトリのバッチが実行ごとに全削除して再生成する。
- **000037**: ユーザー所有の `goal_groups` を追加し、`goals.group_id` とグループ内 `sort_order` による目標分類・並び替えへ変更。既存目標は未分類のまま従来順を保持する。
- **000038**: `api_tokens` に名前、表示用prefix、最終利用日時を追加し、`user_id` の単独一意制約を削除して1ユーザー複数トークンに対応。既存ハッシュは変更せず、名前を「既存のトークン」、prefixと最終利用日時をNULLで移行する。
- **000039**: APIメンテナンス状態を永続化する `system_maintenance` テーブルと、無効状態の初期singleton行を追加。
- **000041**: プレイヤーの公式RATING・公式OVER POWER履歴を追加し、取得日時をマイクロ秒精度へ変更。
- **000042**: プレイヤー現在値と公式指標履歴の取得日時を秒精度へ変更。

#### 000028の失敗時復旧

`000028` は列追加、既存行への採番、`NOT NULL`化、インデックス置換を別々に実行します。失敗時に`migrate force 28`を実行する前に、必ず以下を確認します。

```sql
SHOW COLUMNS FROM goals LIKE 'sort_order';
SELECT COUNT(*) AS null_sort_order_count FROM goals WHERE sort_order IS NULL;
SHOW INDEX FROM goals WHERE Key_name IN (
  'idx_goals_user_created_id',
  'idx_goals_user_sort_order_id'
);
```

- `sort_order`列が存在しない場合は、`migrate force 27`でdirty状態を戻してから、修正済みの`000028`を再実行する。
- `sort_order`がNULL許容、またはNULL値が1件でもある場合は、書き込み停止を維持したまま次を実行する。

```sql
UPDATE goals g
INNER JOIN (
    SELECT
        id,
        ROW_NUMBER() OVER (
            PARTITION BY user_id
            ORDER BY created_at ASC, id ASC
        ) AS sort_order
    FROM goals
) ranked ON ranked.id = g.id
SET g.sort_order = ranked.sort_order;

ALTER TABLE goals
    MODIFY COLUMN sort_order SMALLINT UNSIGNED NOT NULL;
```

- 新インデックスが存在しない場合は、先に作成する。旧インデックスは新インデックスの存在を確認してから削除する。

```sql
CREATE INDEX idx_goals_user_sort_order_id
    ON goals(user_id, sort_order, id);

DROP INDEX idx_goals_user_created_id ON goals;
```

最後に`sort_order`が`NOT NULL`で、NULL件数が0件、新インデックスのみが存在することを確認してから`migrate force 28`でdirty状態を解消する。すでに新インデックスが存在する場合は、重複作成せず旧インデックスの削除から再開する。

### 000030 コースレコード

`course_classes`、`courses`、`player_course_records`を追加する。コースクラスは`1`～`5`、`inf`、`extra`の固定値であり、コースは論理削除に対応する。コーススコアは0～3,030,000点を保持する。

### 000037 目標グループ

適用中はGoalとGoalGroupの書き込みを停止する。MySQLのDDLは暗黙コミットされるため、失敗時は `goal_groups`、`goals.group_id`、`idx_goals_user_group_sort_order_id`、`fk_goals_group_user` の有無を確認し、up SQLの順序どおり不足分だけを適用してからバージョンを修復する。既存Goalは `group_id = NULL` の未分類となり、従来の `sort_order` を維持する。downでは現在のグループ順・グループ内順・未分類末尾の表示順をユーザー全体の連番へ変換してから `group_id` を削除する。

### 000038 APIトークン複数発行

upでは既存レコードと `hashed_token` を保持したまま管理用カラムを追加する。`name` は旧バイナリからのname未指定INSERTでNULL行が再発しないよう、「既存のトークン」を既定値とする。

移行中も外部APIのトークン認証は継続できる。一方、旧バイナリの `DELETE /internal/auth/api-tokens` はユーザーの全トークンを削除するため、マイグレーション開始前から旧インスタンスの排出完了までAPIトークン管理CRUDを停止する。適用順は「管理CRUD停止 → 000038 up → 全インスタンスを新バイナリへ切替 → 管理CRUD再開」とする。

downで1ユーザー1トークンへ戻す場合、複数発行済みのユーザーは `created_at DESC, id DESC` で最新の1件だけを保持し、それ以外を削除する。ロールバック前に必ず影響を確認すること。

### 000039 システムメンテナンス状態

`system_maintenance` は `id = 1` の単一行だけを保持します。MySQL 8.4で有効な `CHECK (id = 1)`、主キー、リポジトリ実装の固定ID指定を組み合わせてsingletonを保証します。`updated_by_user_id` は `users.id` を参照し、ユーザー削除時も運用状態を維持できるよう `ON DELETE SET NULL` とします。

APIバイナリは起動時にこの行を必須で読み込むため、デプロイでは `000039` のupを新バイナリより先に適用してください。up直後の初期状態は通常運用です。

downはテーブルと保存済みの状態・監査情報を削除します。旧バイナリへのロールバックでは原則としてテーブルを残し、downは保存データを破棄してよい場合だけ適用してください。

### 000031 コース作成日時の削除

`courses` テーブルの `created_at` カラムを削除する。コースマスタでは作成日時を利用しないため、更新日時のみを保持する。

### 000033 コース表示用ID

`courses` テーブルに16文字の小文字16進数で表す一意な `display_id` を追加する。既存コースにはマイグレーションで暗号学的乱数を採番し、以後の個別コースAPIでは `official_idx` ではなく `display_id` を外部識別子として利用する。

### 000041 プレイヤー公式RATING・公式OVER POWER履歴

`players.official_player_rating`を`NOT NULL`へ変更し、公式RATINGと公式OVER POWERを同一取得時点の組として保存する`player_metric_histories`を追加する。連続取得の順序と履歴主キーを正確に扱うため、現在値と履歴の`data_collected_at`はともにマイクロ秒精度の`TIMESTAMP(6)`とする。

適用前に次のSQLで既存NULLを必ず監査する。該当行を`0`などの推測値で補完してはならず、CHUNITHM-NETから公式データを再取得してからマイグレーションを適用する。

```sql
SELECT id, user_id
FROM players
WHERE official_player_rating IS NULL;
```

該当行が残っている場合、`ALTER TABLE`は失敗して移行を停止する。履歴は変更時のみ保存し、保持件数に上限を設けない。

デプロイ順は「プレイヤーデータ登録を停止 → NULL監査と公式データ再取得 → `000041` up → 新バイナリへ切替 → 登録再開」とする。旧バイナリは`rating: null`を受理し得る一方、新バイナリは履歴テーブルを必要とするため、登録を継続したままのローリング移行は行わない。

### 000043 公式OP%履歴

`players`と`player_metric_histories`に`official_overpower_percent DECIMAL(5,2) NULL`を追加する。既存の取得時点には公式OP%の原本がないためバックフィルせず、記録開始前の値は`NULL`のまま保持する。新バイナリへの切替後はプレイヤーデータ登録の`overpower.percentage`を必須とし、RATING・公式OVER POWER・公式OP%のいずれかが変化した場合に更新前の組を履歴化する。
