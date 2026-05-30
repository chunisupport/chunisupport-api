#!/usr/bin/env python3
import sys

# .pyc / __pycache__ の生成を完全に抑制（CI 実行時にディレクトリを汚さないため）
sys.dont_write_bytecode = True

"""
Discord Webhook 通知送信スクリプト（依存なし版）

- urllib と標準ライブラリのみを使用（pip install 不要）
- GitHub Actions の ubuntu-latest でそのまま実行可能
- 環境変数で通知内容を指定し、埋め込みメッセージ（embed）を送信
- DISCORD_WEBHOOK_URL が未設定/空の場合は通知をスキップ（エラー終了しない）

使用例（build-start）:
  env:
    DISCORD_WEBHOOK_URL: ${{ secrets.DISCORD_WEBHOOK_URL }}
    DISCORD_NOTIFY_MODE: build-start
    REPO: ...
    ...
  run: python .github/scripts/send_discord.py

対応モード:
  - build-start
  - build-complete
"""

import json
import os
import sys
import time
from datetime import datetime, timezone
from urllib.error import HTTPError, URLError
from urllib.parse import parse_qsl, urlencode, urlsplit, urlunsplit
from urllib.request import Request, urlopen


def get_env(key: str, default: str = "") -> str:
    """環境変数を取得。デフォルト値を指定可能。"""
    return os.environ.get(key, default)


def with_query_param(url: str, key: str, value: str) -> str:
    parts = urlsplit(url)
    query = dict(parse_qsl(parts.query, keep_blank_values=True))
    query[key] = value
    return urlunsplit((parts.scheme, parts.netloc, parts.path, urlencode(query), parts.fragment))


def discord_message_url(webhook_url: str, message_id: str) -> str:
    base_url = webhook_url.split("?", 1)[0].rstrip("/")
    return f"{base_url}/messages/{message_id}"


def send_discord(webhook_url: str, payload: dict) -> None:
    """
    Discord Webhook に JSON ペイロードを送信する。
    送信失敗時は標準エラーに出力するが、終了コードは 0 のまま（CI 継続）。
    """
    data = json.dumps(payload, ensure_ascii=False).encode("utf-8")
    req = Request(
        webhook_url,
        data=data,
        headers={
            "Content-Type": "application/json",
            "User-Agent": "chunisupport-api-ci/1.0 (urllib)",
        },
    )

    try:
        with urlopen(req, timeout=10) as resp:
            print(f"Discord通知を送信しました (HTTP {resp.status})")
    except HTTPError as e:
        # Discord 側で 400/429 などが返る場合も CI は止めない
        print(f"Discord通知の送信に失敗しました（継続します）: {e.code} {e.reason}", file=sys.stderr)
    except URLError as e:
        print(f"Discord通知の送信に失敗しました（継続します）: {e.reason}", file=sys.stderr)
    except Exception as e:
        print(f"Discord通知の送信中に予期しないエラー（継続します）: {e}", file=sys.stderr)


def send_discord_and_get_message_id(webhook_url: str, payload: dict) -> str:
    data = json.dumps(payload, ensure_ascii=False).encode("utf-8")
    req = Request(
        with_query_param(webhook_url, "wait", "true"),
        data=data,
        headers={
            "Content-Type": "application/json",
            "User-Agent": "chunisupport-api-ci/1.0 (urllib)",
        },
    )

    try:
        with urlopen(req, timeout=10) as resp:
            body = resp.read().decode("utf-8")
            message = json.loads(body)
            message_id = str(message.get("id", ""))
            print(f"Discord通知を送信しました (HTTP {resp.status})")
            return message_id
    except HTTPError as e:
        # Discord 側で 400/429 などが返る場合も CI は止めない
        print(f"Discord通知の送信に失敗しました（継続します）: {e.code} {e.reason}", file=sys.stderr)
    except URLError as e:
        print(f"Discord通知の送信に失敗しました（継続します）: {e.reason}", file=sys.stderr)
    except Exception as e:
        print(f"Discord通知の送信中に予期しないエラー（継続します）: {e}", file=sys.stderr)
    return ""


