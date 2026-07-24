CREATE TABLE `inventory` (
  `id` int NOT NULL AUTO_INCREMENT,
  `add_time` datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3),
  `update_time` datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  `deleted_at` datetime(3) NULL DEFAULT NULL,
  `is_deleted` tinyint(1) NOT NULL DEFAULT 0,
  `goods` int NOT NULL DEFAULT 0,
  `stocks` int NOT NULL DEFAULT 0,
  `version` int NOT NULL DEFAULT 0,
  PRIMARY KEY (`id`),
  KEY `idx_inventory_goods` (`goods`),
  KEY `idx_inventory_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `stockselldetail` (
  `order_sn` varchar(200) NOT NULL,
  `status` int NOT NULL,
  `detail` varchar(200) NOT NULL,
  UNIQUE KEY `idx_order_sn` (`order_sn`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
