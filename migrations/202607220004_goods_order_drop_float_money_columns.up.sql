ALTER TABLE `ordergoods`
  DROP COLUMN `goods_price`;

ALTER TABLE `orderinfo`
  DROP COLUMN `order_mount`;

ALTER TABLE `goods`
  DROP COLUMN `shop_price`,
  DROP COLUMN `market_price`;
