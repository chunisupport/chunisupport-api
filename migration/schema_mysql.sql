CREATE TABLE `account_types` (
  `id` tinyint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(15) COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `achievement_types` (
  `id` tinyint unsigned NOT NULL AUTO_INCREMENT,
  `code` varchar(30) COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uq_achievement_types_code` (`code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `api_tokens` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` int unsigned NOT NULL,
  `name` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '既存のトークン',
  `hashed_token` char(64) COLLATE utf8mb4_unicode_ci NOT NULL,
  `token_prefix` char(5) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `last_used_at` datetime DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uq_api_tokens_hashed_token` (`hashed_token`),
  UNIQUE KEY `uq_api_tokens_user_name` (`user_id`,`name`),
  KEY `idx_api_tokens_user_created_id` (`user_id`,`created_at` DESC,`id` DESC),
  CONSTRAINT `api_tokens_ibfk_1` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `chart_best_slot_stats_by_rating_band` (
  `chart_id` mediumint unsigned NOT NULL,
  `rating_band_id` tinyint unsigned NOT NULL,
  `best_player_count` int unsigned NOT NULL,
  `eligible_player_count` int unsigned NOT NULL,
  `best_player_percentage` decimal(7,4) DEFAULT NULL,
  PRIMARY KEY (`chart_id`,`rating_band_id`),
  KEY `idx_chart_best_slot_stats_ranking` (`rating_band_id`,`best_player_percentage` DESC,`best_player_count` DESC,`chart_id`),
  CONSTRAINT `chart_best_slot_stats_by_rating_band_ibfk_1` FOREIGN KEY (`chart_id`) REFERENCES `charts` (`id`) ON DELETE CASCADE,
  CONSTRAINT `chart_best_slot_stats_by_rating_band_ibfk_2` FOREIGN KEY (`rating_band_id`) REFERENCES `rating_bands` (`id`),
  CONSTRAINT `chart_best_slot_stats_by_rating_band_chk_1` CHECK ((`best_player_count` <= `eligible_player_count`)),
  CONSTRAINT `chart_best_slot_stats_by_rating_band_chk_2` CHECK (((`best_player_percentage` is null) or (`best_player_percentage` between 0 and 100))),
  CONSTRAINT `chart_best_slot_stats_by_rating_band_chk_3` CHECK ((((`eligible_player_count` = 0) and (`best_player_percentage` is null)) or ((`eligible_player_count` > 0) and (`best_player_percentage` is not null))))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `chart_stats_by_rating_band` (
  `chart_id` mediumint unsigned NOT NULL,
  `rating_band_id` tinyint unsigned NOT NULL,
  `rank_aaal` int unsigned NOT NULL DEFAULT '0',
  `rank_s` int unsigned NOT NULL DEFAULT '0',
  `rank_sp` int unsigned NOT NULL DEFAULT '0',
  `rank_ss` int unsigned NOT NULL DEFAULT '0',
  `rank_ssp` int unsigned NOT NULL DEFAULT '0',
  `rank_sss` int unsigned NOT NULL DEFAULT '0',
  `rank_sssp` int unsigned NOT NULL DEFAULT '0',
  `rank_max` int unsigned NOT NULL DEFAULT '0',
  `combo_none` int unsigned NOT NULL DEFAULT '0',
  `combo_fc` int unsigned NOT NULL DEFAULT '0',
  `combo_aj` int unsigned NOT NULL DEFAULT '0',
  `combo_ajc` int unsigned NOT NULL DEFAULT '0',
  `clear_failed` int unsigned NOT NULL DEFAULT '0',
  `clear_clear` int unsigned NOT NULL DEFAULT '0',
  `clear_hard` int unsigned NOT NULL DEFAULT '0',
  `clear_brave` int unsigned NOT NULL DEFAULT '0',
  `clear_absolute` int unsigned NOT NULL DEFAULT '0',
  `clear_catastrophy` int unsigned NOT NULL DEFAULT '0',
  `average_score` double DEFAULT NULL,
  `median_score` double DEFAULT NULL,
  `player_count` int unsigned NOT NULL DEFAULT '0',
  PRIMARY KEY (`chart_id`,`rating_band_id`),
  KEY `rating_band_id` (`rating_band_id`),
  CONSTRAINT `chart_stats_by_rating_band_ibfk_1` FOREIGN KEY (`chart_id`) REFERENCES `charts` (`id`) ON DELETE CASCADE,
  CONSTRAINT `chart_stats_by_rating_band_ibfk_2` FOREIGN KEY (`rating_band_id`) REFERENCES `rating_bands` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `charts` (
  `id` mediumint unsigned NOT NULL AUTO_INCREMENT,
  `song_id` int unsigned NOT NULL,
  `difficulty_id` tinyint unsigned NOT NULL,
  `const` decimal(3,1) NOT NULL,
  `is_const_unknown` tinyint(1) NOT NULL DEFAULT '1',
  `notes` int DEFAULT NULL,
  `notes_designer` varchar(100) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `unique_song_difficulty` (`song_id`,`difficulty_id`),
  KEY `difficulty_id` (`difficulty_id`),
  CONSTRAINT `charts_ibfk_1` FOREIGN KEY (`song_id`) REFERENCES `songs` (`id`) ON DELETE CASCADE,
  CONSTRAINT `charts_ibfk_2` FOREIGN KEY (`difficulty_id`) REFERENCES `difficulties` (`id`),
  CONSTRAINT `charts_chk_1` CHECK ((`const` >= 0)),
  CONSTRAINT `charts_chk_2` CHECK (((`notes` is null) or (`notes` >= 0)))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `class_emblem_bases` (
  `id` tinyint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(10) COLLATE utf8mb4_unicode_ci NOT NULL,
  `sort_order` tinyint unsigned NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `class_emblems` (
  `id` tinyint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(10) COLLATE utf8mb4_unicode_ci NOT NULL,
  `sort_order` tinyint unsigned NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `clear_lamp_types` (
  `id` tinyint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(20) COLLATE utf8mb4_unicode_ci NOT NULL,
  `sort_order` tinyint unsigned NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `combo_lamp_types` (
  `id` tinyint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(20) COLLATE utf8mb4_unicode_ci NOT NULL,
  `sort_order` tinyint unsigned NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `course_classes` (
  `id` tinyint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL,
  `sort_order` tinyint unsigned NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uq_course_classes_name` (`name`),
  UNIQUE KEY `uq_course_classes_sort_order` (`sort_order`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `courses` (
  `id` mediumint unsigned NOT NULL AUTO_INCREMENT,
  `display_id` char(16) COLLATE utf8mb4_unicode_ci NOT NULL,
  `official_idx` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `course_class_id` tinyint unsigned NOT NULL,
  `is_deleted` tinyint(1) NOT NULL DEFAULT '0',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uq_courses_official_idx` (`official_idx`),
  UNIQUE KEY `uq_courses_display_id` (`display_id`),
  KEY `idx_courses_class_deleted_idx` (`course_class_id`,`is_deleted`,`official_idx`),
  CONSTRAINT `fk_courses_course_class` FOREIGN KEY (`course_class_id`) REFERENCES `course_classes` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `difficulties` (
  `id` tinyint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(15) COLLATE utf8mb4_unicode_ci NOT NULL,
  `sort_order` tinyint unsigned NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `friendship_statuses` (
  `id` tinyint unsigned NOT NULL,
  `name` varchar(20) COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `friendships` (
  `user_id` int unsigned NOT NULL,
  `friend_user_id` int unsigned NOT NULL,
  `status_id` tinyint unsigned NOT NULL,
  `requested_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `accepted_at` timestamp NULL DEFAULT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`user_id`,`friend_user_id`),
  KEY `idx_friendships_friend_user_status` (`friend_user_id`,`status_id`,`requested_at`),
  KEY `idx_friendships_user_status` (`user_id`,`status_id`,`accepted_at`),
  KEY `fk_friendships_status_id` (`status_id`),
  CONSTRAINT `fk_friendships_friend_user_id` FOREIGN KEY (`friend_user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_friendships_status_id` FOREIGN KEY (`status_id`) REFERENCES `friendship_statuses` (`id`),
  CONSTRAINT `fk_friendships_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE,
  CONSTRAINT `chk_friendships_accepted_at` CHECK ((((`status_id` = 2) and (`accepted_at` is not null)) or ((`status_id` <> 2) and (`accepted_at` is null)))),
  CONSTRAINT `chk_friendships_not_self` CHECK ((`user_id` <> `friend_user_id`))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `full_chain_types` (
  `id` tinyint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(25) COLLATE utf8mb4_unicode_ci NOT NULL,
  `sort_order` tinyint unsigned NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `genres` (
  `id` tinyint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(30) COLLATE utf8mb4_unicode_ci NOT NULL,
  `sort_order` tinyint unsigned NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `goal_groups` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `user_id` int unsigned NOT NULL,
  `name` varchar(30) COLLATE utf8mb4_unicode_ci NOT NULL,
  `sort_order` smallint unsigned NOT NULL,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uq_goal_groups_user_name` (`user_id`,`name`),
  UNIQUE KEY `uq_goal_groups_user_id` (`user_id`,`id`),
  KEY `idx_goal_groups_user_sort_order_id` (`user_id`,`sort_order`,`id`),
  CONSTRAINT `fk_goal_groups_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `goals` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `user_id` int unsigned NOT NULL,
  `group_id` int unsigned DEFAULT NULL,
  `title` varchar(30) COLLATE utf8mb4_unicode_ci NOT NULL,
  `achievement_type_id` tinyint unsigned NOT NULL,
  `achievement_params` json NOT NULL,
  `attributes` json NOT NULL,
  `invert_value` tinyint(1) NOT NULL DEFAULT '0',
  `invert_percentage` tinyint(1) NOT NULL DEFAULT '0',
  `sort_order` smallint unsigned NOT NULL,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `fk_goals_achievement_type_id` (`achievement_type_id`),
  KEY `idx_goals_user_group_sort_order_id` (`user_id`,`group_id`,`sort_order`,`id`),
  CONSTRAINT `fk_goals_achievement_type_id` FOREIGN KEY (`achievement_type_id`) REFERENCES `achievement_types` (`id`) ON DELETE RESTRICT,
  CONSTRAINT `fk_goals_group_user` FOREIGN KEY (`user_id`, `group_id`) REFERENCES `goal_groups` (`user_id`, `id`) ON DELETE RESTRICT,
  CONSTRAINT `fk_goals_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `honor_types` (
  `id` tinyint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(20) COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `honors` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(500) COLLATE utf8mb4_unicode_ci NOT NULL,
  `honor_type_id` tinyint unsigned NOT NULL,
  `image_url` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `unique_honor_name_type` (`name`,`honor_type_id`),
  UNIQUE KEY `unique_honor_image_url` (`image_url`),
  KEY `honor_type_id` (`honor_type_id`),
  CONSTRAINT `honors_ibfk_1` FOREIGN KEY (`honor_type_id`) REFERENCES `honor_types` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `player_course_records` (
  `player_id` mediumint unsigned NOT NULL,
  `course_id` mediumint unsigned NOT NULL,
  `score` mediumint unsigned NOT NULL,
  `is_clear` tinyint(1) NOT NULL,
  `combo_lamp_id` tinyint unsigned NOT NULL,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`player_id`,`course_id`),
  KEY `idx_player_course_records_course_id` (`course_id`),
  KEY `idx_player_course_records_player_updated_at` (`player_id`,`updated_at` DESC),
  KEY `fk_player_course_records_combo_lamp` (`combo_lamp_id`),
  CONSTRAINT `fk_player_course_records_combo_lamp` FOREIGN KEY (`combo_lamp_id`) REFERENCES `combo_lamp_types` (`id`),
  CONSTRAINT `fk_player_course_records_course` FOREIGN KEY (`course_id`) REFERENCES `courses` (`id`),
  CONSTRAINT `fk_player_course_records_player` FOREIGN KEY (`player_id`) REFERENCES `players` (`id`) ON DELETE CASCADE,
  CONSTRAINT `chk_player_course_records_score` CHECK ((`score` between 0 and 3030000))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `player_favorite_songs` (
  `player_id` mediumint unsigned NOT NULL,
  `song_id` int unsigned NOT NULL,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`player_id`,`song_id`),
  KEY `fk_player_favorite_songs_song_id` (`song_id`),
  CONSTRAINT `fk_player_favorite_songs_player_id` FOREIGN KEY (`player_id`) REFERENCES `players` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_player_favorite_songs_song_id` FOREIGN KEY (`song_id`) REFERENCES `songs` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `player_honors` (
  `player_id` mediumint unsigned NOT NULL,
  `honor_id` int unsigned NOT NULL,
  `slot` tinyint NOT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`player_id`,`slot`),
  KEY `honor_id` (`honor_id`),
  CONSTRAINT `player_honors_ibfk_1` FOREIGN KEY (`player_id`) REFERENCES `players` (`id`) ON DELETE CASCADE,
  CONSTRAINT `player_honors_ibfk_2` FOREIGN KEY (`honor_id`) REFERENCES `honors` (`id`),
  CONSTRAINT `player_honors_chk_1` CHECK ((`slot` between 1 and 3))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `player_latest_updates` (
  `player_id` mediumint unsigned NOT NULL,
  `schema_version` int unsigned NOT NULL,
  `result_gzip` mediumblob NOT NULL,
  `source_updated_at` datetime(6) NOT NULL,
  `imported_at` datetime(6) NOT NULL,
  `body_hash` char(64) COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`player_id`),
  CONSTRAINT `fk_player_latest_updates_player` FOREIGN KEY (`player_id`) REFERENCES `players` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `player_locked_songs` (
  `player_id` mediumint unsigned NOT NULL,
  `song_id` int unsigned NOT NULL,
  `is_ultima` tinyint(1) NOT NULL,
  PRIMARY KEY (`player_id`,`song_id`,`is_ultima`),
  KEY `fk_player_locked_songs_song_id` (`song_id`),
  CONSTRAINT `fk_player_locked_songs_player_id` FOREIGN KEY (`player_id`) REFERENCES `players` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_player_locked_songs_song_id` FOREIGN KEY (`song_id`) REFERENCES `songs` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `player_record_histories` (
  `player_id` mediumint unsigned NOT NULL,
  `chart_id` mediumint unsigned NOT NULL,
  `score` mediumint unsigned NOT NULL,
  `clear_lamp_id` tinyint unsigned NOT NULL,
  `combo_lamp_id` tinyint unsigned NOT NULL,
  `full_chain_id` tinyint unsigned NOT NULL,
  `updated_at` timestamp NOT NULL,
  PRIMARY KEY (`player_id`,`chart_id`,`updated_at`),
  KEY `fk_player_record_histories_chart` (`chart_id`),
  CONSTRAINT `fk_player_record_histories_chart` FOREIGN KEY (`chart_id`) REFERENCES `charts` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_player_record_histories_player` FOREIGN KEY (`player_id`) REFERENCES `players` (`id`) ON DELETE CASCADE,
  CONSTRAINT `chk_player_record_histories_score` CHECK ((`score` between 0 and 1010000))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `player_records` (
  `player_id` mediumint unsigned NOT NULL,
  `chart_id` mediumint unsigned NOT NULL,
  `score` mediumint unsigned NOT NULL,
  `clear_lamp_id` tinyint unsigned NOT NULL DEFAULT '1',
  `combo_lamp_id` tinyint unsigned NOT NULL DEFAULT '1',
  `full_chain_id` tinyint unsigned NOT NULL DEFAULT '1',
  `slot_id` tinyint unsigned NOT NULL,
  `slot_order` tinyint unsigned DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`player_id`,`chart_id`),
  KEY `clear_lamp_id` (`clear_lamp_id`),
  KEY `combo_lamp_id` (`combo_lamp_id`),
  KEY `full_chain_id` (`full_chain_id`),
  KEY `slot_id` (`slot_id`),
  KEY `idx_player_records_chart_id` (`chart_id`),
  KEY `idx_player_records_player_updated_at` (`player_id`,`updated_at` DESC),
  CONSTRAINT `player_records_ibfk_1` FOREIGN KEY (`player_id`) REFERENCES `players` (`id`) ON DELETE CASCADE,
  CONSTRAINT `player_records_ibfk_2` FOREIGN KEY (`chart_id`) REFERENCES `charts` (`id`) ON DELETE CASCADE,
  CONSTRAINT `player_records_ibfk_3` FOREIGN KEY (`clear_lamp_id`) REFERENCES `clear_lamp_types` (`id`),
  CONSTRAINT `player_records_ibfk_4` FOREIGN KEY (`combo_lamp_id`) REFERENCES `combo_lamp_types` (`id`),
  CONSTRAINT `player_records_ibfk_5` FOREIGN KEY (`full_chain_id`) REFERENCES `full_chain_types` (`id`),
  CONSTRAINT `player_records_ibfk_6` FOREIGN KEY (`slot_id`) REFERENCES `slots` (`id`),
  CONSTRAINT `player_records_chk_1` CHECK ((`score` between 0 and 1010000)),
  CONSTRAINT `player_records_chk_2` CHECK (((`slot_order` is null) or (`slot_order` between 1 and 255)))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `player_worldsend_record_histories` (
  `player_id` mediumint unsigned NOT NULL,
  `worldsend_chart_id` mediumint unsigned NOT NULL,
  `score` mediumint unsigned NOT NULL,
  `clear_lamp_id` tinyint unsigned NOT NULL,
  `combo_lamp_id` tinyint unsigned NOT NULL,
  `full_chain_id` tinyint unsigned NOT NULL,
  `updated_at` timestamp NOT NULL,
  PRIMARY KEY (`player_id`,`worldsend_chart_id`,`updated_at`),
  KEY `fk_player_worldsend_record_histories_chart` (`worldsend_chart_id`),
  CONSTRAINT `fk_player_worldsend_record_histories_chart` FOREIGN KEY (`worldsend_chart_id`) REFERENCES `worldsend_charts` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_player_worldsend_record_histories_player` FOREIGN KEY (`player_id`) REFERENCES `players` (`id`) ON DELETE CASCADE,
  CONSTRAINT `chk_player_worldsend_record_histories_score` CHECK ((`score` between 0 and 1010000))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `player_worldsend_records` (
  `player_id` mediumint unsigned NOT NULL,
  `worldsend_chart_id` mediumint unsigned NOT NULL,
  `score` mediumint unsigned NOT NULL,
  `clear_lamp_id` tinyint unsigned NOT NULL DEFAULT '1',
  `combo_lamp_id` tinyint unsigned NOT NULL DEFAULT '1',
  `full_chain_id` tinyint unsigned NOT NULL DEFAULT '1',
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`player_id`,`worldsend_chart_id`),
  KEY `clear_lamp_id` (`clear_lamp_id`),
  KEY `combo_lamp_id` (`combo_lamp_id`),
  KEY `full_chain_id` (`full_chain_id`),
  KEY `idx_player_worldsend_records_worldsend_chart_id` (`worldsend_chart_id`),
  KEY `idx_player_worldsend_records_player_updated_at` (`player_id`,`updated_at` DESC),
  CONSTRAINT `player_worldsend_records_ibfk_1` FOREIGN KEY (`player_id`) REFERENCES `players` (`id`) ON DELETE CASCADE,
  CONSTRAINT `player_worldsend_records_ibfk_2` FOREIGN KEY (`worldsend_chart_id`) REFERENCES `worldsend_charts` (`id`) ON DELETE CASCADE,
  CONSTRAINT `player_worldsend_records_ibfk_3` FOREIGN KEY (`clear_lamp_id`) REFERENCES `clear_lamp_types` (`id`),
  CONSTRAINT `player_worldsend_records_ibfk_4` FOREIGN KEY (`combo_lamp_id`) REFERENCES `combo_lamp_types` (`id`),
  CONSTRAINT `player_worldsend_records_ibfk_5` FOREIGN KEY (`full_chain_id`) REFERENCES `full_chain_types` (`id`),
  CONSTRAINT `player_worldsend_records_chk_1` CHECK ((`score` between 0 and 1010000))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `players` (
  `id` mediumint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` int unsigned NOT NULL,
  `player_name` varchar(20) COLLATE utf8mb4_unicode_ci NOT NULL,
  `player_level` int NOT NULL,
  `official_player_rating` decimal(4,2) NOT NULL,
  `calculated_player_rating` decimal(6,4) DEFAULT NULL,
  `new_average_rating` decimal(6,4) DEFAULT NULL,
  `best_average_rating` decimal(6,4) DEFAULT NULL,
  `class_emblem_id` tinyint unsigned DEFAULT NULL,
  `class_emblem_base_id` tinyint unsigned DEFAULT NULL,
  `last_played_at` datetime DEFAULT NULL,
  `overpower_value` decimal(9,3) DEFAULT NULL,
  `official_overpower` decimal(8,2) NOT NULL DEFAULT '0.00',
  `data_collected_at` timestamp NULL DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uq_players_user_id` (`user_id`),
  KEY `class_emblem_id` (`class_emblem_id`),
  KEY `class_emblem_base_id` (`class_emblem_base_id`),
  KEY `idx_players_player_name` (`player_name`),
  CONSTRAINT `fk_players_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE,
  CONSTRAINT `players_ibfk_1` FOREIGN KEY (`class_emblem_id`) REFERENCES `class_emblems` (`id`),
  CONSTRAINT `players_ibfk_2` FOREIGN KEY (`class_emblem_base_id`) REFERENCES `class_emblem_bases` (`id`),
  CONSTRAINT `players_chk_1` CHECK ((`player_level` >= 1))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `player_metric_histories` (
  `player_id` mediumint unsigned NOT NULL,
  `official_rating` decimal(4,2) NOT NULL,
  `official_overpower` decimal(8,2) NOT NULL,
  `data_collected_at` timestamp NOT NULL,
  PRIMARY KEY (`player_id`,`data_collected_at`),
  CONSTRAINT `fk_player_metric_histories_player` FOREIGN KEY (`player_id`) REFERENCES `players` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `rating_bands` (
  `id` tinyint unsigned NOT NULL,
  `label` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL,
  `min_inclusive` decimal(6,4) DEFAULT NULL,
  `max_exclusive` decimal(6,4) DEFAULT NULL,
  `sort_order` tinyint unsigned NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uq_rating_bands_label` (`label`),
  UNIQUE KEY `uq_rating_bands_sort_order` (`sort_order`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `record_filters` (
  `id` binary(16) NOT NULL,
  `user_id` int unsigned NOT NULL,
  `name` varchar(30) COLLATE utf8mb4_unicode_ci NOT NULL,
  `filter_value_gzip` blob NOT NULL,
  `is_worldsend` tinyint(1) NOT NULL DEFAULT '0',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_record_filters_user_updated_id` (`user_id`,`updated_at` DESC,`id`),
  CONSTRAINT `fk_record_filters_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `schema_migrations` (
  `version` bigint NOT NULL,
  `dirty` tinyint(1) NOT NULL,
  PRIMARY KEY (`version`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `slots` (
  `id` tinyint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(25) COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `songs` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `display_id` char(16) COLLATE utf8mb4_unicode_ci NOT NULL,
  `title` varchar(300) COLLATE utf8mb4_unicode_ci NOT NULL,
  `reading` varchar(300) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `artist` varchar(300) COLLATE utf8mb4_unicode_ci NOT NULL,
  `genre_id` tinyint unsigned NOT NULL,
  `bpm` int DEFAULT NULL,
  `released_at` date DEFAULT NULL,
  `official_idx` varchar(10) COLLATE utf8mb4_unicode_ci NOT NULL,
  `jacket` varchar(20) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `is_worldsend` tinyint(1) NOT NULL DEFAULT '0',
  `is_new` tinyint(1) NOT NULL DEFAULT '0',
  `is_deleted` tinyint(1) NOT NULL DEFAULT '0',
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `display_id` (`display_id`),
  UNIQUE KEY `official_idx` (`official_idx`),
  KEY `genre_id` (`genre_id`),
  KEY `idx_songs_worldsend_deleted` (`is_worldsend`,`is_deleted`),
  CONSTRAINT `songs_ibfk_1` FOREIGN KEY (`genre_id`) REFERENCES `genres` (`id`),
  CONSTRAINT `songs_chk_1` CHECK (((`bpm` is null) or (`bpm` > 0)))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `system_maintenance` (
  `id` tinyint unsigned NOT NULL,
  `enabled` tinyint(1) NOT NULL,
  `comment` varchar(1000) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `updated_by_user_id` int unsigned DEFAULT NULL,
  `updated_at` datetime(6) NOT NULL,
  PRIMARY KEY (`id`),
  KEY `fk_system_maintenance_updated_by_user` (`updated_by_user_id`),
  CONSTRAINT `fk_system_maintenance_updated_by_user` FOREIGN KEY (`updated_by_user_id`) REFERENCES `users` (`id`) ON DELETE SET NULL,
  CONSTRAINT `chk_system_maintenance_singleton` CHECK ((`id` = 1))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `users` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `username` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL,
  `firebase_uid` varchar(128) CHARACTER SET ascii COLLATE ascii_bin DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `account_type_id` tinyint unsigned NOT NULL DEFAULT '1',
  `player_id` mediumint unsigned DEFAULT NULL,
  `is_private` tinyint(1) NOT NULL DEFAULT '0',
  `is_suspicious` tinyint(1) NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`),
  UNIQUE KEY `username` (`username`),
  UNIQUE KEY `uq_users_player_id` (`player_id`),
  UNIQUE KEY `uk_users_firebase_uid` (`firebase_uid`),
  KEY `account_type_id` (`account_type_id`),
  KEY `idx_users_private` (`is_private`,`player_id`),
  CONSTRAINT `fk_users_player_id` FOREIGN KEY (`player_id`) REFERENCES `players` (`id`) ON DELETE SET NULL,
  CONSTRAINT `users_ibfk_1` FOREIGN KEY (`account_type_id`) REFERENCES `account_types` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `versions` (
  `id` tinyint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL,
  `released_at` date NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `worldsend_chart_stats_by_rating_band` (
  `worldsend_chart_id` mediumint unsigned NOT NULL,
  `rating_band_id` tinyint unsigned NOT NULL,
  `rank_aaal` int unsigned NOT NULL DEFAULT '0',
  `rank_s` int unsigned NOT NULL DEFAULT '0',
  `rank_sp` int unsigned NOT NULL DEFAULT '0',
  `rank_ss` int unsigned NOT NULL DEFAULT '0',
  `rank_ssp` int unsigned NOT NULL DEFAULT '0',
  `rank_sss` int unsigned NOT NULL DEFAULT '0',
  `rank_sssp` int unsigned NOT NULL DEFAULT '0',
  `rank_max` int unsigned NOT NULL DEFAULT '0',
  `combo_none` int unsigned NOT NULL DEFAULT '0',
  `combo_fc` int unsigned NOT NULL DEFAULT '0',
  `combo_aj` int unsigned NOT NULL DEFAULT '0',
  `combo_ajc` int unsigned NOT NULL DEFAULT '0',
  `clear_failed` int unsigned NOT NULL DEFAULT '0',
  `clear_clear` int unsigned NOT NULL DEFAULT '0',
  `clear_hard` int unsigned NOT NULL DEFAULT '0',
  `clear_brave` int unsigned NOT NULL DEFAULT '0',
  `clear_absolute` int unsigned NOT NULL DEFAULT '0',
  `clear_catastrophy` int unsigned NOT NULL DEFAULT '0',
  `average_score` double DEFAULT NULL,
  `median_score` double DEFAULT NULL,
  `player_count` int unsigned NOT NULL DEFAULT '0',
  PRIMARY KEY (`worldsend_chart_id`,`rating_band_id`),
  KEY `rating_band_id` (`rating_band_id`),
  CONSTRAINT `worldsend_chart_stats_by_rating_band_ibfk_1` FOREIGN KEY (`worldsend_chart_id`) REFERENCES `worldsend_charts` (`id`) ON DELETE CASCADE,
  CONSTRAINT `worldsend_chart_stats_by_rating_band_ibfk_2` FOREIGN KEY (`rating_band_id`) REFERENCES `rating_bands` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `worldsend_charts` (
  `id` mediumint unsigned NOT NULL AUTO_INCREMENT,
  `song_id` int unsigned NOT NULL,
  `level_star` tinyint DEFAULT NULL,
  `attribute` char(1) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `notes` int DEFAULT NULL,
  `notes_designer` varchar(100) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `song_id` (`song_id`),
  CONSTRAINT `worldsend_charts_ibfk_1` FOREIGN KEY (`song_id`) REFERENCES `songs` (`id`) ON DELETE CASCADE,
  CONSTRAINT `worldsend_charts_chk_1` CHECK (((`level_star` is null) or (`level_star` between 1 and 5))),
  CONSTRAINT `worldsend_charts_chk_2` CHECK (((`notes` is null) or (`notes` >= 0)))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
