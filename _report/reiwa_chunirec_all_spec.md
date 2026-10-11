# reiwa chunirec_all 互換API仕様案

この文書は実装方針をまとめた作業資料です。正式な仕様は `docs/API.md` を参照してください。

## 提供先

- 動的API: `GET /compat/reiwa/1/chunirec_all.json`
- 静的JSON: `https://static.chunisupport.net/compat/reiwa/1/chunirec_all.json`
- 開発・ステージングの静的JSON: `https://static.chunisup-dev.f5.si/compat/reiwa/1/chunirec_all.json`
- 動的APIは既存のreiwa互換APIと同じAPIトークン認証、レート制限、メンテナンス制御、互換エラー形式を使用します。
- 静的JSONは認証不要です。公開用CORSとキャッシュ設定は既存の静的JSONに合わせます。
- クエリパラメータはありません。

## 収録対象と識別子

通常楽曲とWORLD'S END楽曲を返します。削除済み楽曲は除外します。
データの取得元はchunisupportの楽曲・譜面マスタです。

`meta.id`にはchunisupportの楽曲識別子を使用します。chunirecの楽曲IDとの一致は保証しません。
同名曲や通常楽曲とWORLD'S END楽曲は統合せず、別々の楽曲要素として返します。
WORLD'S ENDはchunisupportの1楽曲1譜面の単位で返します。

## レスポンス

HTTP 200で楽曲単位のトップレベル配列を返します。
各要素は `meta` と `data` を持ちます。
JSONの項目名、型、難易度の省略規則をreiwaの `chunirec_all.json` に合わせます。
以下に記載する項目をすべて提供し、chunisupport固有の項目は追加しません。

### meta

| 項目 | 型 | 通常楽曲 | WORLD'S END |
| --- | --- | --- | --- |
| `id` | string | 楽曲識別子 | 楽曲識別子 |
| `title` | string | 保存された曲名 | 保存された曲名の末尾に `【属性】` を付加 |
| `genre` | string | マスタのジャンル名。`POPS & ANIME` は `POPS&ANIME` に変換 | `WORLD'S END` |
| `artist` | string | アーティスト名 | アーティスト名 |
| `release` | string | リリース日を `YYYY-MM-DD` で表現 | 同左 |
| `bpm` | number | BPM | BPM |
| `img` | string | 拡張子なしのジャケット画像識別子 | 項目を省略 |
| `idx` | string | 公式インデックス。数値化せず文字列で返す | 項目を省略 |

未設定の文字列は空文字、未設定の数値は0を返します。`null`は返しません。
WORLD'S ENDの属性は譜面に保存された値を使用し、星数はタイトルに含めません。
属性が未設定または空文字の場合は、属性の括弧を付けず曲名のみを返します。

### data

通常楽曲は存在する難易度の `BAS`、`ADV`、`EXP`、`MAS`、`ULT` を返します。
存在しない難易度は項目ごと省略します。空オブジェクトや `null` は返しません。
WORLD'S ENDは `WE` のみを返します。

| 各難易度の項目 | 型 | 通常譜面 | WORLD'S END |
| --- | --- | --- | --- |
| `level` | number | 既存互換APIと同じ表記レベル。定数の小数部が0.5未満なら整数、0.5以上なら整数部+0.5 | 0 |
| `const` | number | 保存された譜面定数 | 0 |
| `maxcombo` | number | ノーツ数。未設定なら0 | ノーツ数。未設定なら0 |
| `is_const_unknown` | number | 定数が推定値なら1、それ以外は0 | 常に0 |

`is_const_unknown`はbooleanではなく整数の0または1です。
WORLD'S ENDの `level` は星数ではありません。

### 例

```json
[
  {
    "meta": {
      "id": "0000000000000001",
      "title": "通常楽曲",
      "genre": "ORIGINAL",
      "artist": "アーティスト",
      "release": "2026-10-01",
      "bpm": 180,
      "img": "jacket_identifier",
      "idx": "100"
    },
    "data": {
      "MAS": {
        "level": 14.5,
        "const": 14.7,
        "maxcombo": 1000,
        "is_const_unknown": 0
      }
    }
  },
  {
    "meta": {
      "id": "0000000000000002",
      "title": "WORLD'S END楽曲【狂】",
      "genre": "WORLD'S END",
      "artist": "アーティスト",
      "release": "2026-10-01",
      "bpm": 180
    },
    "data": {
      "WE": {
        "level": 0,
        "const": 0,
        "maxcombo": 1500,
        "is_const_unknown": 0
      }
    }
  }
]
```

## 並び順

通常楽曲を先に、WORLD'S END楽曲を後に返します。
通常楽曲は公式インデックスを数値として昇順に並べ、同値の場合は楽曲識別子の昇順に並べます。
公式インデックスが空または数値として解釈できない通常楽曲は、有効なインデックスを持つ通常楽曲の後に楽曲識別子の昇順で並べます。
WORLD'S END楽曲は楽曲識別子の昇順で並べます。
オブジェクト内のキーの順序は契約に含めません。

## 静的出力

既存の楽曲・譜面・バージョン一覧エクスポートにこのJSONを追加します。
オブジェクトキーは `compat/reiwa/1/chunirec_all.json` です。
動的APIと静的JSONは同じDTO変換を使用します。
通常楽曲またはWORLD'S END楽曲が0件の場合は、異常なスナップショットでの上書きを防ぐためバッチを失敗させます。
アップロードと成功後のキャッシュパージは既存のエクスポート方式に合わせます。

## 実装時の検証対象

- 通常楽曲の全項目、各型、存在しない難易度の省略。
- WORLD'S ENDの属性付きタイトル、星数を含まないこと、`WE`の固定値、`img`・`idx`の省略。
- 未設定値の空文字・0、および属性未設定時のタイトル。
- 削除済み楽曲の除外と楽曲識別子の一貫性。
- 通常楽曲とWORLD'S ENDの同名曲を別要素として維持すること。
- 並び順と無効な公式インデックスの扱い。
- 動的APIのトークン認証、メンテナンス制御、原因エラーを保持した切断処理。
- 静的JSONの保存先、内容、空データ時の上書き防止、キャッシュパージ対象。

正式な仕様は `docs/API.md`、`docs/static_data_export.md`、公開ガイドに記載します。
