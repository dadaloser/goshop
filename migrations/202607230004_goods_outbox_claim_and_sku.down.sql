ALTER TABLE `goods`
  DROP KEY `idx_goods_on_sale`,
  DROP KEY `idx_goods_spu_code`,
  DROP KEY `uk_goods_sku_code`,
  DROP COLUMN `sku_code`,
  DROP COLUMN `spu_code`;

ALTER TABLE `outbox_events`
  DROP KEY `idx_outbox_claimed_at`,
  DROP COLUMN `claimed_at`;
