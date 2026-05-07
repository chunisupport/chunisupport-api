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
  `hashed_token` char(64) COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uq_api_tokens_user_id` (`user_id`),
  UNIQUE KEY `uq_api_tokens_hashed_token` (`hashed_token`),
  CONSTRAINT `api_tokens_ibfk_1` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
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
CREATE TABLE `difficulties` (
  `id` tinyint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(15) COLLATE utf8mb4_unicode_ci NOT NULL,
  `sort_order` tinyint unsigned NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `name` (`name`)
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
  PRIMARY KEY (`id`),
  UNIQUE KEY `name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `goals` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `user_id` int unsigned NOT NULL,
  `title` varchar(30) COLLATE utf8mb4_unicode_ci NOT NULL,
  `achievement_type_id` tinyint unsigned NOT NULL,
  `achievement_params` json NOT NULL,
  `attributes` json NOT NULL,
  `invert` tinyint(1) NOT NULL DEFAULT '0',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `fk_goals_achievement_type_id` (`achievement_type_id`),
  KEY `idx_goals_user_created_id` (`user_id`,`created_at`,`id`),
  CONSTRAINT `fk_goals_achievement_type_id` FOREIGN KEY (`achievement_type_id`) REFERENCES `achievement_types` (`id`) ON DELETE RESTRICT,
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
  KEY `honor_type_id` (`honor_type_id`),
  CONSTRAINT `honors_ibfk_1` FOREIGN KEY (`honor_type_id`) REFERENCES `honor_types` (`id`)
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
CREATE TABLE `player_locked_songs` (
  `player_id` mediumint unsigned NOT NULL,
  `song_id` int unsigned NOT NULL,
  `is_ultima` tinyint(1) NOT NULL,
  PRIMARY KEY (`player_id`,`song_id`,`is_ultima`),
  KEY `fk_player_locked_songs_song_id` (`song_id`),
  CONSTRAINT `fk_player_locked_songs_player_id` FOREIGN KEY (`player_id`) REFERENCES `players` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_player_locked_songs_song_id` FOREIGN KEY (`song_id`) REFERENCES `songs` (`id`) ON DELETE CASCADE
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
  `official_player_rating` decimal(4,2) DEFAULT NULL,
  `calculated_player_rating` decimal(6,4) DEFAULT NULL,
  `new_average_rating` decimal(6,4) DEFAULT NULL,
  `best_average_rating` decimal(6,4) DEFAULT NULL,
  `class_emblem_id` tinyint unsigned DEFAULT NULL,
  `class_emblem_base_id` tinyint unsigned DEFAULT NULL,
  `last_played_at` datetime DEFAULT NULL,
  `overpower_value` decimal(9,3) DEFAULT NULL,
  `overpower_percentage` decimal(7,4) DEFAULT NULL,
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
  `artist` varchar(300) COLLATE utf8mb4_unicode_ci NOT NULL,
  `genre_id` tinyint unsigned NOT NULL,
  `bpm` int DEFAULT NULL,
  `released_at` date DEFAULT NULL,
  `official_idx` varchar(10) COLLATE utf8mb4_unicode_ci NOT NULL,
  `jacket` varchar(20) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `is_worldsend` tinyint(1) NOT NULL DEFAULT '0',
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
