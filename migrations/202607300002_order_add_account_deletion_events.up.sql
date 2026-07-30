CREATE TABLE `order_account_deletion_inbox` (
  `request_id` char(36) NOT NULL,
  `user_id` int NOT NULL,
  `decision` varchar(16) NOT NULL,
  `reason` varchar(255) NOT NULL DEFAULT '',
  `created_at` datetime(3) NOT NULL,
  PRIMARY KEY (`request_id`),
  KEY `idx_order_account_deletion_inbox_user` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `order_account_deletion_outbox` (
  `id` char(36) NOT NULL,
  `request_id` char(36) NOT NULL,
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
  UNIQUE KEY `uk_order_account_deletion_outbox_request` (`request_id`),
  KEY `idx_order_account_deletion_outbox_claim` (`status`, `available_at`, `user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
