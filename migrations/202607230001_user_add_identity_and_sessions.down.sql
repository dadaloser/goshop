DROP TABLE IF EXISTS `verification_codes`;
DROP TABLE IF EXISTS `user_sessions`;

ALTER TABLE `user`
  DROP COLUMN `last_login_at`,
  DROP COLUMN `email_verified`,
  DROP COLUMN `mobile_verified`;
