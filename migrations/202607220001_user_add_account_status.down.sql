SET @has_account_status := (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'user'
    AND COLUMN_NAME = 'account_status'
);

SET @drop_account_status_sql := IF(
  @has_account_status = 1,
  'ALTER TABLE `user` DROP COLUMN `account_status`',
  'SELECT 1'
);

PREPARE drop_account_status_stmt FROM @drop_account_status_sql;
EXECUTE drop_account_status_stmt;
DEALLOCATE PREPARE drop_account_status_stmt;
