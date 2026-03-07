-- +goose Up
CREATE INDEX IF NOT EXISTS idx_transactions_raw_date ON transactions(raw_date);
CREATE INDEX IF NOT EXISTS idx_transactions_category_id ON transactions(category_id);
CREATE INDEX IF NOT EXISTS idx_transactions_statement_id ON transactions(statement_id);
CREATE INDEX IF NOT EXISTS idx_import_items_session_id ON import_items(session_id);
CREATE INDEX IF NOT EXISTS idx_receipts_transaction_id ON receipts(transaction_id);
CREATE INDEX IF NOT EXISTS idx_receipts_image_hash ON receipts(image_hash);

-- +goose Down
DROP INDEX IF EXISTS idx_transactions_raw_date;
DROP INDEX IF EXISTS idx_transactions_category_id;
DROP INDEX IF EXISTS idx_transactions_statement_id;
DROP INDEX IF EXISTS idx_import_items_session_id;
DROP INDEX IF EXISTS idx_receipts_transaction_id;
DROP INDEX IF EXISTS idx_receipts_image_hash;
