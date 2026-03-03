# REF-G06 修正計画書: 貧血症モデル解消と集約振る舞いの集約

## 概要

REF-G06は5つの課題（DOM-004, DOM-005, DOM-019, INFRA-008, UC-002）を統合したリファクタリングです。
核心は「エンティティに振る舞いがなく、ドメインロジックがUsecase/Infraに流出している」問題の解消です。

### 対象課題

| ID | 概要 |
|---|---|
| DOM-004 | `Song` エンティティが貧血症モデル（メソッドなし、コンストラクタなし） |
| DOM-005 | `Session` エンティティにメソッドなし（有効期限判定がドメイン外に流出） |
| DOM-019 | `Player.Users *User` フィールドがDDD集約境界を侵害（デッドコード） |
| INFRA-008 | `song_repository_impl.go` の `DeleteSong`/`RestoreSong` で `RowsAffected` 未確認 |
| UC-002 | `DeleteUser` の重複定義（`UserCredentialUsecase` と `UserUsecase` で同名異義） |

### 設計判断

| 項目 | 決定事項 |
|---|---|
| Song Save | 新規 `Save(song)` メソッドを `SongRepository` に追加。`DeleteSong`/`RestoreSong` は廃止 |
| WorldsEnd | Song と同時に統一し `SaveSong(song)` パターンに移行 |
| Session.IsExpired | `IsExpired(now time.Time) bool` として引数で現在時刻を受け取る方式（テスタビリティ優先） |
| Player.Users | デッドコードのため単純削除 |
| UC-002 | `UserCredentialUsecase.DeleteUser` → `DeleteOwnAccount` にリネーム + 削除済みチェック追加 |

---

## Phase 1: Song エンティティの Rich Model 化（DOM-004）

### 1.1 Song エンティティにメソッド追加

**対象ファイル**: `internal/domain/entity/song.go`

以下のメソッドを追加する（`User` エンティティの `Delete()`/`Restore()`/`IsActive()` パターンを模範とする）:

- `Delete()` — `IsDeleted = true` に設定
- `Restore()` — `IsDeleted = false` に設定
- `IsActive() bool` — `!IsDeleted` を返す

**注意**: Song には `UpdatedAt` フィールドがないため、`User` と異なり `UpdatedAt` 更新は不要。

### 1.2 Song エンティティのユニットテスト作成

**新規ファイル**: `internal/domain/entity/song_test.go`

テーブルテスト + Given-When-Then パターンで以下をテスト:
- `Delete()` で `IsDeleted` が `true` になること
- `Restore()` で `IsDeleted` が `false` になること
- `IsActive()` が `IsDeleted` の逆を返すこと

### 1.3 Usecase層の直接フィールド参照をメソッド呼び出しに置換

| ファイル | 変更箇所 | Before | After |
|---|---|---|---|
| `song_usecase_impl.go` | L63 | `if song.IsDeleted {` | `if !song.IsActive() {` |
| `chart_stats_usecase.go` | L72, L156 | `if song.IsDeleted {` | `if !song.IsActive() {` |
| `worldsend_usecase.go` | L81 | `if songWithChart.Song.IsDeleted {` | `if !songWithChart.Song.IsActive() {` |

---

## Phase 2: SongRepository の Save パターン導入（INFRA-008）

### 2.1 SongRepository インターフェース変更

**対象ファイル**: `internal/domain/repository/song_repository.go`

- `Save(ctx context.Context, exec Executor, song *entity.Song) error` を追加
- `DeleteSong(ctx, exec, displayID)` と `RestoreSong(ctx, exec, displayID)` を削除

### 2.2 SongRepository 実装に Save メソッド追加

**対象ファイル**: `internal/infra/repository/song_repository_impl.go`

- `Save` メソッドで `is_deleted` を含むUPDATE文を発行（`display_id` で特定）
- `RowsAffected` をチェックし、0件なら `repository.ErrSongNotFound` を返す
- 旧 `DeleteSong`/`RestoreSong` メソッドを削除

### 2.3 Usecase層を「取得 → エンティティ操作 → Save」パターンに移行

**対象ファイル**: `internal/usecase/song_usecase_impl.go`

```go
// Before
func (s *songUsecaseImpl) DeleteSong(ctx context.Context, displayID string) error {
    return s.tm.Transactional(ctx, func(tx repository.Executor) error {
        _, err := s.songRepo.FindByDisplayID(ctx, tx, displayID)
        if err != nil { return err }
        return s.songRepo.DeleteSong(ctx, tx, displayID)
    })
}

// After
func (s *songUsecaseImpl) DeleteSong(ctx context.Context, displayID string) error {
    return s.tm.Transactional(ctx, func(tx repository.Executor) error {
        song, err := s.songRepo.FindByDisplayID(ctx, tx, displayID)
        if err != nil { return err }
        song.Delete()
        return s.songRepo.Save(ctx, tx, song)
    })
}
```