def get_discord_message(webhook_url: str, message_id: str) -> dict:
    req = Request(
        discord_message_url(webhook_url, message_id),
        headers={
            "User-Agent": "chunisupport-api-ci/1.0 (urllib)",
        },
        method="GET",
    )

    try:
        with urlopen(req, timeout=10) as resp:
            return json.loads(resp.read().decode("utf-8"))
    except HTTPError as e:
        # Discord 側で 400/429 などが返る場合も CI は止めない
        print(f"Discord通知の取得に失敗しました（継続します）: {e.code} {e.reason}", file=sys.stderr)
    except URLError as e:
        print(f"Discord通知の取得に失敗しました（継続します）: {e.reason}", file=sys.stderr)
    except Exception as e:
        print(f"Discord通知の取得中に予期しないエラー（継続します）: {e}", file=sys.stderr)
    return {}


def update_discord_message(webhook_url: str, message_id: str, payload: dict) -> bool:
    data = json.dumps(payload, ensure_ascii=False).encode("utf-8")
    req = Request(
        discord_message_url(webhook_url, message_id),
        data=data,
        headers={
            "Content-Type": "application/json",
            "User-Agent": "chunisupport-api-ci/1.0 (urllib)",
        },
        method="PATCH",
    )

    try:
        with urlopen(req, timeout=10) as resp:
            print(f"Discord通知を更新しました (HTTP {resp.status})")
            return True
    except HTTPError as e:
        # Discord 側で 400/429 などが返る場合も CI は止めない
        print(f"Discord通知の更新に失敗しました（継続します）: {e.code} {e.reason}", file=sys.stderr)
    except URLError as e:
        print(f"Discord通知の更新に失敗しました（継続します）: {e.reason}", file=sys.stderr)
    except Exception as e:
        print(f"Discord通知の更新中に予期しないエラー（継続します）: {e}", file=sys.stderr)
    return False


def github_repo_url(env: dict) -> str:
    server_url = env.get("GITHUB_SERVER_URL", "https://github.com").rstrip("/")
    repo = env.get("REPO", "unknown")
    return f"{server_url}/{repo}"


def build_build_start_embed(env: dict) -> dict:
    """ビルド開始通知用の embed を構築。"""
    repo = env.get("REPO", "unknown")
    branch = env.get("BRANCH", "unknown")
    sha = env.get("SHA", "unknown")
    short_sha = sha[:7] if len(sha) >= 7 else sha
    arch = env.get("TARGET_ARCH", "unknown")
    arch_label = env.get("TARGET_ARCH_LABEL", f"linux/{arch}")

    return {
        "author": {"name": repo, "url": github_repo_url(env)},
        "title": f"🚀 ビルド開始 ({arch})",
        "description": f"{arch_label} ビルドを開始しました",
        "color": 3447003,
        "fields": [
            {"name": "コミット", "value": short_sha, "inline": True},
        ],
        "footer": {"text": branch},
        "timestamp": datetime.now(timezone.utc).isoformat(),
    }


def build_build_start_embeds(env: dict) -> list[dict]:
    return [
        build_build_start_embed(env | {"TARGET_ARCH": arch, "TARGET_ARCH_LABEL": label})
        for arch, label in target_arches()
    ]


def target_arches() -> tuple[tuple[str, str], ...]:
    return (("amd64", "linux/amd64"), ("arm64", "linux/arm64"))


def arch_label(arch: str) -> str:
    for target_arch, label in target_arches():
        if target_arch == arch:
            return label
    return f"linux/{arch}"


def build_build_complete_embed(env: dict) -> dict:
    """ビルド完了通知用の embed を構築（結果によりタイトル・色・文言を切り替え）。"""
    repo = env.get("REPO", "unknown")
    branch = env.get("BRANCH", "unknown")
    sha = env.get("SHA", "unknown")
    short_sha = sha[:7] if len(sha) >= 7 else sha
    build_result = env.get("BUILD_RESULT", "failure")
    arch = env.get("TARGET_ARCH", "unknown")
    arch_label = env.get("TARGET_ARCH_LABEL", f"linux/{arch}")

    if build_result == "success":
        status = "✅"
        title_text = "ビルド完了"
        color = 3066993
        desc = f"{arch_label} ビルドが正常に完了しました"
    elif build_result == "cancelled":
        status = "⚠️"
        title_text = "ビルドキャンセル"
        color = 16776960
        desc = f"{arch_label} ビルドがキャンセルされました"
    else:
        status = "❌"
        title_text = "ビルド失敗"
        color = 15158332
        desc = f"{arch_label} ビルドが失敗しました"

    return {
        "author": {"name": repo, "url": github_repo_url(env)},
        "title": f"{status} {title_text} ({arch})",
        "description": desc,
        "color": color,
        "fields": [
            {"name": "コミット", "value": short_sha, "inline": True},
        ],
        "footer": {"text": branch},
        "timestamp": datetime.now(timezone.utc).isoformat(),
    }


