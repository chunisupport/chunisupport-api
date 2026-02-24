# Goal attributes 内部短縮キー設計書

## 1. 背景

現行の `goals.attributes` は JSON カラムで保持されており、API 仕様上のキーは `diff` / `const` / `genre` / `ver` で固定されている。
一方で、保存件数の増加に伴い JSON キー名の重複コストを抑えたいという要求がある。

本設計では、**API 契約は維持したまま、DB 内部表現のみ短縮キーへ変換**する方針を定義する。

- 外部（API）: 既存の可読キーを維持
- 内部（DB）: 短縮キーで保存
- 既存データ: 旧フォーマット（可読キー）と新フォーマット（短縮キー）の両方を読める

## 2. 目的

- API 互換性を壊さずに `attributes` の保存サイズを削減する。
- 既存データを無停止で移行可能な設計にする。
- クリーンアーキテクチャの責務分離を維持し、変換ロジックを Usecase 層に閉じ込める。

## 3. 非目的

- `achievement_params` の短縮化（今回は対象外）
- API リクエスト/レスポンスのキー名変更
- DB スキーマ変更（カラム追加・型変更）は必須ではない

## 4. 用語

- **可読キー形式（Legacy/Public 形式）**: `diff` / `const` / `genre` / `ver`
- **短縮キー形式（Compact 形式）**: `d` / `c` / `g` / `v`
- **正規化形式（Canonical 形式）**: Usecase 内部で検証済みの可読キー形式

## 5. データ形式

### 5.1 可読キー形式（API 境界）

```json
{
  "diff": 4,
  "const": { "min": 14.0, "max": 14.4 },
  "genre": 1,
  "ver": 20
}
```

### 5.2 短縮キー形式（DB 保存）

```json
{
  "d": 4,
  "c": { "n": 14.0, "x": 14.4 },
  "g": 1,
  "v": 20
}
```

### 5.3 キーマップ

| 意味 | 可読キー | 短縮キー |
|---|---|---|
| 難易度 | `diff` | `d` |
| 定数条件 | `const` | `c` |
| 定数最小 | `min` | `n` |
| 定数最大 | `max` | `x` |
| ジャンル | `genre` | `g` |
| バージョン | `ver` | `v` |

## 6. 互換方針（最重要）

### 6.1 読み込み互換（Backward Compatibility）

`attributes` 読み込み時は以下の優先順で解釈する。

1. 可読キー形式として decode
2. 失敗またはキー不一致時は短縮キー形式として decode
3. どちらも不正なら `ErrInvalidGoalAttributes`

最終的には **正規化形式（可読キー）へ変換**して以降の業務ロジックへ渡す。

### 6.2 書き込み方針（Forward Compatibility）

- 新規作成/更新は短縮キー形式で保存する。
- API レスポンスは常に可読キー形式で返却する。

## 7. レイヤ責務

### 7.1 Handler / DTO

- 現状維持。可読キー形式のみ受け付ける。
- 短縮キーは外部公開しない。

### 7.2 Usecase

- `validateAttributes` の入口で **dual decode**（可読/短縮）を実装。
- 検証処理は既存ロジックを流用し、正規化形式を生成。
- 保存直前に正規化形式から短縮キー形式へ encode する。

### 7.3 Repository / Infra

- バイト列の永続化のみ担当（変換ロジックを持たない）。
- 既存の `[]byte` 入出力契約を維持可能。

## 8. 変更案（最小差分）

1. `internal/usecase/goal_usecase_impl.go` に以下の純関数を追加。
   - `decodeAttributesDual(raw []byte) (map[string]json.RawMessage, error)`
   - `toCompactAttributes(canonical map[string]json.RawMessage) ([]byte, error)`
   - `toCanonicalAttributesFromCompact(compact map[string]json.RawMessage) (map[string]json.RawMessage, error)`
2. 既存 `validateAttributes` の冒頭 decode を `decodeAttributesDual` に置換。
3. `validateAttributes` の戻り値 `canon` は「可読キー canonical」を返し、
   `Create/Update` 内で保存時に `toCompactAttributes` を適用。
4. `toOutputs` で `g.Attributes` を返す際、短縮キーが混在していても可読キーへ復元。

## 9. マイグレーション戦略

### Phase 0: コード先行デプロイ

- dual decode + compact write を先に導入。
- これにより旧データ（可読キー）の読取を壊さず、新規データは短縮化される。

### Phase 1: バックフィル（任意）

- バッチで既存レコードを順次再保存し、短縮形式へ統一。
- 途中で停止しても、dual decode により運用継続可能。

### Phase 2: 監視

- 失敗件数（attributes decode error）をメトリクス化して監視。
- 問題なければ運用継続。

## 10. テスト戦略

### 10.1 Unit Test（Usecase）

最低限追加すべきケース:

1. 可読キー入力を受け付け、保存は短縮キーである。
2. DB から短縮キーを読んだとき、レスポンスは可読キーになる。
3. DB から可読キー（旧データ）を読んでも正常に動作する。
4. 可読/短縮キー混在や未知キーは不正として落とす。
5. `const.min/max` のスケール・範囲検証が既存通りに効く。

### 10.2 回帰対象

- Goal 作成/更新/一覧 API の既存テスト
- 動的上限チェック（target stats 使用箇所）

## 11. 障害時ロールバック

- compact write を Feature Flag 化しておけば即時切り戻し可能。
- dual decode は残したままで問題ないため、安全なロールバックが可能。

## 12. 期待効果

- JSON キー名の重複を削減し、`attributes` カラムの平均保存サイズを低減。
- API 契約は維持されるため、フロントエンドや外部連携影響を最小化。
- 既存データを破棄せず段階移行できる。

## 13. リスクと対策

- リスク: 変換ロジック分岐による実装ミス
  - 対策: 変換関数を純関数化し、双方向テストを追加
- リスク: SQL 直接参照時に可読性が下がる
  - 対策: 運用向けに短縮キー対応表を共有
- リスク: 将来のキー追加時に互換性崩壊
  - 対策: 変換関数を 1 箇所に集中させ、追加時は table-driven test を更新

## 14. 今後の拡張

- `achievement_params` への同様の圧縮適用（必要時）
- attributes の完全 RDB 正規化（複数選択要件が強くなった場合）

