ALTER TABLE `ordergoods`
  DROP COLUMN `goods_price_fen`;

ALTER TABLE `orderinfo`
  DROP COLUMN `order_mount_fen`;

ALTER TABLE `goods`
  DROP COLUMN `shop_price_fen`,
  DROP COLUMN `market_price_fen`;
