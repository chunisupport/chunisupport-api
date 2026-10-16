-- sort_order は一意制約を持つため、一度既存値と重ならない範囲へ退避してから元の値へ戻す。
UPDATE name_folders SET sort_order = sort_order + 100;

UPDATE name_folders
SET sort_order = CASE code
    WHEN 'ABCD' THEN 1
    WHEN 'EFGH' THEN 2
    WHEN 'IJKL' THEN 3
    WHEN 'MNOP' THEN 4
    WHEN 'QRST' THEN 5
    WHEN 'UVWXYZ' THEN 6
    WHEN 'A' THEN 7
    WHEN 'KA' THEN 8
    WHEN 'SA' THEN 9
    WHEN 'TA' THEN 10
    WHEN 'NA' THEN 11
    WHEN 'HA' THEN 12
    WHEN 'MA' THEN 13
    WHEN 'YA' THEN 14
    WHEN 'RA' THEN 15
    WHEN 'WA' THEN 16
    WHEN 'NUMBER' THEN 17
END;
