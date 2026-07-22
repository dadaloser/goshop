CREATE TABLE `category` (
  `id` int NOT NULL AUTO_INCREMENT,
  `add_time` datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3),
  `update_time` datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  `deleted_at` datetime(3) NULL DEFAULT NULL,
  `is_deleted` tinyint(1) NOT NULL DEFAULT 0,
  `name` varchar(20) NOT NULL,
  `parent_category_id` int NOT NULL DEFAULT 0,
  `level` int NOT NULL DEFAULT 1,
  `is_tab` tinyint(1) NOT NULL DEFAULT 0,
  PRIMARY KEY (`id`),
  KEY `idx_category_parent_category_id` (`parent_category_id`),
  KEY `idx_category_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `brands` (
  `id` int NOT NULL AUTO_INCREMENT,
  `add_time` datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3),
  `update_time` datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  `deleted_at` datetime(3) NULL DEFAULT NULL,
  `is_deleted` tinyint(1) NOT NULL DEFAULT 0,
  `name` varchar(20) NOT NULL,
  `logo` varchar(200) NOT NULL DEFAULT '',
  PRIMARY KEY (`id`),
  KEY `idx_brands_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `banner` (
  `id` int NOT NULL AUTO_INCREMENT,
  `add_time` datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3),
  `update_time` datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  `deleted_at` datetime(3) NULL DEFAULT NULL,
  `is_deleted` tinyint(1) NOT NULL DEFAULT 0,
  `image` varchar(200) NOT NULL,
  `url` varchar(200) NOT NULL,
  `index` int NOT NULL DEFAULT 1,
  PRIMARY KEY (`id`),
  KEY `idx_banner_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `goodscategorybrand` (
  `id` int NOT NULL AUTO_INCREMENT,
  `add_time` datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3),
  `update_time` datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  `deleted_at` datetime(3) NULL DEFAULT NULL,
  `is_deleted` tinyint(1) NOT NULL DEFAULT 0,
  `category_id` int NOT NULL,
  `brands_id` int NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_category_brand` (`category_id`, `brands_id`),
  KEY `idx_goodscategorybrand_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `goods` (
  `id` int NOT NULL AUTO_INCREMENT,
  `add_time` datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3),
  `update_time` datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  `deleted_at` datetime(3) NULL DEFAULT NULL,
  `is_deleted` tinyint(1) NOT NULL DEFAULT 0,
  `category_id` int NOT NULL,
  `brands_id` int NOT NULL,
  `on_sale` tinyint(1) NOT NULL DEFAULT 0,
  `ship_free` tinyint(1) NOT NULL DEFAULT 0,
  `is_new` tinyint(1) NOT NULL DEFAULT 0,
  `is_hot` tinyint(1) NOT NULL DEFAULT 0,
  `name` varchar(50) NOT NULL,
  `goods_sn` varchar(50) NOT NULL,
  `click_num` int NOT NULL DEFAULT 0,
  `sold_num` int NOT NULL DEFAULT 0,
  `fav_num` int NOT NULL DEFAULT 0,
  `market_price` float NOT NULL DEFAULT 0,
  `shop_price` float NOT NULL DEFAULT 0,
  `goods_brief` varchar(100) NOT NULL,
  `goods_desc` text NOT NULL,
  `images` varchar(1000) NOT NULL,
  `desc_images` varchar(1000) NOT NULL,
  `goods_front_image` varchar(200) NOT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_goods_category_id` (`category_id`),
  KEY `idx_goods_brands_id` (`brands_id`),
  KEY `idx_goods_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
