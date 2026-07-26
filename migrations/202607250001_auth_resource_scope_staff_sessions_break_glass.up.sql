ALTER TABLE `user_resource_scopes`
  DROP KEY `uk_user_resource_scope`,
  ADD COLUMN `resource_type` varchar(32) NOT NULL DEFAULT '' AFTER `team_id`,
  ADD COLUMN `resource_id` varchar(128) NOT NULL DEFAULT '' AFTER `resource_type`,
  ADD UNIQUE KEY `uk_user_resource_scope` (`user_id`, `domain`, `store_id`, `team_id`, `resource_type`, `resource_id`);

ALTER TABLE `user_sessions`
  ADD COLUMN `principal_type` varchar(32) NOT NULL DEFAULT 'customer' AFTER `user_id`,
  ADD KEY `idx_user_sessions_principal_active` (`principal_type`, `user_id`, `revoked_at`, `expires_at`);

CREATE TABLE `break_glass_approvals` (
  `id` char(36) NOT NULL,
  `requester_user_id` int NOT NULL,
  `approver_user_id` int NOT NULL DEFAULT 0,
  `status` varchar(24) NOT NULL,
  `reason` varchar(255) NOT NULL,
  `request_id` varchar(128) NOT NULL,
  `created_at` datetime(3) NOT NULL,
  `approved_at` datetime(3) NULL,
  `expires_at` datetime(3) NOT NULL,
  `used_at` datetime(3) NULL,
  PRIMARY KEY (`id`),
  KEY `idx_break_glass_status_expiry` (`status`, `expires_at`),
  KEY `idx_break_glass_requester` (`requester_user_id`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
