ALTER TABLE `role_domains`
  DROP FOREIGN KEY `fk_role_domains_role`;

ALTER TABLE `role_permissions`
  DROP FOREIGN KEY `fk_role_permissions_role`;

ALTER TABLE `user_roles`
  DROP FOREIGN KEY `fk_user_roles_role`;
