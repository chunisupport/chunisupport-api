#!/usr/bin/env python3
"""smalldata.db のレコードフィルタをMySQLへ一度だけ移送します。"""

from __future__ import annotations

import argparse
import importlib
import sqlite3
import sys
from dataclasses import dataclass
from datetime import datetime, timezone
from pathlib import Path
from typing import Any, Sequence


if hasattr(sys.stdout, "reconfigure"):
    sys.stdout.reconfigure(encoding="utf-8")
if hasattr(sys.stderr, "reconfigure"):
    sys.stderr.reconfigure(encoding="utf-8")


# 実行対象のMySQL接続情報です。移送先に合わせて直接編集してください。
MYSQL_HOST = "127.0.0.1"
MYSQL_PORT = 3306
MYSQL_DATABASE = "chunisupport"
MYSQL_USER = "chunisupport"
MYSQL_PASSWORD = "Chunisupportchunisupport1@"

PYMYSQL_REQUIREMENT = "PyMySQL[rsa]==1.1.2"
DEPENDENCY_DIR_NAME = ".record_filter_migration_deps"
USER_ID_QUERY_CHUNK_SIZE = 1000

SELECT_RECORD_FILTERS = """
SELECT id, user_id, name, filter_value_gzip, is_worldsend, created_at, updated_at
FROM record_filters
ORDER BY id ASC
"""

INSERT_RECORD_FILTER = """
INSERT INTO record_filters (
    id,
    user_id,
    name,
    filter_value_gzip,
    is_worldsend,
    created_at,
    updated_at
)
VALUES (%s, %s, %s, %s, %s, %s, %s)
"""


def pip_install_command() -> str:
    return (
        f'python -m pip install --upgrade --target {DEPENDENCY_DIR_NAME} '
        f'"{PYMYSQL_REQUIREMENT}"'
    )


@dataclass(frozen=True)
class RecordFilterRow:
    id: bytes
    user_id: int
    name: str
    filter_value_gzip: bytes
    is_worldsend: bool
    created_at: datetime
    updated_at: datetime

    def insert_parameters(self) -> tuple[Any, ...]:
        return (
            self.id,
            self.user_id,
            self.name,
            self.filter_value_gzip,
            self.is_worldsend,
            self.created_at,
            self.updated_at,
        )


def parse_arguments() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="SQLiteの保存済みレコードフィルタを空のMySQLテーブルへ移送します。"
    )
    parser.add_argument(
        "sqlite_path",
        type=Path,
        help="smalldata.dbの絶対パス",
    )
    return parser.parse_args()


def resolve_sqlite_path(value: Path) -> Path:
    if not value.is_absolute():
        raise ValueError("SQLiteのパスには絶対パスを指定してください")

    resolved = value.resolve(strict=True)
    if not resolved.is_file():
        raise ValueError(f"SQLiteのパスがファイルではありません: {resolved}")
    return resolved


def load_pymysql() -> Any:
    dependency_dir = Path.cwd() / DEPENDENCY_DIR_NAME
    package_dir = dependency_dir / "pymysql"

    if not package_dir.is_dir():
        raise RuntimeError(
            f"PyMySQLが{dependency_dir}にありません。実行前に次のコマンドを実行してください: "
            f"{pip_install_command()}"
        )

    sys.path.insert(0, str(dependency_dir))
    importlib.invalidate_caches()
    try:
        pymysql = importlib.import_module("pymysql")
        importlib.import_module("cryptography")
    except ModuleNotFoundError as error:
        raise RuntimeError(
            f"PyMySQLまたはcryptographyを{dependency_dir}から読み込めません。"
            f"次のコマンドで依存を更新してください: {pip_install_command()}"
        ) from error
    return pymysql


def normalize_datetime(value: Any, row_number: int, column_name: str) -> datetime:
    if isinstance(value, datetime):
        parsed = value
    elif isinstance(value, str):
        try:
            parsed = datetime.fromisoformat(value)
        except ValueError as error:
            raise ValueError(
                f"{row_number}行目の{column_name}が日時形式ではありません"
            ) from error
    else:
        raise ValueError(f"{row_number}行目の{column_name}が日時ではありません")

    if parsed.tzinfo is not None:
        parsed = parsed.astimezone(timezone.utc).replace(tzinfo=None)
    return parsed


def normalize_sqlite_row(row: Sequence[Any], row_number: int) -> RecordFilterRow:
    if len(row) != 7:
        raise ValueError(f"{row_number}行目のカラム数が7ではありません")

    raw_id, user_id, name, raw_filter, is_worldsend, created_at, updated_at = row
    record_filter_id = bytes(raw_id) if isinstance(raw_id, (bytes, bytearray, memoryview)) else b""
    if len(record_filter_id) != 16:
        raise ValueError(f"{row_number}行目のIDが16バイトではありません")
    if not isinstance(user_id, int) or user_id <= 0:
        raise ValueError(f"{row_number}行目のuser_idが正の整数ではありません")
    if not isinstance(name, str) or not name or len(name) > 30:
        raise ValueError(f"{row_number}行目のnameが1文字以上30文字以下ではありません")

    filter_value_gzip = (
        bytes(raw_filter)
        if isinstance(raw_filter, (bytes, bytearray, memoryview))
        else b""
    )
    if not filter_value_gzip:
        raise ValueError(f"{row_number}行目のfilter_value_gzipが空です")
    if is_worldsend not in (0, 1, False, True):
        raise ValueError(f"{row_number}行目のis_worldsendが0または1ではありません")

    return RecordFilterRow(
        id=record_filter_id,
        user_id=user_id,
        name=name,
        filter_value_gzip=filter_value_gzip,
        is_worldsend=bool(is_worldsend),
        created_at=normalize_datetime(created_at, row_number, "created_at"),
        updated_at=normalize_datetime(updated_at, row_number, "updated_at"),
    )


