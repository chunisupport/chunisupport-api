-- 楽曲データ収集バッチ（song-batch）の実行履歴を保持します。
-- CLI と管理画面のどちらから起動した実行も記録し、管理画面で実行中・直近の結果を確認できるようにします。
CREATE TABLE song_batch_jobs (
    id BINARY(16) NOT NULL,
    mode VARCHAR(32) NOT NULL,
    fill_missing_release_date BOOLEAN NOT NULL,
    trigger_type VARCHAR(16) NOT NULL,
    status VARCHAR(32) NOT NULL,
    requested_by_user_id INT UNSIGNED NULL,
    started_at DATETIME(6) NOT NULL,
    finished_at DATETIME(6) NULL,
    warning_count INT UNSIGNED NOT NULL DEFAULT 0,
    error_message VARCHAR(1000) NULL,
    PRIMARY KEY (id),
    CONSTRAINT fk_song_batch_jobs_requested_by_user
        FOREIGN KEY (requested_by_user_id)
        REFERENCES users (id)
        ON DELETE SET NULL,
    INDEX idx_song_batch_jobs_started_at (started_at),
    INDEX idx_song_batch_jobs_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
