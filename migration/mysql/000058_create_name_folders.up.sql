CREATE TABLE name_folders (
    id TINYINT UNSIGNED NOT NULL AUTO_INCREMENT,
    code VARCHAR(10) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    name VARCHAR(20) NOT NULL,
    sort_order TINYINT UNSIGNED NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_name_folders_code (code),
    UNIQUE KEY uq_name_folders_name (name),
    UNIQUE KEY uq_name_folders_sort_order (sort_order)
);

INSERT INTO name_folders (id, code, name, sort_order) VALUES
    (1, 'ABCD', 'ABCD', 1),
    (2, 'EFGH', 'EFGH', 2),
    (3, 'IJKL', 'IJKL', 3),
    (4, 'MNOP', 'MNOP', 4),
    (5, 'QRST', 'QRST', 5),
    (6, 'UVWXYZ', 'UVWXYZ', 6),
    (7, 'A', 'あ行', 7),
    (8, 'KA', 'か行', 8),
    (9, 'SA', 'さ行', 9),
    (10, 'TA', 'た行', 10),
    (11, 'NA', 'な行', 11),
    (12, 'HA', 'は行', 12),
    (13, 'MA', 'ま行', 13),
    (14, 'YA', 'や行', 14),
    (15, 'RA', 'ら行', 15),
    (16, 'WA', 'わ行', 16),
    (17, 'NUMBER', '数字', 17);
