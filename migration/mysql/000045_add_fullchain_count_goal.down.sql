-- ユーザーの目標を暗黙に削除しないため、参照が残る場合は外部キー制約によりdownを中止する。
DELETE FROM achievement_types WHERE code = 'fullchain_count';