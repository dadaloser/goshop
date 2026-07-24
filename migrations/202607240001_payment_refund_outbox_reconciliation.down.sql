DROP TABLE IF EXISTS `payment_reconciliation_items`;
ALTER TABLE `payment_reconciliation_runs`
  DROP COLUMN `window_end`, DROP COLUMN `window_start`, DROP COLUMN `provider`;
DROP TABLE IF EXISTS `order_refund_outbox`;
ALTER TABLE `order_refund_requests`
  DROP INDEX `idx_order_refund_provider_id`,
  DROP COLUMN `failure_reason`, DROP COLUMN `provider_refund_id`, DROP COLUMN `provider`;
