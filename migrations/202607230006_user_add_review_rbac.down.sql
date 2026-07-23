DELETE rp FROM `role_permissions` rp JOIN `roles` r ON r.id = rp.role_id WHERE r.name IN ('review','admin','super_admin') AND rp.permission IN ('review:moderate:any','review:reply:any','review:aggregate:rebuild:any');
DELETE rd FROM `role_domains` rd JOIN `roles` r ON r.id = rd.role_id WHERE r.name = 'review' AND rd.domain = 'review';
DELETE FROM `roles` WHERE `name` = 'review';
