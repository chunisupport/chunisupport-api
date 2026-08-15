-- EXTDEVユーザーをPLAYERへ移行
UPDATE users SET account_type_id = 1 WHERE account_type_id = 4;

-- EXTDEVロールを削除
DELETE FROM account_types WHERE id = 4;
