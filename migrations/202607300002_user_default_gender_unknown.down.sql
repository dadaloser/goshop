-- Keep varchar(7): shrinking it would fail once rows contain "unknown".
ALTER TABLE `user`
  ALTER COLUMN `gender` SET DEFAULT 'male';
