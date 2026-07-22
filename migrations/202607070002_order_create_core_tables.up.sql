CREATE TABLE `orderinfo` (
  `id` int NOT NULL AUTO_INCREMENT,
  `add_time` datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3),
  `update_time` datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  `deleted_at` datetime(3) NULL DEFAULT NULL,
  `is_deleted` tinyint(1) NOT NULL DEFAULT 0,
  `user` int NOT NULL DEFAULT 0,
  `order_sn` varchar(30) NOT NULL,
  `pay_type` varchar(20) NOT NULL DEFAULT '',
  `status` varchar(20) NOT NULL DEFAULT '',
  `trade_no` varchar(100) NOT NULL DEFAULT '',
  `order_mount` float NOT NULL DEFAULT 0,
  `pay_time` datetime(3) NULL DEFAULT NULL,
  `address` varchar(100) NOT NULL DEFAULT '',
  `signer_name` varchar(20) NOT NULL DEFAULT '',
  `singer_mobile` varchar(11) NOT NULL DEFAULT '',
  `post` varchar(20) NOT NULL DEFAULT '',
  PRIMARY KEY (`id`),
  KEY `idx_orderinfo_user` (`user`),
  KEY `idx_orderinfo_order_sn` (`order_sn`),
  KEY `idx_orderinfo_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ordergoods` (
  `id` int NOT NULL AUTO_INCREMENT,
  `add_time` datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3),
  `update_time` datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  `deleted_at` datetime(3) NULL DEFAULT NULL,
  `is_deleted` tinyint(1) NOT NULL DEFAULT 0,
  `order` int NOT NULL DEFAULT 0,
  `goods` int NOT NULL DEFAULT 0,
  `goods_name` varchar(100) NOT NULL DEFAULT '',
  `goods_image` varchar(200) NOT NULL DEFAULT '',
  `goods_price` float NOT NULL DEFAULT 0,
  `nums` int NOT NULL DEFAULT 0,
  PRIMARY KEY (`id`),
  KEY `idx_ordergoods_order` (`order`),
  KEY `idx_ordergoods_goods` (`goods`),
  KEY `idx_ordergoods_goods_name` (`goods_name`),
  KEY `idx_ordergoods_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `shoppingcart` (
  `id` int NOT NULL AUTO_INCREMENT,
  `add_time` datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3),
  `update_time` datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  `deleted_at` datetime(3) NULL DEFAULT NULL,
  `is_deleted` tinyint(1) NOT NULL DEFAULT 0,
  `user` int NOT NULL DEFAULT 0,
  `goods` int NOT NULL DEFAULT 0,
  `nums` int NOT NULL DEFAULT 0,
  `checked` tinyint(1) NOT NULL DEFAULT 0,
  PRIMARY KEY (`id`),
  KEY `idx_shoppingcart_user` (`user`),
  KEY `idx_shoppingcart_goods` (`goods`),
  KEY `idx_shoppingcart_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
