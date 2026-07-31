-- Repair installations where an earlier session migration was marked complete
-- without adding every column required by the device-session API.
-- MySQL 5.7 does not support ALTER TABLE ... ADD COLUMN IF NOT EXISTS, so each
-- column is checked through information_schema before its DDL is executed.

SET @sql = IF(
  (SELECT COUNT(*) FROM information_schema.columns
   WHERE table_schema = DATABASE()
     AND table_name = 'user_sessions'
     AND column_name = 'principal_type') = 0,
  'ALTER TABLE `user_sessions` ADD COLUMN `principal_type` varchar(32) NOT NULL DEFAULT ''customer'' AFTER `user_id`',
  'SELECT 1'
);
PREPARE user_session_schema_repair_stmt FROM @sql;
EXECUTE user_session_schema_repair_stmt;
DEALLOCATE PREPARE user_session_schema_repair_stmt;

-- The list query filters by principal_type and user_id, then sorts by the
-- latest operation. Install the covering access path when an older schema was
-- incorrectly marked as migrated; otherwise the full scan can exceed the
-- API-to-gRPC timeout on a large session table.
SET @sql = IF(
  (SELECT COUNT(*) FROM information_schema.statistics
   WHERE table_schema = DATABASE()
     AND table_name = 'user_sessions'
     AND index_name = 'idx_user_sessions_principal_last_used') = 0,
  'ALTER TABLE `user_sessions` ADD KEY `idx_user_sessions_principal_last_used` (`principal_type`, `user_id`, `last_used_at`)',
  'SELECT 1'
);
PREPARE user_session_schema_repair_stmt FROM @sql;
EXECUTE user_session_schema_repair_stmt;
DEALLOCATE PREPARE user_session_schema_repair_stmt;

SET @sql = IF(
  (SELECT COUNT(*) FROM information_schema.columns
   WHERE table_schema = DATABASE()
     AND table_name = 'user_sessions'
     AND column_name = 'client_ip') = 0,
  'ALTER TABLE `user_sessions` ADD COLUMN `client_ip` varchar(45) NOT NULL DEFAULT '''' AFTER `device_name`',
  'SELECT 1'
);
PREPARE user_session_schema_repair_stmt FROM @sql;
EXECUTE user_session_schema_repair_stmt;
DEALLOCATE PREPARE user_session_schema_repair_stmt;

SET @sql = IF(
  (SELECT COUNT(*) FROM information_schema.columns
   WHERE table_schema = DATABASE()
     AND table_name = 'user_sessions'
     AND column_name = 'location') = 0,
  'ALTER TABLE `user_sessions` ADD COLUMN `location` varchar(255) NOT NULL DEFAULT '''' AFTER `client_ip`',
  'SELECT 1'
);
PREPARE user_session_schema_repair_stmt FROM @sql;
EXECUTE user_session_schema_repair_stmt;
DEALLOCATE PREPARE user_session_schema_repair_stmt;
