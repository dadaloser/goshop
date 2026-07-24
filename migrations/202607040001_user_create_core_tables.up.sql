CREATE TABLE `user` (
  `id` int NOT NULL AUTO_INCREMENT,
  `add_time` datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3),
  `update_time` datetime(3) NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  `deleted_at` datetime(3) NULL DEFAULT NULL,
  `is_deleted` tinyint(1) NOT NULL DEFAULT 0,
  `mobile` varchar(11) NOT NULL,
  `password` varchar(100) NOT NULL,
  `nick_name` varchar(20) NOT NULL DEFAULT '',
  `birthday` datetime NULL DEFAULT NULL,
  `gender` varchar(6) NOT NULL DEFAULT 'male',
  `role` int NOT NULL DEFAULT 1,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_mobile` (`mobile`),
  KEY `idx_user_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
