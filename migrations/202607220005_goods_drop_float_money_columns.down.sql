ALTER TABLE `goods`
  ADD COLUMN `market_price` FLOAT NOT NULL DEFAULT 0 AFTER `fav_num`,
  ADD COLUMN `shop_price` FLOAT NOT NULL DEFAULT 0 AFTER `market_price`;

UPDATE `goods`
SET
  `market_price` = `market_price_fen` / 100.0,
  `shop_price` = `shop_price_fen` / 100.0;
