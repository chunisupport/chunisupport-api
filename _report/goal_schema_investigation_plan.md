# goalsテーブル `user_id` 型整合性 調査計画と結果

## 背景

`goals.user_id` の型について、既存スキーマとドメイン定義のどちらをソースオブトゥルースにすべきかを明確化する。
レビュー指摘に基づき、先に調査計画を定義し、その結果を記録する。

## 調査計画（Rule準拠）

### 1. ソースオブトゥルース候補の列挙

- DBスキーマ（`migration/schema_mysql.sql`）
- ドメイン/インフラの型定義（`internal/domain` / `internal/infra/models`）
- 既存の外部キー運用方針（ユーザー配下テーブル）

### 2. 確認項目

- `users.id` の実型
- `user_id` を持つ既存テーブル（`api_tokens`, `sessions`, `players`, `user_recovery_codes`）の実型
- Goコード上でのUserIDの実型
- 参照整合性（外部キー + ON DELETE CASCADE）運用の一貫性

### 3. 影響範囲評価

- `goals` DDL案
- Goの型定義例（`Goal.UserID`）
- 将来のRepository実装時のスキャン型/クエリ条件

### 4. 意思決定基準

- 既存DBスキーマと既存実装（Goコード）が一致している場合、それをソースオブトゥルースとする。
- 提案設計は既存方針に合わせる（新規領域のみ独自ルールを導入しない）。

## 調査結果

- `users.id` は `int unsigned`。
- 既存の `user_id` カラム（`api_tokens`, `sessions`, `players`, `user_recovery_codes`）は `int unsigned` で統一され、`users(id)` への外部キーを持つ。
- Go側もUserIDは `int` で扱われている（例: `internal/infra/models/user_model.go`）。

## 結論（採用方針）

ソースオブトゥルースは既存スキーマ + 既存Go型定義。
したがって `goals` の設計は以下を採用する。

- `user_id INT UNSIGNED NOT NULL`
- `FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE`
- Go例の `Goal.UserID` は `int`

## 反映状況

上記は `_report/goal_achievement_design.md` の完成仕様に反映済み。
