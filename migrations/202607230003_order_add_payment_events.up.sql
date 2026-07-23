CREATE TABLE `payment_events` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `provider` varchar(32) NOT NULL,
  `event_id` varchar(128) NOT NULL,
  `order_sn` varchar(64) NOT NULL,
  `trade_no` varchar(128) NOT NULL,
  `event_type` varchar(32) NOT NULL,
  `order_amount_fen` bigint NOT NULL,
  `provider_amount_fen` bigint NOT NULL,
  `refund_amount_fen` bigint NOT NULL DEFAULT 0,
  `status` varchar(24) NOT NULL,
  `error_detail` varchar(255) NOT NULL DEFAULT '',
  `received_at` datetime(3) NOT NULL,
  `completed_at` datetime(3) NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_payment_event_provider_event` (`provider`, `event_id`),
  KEY `idx_payment_event_order` (`order_sn`, `received_at`),
  KEY `idx_payment_event_reconcile` (`status`, `received_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `payment_reconciliation_runs` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `started_at` datetime(3) NOT NULL,
  `finished_at` datetime(3) NULL,
  `checked_count` int NOT NULL DEFAULT 0,
  `mismatch_count` int NOT NULL DEFAULT 0,
  `status` varchar(24) NOT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_payment_reconciliation_started` (`started_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