def read_sqlite_rows(sqlite_path: Path) -> list[RecordFilterRow]:
    sqlite_uri = f"{sqlite_path.as_uri()}?mode=ro"
    with sqlite3.connect(sqlite_uri, uri=True) as connection:
        raw_rows = connection.execute(SELECT_RECORD_FILTERS).fetchall()

    rows = [
        normalize_sqlite_row(row, row_number)
        for row_number, row in enumerate(raw_rows, start=1)
    ]
    ids = {row.id for row in rows}
    if len(ids) != len(rows):
        raise ValueError("SQLiteに重複したレコードフィルタIDがあります")
    return rows


def connect_mysql(pymysql: Any) -> Any:
    return pymysql.connect(
        host=MYSQL_HOST,
        port=MYSQL_PORT,
        user=MYSQL_USER,
        password=MYSQL_PASSWORD,
        database=MYSQL_DATABASE,
        charset="utf8mb4",
        autocommit=False,
    )


def ensure_target_empty(cursor: Any) -> None:
    cursor.execute("SELECT COUNT(*) FROM record_filters")
    count = cursor.fetchone()[0]
    if count != 0:
        raise ValueError(f"MySQLのrecord_filtersが空ではありません: {count}件")


def select_existing_user_ids(cursor: Any, user_ids: set[int], lock: bool) -> set[int]:
    existing_user_ids: set[int] = set()
    sorted_user_ids = sorted(user_ids)
    for start in range(0, len(sorted_user_ids), USER_ID_QUERY_CHUNK_SIZE):
        chunk = sorted_user_ids[start : start + USER_ID_QUERY_CHUNK_SIZE]
        placeholders = ", ".join(["%s"] * len(chunk))
        lock_clause = " FOR UPDATE" if lock else ""
        cursor.execute(
            f"SELECT id FROM users WHERE id IN ({placeholders}){lock_clause}",
            chunk,
        )
        existing_user_ids.update(row[0] for row in cursor.fetchall())
    return existing_user_ids


def ensure_users_exist(cursor: Any, rows: Sequence[RecordFilterRow], lock: bool) -> None:
    user_ids = {row.user_id for row in rows}
    if not user_ids:
        return

    existing_user_ids = select_existing_user_ids(cursor, user_ids, lock)
    missing_user_ids = sorted(user_ids - existing_user_ids)
    if missing_user_ids:
        values = ", ".join(str(user_id) for user_id in missing_user_ids)
        raise ValueError(f"MySQLに存在しないuser_idがあります: {values}")


def read_mysql_rows(cursor: Any) -> list[RecordFilterRow]:
    cursor.execute(SELECT_RECORD_FILTERS)
    return [
        normalize_sqlite_row(row, row_number)
        for row_number, row in enumerate(cursor.fetchall(), start=1)
    ]


def migrate_rows(connection: Any, rows: Sequence[RecordFilterRow]) -> None:
    # 事前検証の読み取りトランザクションを終了してから、本移送を開始します。
    try:
        with connection.cursor() as cursor:
            cursor.execute("SET time_zone = '+00:00'")
            ensure_target_empty(cursor)
            ensure_users_exist(cursor, rows, lock=False)
    finally:
        connection.rollback()

    connection.begin()
    try:
        with connection.cursor() as cursor:
            ensure_target_empty(cursor)
            ensure_users_exist(cursor, rows, lock=True)
            if rows:
                cursor.executemany(
                    INSERT_RECORD_FILTER,
                    [row.insert_parameters() for row in rows],
                )

            migrated_rows = read_mysql_rows(cursor)
            if migrated_rows != list(rows):
                raise ValueError("MySQLへの移送結果がSQLiteの元データと一致しません")
        connection.commit()
    except BaseException:
        connection.rollback()
        raise


def main() -> int:
    try:
        arguments = parse_arguments()
        sqlite_path = resolve_sqlite_path(arguments.sqlite_path)
        rows = read_sqlite_rows(sqlite_path)
        print(f"SQLiteから{len(rows)}件を検証しました: {sqlite_path}")
        print("注意: 実行中はAPIとバッチからrecord_filtersへの書き込みを停止してください。")

        pymysql = load_pymysql()
        connection = connect_mysql(pymysql)
        try:
            migrate_rows(connection, rows)
        finally:
            connection.close()

        print(
            f"MySQLへ{len(rows)}件を移送しました: "
            f"{MYSQL_USER}@{MYSQL_HOST}:{MYSQL_PORT}/{MYSQL_DATABASE}"
        )
        print(f"移送後は {Path.cwd() / DEPENDENCY_DIR_NAME} を削除できます。")
        return 0
    except (OSError, sqlite3.Error, ValueError) as error:
        print(f"移送を中止しました: {error}", file=sys.stderr)
        return 1
    except Exception as error:
        print(f"移送を中止しました: {error}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
