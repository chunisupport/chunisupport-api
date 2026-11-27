# 期間限定のお知らせ（app_announcements）

プロフィールカード上部に表示する期間限定のお知らせを `app_announcements` テーブルで管理し、`GET /internal/app-announcements`（仕様は `docs/API.md`）で配信します。Docsの記事配信（Markdown/MDX、`/announcements.json`）とは独立しています。

登録・変更・削除は、フロントエンドの管理メニュー（ADMIN専用の `/internal/admin/app-announcements`）で行います。以下は管理画面を使えない場合に、本番DBへの書き込み権限を持つ運営者がSQLで行う手順です。

## 項目と制約

| 列 | 内容 |
| --- | --- |
| `id` | 主キー兼APIの `id`。`LOWER(HEX(RANDOM_BYTES(8)))` で採番する16桁の16進数 |
| `kind` | 選択したときの動作。`message`（本文型: バナーを選ぶとダイアログで `title` と `body` を表示）または `link`（リンク型: バナー自体がリンクで、選ぶと遷移） |
| `title` | 表示する文言（必須、200文字以内） |
| `body` | 本文型で必須のプレーンテキスト（改行可）。`[リンクテキスト](URL)` と書くとフロントエンドでリンクになる。リンク型では `NULL` |
| `target_url` | リンク型で必須の遷移先。本文型では `NULL`。アプリ内パス（例: `/tools/rating-calculator`）または外部URL |
| `starts_at` / `expires_at` | 表示期間（**UTC**）。`starts_at <= 現在 < expires_at` の間だけ配信する |
| `revision` | 1以上。閉じた利用者にも再表示させたいときに増やす |

`starts_at < expires_at` と `revision >= 1` はDB制約で拒否されます。種別と `body` / `target_url` の組み合わせは、種別の追加に追従しやすいようDB制約にせずAPI取得時に検証します。リンク先の内容は検証しないので、URLは登録前によく確認してください。終了前の行に不正なデータがあるとAPI全体が 500 を返すため、登録後に必ず下記の確認を行ってください（開始前の行も検証対象です）。

## 登録

日時はUTCで保存します。日本時間で指定する場合は `CONVERT_TZ` で変換します。`updated_at` もUTCで記録されるよう、先に [手作業SQLのタイムゾーン](configuration.md#手作業sqlのタイムゾーン) に従ってセッションをUTCにしてください。

本文型:

```sql
INSERT INTO app_announcements (id, kind, title, body, target_url, starts_at, expires_at)
VALUES (
    LOWER(HEX(RANDOM_BYTES(8))),
    'message',
    'メンテナンスのお知らせ',
    '12月10日 2:00〜4:00 にメンテナンスを行います。\nこの間はデータ登録ができません。',
    NULL,
    CONVERT_TZ('2026-11-30 12:00:00', '+09:00', '+00:00'),
    CONVERT_TZ('2026-12-10 04:00:00', '+09:00', '+00:00')
);
```

リンク型（Docs記事・アプリ内パス）:

```sql
INSERT INTO app_announcements (id, kind, title, body, target_url, starts_at, expires_at)
VALUES (
    LOWER(HEX(RANDOM_BYTES(8))),
    'link',
    '大型アップデートのお知らせ',
    NULL,
    'https://docs.chunisupport.net/announcements/2026-12-01/',
    CONVERT_TZ('2026-12-01 00:00:00', '+09:00', '+00:00'),
    CONVERT_TZ('2026-12-15 00:00:00', '+09:00', '+00:00')
);
```

## 登録後の確認

ログインした状態で `GET /internal/app-announcements` を呼び、200 が返ることを確認します。500 が返った場合はアプリログ（`restore app announcement <id>`）で対象行を特定し、修正するか削除します。不正な行がある間は管理画面の一覧と更新も 500 になるため、SQLで直してください。

## 変更・停止

```sql
-- 停止（即時に反映される）
DELETE FROM app_announcements WHERE id = '<id>';

-- 内容を改訂して、閉じた利用者にも再表示させる
UPDATE app_announcements SET title = '...', revision = revision + 1 WHERE id = '<id>';
```

終了したお知らせはAPIから返らないため、削除は不要です。
