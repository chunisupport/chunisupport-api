-- 表示幅を最小限に抑えるための超ショート名を追加する。
-- ジャンルはCHUNITHM Wikiの略記、バージョンは3〜4文字の略記に揃える。
ALTER TABLE genres
    ADD COLUMN short_name VARCHAR(10) NULL AFTER name;

UPDATE genres SET short_name = 'P&A' WHERE name = 'POPS & ANIME';
UPDATE genres SET short_name = 'nico' WHERE name = 'niconico';
UPDATE genres SET short_name = '東方' WHERE name = '東方Project';
UPDATE genres SET short_name = 'VAR' WHERE name = 'VARIETY';
UPDATE genres SET short_name = 'イロ' WHERE name = 'イロドリミドリ';
UPDATE genres SET short_name = '撃舞' WHERE name = 'ゲキマイ';
UPDATE genres SET short_name = 'ORI' WHERE name = 'ORIGINAL';

-- 未投入の行が残っている場合はNOT NULL化で失敗させ、略記の登録漏れに気づけるようにする。
ALTER TABLE genres
    MODIFY COLUMN short_name VARCHAR(10) NOT NULL,
    ADD UNIQUE KEY uq_genres_short_name (short_name);

ALTER TABLE versions
    ADD COLUMN short_name VARCHAR(10) NULL AFTER name;

UPDATE versions SET short_name = 'ORI' WHERE name = 'CHUNITHM';
UPDATE versions SET short_name = 'ORI+' WHERE name = 'CHUNITHM PLUS';
UPDATE versions SET short_name = 'AIR' WHERE name = 'CHUNITHM AIR';
UPDATE versions SET short_name = 'AIR+' WHERE name = 'CHUNITHM AIR PLUS';
UPDATE versions SET short_name = 'STR' WHERE name = 'CHUNITHM STAR';
UPDATE versions SET short_name = 'STR+' WHERE name = 'CHUNITHM STAR PLUS';
UPDATE versions SET short_name = 'AMZ' WHERE name = 'CHUNITHM AMAZON';
UPDATE versions SET short_name = 'AMZ+' WHERE name = 'CHUNITHM AMAZON PLUS';
UPDATE versions SET short_name = 'CRY' WHERE name = 'CHUNITHM CRYSTAL';
UPDATE versions SET short_name = 'CRY+' WHERE name = 'CHUNITHM CRYSTAL PLUS';
UPDATE versions SET short_name = 'PAR' WHERE name = 'CHUNITHM PARADISE';
UPDATE versions SET short_name = 'PAR×' WHERE name = 'CHUNITHM PARADISE LOST';
UPDATE versions SET short_name = 'NEW' WHERE name = 'CHUNITHM NEW';
UPDATE versions SET short_name = 'NEW+' WHERE name = 'CHUNITHM NEW PLUS';
UPDATE versions SET short_name = 'SUN' WHERE name = 'CHUNITHM SUN';
UPDATE versions SET short_name = 'SUN+' WHERE name = 'CHUNITHM SUN PLUS';
UPDATE versions SET short_name = 'LMN' WHERE name = 'CHUNITHM LUMINOUS';
UPDATE versions SET short_name = 'LMN+' WHERE name = 'CHUNITHM LUMINOUS PLUS';
UPDATE versions SET short_name = 'VRS' WHERE name = 'CHUNITHM VERSE';
UPDATE versions SET short_name = 'XVRS' WHERE name = 'CHUNITHM X-VERSE';
UPDATE versions SET short_name = 'XVSX' WHERE name = 'CHUNITHM X-VERSE-X';
UPDATE versions SET short_name = 'MAT' WHERE name = 'CHUNITHM Mate';

ALTER TABLE versions
    MODIFY COLUMN short_name VARCHAR(10) NOT NULL,
    ADD UNIQUE KEY uq_versions_short_name (short_name);
