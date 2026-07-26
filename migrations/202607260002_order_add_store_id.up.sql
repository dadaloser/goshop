ALTER TABLE `orderinfo`
  ADD COLUMN `store_id` varchar(64) NOT NULL DEFAULT '' AFTER `order_mount_fen`,
  ADD KEY `idx_orderinfo_store_id` (`store_id`);
