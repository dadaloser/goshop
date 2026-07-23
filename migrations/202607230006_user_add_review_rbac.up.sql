INSERT IGNORE INTO `roles` (`name`, `description`) VALUES ('review', 'review moderation and merchant replies');
INSERT IGNORE INTO `role_domains` (`role_id`, `domain`) SELECT `id`, 'review' FROM `roles` WHERE `name` = 'review';
INSERT IGNORE INTO `role_permissions` (`role_id`, `permission`) SELECT r.id, p.permission FROM `roles` r CROSS JOIN (SELECT 'review:moderate:any' AS permission UNION ALL SELECT 'review:reply:any') p WHERE r.name = 'review';
INSERT IGNORE INTO `role_permissions` (`role_id`, `permission`) SELECT r.id, p.permission FROM `roles` r CROSS JOIN (SELECT 'review:moderate:any' AS permission UNION ALL SELECT 'review:reply:any' UNION ALL SELECT 'review:aggregate:rebuild:any') p WHERE r.name IN ('admin', 'super_admin');
