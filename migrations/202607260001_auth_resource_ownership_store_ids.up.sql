ALTER TABLE `goods`
  ADD COLUMN `store_id` varchar(64) NOT NULL DEFAULT '' AFTER `sku_code`,
  ADD KEY `idx_goods_store_id` (`store_id`);
