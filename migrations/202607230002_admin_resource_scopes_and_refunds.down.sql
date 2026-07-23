DROP TABLE IF EXISTS `inventory_adjustment_logs`;
DROP TABLE IF EXISTS `order_refund_requests`;
DROP TABLE IF EXISTS `user_resource_scopes`;
ALTER TABLE `admin_audit_logs`
  DROP INDEX `idx_admin_audit_target_resource`,
  DROP INDEX `uk_admin_audit_correlation`,
  DROP COLUMN `team_id`, DROP COLUMN `store_id`, DROP COLUMN `domain`,
  DROP COLUMN `target_id`, DROP COLUMN `target_type`, DROP COLUMN `request_id`, DROP COLUMN `correlation_id`;
DELETE rp FROM `role_permissions` rp JOIN `roles` r ON r.id = rp.role_id
WHERE rp.permission IN ('inventory:read:any', 'inventory:audit:read:any');
INSERT IGNORE INTO `role_permissions` (`role_id`, `permission`)
SELECT id, 'inventory:write:any' FROM `roles` WHERE name = 'catalog';
