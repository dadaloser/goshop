DELETE ur FROM `user_roles` ur
LEFT JOIN `roles` r ON r.id = ur.role_id
WHERE r.id IS NULL;

DELETE rp FROM `role_permissions` rp
LEFT JOIN `roles` r ON r.id = rp.role_id
WHERE r.id IS NULL;

DELETE rd FROM `role_domains` rd
LEFT JOIN `roles` r ON r.id = rd.role_id
WHERE r.id IS NULL;

ALTER TABLE `user_roles`
  ADD CONSTRAINT `fk_user_roles_role`
    FOREIGN KEY (`role_id`) REFERENCES `roles` (`id`)
    ON DELETE RESTRICT
    ON UPDATE CASCADE;

ALTER TABLE `role_permissions`
  ADD CONSTRAINT `fk_role_permissions_role`
    FOREIGN KEY (`role_id`) REFERENCES `roles` (`id`)
    ON DELETE CASCADE
    ON UPDATE CASCADE;

ALTER TABLE `role_domains`
  ADD CONSTRAINT `fk_role_domains_role`
    FOREIGN KEY (`role_id`) REFERENCES `roles` (`id`)
    ON DELETE CASCADE
    ON UPDATE CASCADE;
