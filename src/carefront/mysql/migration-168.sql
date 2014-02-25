alter table rx_refill_status_events drop column reason;
alter table rx_refill_status_events add column reason_id int unsigned;
alter table rx_refill_status_events add foreign key (reason_id) references deny_refill_reason(id);