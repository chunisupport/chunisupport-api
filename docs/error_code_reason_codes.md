# エラーコード / 内部理由コード一覧

最終更新: 2026-04-03

## 更新ルール

- APIエラーコード（`internal/app/apierror/codes.go`）を追加・変更した場合は、このドキュメントを同時に更新してください。
- 内部理由コード（`reason`）を追加・変更した場合も、このドキュメントを同時に更新してください。
- 公開APIで返す値（`error.code`）と、ログ・運用調査で使う値（`reason`）を混同しないでください。

## APIエラーコード一覧

| コード | 用途 |
| --- | --- |
| `bad_request` | リクエスト形式不正 |
| `internal_error` | サーバー内部エラー |
| `unauthorized` | 認証失敗 |
| `invalid_credentials` | 認証情報不正 |
| `invalid_token` | トークン不正 |
| `token_expired` | トークン期限切れ |
| `missing_token` | トークン欠落 |
| `invalid_session` | セッション無効 |
| `invalid_recovery_credentials` | リカバリ認証情報不正 |
| `forbidden` | 権限不足 |
| `firebase_uid_already_linked` | Firebase UID が他ユーザーまたは削除済みユーザーに連携済み |
| `registration_failed` | ユーザー登録失敗 |
| `user_not_found` | ユーザー未検出 |
| `operation_failed` | 操作失敗 |
| `player_not_linked` | プレイヤー未連携 |
| `player_not_found` | プレイヤー未検出 |
| `song_not_found` | 楽曲未検出 |
| `chart_not_found` | 譜面未検出 |
| `invalid_genre_id` | ジャンルID不正 |
| `invalid_difficulty_id` | 難易度ID不正 |
| `invalid_difficulty` | 難易度指定不正 |
| `validation_failed` | バリデーション失敗 |
| `resource_not_found` | リソース未検出 |
| `conflict` | 競合 |
| `api_token_not_found` | APIトークン未検出 |
| `payload_too_large` | ペイロード過大 |
| `unsupported_media_type` | Content-Type不正 |
| `method_not_allowed` | HTTPメソッド不正 |
| `not_found` | エンドポイント未検出 |
| `too_many_requests` | レート制限 |
| `service_unavailable` | サービス利用不可 |
| `username_empty` | ユーザー名空 |
| `username_too_short` | ユーザー名が短すぎる |
| `username_too_long` | ユーザー名が長すぎる |
| `username_invalid_char` | ユーザー名の文字種不正 |
| `password_too_short` | パスワードが短すぎる |
| `password_too_long` | パスワードが長すぎる |
| `invalid_password` | パスワード不正 |
| `app_version_unsupported` | アプリバージョン非対応 |
| `goal_not_found` | goal未検出 |
| `goal_limit_exceeded` | goal上限超過 |
| `goal_invalid_title` | goalタイトル不正 |
| `goal_invalid_achievement_type` | goal達成種別不正 |
| `goal_invalid_achievement_params` | goal達成条件不正 |
| `goal_invalid_attributes` | goal属性不正 |
| `invalid_goal_input` | goal入力不正 |

## 内部理由コード一覧（運用・調査用）

### Goal バリデーションログ（`slog`）

| reason | 発生条件 |
| --- | --- |
| `count_over_dynamic_max` | `count` が属性絞り込み後の譜面数上限を超過 |
| `total_score_over_dynamic_max` | `total_score.total` が理論上限超過 |
| `overpower_value_over_dynamic_max` | `overpower_value.total` が理論上限超過 |

### PlayerData `skipped_records[].reason`

| reason | 発生条件 |
| --- | --- |
| `invalid slot {slot}` | 称号スロットキー不正 |
| `slot out of range: {slot}` | 称号スロット範囲外 |
| `honor_type not found: {type}` | 称号タイプ未解決 |
| `failed to create honor` | 称号エンティティ生成失敗 |
| `failed to insert player_honor (bulk)` | 称号一括保存失敗 |
| `failed to resolve chart` | 通常譜面解決失敗 |
| `failed to resolve clear_lamp` | クリアランプ解決失敗 |
| `failed to resolve combo_lamp` | コンボランプ解決失敗 |
| `failed to resolve full_chain` | フルチェイン解決失敗 |
| `failed to resolve slot` | スロット解決失敗 |
| `failed to resolve worldsend chart` | WORLD'S END譜面解決失敗 |
| `score out of range: {score}` | WORLD'S ENDスコア範囲外 |
