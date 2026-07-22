CREATE TABLE `admin_audit_logs` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `target_user_id` int NOT NULL DEFAULT 0,
  `actor_user_id` int NOT NULL DEFAULT 0,
  `actor_principal_type` varchar(32) NOT NULL,
  `action` varchar(64) NOT NULL,
  `detail` text NULL,
  `add_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_admin_audit_logs_target` (`target_user_id`),
  KEY `idx_admin_audit_logs_actor` (`actor_user_id`)
);
