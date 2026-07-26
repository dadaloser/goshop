DROP TABLE IF EXISTS `break_glass_approvals`;

ALTER TABLE `user_sessions`
  DROP KEY `idx_user_sessions_principal_active`,
  DROP COLUMN `principal_type`;

ALTER TABLE `user_resource_scopes`
  DROP KEY `uk_user_resource_scope`,
  DROP COLUMN `resource_id`,
  DROP COLUMN `resource_type`,
  ADD UNIQUE KEY `uk_user_resource_scope` (`user_id`, `domain`, `store_id`, `team_id`);
