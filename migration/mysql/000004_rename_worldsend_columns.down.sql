-- WORLD'S END用語変更のロールバック
-- attribute → we_kanji
-- level_star → we_star

-- CHECK制約の削除と再作成
ALTER TABLE worldsend_charts DROP CHECK worldsend_charts_chk_1;
ALTER TABLE worldsend_charts ADD CONSTRAINT worldsend_charts_chk_1 
    CHECK (we_star IS NULL OR we_star BETWEEN 1 AND 5);

ALTER TABLE worldsend_charts
    CHANGE COLUMN attribute we_kanji CHAR(1),
    CHANGE COLUMN level_star we_star TINYINT;
