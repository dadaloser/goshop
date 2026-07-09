ALTER TABLE `inventory`
  ADD COLUMN `total` int NOT NULL DEFAULT 0 AFTER `goods`,
  ADD COLUMN `available` int NOT NULL DEFAULT 0 AFTER `total`,
  ADD COLUMN `locked` int NOT NULL DEFAULT 0 AFTER `available`,
  ADD COLUMN `sold` int NOT NULL DEFAULT 0 AFTER `locked`;

UPDATE `inventory`
SET
  `total` = `stocks`,
  `available` = `stocks`,
  `locked` = 0,
  `sold` = 0;
