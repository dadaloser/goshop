ALTER TABLE `goods`
  ADD COLUMN `market_price` FLOAT NOT NULL DEFAULT 0 AFTER `fav_num`,
  ADD COLUMN `shop_price` FLOAT NOT NULL DEFAULT 0 AFTER `market_price`;

UPDATE `goods`
SET
  `market_price` = `market_price_fen` / 100.0,
  `shop_price` = `shop_price_fen` / 100.0;

ALTER TABLE `orderinfo`
  ADD COLUMN `order_mount` FLOAT NOT NULL DEFAULT 0 AFTER `trade_no`;

UPDATE `orderinfo`
SET `order_mount` = `order_mount_fen` / 100.0;

ALTER TABLE `ordergoods`
  ADD COLUMN `goods_price` FLOAT NOT NULL DEFAULT 0 AFTER `goods_image`;

UPDATE `ordergoods`
SET `goods_price` = `goods_price_fen` / 100.0;
