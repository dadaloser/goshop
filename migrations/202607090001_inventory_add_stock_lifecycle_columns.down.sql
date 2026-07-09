-- Destructive rollback: this removes lifecycle stock columns and any data
-- accumulated after migration. Review production data before applying.
ALTER TABLE `inventory`
  DROP COLUMN `sold`,
  DROP COLUMN `locked`,
  DROP COLUMN `available`,
  DROP COLUMN `total`;
