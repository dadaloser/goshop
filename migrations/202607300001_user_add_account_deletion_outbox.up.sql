CREATE TABLE `user_account_deletion_outbox` (
  `id` char(36) NOT NULL,
  `event_type` varchar(64) NOT NULL,
  `user_id` int NOT NULL,
  `payload` json NOT NULL,
  `status` varchar(16) NOT NULL DEFAULT 'PENDING',
  `retry_count` int NOT NULL DEFAULT 0,
  `available_at` datetime(3) NOT NULL,
  `locked_at` datetime(3) NULL,
  `published_at` datetime(3) NULL,
  `last_error` varchar(500) NOT NULL DEFAULT '',
  `created_at` datetime(3) NOT NULL,
  `updated_at` datetime(3) NOT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_user_deletion_outbox_claim` (`status`, `available_at`, `user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
