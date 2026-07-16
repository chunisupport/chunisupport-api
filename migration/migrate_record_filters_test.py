import unittest
from datetime import datetime
from pathlib import Path
from tempfile import TemporaryDirectory
from unittest.mock import MagicMock, patch

from migration.temp import migrate_record_filters


class NormalizeSQLiteRowTest(unittest.TestCase):
    def test_正常な行を移送用データへ変換する(self) -> None:
        # Given
        row = (
            b"1234567890123456",
            10,
            "通常枠",
            b"\x1f\x8b\x08",
            0,
            "2026-07-16 01:02:03",
            "2026-07-16 04:05:06",
        )

        # When
        actual = migrate_record_filters.normalize_sqlite_row(row, 1)

        # Then
        self.assertEqual(b"1234567890123456", actual.id)
        self.assertEqual(10, actual.user_id)
        self.assertEqual("通常枠", actual.name)
        self.assertEqual(b"\x1f\x8b\x08", actual.filter_value_gzip)
        self.assertFalse(actual.is_worldsend)
        self.assertEqual(datetime(2026, 7, 16, 1, 2, 3), actual.created_at)
        self.assertEqual(datetime(2026, 7, 16, 4, 5, 6), actual.updated_at)

    def test_IDが16バイトでなければ停止する(self) -> None:
        # Given
        row = (
            b"invalid",
            10,
            "通常枠",
            b"\x1f\x8b\x08",
            0,
            "2026-07-16 01:02:03",
            "2026-07-16 04:05:06",
        )

        # When / Then
        with self.assertRaisesRegex(ValueError, "IDが16バイトではありません"):
            migrate_record_filters.normalize_sqlite_row(row, 1)

    def test_不正なWorldsend値なら停止する(self) -> None:
        # Given
        row = (
            b"1234567890123456",
            10,
            "通常枠",
            b"\x1f\x8b\x08",
            2,
            "2026-07-16 01:02:03",
            "2026-07-16 04:05:06",
        )

        # When / Then
        with self.assertRaisesRegex(ValueError, "is_worldsendが0または1ではありません"):
            migrate_record_filters.normalize_sqlite_row(row, 1)


class MigrateRowsTest(unittest.TestCase):
    def test_空のデータを検証してコミットする(self) -> None:
        # Given
        connection = MagicMock()
        cursor = connection.cursor.return_value.__enter__.return_value
        cursor.fetchone.side_effect = [(0,), (0,)]
        cursor.fetchall.return_value = []

        # When
        migrate_record_filters.migrate_rows(connection, [])

        # Then
        connection.begin.assert_called_once_with()
        connection.commit.assert_called_once_with()
        connection.rollback.assert_called_once_with()

    def test_非空データを挿入して照合後にコミットする(self) -> None:
        # Given
        row = migrate_record_filters.RecordFilterRow(
            id=b"1234567890123456",
            user_id=10,
            name="通常枠",
            filter_value_gzip=b"\x1f\x8b\x08",
            is_worldsend=False,
            created_at=datetime(2026, 7, 16, 1, 2, 3),
            updated_at=datetime(2026, 7, 16, 4, 5, 6),
        )
        connection = MagicMock()
        cursor = connection.cursor.return_value.__enter__.return_value
        cursor.fetchone.side_effect = [(0,), (0,)]
        cursor.fetchall.side_effect = [
            [(10,)],
            [(10,)],
            [row.insert_parameters()],
        ]

        # When
        migrate_record_filters.migrate_rows(connection, [row])

        # Then
        cursor.executemany.assert_called_once_with(
            migrate_record_filters.INSERT_RECORD_FILTER,
            [row.insert_parameters()],
        )
        connection.commit.assert_called_once_with()
        connection.rollback.assert_called_once_with()

    def test_孤立ユーザーがあれば挿入前に停止する(self) -> None:
        # Given
        row = migrate_record_filters.RecordFilterRow(
            id=b"1234567890123456",
            user_id=10,
            name="通常枠",
            filter_value_gzip=b"\x1f\x8b\x08",
            is_worldsend=False,
            created_at=datetime(2026, 7, 16, 1, 2, 3),
            updated_at=datetime(2026, 7, 16, 4, 5, 6),
        )
        connection = MagicMock()
        cursor = connection.cursor.return_value.__enter__.return_value
        cursor.fetchone.return_value = (0,)
        cursor.fetchall.return_value = []

        # When / Then
        with self.assertRaisesRegex(ValueError, "MySQLに存在しないuser_idがあります: 10"):
            migrate_record_filters.migrate_rows(connection, [row])
        cursor.executemany.assert_not_called()
        connection.begin.assert_not_called()
        connection.commit.assert_not_called()
        connection.rollback.assert_called_once_with()

    def test_移送後のデータが一致しなければロールバックする(self) -> None:
        # Given
        row = migrate_record_filters.RecordFilterRow(
            id=b"1234567890123456",
            user_id=10,
            name="通常枠",
            filter_value_gzip=b"\x1f\x8b\x08",
            is_worldsend=False,
            created_at=datetime(2026, 7, 16, 1, 2, 3),
            updated_at=datetime(2026, 7, 16, 4, 5, 6),
        )
        connection = MagicMock()
        cursor = connection.cursor.return_value.__enter__.return_value
        cursor.fetchone.side_effect = [(0,), (0,)]
        cursor.fetchall.side_effect = [[(10,)], [(10,)], []]

        # When / Then
        with self.assertRaisesRegex(ValueError, "移送結果がSQLiteの元データと一致しません"):
            migrate_record_filters.migrate_rows(connection, [row])
        connection.commit.assert_not_called()
        self.assertEqual(2, connection.rollback.call_count)


class LoadPyMySQLTest(unittest.TestCase):
    def test_RSA認証用のextra依存を指定する(self) -> None:
        self.assertEqual("PyMySQL[rsa]==1.1.2", migrate_record_filters.PYMYSQL_REQUIREMENT)

    def test_事前インストールされていなければpipコマンドを案内して停止する(self) -> None:
        # Given
        with TemporaryDirectory() as temporary_directory:
            current_directory = Path(temporary_directory)

            # When / Then
            with patch.object(Path, "cwd", return_value=current_directory):
                with self.assertRaisesRegex(
                    RuntimeError,
                    "python -m pip install --upgrade --target .record_filter_migration_deps",
                ):
                    migrate_record_filters.load_pymysql()

    def test_cryptographyを読み込めなければ再インストールを案内する(self) -> None:
        # Given
        with TemporaryDirectory() as temporary_directory:
            current_directory = Path(temporary_directory)
            (current_directory / ".record_filter_migration_deps" / "pymysql").mkdir(
                parents=True
            )

            # When / Then
            with patch.object(Path, "cwd", return_value=current_directory):
                with patch(
                    "migration.temp.migrate_record_filters.importlib.import_module",
                    side_effect=[MagicMock(), ModuleNotFoundError("cryptography")],
                ):
                    with self.assertRaisesRegex(RuntimeError, "cryptography"):
                        migrate_record_filters.load_pymysql()


if __name__ == "__main__":
    unittest.main()
