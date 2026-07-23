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
