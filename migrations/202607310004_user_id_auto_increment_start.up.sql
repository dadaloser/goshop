-- New users use the database-generated sequence. Start it at the first
-- nine-digit value while preserving any already allocated larger ID.
ALTER TABLE `user` AUTO_INCREMENT = 100000000;
