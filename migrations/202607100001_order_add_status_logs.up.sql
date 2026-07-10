CREATE TABLE `order_status_logs` (
  `id` int NOT NULL AUTO_INCREMENT,
  `add_time` datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3),
  `update_time` datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  `deleted_at` datetime(3) NULL DEFAULT NULL,
  `is_deleted` tinyint(1) NOT NULL DEFAULT 0,
  `order_id` int NOT NULL DEFAULT 0,
  `order_sn` varchar(30) NOT NULL,
  `from_status` varchar(20) NOT NULL DEFAULT '',
  `to_status` varchar(20) NOT NULL,
  `reason` varchar(128) NOT NULL DEFAULT '',
  `source` varchar(64) NOT NULL DEFAULT '',
  `operator` varchar(64) NOT NULL DEFAULT '',
  PRIMARY KEY (`id`),
  KEY `idx_order_status_logs_order_id` (`order_id`),
  KEY `idx_order_status_logs_order_sn` (`order_sn`),
  KEY `idx_order_status_logs_to_status` (`to_status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
