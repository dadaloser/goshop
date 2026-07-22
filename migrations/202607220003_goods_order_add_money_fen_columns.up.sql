ALTER TABLE `goods`
  ADD COLUMN `market_price_fen` BIGINT NOT NULL DEFAULT 0 AFTER `market_price`,
  ADD COLUMN `shop_price_fen` BIGINT NOT NULL DEFAULT 0 AFTER `shop_price`;

UPDATE `goods`
SET
  `market_price_fen` = ROUND(`market_price` * 100),
  `shop_price_fen` = ROUND(`shop_price` * 100)
WHERE `market_price_fen` = 0
  AND `shop_price_fen` = 0
  AND (`market_price` <> 0 OR `shop_price` <> 0);

ALTER TABLE `orderinfo`
  ADD COLUMN `order_mount_fen` BIGINT NOT NULL DEFAULT 0 AFTER `order_mount`;

UPDATE `orderinfo`
SET `order_mount_fen` = ROUND(`order_mount` * 100)
WHERE `order_mount_fen` = 0
  AND `order_mount` <> 0;

ALTER TABLE `ordergoods`
  ADD COLUMN `goods_price_fen` BIGINT NOT NULL DEFAULT 0 AFTER `goods_price`;

UPDATE `ordergoods`
SET `goods_price_fen` = ROUND(`goods_price` * 100)
WHERE `goods_price_fen` = 0
  AND `goods_price` <> 0;
