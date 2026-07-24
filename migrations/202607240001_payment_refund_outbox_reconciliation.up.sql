CREATE TABLE IF NOT EXISTS `order_refund_requests` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `order_sn` varchar(64) NOT NULL,
  `actor_user_id` int NOT NULL,
  `amount_fen` bigint NOT NULL,
  `reason` varchar(255) NOT NULL,
  `status` varchar(24) NOT NULL,
  `correlation_id` char(36) NOT NULL,
  `created_at` datetime(3) NOT NULL,
  `updated_at` datetime(3) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_order_refund_correlation` (`correlation_id`),
  KEY `idx_order_refund_order` (`order_sn`, `status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

ALTER TABLE `order_refund_requests`
  ADD COLUMN `provider` varchar(32) NOT NULL DEFAULT '' AFTER `status`,
  ADD COLUMN `provider_refund_id` varchar(128) NOT NULL DEFAULT '' AFTER `provider`,
  ADD COLUMN `failure_reason` varchar(255) NOT NULL DEFAULT '' AFTER `provider_refund_id`,
  ADD KEY `idx_order_refund_provider_id` (`provider`, `provider_refund_id`);

CREATE TABLE `order_refund_outbox` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `refund_request_id` bigint unsigned NOT NULL,
  `status` varchar(24) NOT NULL DEFAULT 'pending',
  `attempts` int NOT NULL DEFAULT 0,
  `available_at` datetime(3) NOT NULL,
  `locked_at` datetime(3) NULL,
  `last_error` varchar(255) NOT NULL DEFAULT '',
  `created_at` datetime(3) NOT NULL,
  `updated_at` datetime(3) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_refund_outbox_request` (`refund_request_id`),
  KEY `idx_refund_outbox_claim` (`status`, `available_at`, `id`),
  CONSTRAINT `fk_refund_outbox_request` FOREIGN KEY (`refund_request_id`) REFERENCES `order_refund_requests` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

ALTER TABLE `payment_reconciliation_runs`
  ADD COLUMN `provider` varchar(32) NOT NULL DEFAULT '' AFTER `id`,
  ADD COLUMN `window_start` datetime(3) NOT NULL DEFAULT '1970-01-01 00:00:00.000' AFTER `provider`,
  ADD COLUMN `window_end` datetime(3) NOT NULL DEFAULT '1970-01-01 00:00:00.000' AFTER `window_start`;

CREATE TABLE `payment_reconciliation_items` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `run_id` bigint unsigned NOT NULL,
  `provider_event_id` varchar(128) NOT NULL,
  `order_sn` varchar(64) NOT NULL,
  `trade_no` varchar(128) NOT NULL DEFAULT '',
  `event_type` varchar(32) NOT NULL,
  `provider_amount_fen` bigint NOT NULL,
  `local_amount_fen` bigint NOT NULL,
  `result` varchar(24) NOT NULL,
  `detail` varchar(255) NOT NULL DEFAULT '',
  `created_at` datetime(3) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_reconciliation_run_event` (`run_id`, `provider_event_id`),
  KEY `idx_reconciliation_result` (`run_id`, `result`),
  CONSTRAINT `fk_reconciliation_item_run` FOREIGN KEY (`run_id`) REFERENCES `payment_reconciliation_runs` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
