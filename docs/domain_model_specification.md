# ドメインモデル仕様書

このドキュメントでは、chunisupport-api のドメインモデル（エンティティ、値オブジェクト）の仕様を定義します。

## 目次

- [基本ルール](#基本ルール)
- [エンティティ一覧](#エンティティ一覧)
- [値オブジェクト一覧](#値オブジェクト一覧)
- [ドメインサービス](#ドメインサービス)

---

## 基本ルール

### 難易度の文字列表記

**難易度(Difficulty)の文字列は常に大文字で扱います。**

- **該当する値**: `BASIC`, `ADVANCED`, `EXPERT`, `MASTER`, `ULTIMA`
- **適用範囲**: 
  - データベースのマスターデータ（`difficulties`テーブル）
  - マスターデータのキャッシュキー
  - API入出力における難易度の表現
  - コード内での難易度の比較・検索
- **理由**: システム全体で難易度を大文字で統一することで、キャッシュキーの不一致によるリソース検索エラーを防止します。

**注意**: 短縮形（`BAS`, `ADV`, `EXP`, `MAS`, `ULT`）は入力時に受け付けますが、内部処理では必ず正式名称の大文字表記に変換してください。

---

## エンティティ一覧

### User（ユーザー集約ルート）

#### 概要
システム利用者を表す集約ルート。認証情報、プレイヤー紐付け、プライバシー設定を管理します。

#### フィールド

| フィールド名 | 型 | 必須 | 説明 |
|------------|-----|-----|------|
| ID | int | ✓ | ユーザーID（主キー） |
| Username | username.UserName | ✓ | ユーザー名（値オブジェクト） |
| PasswordHash | passwordhash.PasswordHash | ✓ | パスワードハッシュ（値オブジェクト） |
| CreatedAt | time.Time | ✓ | 作成日時 |
| UpdatedAt | time.Time | ✓ | 更新日時 |
| PlayerID | *int | - | 紐付けられたプレイヤーID |
| AccountTypeID | int | ✓ | アカウント種別ID |
| IsDeleted | bool | ✓ | 論理削除フラグ |
| IsPrivate | bool | ✓ | 非公開設定フラグ |

#### 振る舞い（メソッド）

##### クエリメソッド

- **IsActive() bool**
  - ユーザーが有効（削除されていない）かを判定
  - 返り値: `!IsDeleted`

- **IsPublic() bool**
  - ユーザーが公開設定かを判定
  - 返り値: `!IsPrivate`

- **HasLinkedPlayer() bool**
  - ユーザーにプレイヤーが紐づいているかを判定
  - 返り値: `PlayerID != nil`

##### コマンドメソッド

- **ChangePrivacy(isPrivate bool)**
  - ユーザーの公開/非公開設定を変更
  - 副作用: `UpdatedAt` を現在時刻に更新

- **ChangePassword(hash passwordhash.PasswordHash)**
  - ユーザーのパスワードハッシュを変更
  - 副作用: `UpdatedAt` を現在時刻に更新

- **Delete()**
  - ユーザーを論理削除
  - 副作用: `IsDeleted = true`, `UpdatedAt` を現在時刻に更新

- **LinkPlayer(playerID int)**
  - ユーザーにプレイヤーを紐付け
  - 副作用: `PlayerID` を設定、`UpdatedAt` を現在時刻に更新

#### 不変条件

- `Username` は5文字以上50文字以内の小文字英数字
- `PasswordHash` はbcryptハッシュ形式
- 削除済みユーザー（`IsDeleted = true`）は無効とみなされる

---

### Player（プレイヤーエンティティ）

#### 概要
CHUNITHM プレイヤーの情報を表すエンティティ。レーティング、レベル、オーバーパワーなどの統計情報を管理します。

#### フィールド

| フィールド名 | 型 | 必須 | 説明 |
|------------|-----|-----|------|
| ID | int | ✓ | プレイヤーID（主キー） |
| UserID | int | ✓ | 所属ユーザーID（外部キー） |
| Name | playername.PlayerName | ✓ | プレイヤー名（値オブジェクト） |
| Level | int | ✓ | プレイヤーレベル |
| OfficialRating | *float64 | - | 公式レーティング |
| CalculatedRating | *float64 | - | 計算レーティング |
| NewAverageRating | *float64 | - | 新曲枠平均レーティング |
| BestAverageRating | *float64 | - | ベスト枠平均レーティング |
| ClassEmblemID | *int | - | クラスエンブレムID |
| ClassEmblemBaseID | *int | - | クラスエンブレムベースID |
| LastPlayedAt | *time.Time | - | 最終プレイ日時 |
| OverpowerValue | *float64 | - | オーバーパワー値 |
| OverpowerPercent | *float64 | - | オーバーパワー割合（%） |
| CreatedAt | time.Time | ✓ | 作成日時 |
| UpdatedAt | time.Time | ✓ | 更新日時 |
| Users | *User | - | 紐づくユーザー（関連エンティティ） |

#### 振る舞い（メソッド）

現在、クエリメソッドのみ。将来的にレーティング再計算などのコマンドメソッドを追加予定。

#### 不変条件

- `Name` は1文字以上20文字以内
- `Level` は0以上
- `OfficialRating`, `CalculatedRating` は0.0以上
- `OverpowerPercent` は0.0～100.0の範囲

---

### PlayerRecord（プレイヤー記録エンティティ）

#### 概要
プレイヤーの譜面ごとの記録を表すエンティティ。スコア、ランプ、スロット情報を管理します。

#### フィールド

| フィールド名 | 型 | 必須 | 説明 |
|------------|-----|-----|------|
| PlayerID | int | ✓ | プレイヤーID（複合主キー） |
| ChartID | int | ✓ | 譜面ID（複合主キー） |
| Score | score.Score | ✓ | スコア（値オブジェクト） |
| ClearLampID | int | ✓ | クリアランプID |
| ComboLampID | int | ✓ | コンボランプID |
| FullChainID | int | ✓ | フルチェーンID |
| SlotID | int | ✓ | スロットID |
| SlotOrder | *int | - | スロット内順位 |
| UpdatedAt | time.Time | ✓ | 更新日時 |
| Chart | *Chart | - | 譜面（関連エンティティ） |
| Song | *Song | - | 楽曲（関連エンティティ） |
| ClearLamp | *ClearLampType | - | クリアランプ種別 |
| ComboLamp | *ComboLampType | - | コンボランプ種別 |
| FullChain | *FullChainType | - | フルチェーン種別 |
| Slot | *Slot | - | スロット種別 |
| ChartDifficulty | *ChartDifficulty | - | 譜面難易度 |

#### 振る舞い（メソッド）

##### クエリメソッド

- **IsRanked() bool**
  - このレコードがランキング対象（スロット指定あり）かを判定
  - 返り値: `Slot != nil && Slot.Name != "" && Slot.Name != "none"`

- **SlotKey() string**
  - レコードのスロット種別を示すキーを返す
  - 返り値: ランキング対象の場合は `Slot.Name`、それ以外は空文字列

#### 不変条件

- `Score` は0～1,010,000の範囲
- `PlayerID` + `ChartID` の組み合わせは一意
- `Slot.Name = "none"` のレコードはランキング対象外

---

### Song（楽曲エンティティ）

#### 概要
CHUNITHM の楽曲情報を表すエンティティ。

#### フィールド

| フィールド名 | 型 | 必須 | 説明 |
|------------|-----|-----|------|
| ID | int | ✓ | 楽曲ID（主キー） |
| DisplayID | string | ✓ | 表示用ID（例: "song001"） |
| Title | string | ✓ | 楽曲タイトル |
| Artist | string | ✓ | アーティスト名 |
| GenreID | *int | - | ジャンルID |
| BPM | *int | - | BPM（テンポ） |
| ReleaseDate | *time.Time | - | リリース日 |
| OfficialIdx | *string | - | 公式インデックス |
| Img | *string | - | ジャケット画像URL |
| IsWorldsend | bool | ✓ | WORLD'S END楽曲フラグ |
| IsDeleted | bool | ✓ | 削除フラグ |

#### 振る舞い（メソッド）

現在、振る舞いメソッドなし（データ保持のみ）。

#### 不変条件

- `DisplayID` は一意
- `Title`, `Artist` は非空
- `BPM` は正の整数

---

### Chart（譜面エンティティ）

#### 概要
楽曲の譜面情報を表すエンティティ。難易度、譜面定数、ノーツ数を管理します。

#### フィールド

| フィールド名 | 型 | 必須 | 説明 |
|------------|-----|-----|------|
| ID | int | ✓ | 譜面ID（主キー） |
| SongID | int | ✓ | 楽曲ID（外部キー） |
| DifficultyID | int | ✓ | 難易度ID（BASIC, ADVANCED, EXPERT, MASTER, ULTIMA, WORLD'S END） |
| Const | chartconstant.ChartConstant | ✓ | 譜面定数（値オブジェクト） |
| IsConstUnknown | bool | ✓ | 譜面定数が未確定かどうか |
| Notes | *notes.Notes | - | ノーツ数（値オブジェクト） |

#### 振る舞い（メソッド）

現在、振る舞いメソッドなし（データ保持のみ）。

#### 不変条件

- `Const` は0.0～15.9の範囲（通常譜面の場合）
- `Notes` は正の整数
- `SongID` + `DifficultyID` の組み合わせは一意

---

### その他のエンティティ

以下のエンティティは主にマスターデータとして機能します：

- **AccountType**: アカウント種別（一般、管理者など）
- **ChartDifficulty**: 譜面難易度（BASIC, ADVANCED, EXPERT, MASTER, ULTIMA, WORLD'S END）
- **ClearLampType**: クリアランプ種別（FAILED, CLEAR, HARD, AB, AJなど）
- **ComboLampType**: コンボランプ種別（NONE, FC, AJ）
- **FullChainType**: フルチェーン種別（NONE, FULL CHAIN, FULL CHAIN PLATINUM）
- **Genre**: 楽曲ジャンル
- **Slot**: レーティング枠種別（best, new, none）
- **ClassEmblem**: クラスエンブレム（称号）
- **ClassEmblemBase**: クラスエンブレムベース（称号の基礎デザイン）
- **APIToken**: API認証トークン
- **Session**: ユーザーセッション

---

### RatingBand（レーティング帯マスタ）

#### 概要
レーティング帯の範囲を定義するマスターデータエンティティ。統計データの集計軸として使用されます。

#### フィールド

| フィールド名 | 型 | 必須 | 説明 |
|------------|-----|-----|------|
| ID | int | ✓ | レーティング帯ID（主キー） |
| Label | string | ✓ | 表示ラベル（例: "15.0", "17.6+"） |
| MinInclusive | *float64 | - | 下限（含む）。nilの場合は下限なし |
| MaxExclusive | *float64 | - | 上限（含まない）。nilの場合は上限なし |
| SortOrder | int | ✓ | 表示順序 |

#### 不変条件

- `Label` は非空
- `MinInclusive` と `MaxExclusive` の両方がnilであってはならない
  - **例外**: `ID = 0` (ラベル "ALL") は全プレイヤー統計を表す特殊行であり、レーティング帯範囲を持たないため、両方nilを許容します
- レーティング帯は 0.1刻みで定義（15.00未満、15.00～17.50、17.60+）
- 判定式: `MinInclusive <= rating < MaxExclusive`

---

### ChartStatsByRatingBand（譜面×レーティング帯統計）

#### 概要
譜面ごとのレーティング帯別統計データを表すエンティティ。プレイヤーのランク分布、コンボランプ分布、クリアランプ分布、平均スコアを管理します。

#### フィールド

| フィールド名 | 型 | 必須 | 説明 |
|------------|-----|-----|------|
| ChartID | int | ✓ | 譜面ID（複合主キー） |
| RatingBandID | int | ✓ | レーティング帯ID（複合主キー） |
| Rank | ChartRankStats | ✓ | ランク別人数統計 |
| Combo | ChartComboStats | ✓ | コンボランプ別人数統計 |
| Clear | ChartClearStats | ✓ | クリアランプ別人数統計 |
| AverageScore | *float64 | - | レーティング帯別平均スコア（レコード数0件の場合はnil） |
| PlayerCount | int | ✓ | レーティング帯別プレイヤー数 |

#### 不変条件

- `ChartID` + `RatingBandID` の組み合わせは一意
- `AverageScore` は 0.0～1,010,000.0 の範囲（nilを除く）
- 人数カウント（Rank, Combo, Clear）は非負整数

#### 統計データの仕様

- **レーティング帯の判定基準**: プレイヤーの「ベスト枠平均レーティング」を小数点1桁で切り捨てた値
- **平均スコアの計算**: 各レーティング帯のプレイヤーレコードのスコア平均値（AVG）
- **NULL値の扱い**: 該当レーティング帯にレコードが0件の場合、`AverageScore` は `nil`
- **更新タイミング**: 統計データは定期バッチで更新され、過去データは保持しない

---

### ChartRankStats（ランク別人数統計）

#### 概要
譜面のランク別人数を表す値オブジェクト。

#### フィールド

| フィールド名 | 型 | 説明 |
|------------|-----|------|
| AAAL | int | AAA以下人数 |
| S | int | S人数 |
| SP | int | S+人数 |
| SS | int | SS人数 |
| SSP | int | SS+人数 |
| SSS | int | SSS人数 |
| SSSP | int | SSS+人数 |
| Max | int | 理論値（1,010,000点）人数 |

---

### ChartComboStats（コンボランプ別人数統計）

#### 概要
譜面のコンボランプ別人数を表す値オブジェクト。

#### フィールド

| フィールド名 | 型 | 説明 |
|------------|-----|------|
| None | int | コンボランプなし人数 |
| FC | int | FULL COMBO人数 |
| AJ | int | ALL JUSTICE人数 |

---

### ChartClearStats（クリアランプ別人数統計）

#### 概要
譜面のクリアランプ別人数を表す値オブジェクト。

#### フィールド

| フィールド名 | 型 | 説明 |
|------------|-----|------|
| Failed | int | FAILED人数 |
| Clear | int | CLEAR人数 |
| Hard | int | HARD人数 |
| Brave | int | BRAVE人数 |
| Absolute | int | ABSOLUTE人数 |
| Catastrophy | int | CATASTROPHY人数 |

---

## 値オブジェクト一覧

### username.UserName

#### 概要
ユーザー名を表す値オブジェクト。不変かつバリデーションを持ちます。

#### 制約

- 5文字以上50文字以内
- 小文字英数字のみ（`^[a-z0-9]+$`）
- 空文字列は不可

#### ファクトリメソッド

- `NewUserName(value string) (UserName, error)`: バリデーション付き生成
- `MustNewUserName(value string) UserName`: バリデーションなし生成（パニックあり）

#### メソッド

- `String() string`: 文字列値を取得
- `Value() (driver.Value, error)`: DB保存用（driver.Valuer実装）
- `Scan(src any) error`: DB読み込み用（sql.Scanner実装）
- `MarshalJSON() ([]byte, error)`: JSON出力用
- `UnmarshalJSON(data []byte) error`: JSON入力用

---

### playername.PlayerName

#### 概要
プレイヤー名を表す値オブジェクト。

#### 制約

- 1文字以上8文字以内（UTF-8のルーン数でカウント）
- 空文字列は不可
- 半角英数字（a-z, A-Z, 0-9）を含まない
- 半角カタカナ（U+FF61〜U+FF9F）を含まない

#### ファクトリメソッド

- `NewPlayerName(value string) (PlayerName, error)`: バリデーション付き生成
- `MustNewPlayerName(value string) PlayerName`: バリデーションなし生成

#### メソッド

- `String() string`
- `Value() (driver.Value, error)`
- `Scan(src any) error`
- `MarshalJSON() ([]byte, error)`
- `UnmarshalJSON(data []byte) error`

---

### score.Score

#### 概要
スコア値を表す値オブジェクト。CHUNITHMのスコア範囲を保証します。

#### 制約

- 型: `uint32`
- 範囲: 0～1,010,000
- 負の値は不可
- 1,010,000を超える値は不可

#### ファクトリメソッド

- `NewScore(value uint32) (Score, error)`: バリデーション付き生成

#### メソッド

- `Value() (driver.Value, error)`: int64に変換してDB保存
- `Scan(value any) error`: int64/[]byte/stringからパース

---

### passwordhash.PasswordHash

#### 概要
bcryptハッシュ化されたパスワードを表す値オブジェクト。

#### 制約

- bcrypt形式（`$2a$`で始まる60文字のハッシュ）
- 空文字列は不可

#### ファクトリメソッド

- `NewPasswordHash(hash string) (PasswordHash, error)`: バリデーション付き生成
- `HashFromPassword(password string) (PasswordHash, error)`: 平文パスワードからハッシュ生成

#### メソッド

- `String() string`
- `Value() (driver.Value, error)`
- `Scan(src any) error`
- `ComparePassword(password string) bool`: パスワード一致検証

---

### chartconstant.ChartConstant

#### 概要
譜面定数を表す値オブジェクト。

#### 制約

- 型: `float64`
- 範囲: 通常は0.0～15.9（WORLD'S ENDは例外）
- 0.1刻みで管理

#### ファクトリメソッド

- `NewChartConstant(value float64) (ChartConstant, error)`: バリデーション付き生成

#### メソッド

- `Float64() float64`: float64値を取得
- `Value() (driver.Value, error)`: 文字列に変換してDB保存
- `Scan(value any) error`: float64/[]byte/stringからパース

---

### notes.Notes

#### 概要
ノーツ数を表す値オブジェクト。

#### 制約

- 型: `int`
- 範囲: 正の整数（1以上）

#### ファクトリメソッド

- `NewNotes(value int) (Notes, error)`: バリデーション付き生成

#### メソッド

- `Int() int`: int値を取得
- `Value() (driver.Value, error)`
- `Scan(value any) error`

---

### displayid.DisplayID

#### 概要
楽曲の表示IDを表す値オブジェクト。

#### 制約

- 形式: 英数字とハイフン（例: "song-001", "worldsend-xyz"）
- 空文字列は不可

#### ファクトリメソッド

- `NewDisplayID(value string) (DisplayID, error)`: バリデーション付き生成

#### メソッド

- `String() string`
- `Value() (driver.Value, error)`
- `Scan(src any) error`

---

## ドメインサービス

### domain/rating パッケージ

#### 概要
CHUNITHMのレーティングおよびオーバーパワー計算ロジックを提供するドメインサービス。

#### 提供関数

##### CalcSingleRating

```go
func CalcSingleRating(score uint32, chartConst float64) float64
```

- **概要**: 指定されたスコアと譜面定数から単曲レーティングを計算
- **引数**:
  - `score`: プレイヤーのスコア（0～1,010,000）
  - `chartConst`: 譜面定数
- **返り値**: 単曲レーティング（0.0以上）

**計算式**:

| ランク | スコア範囲 | 計算式 |
|-------|-----------|--------|
| SSS+ | 1,009,000～ | 譜面定数 + 2.15 |
| SSS | 1,007,500～ | 譜面定数 + 2.0 + (score - 1,007,500) / 100 * 0.01 |
| SS+ | 1,005,000～ | 譜面定数 + 1.5 + (score - 1,005,000) / 50 * 0.01 |
| SS | 1,000,000～ | 譜面定数 + 1.0 + (score - 1,000,000) / 100 * 0.01 |
| S+ | 990,000～ | 譜面定数 + 0.6 + (score - 990,000) / 250 * 0.01 |
| S | 975,000～ | 譜面定数 + (score - 975,000) / 2500 * 0.1 |
| AAA | 950,000～ | 譜面定数 - 1.67 + (score - 950,000) / 150 * 0.01 |
| AA | 925,000～ | 譜面定数 - 3.34 + (score - 925,000) / 150 * 0.01 |
| A | 900,000～ | 譜面定数 - 5.0 + (score - 900,000) / 150 * 0.01 |
| BBB | 800,000～ | (譜面定数 - 5.0) / 2 + (score - 800,000) / (2000 / (譜面定数 - 5)) * 0.01 |
| C | 500,000～ | (score - 500,000) / (6000 / (譜面定数 - 5)) * 0.01 |
| D | ～500,000 | 0 |

##### CalcSingleOverpower

```go
func CalcSingleOverpower(score uint32, chartConst float64, comboLampID int) float64
```

- **概要**: 指定されたスコア、譜面定数、コンボランプから単曲オーバーパワーを計算
- **引数**:
  - `score`: プレイヤーのスコア（0～1,010,000）
  - `chartConst`: 譜面定数
  - `comboLampID`: コンボランプID（1=なし、2=FC、3=AJ）
- **返り値**: 単曲オーバーパワー（0.0以上）

**コンボランプ補正**:

- `comboLampID == 2` (FULL COMBO): +0.5
- `comboLampID == 3` (ALL JUSTICE): +1.0
- `score == 1,010,000` (理論値): +1.25

**計算式**:

| ランク | スコア範囲 | 計算式 |
|-------|-----------|--------|
| S以上 | 975,000～1,007,500 | レーティング値 × 5 + 補正1 |
| SSS以上 | 1,007,501～ | (譜面定数 + 2) × 5 + 補正1 + 補正2 |
| AJC | 1,010,000 | (譜面定数 + 3) × 5 |

補正2 = (スコア - 1,007,500) × 0.0015（最大3.75）

**精度**:
- S以上: 0.005単位（小数点以下3桁目を切り捨て）
- S未満: 0.05単位（小数点以下2桁目を切り捨て）

---

## アーキテクチャ上の注意事項

### エンティティの純粋性

- エンティティは `db` タグや `json` タグを持たない
- インフラストラクチャ層の関心事（DB永続化、JSON変換）はエンティティから分離
- `internal/infra/models` パッケージでデータモデルを定義し、`ToEntity()/FromEntity()` マッパーで変換

### 値オブジェクトの責務

- ドメインの制約をコンストラクタでバリデーション
- 不変性を保証（フィールドは非公開）
- `driver.Valuer`, `sql.Scanner` を実装してDB永続化をサポート
- `json.Marshaler`, `json.Unmarshaler` を実装してJSON変換をサポート

### 集約境界

- **User**: 集約ルート。プライバシー設定、パスワード変更、プレイヤー紐付けはUserが管理
- **Player**: 別の集約。UserとはIDで参照（直接の親子関係を持たない）
- **PlayerRecord**: Playerの子エンティティではなく、独立したエンティティとして扱う（Chart/Songとの多対多の関係を持つため）

### リポジトリパターン

- 集約ルートごとにリポジトリを定義
- `Save(ctx, entity)` メソッドで集約全体を永続化（INSERT/UPDATE判定は内部で実施）
- 部分更新メソッド（`UpdatePassword`, `UpdatePrivacy`など）は廃止し、集約指向の永続化を推進
