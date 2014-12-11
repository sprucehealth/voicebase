ALTER TABLE rx_refill_request modify column request_date timestamp not null default current_timestamp;
ALTER TABLE rx_refill_request ADD INDEX (erx_request_queue_item_id);