def build_build_complete_embeds(env: dict) -> list[dict]:
    return [
        build_build_complete_embed(env | {"TARGET_ARCH": arch, "TARGET_ARCH_LABEL": label})
        for arch, label in target_arches()
    ]


def embed_target_arch(embed: dict) -> str:
    for field in embed.get("fields", []):
        if field.get("name") == "対象アーキテクチャ":
            value = field.get("value", "")
            if value.startswith("linux/"):
                return value.removeprefix("linux/")
    title = embed.get("title", "")
    for target_arch, _ in target_arches():
        if title.endswith(f"({target_arch})"):
            return target_arch
    return ""


def replace_arch_embed(embeds: list[dict], arch: str, new_embed: dict) -> list[dict]:
    replaced = False
    next_embeds = []

    for embed in embeds:
        if embed_target_arch(embed) == arch:
            next_embeds.append(new_embed)
            replaced = True
        else:
            next_embeds.append(embed)

    if not replaced:
        next_embeds.append(new_embed)

    return next_embeds


def has_arch_embed(embeds: list[dict], arch: str, title: str) -> bool:
    return any(embed_target_arch(embed) == arch and embed.get("title") == title for embed in embeds)


def update_discord_arch_embed(webhook_url: str, message_id: str, env: dict) -> bool:
    arch = env.get("TARGET_ARCH", "")
    if not arch:
        print("TARGET_ARCH が未設定のためアーキテクチャ別更新をスキップします", file=sys.stderr)
        return False

    target_embed = build_build_complete_embed(env | {"TARGET_ARCH_LABEL": arch_label(arch)})
    for attempt in range(3):
        message = get_discord_message(webhook_url, message_id)
        current_embeds = message.get("embeds", [])
        payload = {
            "username": "Build & Deploy | chunisupport-api",
            "embeds": replace_arch_embed(current_embeds, arch, target_embed),
        }
        if not update_discord_message(webhook_url, message_id, payload):
            time.sleep(2**attempt)
            continue

        time.sleep(1)
        latest_message = get_discord_message(webhook_url, message_id)
        if has_arch_embed(latest_message.get("embeds", []), arch, target_embed["title"]):
            return True

        print("Discord通知が他の更新で上書きされた可能性があるため再試行します")
        time.sleep(2**attempt)

    return False


def main() -> int:
    webhook_url = get_env("DISCORD_WEBHOOK_URL")
    if not webhook_url:
        print("DISCORD_WEBHOOK_URL が未設定のため通知をスキップします")
        return 0

    mode = get_env("DISCORD_NOTIFY_MODE", "build-start")

    # 環境変数から必要な値だけを辞書にまとめて関数に渡す（テスト容易性も考慮）
    env = {
        "REPO": get_env("REPO"),
        "BRANCH": get_env("BRANCH"),
        "GITHUB_SERVER_URL": get_env("GITHUB_SERVER_URL", "https://github.com"),
        "SHA": get_env("SHA"),
        "BUILD_RESULT": get_env("BUILD_RESULT"),
        "TARGET_ARCH": get_env("TARGET_ARCH"),
    }

    message_id_path = get_env("DISCORD_MESSAGE_ID_PATH")
    message_id = ""
    if message_id_path and os.path.exists(message_id_path):
        with open(message_id_path, encoding="utf-8") as f:
            message_id = f.read().strip()

    if mode == "build-arch-complete":
        if message_id:
            update_discord_arch_embed(webhook_url, message_id, env)
        else:
            print("Discord Message ID が未取得のためアーキテクチャ別更新をスキップします")
        return 0

    if mode == "build-start":
        embeds = build_build_start_embeds(env)
    elif mode == "build-complete":
        embeds = build_build_complete_embeds(env)
    else:
        print(f"未知の DISCORD_NOTIFY_MODE: {mode}", file=sys.stderr)
        return 1

    payload = {
        "username": "Build & Deploy | chunisupport-api",
        "embeds": embeds,
    }

    if message_id and update_discord_message(webhook_url, message_id, payload):
        return 0

    if not message_id_path:
        send_discord(webhook_url, payload)
        return 0

    message_id = send_discord_and_get_message_id(webhook_url, payload)
    if message_id:
        message_id_dir = os.path.dirname(message_id_path)
        if message_id_dir:
            os.makedirs(message_id_dir, exist_ok=True)
        with open(message_id_path, "w", encoding="utf-8") as f:
            f.write(message_id)

    return 0


if __name__ == "__main__":
    sys.exit(main())
