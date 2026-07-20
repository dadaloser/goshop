CREATE TABLE `user_audit_logs` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` int NOT NULL,
  `actor_user_id` int NOT NULL DEFAULT 0,
  `actor_principal_type` varchar(32) NOT NULL,
  `action` varchar(64) NOT NULL,
  `from_status` varchar(16) NULL,
  `to_status` varchar(16) NULL,
  `detail` text NULL,
  `add_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_user_audit_logs_user` (`user_id`),
  KEY `idx_user_audit_logs_actor` (`actor_user_id`)
);
