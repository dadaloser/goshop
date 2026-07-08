ALTER TABLE `user`
  ADD COLUMN `username` varchar(32) NULL AFTER `id`,
  ADD COLUMN `email` varchar(100) NULL AFTER `mobile`;

CREATE UNIQUE INDEX `idx_username` ON `user` (`username`);
CREATE UNIQUE INDEX `idx_email` ON `user` (`email`);
