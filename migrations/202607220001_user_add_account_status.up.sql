SET @has_account_status := (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'user'
    AND COLUMN_NAME = 'account_status'
);

SET @add_account_status_sql := IF(
  @has_account_status = 0,
  'ALTER TABLE `user` ADD COLUMN `account_status` varchar(16) NOT NULL DEFAULT ''active'' AFTER `role`',
  'SELECT 1'
);

PREPARE add_account_status_stmt FROM @add_account_status_sql;
EXECUTE add_account_status_stmt;
DEALLOCATE PREPARE add_account_status_stmt;

UPDATE `user`
SET `account_status` = 'active'
WHERE `account_status` IS NULL
   OR `account_status` = '';
