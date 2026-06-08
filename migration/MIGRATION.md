# データベースマイグレーションとスキーマ

## マイグレーションツール

このプロジェクトでは、データベースのスキーマ管理とマイグレーションのために [**golang-migrate**](https://github.com/golang-migrate/migrate) を使用しています。インストールにはバイナリのダウンロードではなく、以下のコマンドを利用してください。

```plaintext
go install -tags 'mysql sqlite' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
``` 

マイグレーションファイルは `migration/mysql` ディレクトリに格納されており、`*.up.sql` ファイルがスキーマの追加・変更、`*.down.sql` ファイルが変更のロールバックを定義します。

静的データ用のSQLiteスキーマは `migration/sqlite` ディレクトリに配置しています。

## 主要テーブルの概要

以下は、アプリケーションのコア機能に関連する主要なテーブルの概要です。

### ユーザー・認証関連

#### `users`
- **役割**: このシステムのユーザーアカウント情報を格納します。
- **主なカラム**:
    - `id`: ユーザーのユニークID。
    - `username`: アプリ内で一意なユーザー名（ユニーク制約）。
    - `firebase_uid`: Firebase Authentication の UID（ユニーク制約、NULL可）。
    - `account_type_id`: `account_types`マスタへの外部キー（PLAYER/EDITOR/ADMIN）。
    - `player_id`: `players`テーブルへの外部キー（ユニーク制約、NULL可）。
    - `is_private`: プライバシー設定（0=公開, 1=非公開）。
    - `is_suspicious`: 不審アカウントフラグ（0=正常, 1=不審）。
    - `created_at`, `updated_at`: 作成日時、更新日時。

#### `api_tokens`
- **役割**: API認証用のトークンを管理します。
- **主なカラム**:
    - `id`: トークンのユニークID。
    - `user_id`: `users`テーブルへの外部キー。
    - `hashed_token`: トークンのハッシュ値。
    - `created_at`: 作成日時。

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
- `account_types`: アカウント種別マスタ（PLAYER、EDITOR、ADMIN）。
- `versions`: バージョンマスタ。CHUNITHMの各バージョン（無印からX-VERSE-Xまで）の情報とリリース日を格納。

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
