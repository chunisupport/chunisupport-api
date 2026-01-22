# データベースマイグレーションとスキーマ

## マイグレーションツール

このプロジェクトでは、データベースのスキーマ管理とマイグレーションのために [**golang-migrate**](https://github.com/golang-migrate/migrate) を使用しています。インストールにはバイナリのダウンロードではなく、以下のコマンドを利用してください。

```plaintext
go install -tags 'mysql sqlite' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
``` 

マイグレーションファイルは `migration/mysql` ディレクトリに格納されており、`*.up.sql` ファイルがスキーマの追加・変更、`*.down.sql` ファイルが変更のロールバックを定義します。

## 主要テーブルの概要

以下は、アプリケーションのコア機能に関連する主要なテーブルの概要です。

### `users`
- **役割**: このシステムのユーザーアカウント情報を格納します。
- **主なカラム**:
    - `id`: ユーザーのユニークID。
    - `username`: ログインに使用するユーザー名。
    - `password_hash`: ハッシュ化されたパスワード。
    - `player_id`: `players`テーブルへの外部キー。ユーザーに紐づくCHUNITHMのプレイヤー情報。

### `sessions`
- **役割**: ユーザーのログインセッションを管理します。JWTと組み合わせた認証方式のバックエンドとして機能します。
- **主なカラム**:
    - `id`: セッションのユニークID（UUID）。
    - `user_id`: `users`テーブルへの外部キー。`ON DELETE CASCADE`が設定されており、ユーザーが削除されるとセッションも自動的に削除されます。
    - `expires_at`: セッションの有効期限。

### `players`
- **役割**: CHUNITHMのプレイヤーとしてのプロフィール情報を格納します。
- **主なカラム**:
    - `id`: プレイヤーのユニークID。
    - `user_id`: `users`テーブルへの外部キー。
    - `player_name`: プレイヤー名。
    - `player_level`, `player_rating`: ゲーム内でのレベル、レーティング。
    - `class_emblem_id`, `class_emblem_base_id`: クラスエンブレム情報。
    - `last_played_at`: 最終プレイ日時。
    - `overpower_value`, `overpower_percentage`: オーバーパワー関連の値。
    - `team_name`, `team_color`: チーム情報。

### `songs`
- **役割**: 楽曲の基本情報を格納します。
- **主なカラム**:
    - `id`: 楽曲のユニークID。
    - `title`, `artist`: 楽曲のタイトルとアーティスト名。
    - `genre_id`: `genres`マスタへの外部キー。
    - `official_idx`: 公式サイトの楽曲JSONにおけるユニークID。バッチ処理での突き合わせに使用されます。

### `charts`
- **役割**: 各楽曲の譜面情報を格納します。一つの楽曲に対して複数の難易度（BASIC, EXPERTなど）の譜面が存在します。
- **主なカラム**:
    - `id`: 譜面のユニークID。
    - `song_id`: `songs`テーブルへの外部キー。
    - `difficulty_id`: `difficulties`マスタへの外部キー。
    - `const`: 譜面定数。レーティング計算の基礎となります。

### `player_records`
- **役割**: プレイヤーのスコア記録を格納します。
- **主なカラム**:
    - `player_id`, `chart_id`: どのプレイヤーがどの譜面をプレイしたかを表す主キー。
    - `score`: スコア。
    - `clear_lamp_id`, `combo_lamp_id`, `full_chain_id`: クリア状況やコンボ状況を示すマスタへの外部キー。
    - `slot_id`, `slot_order`: 種別とスロット内の順序を示すカラム。`slot_id`は`slots`マスタを参照します。

### マスタテーブル
- `genres`, `difficulties`, `clear_lamp_types`など、接尾辞に `_types` や `_emblems` がつくテーブルは、定型的なデータを管理するためのマスタテーブルです。
- `slots`: プレイヤー称号のスロット種別を表すマスタです。初期値として `none`, `best`, `best_candidate`, `new`, `new_candidate` を保持します。

---

## 外部データソース用の一時テーブル

楽曲データの構築は、現在このリポジトリとは別のバッチ処理用リポジトリで行われています。しかし、そのバッチ処理で使用される一時テーブルのスキーマ定義（マイグレーションファイル）は、引き続きこのリポジトリで管理しています。

これらのテーブルは、外部データソース（公式サイト、Chunirecなど）から取得した情報を一時的に保存するためのもので、`official_source_songs`, `chunirec_source_songs`, `natua_source_charts` のように、`{source_name}_source_{entity}`という命名規則になっています。

これらのテーブルは、バッチ処理リポジトリ側でデータが投入され、正規化・統合処理を経て、最終的にこのAPIサーバーが使用する主要テーブル群（`songs`, `charts`など）にデータが格納される、という流れになっています。
