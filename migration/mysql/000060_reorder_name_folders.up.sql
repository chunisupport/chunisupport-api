-- 筐体の楽曲名順に合わせ、あ行〜わ行、ABCD〜UVWXYZ、数字の順に並べ替える。
-- sort_order は一意制約を持つため、一度既存値と重ならない範囲へ退避してから確定値を設定する。
UPDATE name_folders SET sort_order = sort_order + 100;

UPDATE name_folders
SET sort_order = CASE code
    WHEN 'A' THEN 1
    WHEN 'KA' THEN 2
    WHEN 'SA' THEN 3
    WHEN 'TA' THEN 4
    WHEN 'NA' THEN 5
    WHEN 'HA' THEN 6
    WHEN 'MA' THEN 7
    WHEN 'YA' THEN 8
    WHEN 'RA' THEN 9
    WHEN 'WA' THEN 10
    WHEN 'ABCD' THEN 11
    WHEN 'EFGH' THEN 12
    WHEN 'IJKL' THEN 13
    WHEN 'MNOP' THEN 14
    WHEN 'QRST' THEN 15
    WHEN 'UVWXYZ' THEN 16
    WHEN 'NUMBER' THEN 17
END;
