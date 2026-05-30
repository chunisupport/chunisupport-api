ALTER TABLE difficulties
    ADD COLUMN sort_order TINYINT UNSIGNED NULL AFTER name;

UPDATE difficulties SET sort_order = 0 WHERE name = 'BASIC';
UPDATE difficulties SET sort_order = 1 WHERE name = 'ADVANCED';
UPDATE difficulties SET sort_order = 2 WHERE name = 'EXPERT';
UPDATE difficulties SET sort_order = 3 WHERE name = 'MASTER';
UPDATE difficulties SET sort_order = 4 WHERE name = 'ULTIMA';

ALTER TABLE difficulties
    MODIFY COLUMN sort_order TINYINT UNSIGNED NOT NULL;

ALTER TABLE class_emblems
    ADD COLUMN sort_order TINYINT UNSIGNED NULL AFTER name;

UPDATE class_emblems SET sort_order = 0 WHERE name = '1';
UPDATE class_emblems SET sort_order = 1 WHERE name = '2';
UPDATE class_emblems SET sort_order = 2 WHERE name = '3';
UPDATE class_emblems SET sort_order = 3 WHERE name = '4';
UPDATE class_emblems SET sort_order = 4 WHERE name = '5';
UPDATE class_emblems SET sort_order = 5 WHERE name = 'inf';

ALTER TABLE class_emblems
    MODIFY COLUMN sort_order TINYINT UNSIGNED NOT NULL;

ALTER TABLE class_emblem_bases
    ADD COLUMN sort_order TINYINT UNSIGNED NULL AFTER name;

UPDATE class_emblem_bases SET sort_order = 0 WHERE name = '1';
UPDATE class_emblem_bases SET sort_order = 1 WHERE name = '2';
UPDATE class_emblem_bases SET sort_order = 2 WHERE name = '3';
UPDATE class_emblem_bases SET sort_order = 3 WHERE name = '4';
UPDATE class_emblem_bases SET sort_order = 4 WHERE name = '5';
UPDATE class_emblem_bases SET sort_order = 5 WHERE name = 'inf';

ALTER TABLE class_emblem_bases
    MODIFY COLUMN sort_order TINYINT UNSIGNED NOT NULL;

ALTER TABLE clear_lamp_types
    ADD COLUMN sort_order TINYINT UNSIGNED NULL AFTER name;

UPDATE clear_lamp_types SET sort_order = 0 WHERE name = 'FAILED';
UPDATE clear_lamp_types SET sort_order = 1 WHERE name = 'CLEAR';
UPDATE clear_lamp_types SET sort_order = 2 WHERE name = 'HARD';
UPDATE clear_lamp_types SET sort_order = 3 WHERE name = 'BRAVE';
UPDATE clear_lamp_types SET sort_order = 4 WHERE name = 'ABSOLUTE';
UPDATE clear_lamp_types SET sort_order = 5 WHERE name = 'CATASTROPHY';

ALTER TABLE clear_lamp_types
    MODIFY COLUMN sort_order TINYINT UNSIGNED NOT NULL;

ALTER TABLE combo_lamp_types
    ADD COLUMN sort_order TINYINT UNSIGNED NULL AFTER name;

UPDATE combo_lamp_types SET sort_order = 0 WHERE name = 'NONE';
UPDATE combo_lamp_types SET sort_order = 1 WHERE name = 'FULL COMBO';
UPDATE combo_lamp_types SET sort_order = 2 WHERE name = 'ALL JUSTICE';

ALTER TABLE combo_lamp_types
    MODIFY COLUMN sort_order TINYINT UNSIGNED NOT NULL;

ALTER TABLE full_chain_types
    ADD COLUMN sort_order TINYINT UNSIGNED NULL AFTER name;

UPDATE full_chain_types SET sort_order = 0 WHERE name = 'NONE';
UPDATE full_chain_types SET sort_order = 1 WHERE name = 'FULL CHAIN GOLD';
UPDATE full_chain_types SET sort_order = 2 WHERE name = 'FULL CHAIN PLATINUM';

ALTER TABLE full_chain_types
    MODIFY COLUMN sort_order TINYINT UNSIGNED NOT NULL;
