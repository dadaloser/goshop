ALTER TABLE `user_sessions`
  ADD COLUMN `client_ip` varchar(45) NOT NULL DEFAULT '' AFTER `device_name`,
  ADD COLUMN `location` varchar(255) NOT NULL DEFAULT '' AFTER `client_ip`;
