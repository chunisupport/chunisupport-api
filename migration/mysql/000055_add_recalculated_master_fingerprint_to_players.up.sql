-- プレイヤーデータ再計算バッチが直近の再計算に使ったマスタと計算ロジックのフィンガープリント（SHA-256の16進数小文字）を保持します。
-- 値が現在のフィンガープリントと一致するプレイヤーは計算結果が変わらないため、バッチの処理対象から除外します。
-- NULLは次回のバッチで再計算が必要であることを表し、既存行もNULLのため適用後の初回実行では全プレイヤーを再計算します。
ALTER TABLE players
    ADD COLUMN recalculated_master_fingerprint CHAR(64) CHARACTER SET ascii COLLATE ascii_bin NULL;
