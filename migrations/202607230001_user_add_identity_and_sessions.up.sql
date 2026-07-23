ALTER TABLE `user`
  ADD COLUMN `mobile_verified` tinyint(1) NOT NULL DEFAULT 0 AFTER `mobile`,
  ADD COLUMN `email_verified` tinyint(1) NOT NULL DEFAULT 0 AFTER `email`,
  ADD COLUMN `last_login_at` datetime(3) NULL AFTER `account_status`;

CREATE TABLE `user_sessions` (
  `id` char(36) NOT NULL,
  `user_id` int NOT NULL,
  `refresh_token_hash` binary(32) NOT NULL,
  `device_id` varchar(128) NOT NULL,
  `device_name` varchar(128) NOT NULL DEFAULT '',
  `created_at` datetime(3) NOT NULL,
  `last_used_at` datetime(3) NOT NULL,
  `expires_at` datetime(3) NOT NULL,
  `revoked_at` datetime(3) NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_sessions_refresh_hash` (`refresh_token_hash`),
  KEY `idx_user_sessions_user_active` (`user_id`, `revoked_at`, `expires_at`),
  KEY `idx_user_sessions_device` (`user_id`, `device_id`),
  CONSTRAINT `fk_user_sessions_user` FOREIGN KEY (`user_id`) REFERENCES `user` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `verification_codes` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `channel` varchar(16) NOT NULL,
  `purpose` varchar(16) NOT NULL,
  `destination_hash` binary(32) NOT NULL,
  `code_hash` binary(32) NOT NULL,
  `attempts` int unsigned NOT NULL DEFAULT 0,
  `expires_at` datetime(3) NOT NULL,
  `consumed_at` datetime(3) NULL,
  `created_at` datetime(3) NOT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_verification_lookup` (`channel`, `purpose`, `destination_hash`, `consumed_at`, `expires_at`),
  KEY `idx_verification_created` (`destination_hash`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
