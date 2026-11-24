ALTER TABLE balance_entries DROP CONSTRAINT balance_entries_order_id_fkey;
DROP TABLE order_status_history;
DROP TABLE order_items;
DROP TABLE orders;
DROP SEQUENCE order_number_seq;
DROP TYPE order_status;
