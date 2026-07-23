CREATE TABLE `user_resource_scopes` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` int NOT NULL,
  `domain` varchar(32) NOT NULL,
  `store_id` varchar(64) NOT NULL DEFAULT '',
  `team_id` varchar(64) NOT NULL DEFAULT '',
  `created_at` datetime(3) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_resource_scope` (`user_id`, `domain`, `store_id`, `team_id`),
  KEY `idx_user_resource_scope_lookup` (`user_id`, `domain`),
  CONSTRAINT `fk_user_resource_scopes_user` FOREIGN KEY (`user_id`) REFERENCES `user` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

ALTER TABLE `admin_audit_logs`
  ADD COLUMN `correlation_id` char(36) NULL AFTER `detail`,
  ADD COLUMN `request_id` varchar(128) NOT NULL DEFAULT '' AFTER `correlation_id`,
  ADD COLUMN `target_type` varchar(32) NOT NULL DEFAULT '' AFTER `request_id`,
  ADD COLUMN `target_id` varchar(128) NOT NULL DEFAULT '' AFTER `target_type`,
  ADD COLUMN `domain` varchar(32) NOT NULL DEFAULT '' AFTER `target_id`,
  ADD COLUMN `store_id` varchar(64) NOT NULL DEFAULT '' AFTER `domain`,
  ADD COLUMN `team_id` varchar(64) NOT NULL DEFAULT '' AFTER `store_id`,
  ADD UNIQUE KEY `uk_admin_audit_correlation` (`correlation_id`),
  ADD KEY `idx_admin_audit_target_resource` (`target_type`, `target_id`);

CREATE TABLE `inventory_adjustment_logs` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `goods_id` int NOT NULL,
  `before_available` int NOT NULL,
  `after_available` int NOT NULL,
  `actor_user_id` int NOT NULL,
  `correlation_id` char(36) NOT NULL,
  `request_id` varchar(128) NOT NULL,
  `reason` varchar(255) NOT NULL,
  `created_at` datetime(3) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_inventory_adjustment_correlation` (`correlation_id`),
  KEY `idx_inventory_adjustment_goods` (`goods_id`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `order_refund_requests` (
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

DELETE rp FROM `role_permissions` rp JOIN `roles` r ON r.id = rp.role_id
WHERE r.name = 'catalog' AND rp.permission IN ('inventory:write:any', 'inventory:read:any', 'inventory:audit:read:any');

INSERT IGNORE INTO `role_permissions` (`role_id`, `permission`)
SELECT r.id, p.permission FROM `roles` r
CROSS JOIN (
  SELECT 'inventory:read:any' AS permission
  UNION ALL SELECT 'inventory:write:any'
  UNION ALL SELECT 'inventory:audit:read:any'
) p
WHERE r.name IN ('ops', 'admin', 'super_admin');
