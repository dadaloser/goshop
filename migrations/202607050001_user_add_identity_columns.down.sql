DROP INDEX `idx_email` ON `user`;
DROP INDEX `idx_username` ON `user`;

-- Destructive rollback: this removes user identity data added after the
-- migration. Review production data before applying.
ALTER TABLE `user`
  DROP COLUMN `email`,
  DROP COLUMN `username`;
