CREATE TABLE `user_device_blacklist` (
  `user_id` int NOT NULL,
  `device_id` varchar(128) NOT NULL,
  `created_at` datetime(3) NOT NULL,
  PRIMARY KEY (`user_id`, `device_id`),
  CONSTRAINT `fk_user_device_blacklist_user` FOREIGN KEY (`user_id`) REFERENCES `user` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
