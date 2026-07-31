-- Apply independently from 202607310003: installations that already recorded
-- that migration must still receive the list-query index.
SET @sql = IF(
  (SELECT COUNT(*) FROM information_schema.statistics
   WHERE table_schema = DATABASE()
     AND table_name = 'user_sessions'
     AND index_name = 'idx_user_sessions_principal_last_used') = 0,
  'ALTER TABLE `user_sessions` ADD KEY `idx_user_sessions_principal_last_used` (`principal_type`, `user_id`, `last_used_at`)',
  'SELECT 1'
);
PREPARE user_session_list_index_repair_stmt FROM @sql;
EXECUTE user_session_list_index_repair_stmt;
DEALLOCATE PREPARE user_session_list_index_repair_stmt;
