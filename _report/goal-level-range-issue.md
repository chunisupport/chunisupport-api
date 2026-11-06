# 目標の範囲入力モード保存の保留メモ

このファイルは未実装事項の追跡メモであり、現在仕様の根拠にはしない。

- API: https://github.com/chunisupport/chunisupport-api/issues/256
- フロント: https://github.com/chunisupport/chunisupport-front/issues/374

今回はAPIの実装を変更しない。`validateAttributes` が未知の属性を拒否し、`const` 内もmin/maxだけに正規化するため、フロントだけで入力モードを追加保存することはできない。

任意のモード属性を許可・検証・保持し、その後フロントでレベル／譜面定数範囲入力を追加する。属性名はIssueで合意する。モード欠落を許容し、既存目標の定数範囲・進捗計算を変えない。作成・更新・取得とユーザーデータ移行経路を確認する。

受け入れ条件と検討事項は各Issueに記載する。DBカラム追加や既存目標の一括更新は想定しない。
