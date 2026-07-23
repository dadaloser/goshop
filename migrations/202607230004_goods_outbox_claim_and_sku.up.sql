ALTER TABLE `outbox_events`
  ADD COLUMN `claimed_at` bigint NOT NULL DEFAULT 0 AFTER `processing_lock`,
  ADD KEY `idx_outbox_claimed_at` (`status`, `claimed_at`);

ALTER TABLE `goods`
  ADD COLUMN `spu_code` varchar(64) NOT NULL DEFAULT '' AFTER `goods_sn`,
  ADD COLUMN `sku_code` varchar(64) NOT NULL DEFAULT '' AFTER `spu_code`,
  ADD KEY `idx_goods_spu_code` (`spu_code`),
  ADD KEY `idx_goods_on_sale` (`on_sale`, `category_id`, `brands_id`);

UPDATE `goods`
SET `spu_code` = CONCAT('legacy-spu-', `id`),
    `sku_code` = CONCAT('legacy-sku-', `id`)
WHERE `spu_code` = '' OR `sku_code` = '';

ALTER TABLE `goods`
  ADD UNIQUE KEY `uk_goods_sku_code` (`sku_code`);
