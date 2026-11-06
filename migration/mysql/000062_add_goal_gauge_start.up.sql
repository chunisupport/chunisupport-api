-- ゲージのスタート位置は表示専用の値です。0は従来どおり0からゲージを伸ばします。
-- DECIMAL(16,3) は total_score の合計値と OVER POWER の小数第3位を同じ列で扱うための桁数です。
ALTER TABLE goals
    ADD COLUMN gauge_start DECIMAL(16,3) NOT NULL DEFAULT 0 AFTER invert_percentage,
    ADD COLUMN apply_gauge_start_to_percentage BOOLEAN NOT NULL DEFAULT FALSE AFTER gauge_start;
