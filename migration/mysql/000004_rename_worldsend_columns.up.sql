-- WORLD'S END用語の統一的なリネーム
-- we_kanji → attribute (カテゴリ漢字 → 属性)
-- we_star → level_star (星の数 → レベル)

-- CHECK制約の削除（MySQL 8.0.16+）
ALTER TABLE worldsend_charts DROP CHECK worldsend_charts_chk_1;

-- カラム名の変更
ALTER TABLE worldsend_charts
    CHANGE COLUMN we_kanji attribute CHAR(1),
    CHANGE COLUMN we_star level_star TINYINT;

-- CHECK制約の再作成
ALTER TABLE worldsend_charts ADD CONSTRAINT worldsend_charts_chk_1 
    CHECK (level_star IS NULL OR level_star BETWEEN 1 AND 5);
