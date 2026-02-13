# 未プレイ補完 機能設計書

## 1. 背景と目的

現行のユーザーレコード取得API（`GET /internal/users/:username` / `GET /v1/users/:username`）は、プレイ済み譜面のみを返却します。
一方でフロントエンドでは、一覧表示・フィルタ・ソート・集計のために「未プレイ譜面を含む全譜面集合」を必要とするケースがあり、クライアント側でマスタデータとの突合実装が必要になっています。

本設計書では、APIが未プレイ譜面を補完して返却する仕様を、後方互換性を維持しながら導入する方針を定義します。

---

## 2. 設計方針

### 2.1 後方互換性

- デフォルト挙動は変更しない（従来どおり「プレイ済みのみ返却」）。
- クエリパラメータによる **opt-in** 方式で未プレイ補完を有効化する。

### 2.2 責務分離（Clean Architecture / DDD）

- **Domain**: 未プレイ補完そのもののHTTP知識を持たせない。
- **Usecase**: 「プレイ済みレコード集合」と「対象譜面集合」を合成して返却モデルを構築する。
- **Handler**: クエリパラメータを解釈し、Usecaseにフラグを渡す。
- **Infra**: マスタ譜面取得は既存Repository・MasterCacheを利用し、N+1を発生させない。

### 2.3 パフォーマンス

- 補完時は「譜面マスタの一括取得 + プレイ済みレコードのマップ化」で合成する。
- 曲ごと・譜面ごとの逐次問い合わせ（N+1）は禁止する。
- 必要に応じてレスポンス増加を抑えるため、`view=rating` と未プレイ補完は排他または補完対象外とする（詳細は後述）。

---

## 3. 対象APIとインターフェース

## 3.1 対象エンドポイント

- `GET /internal/users/:username`
- `GET /v1/users/:username`

## 3.2 追加クエリパラメータ

- `include_unplayed`（optional, bool, default: `false`）

### 解釈ルール

- `true` の場合のみ補完を有効化。
- それ以外（未指定含む）は `false` 扱い。

---

## 4. レスポンス仕様

## 4.1 基本方針

- `records.all` のみを補完対象とする。
- `records.best` / `best_candidate` / `new` / `new_candidate` は従来どおり「プレイ済み由来」のままとする。
- WORLD'S END は既存仕様に合わせ、`records.worldsend` は補完対象外とする。

## 4.2 レコード要素への追加フィールド

補完時にクライアントが「実プレイ済み」かどうかを判別できるよう、`records.all[*]` に以下フィールドを追加する。

- `is_played: boolean`
  - `true`: 実際にユーザー記録が存在する譜面
  - `false`: APIが補完した未プレイ譜面

> 既存クライアント互換性を最大化するため、`include_unplayed=false` 時も `is_played` を返す案を推奨（常に `true`）。
> ただしレスポンス差分最小化を優先する場合は「補完時のみ返す」案も選択可能。

## 4.3 未プレイ譜面の初期値

`is_played=false` の要素に対しては、以下の初期値を返却する。

- `score`: `0`
- `clear_lamp`: `"NONE"`
- `combo_lamp`: `"NONE"`
- `chain_lamp`: `"NONE"`
- `ajc_lamp`: `"NONE"`
- `rating`: `0`
- `overpower`: `0`
- `is_const_unknown`: 譜面マスタ値を採用
- `const` / `title` / `artist` / `difficulty` / `img`: 譜面・楽曲マスタ値を採用
- `updated_at`: `null`

> 難易度文字列は既存方針どおり **大文字**（`BASIC`/`ADVANCED`/`EXPERT`/`MASTER`/`ULTIMA`）で返却する。

---

## 5. `view=rating` との関係

`view=rating` は軽量レスポンス用途であり、`records.all` 自体を返却しません。
そのため、以下のいずれかで統一します。

- **推奨案A（シンプル）**: `view=rating` 指定時は `include_unplayed` を無視する。
- 代替案B（厳格）: `view=rating&include_unplayed=true` は `400 validation_failed`。

本設計では互換性を重視し、**推奨案A** を採用します。

---

## 6. 実装イメージ

## 6.1 Usecase入力

`GetUserProfileWithRecords` にオプション構造体を追加。

- `IncludeUnplayed bool`
- `View string`（既存）

## 6.2 合成アルゴリズム

1. プレイ済みレコード一覧を取得
2. `include_unplayed=false` なら従来レスポンスを返却
3. 対象譜面マスタ一覧（WORLD'S END除外）を一括取得
4. プレイ済みをキー（`song_id + difficulty`）でマップ化
5. マスタ譜面を走査し、存在すれば既存レコード（`is_played=true`）、なければ補完レコード（`is_played=false`）を生成
6. 既存の並び順ルールに従って整列し、`records.all` として返却

## 6.3 データソース

- 曲・譜面: 既存 master cache / repository
- ユーザーレコード: 既存 user record repository

---

## 7. テスト戦略（TDD）

## 7.1 ユースケーステスト

- `include_unplayed=false`: 従来件数・内容と一致
- `include_unplayed=true`: 未プレイ譜面が追加される
- 補完レコードに `is_played=false` と初期値が設定される
- プレイ済みレコードは `is_played=true` で既存値が保持される
- 難易度が大文字で返る
- `view=rating` 時に `include_unplayed` が無視される

## 7.2 ハンドラテスト

- クエリパラメータ解釈（`true` / 未指定 / 不正値）
- Usecaseへ正しいオプションが渡る

## 7.3 回帰テスト

- 既存の `GET /internal/users/:username` と `GET /v1/users/:username` のレスポンス互換性
- 既存ランキング・レーティング計算への影響なし

---

## 8. APIドキュメント更新方針

実装時に `docs/API.md` の以下を更新する。

- `GET /internal/users/:username` のクエリパラメータに `include_unplayed` を追加
- `GET /v1/users/:username` 側の同等説明を追加
- `records.all[*].is_played` フィールド説明を追加
- `view=rating` 時の `include_unplayed` 扱いを明記

---

## 9. リリース計画

1. 設計確定（本書）
2. Usecaseテスト追加（Red）
3. 最小実装（Green）
4. リファクタリング（Refactor）
5. Handler/API.md 更新
6. ステージングでレスポンスサイズ・レイテンシ確認
7. 本番リリース

---

## 10. 意思決定メモ（推奨）

- 採用: `include_unplayed` の opt-in 方式
- 補完範囲: `records.all` のみ
- 判別子: `is_played` を追加
- `view=rating` との併用: `include_unplayed` 無視

上記により、既存互換性を維持しつつ、フロントエンド実装コストを削減できます。