### 2.4 テストモック・テストの更新

- `internal/usecase/song_usecase_impl_test.go` のモックから `DeleteSong`/`RestoreSong` を削除し `Save` を追加
- `internal/testutil/song_usecase_mock.go` はUsecaseインターフェースのモックなので影響なし（Repository側の変更のみ）
- Song Delete/Restore のユースケーステストを新規追加

---

## Phase 3: WorldsEnd 側の統一

### 3.1 WorldsendChartRepository インターフェース変更

**対象ファイル**: `internal/domain/repository/worldsend_chart_repository.go`

- `SaveSong(ctx context.Context, exec Executor, song *entity.Song) error` を追加
- `DeleteSong`/`RestoreSong` を削除

### 3.2 WorldsendChartRepository 実装に SaveSong 追加

**対象ファイル**: `internal/infra/repository/worldsend_chart_repository_impl.go`

- `UPDATE songs SET is_deleted = ? WHERE display_id = ? AND is_worldsend = 1` で更新
- `RowsAffected == 0` で `ErrSongNotFound`（既存パターンを維持）
- 旧 `DeleteSong`/`RestoreSong` を削除

### 3.3 WorldsEnd Usecase の変更

**対象ファイル**: `internal/usecase/worldsend_usecase.go`

「取得 → エンティティ操作 → SaveSong」パターンに移行。

### 3.4 テストモック・テストの更新

- WorldsEnd Delete/Restore のユースケーステストを追加
- モックの `DeleteSong`/`RestoreSong` を `SaveSong` に置換

---

## Phase 4: Session エンティティの振る舞い追加（DOM-005）

### 4.1 Session に IsExpired メソッド追加

**対象ファイル**: `internal/domain/entity/session.go`

```go
// IsExpired は指定された時刻においてセッションが有効期限切れかどうかを判定する。
// 引数で現在時刻を受け取ることでテスタビリティを確保する。
func (s *Session) IsExpired(now time.Time) bool {
    return s.ExpiresAt.Before(now)
}
```

### 4.2 Session のユニットテスト作成

**新規ファイル**: `internal/domain/entity/session_test.go`

### 4.3 auth_usecase_impl.go の直接比較を置換

```go
// Before
if session.ExpiresAt.Before(time.Now()) {

// After
if session.IsExpired(time.Now()) {
```

---

## Phase 5: Player.Users デッドフィールドの削除（DOM-019）

### 5.1 フィールド削除

**対象ファイル**: `internal/domain/entity/player.go`

`Users *User` フィールドとコメントを削除。既に `UserID int` でID参照しており、コードベース内で一切使用されていないことを確認済み。

---

## Phase 6: UserCredentialUsecase.DeleteUser のリネーム（UC-002）

### 6.1 インターフェース・実装のリネーム

**対象ファイル**: `internal/usecase/user_credential_usecase.go`

- `DeleteUser(ctx, userID)` → `DeleteOwnAccount(ctx, userID)` にリネーム
- 実装に `user.IsDeleted` チェックを追加（削除済みなら `ErrUserAlreadyDeleted` を返す）

### 6.2 legacyAuthService の更新

**対象ファイル**: `internal/usecase/auth_service_compat.go`

- 委譲メソッドを `DeleteOwnAccount` に合わせて更新

### 6.3 ハンドラー側の呼び出し更新

**対象ファイル**: `internal/app/handler/api_internal/profile_handler.go`

- `h.userCredentialUsecase.DeleteUser` → `h.userCredentialUsecase.DeleteOwnAccount`

### 6.4 テストの更新

**対象ファイル**: `internal/usecase/user_security_usecase_test.go`

- メソッド名をリネーム
- 削除済みユーザーに対する呼び出しのエラーテストを追加

---

## Phase 7: 全体検証

1. `go test ./...` 全テスト実行
2. `gofmt` 全ファイルフォーマット
3. AGENTS.md に基づくセルフレビュー × 3回（各回 `go test` + `gofmt` 実行）
4. `_report/refactor.md` から解決済みの課題を削除

---

## 検証コマンド

```bash
# 新規エンティティテスト
go test ./internal/domain/entity/...

# Usecase テスト
go test ./internal/usecase/...

# Repository テスト
go test ./internal/infra/repository/...

# 全テスト
go test ./...

# フォーマット確認
gofmt -l ./internal/

# Song の直接フィールド参照が残っていないことを確認
grep -r "\.IsDeleted" internal/usecase/

# 旧メソッドがインターフェースから消えていることを確認
grep -r "DeleteSong\|RestoreSong" internal/domain/repository/
```
