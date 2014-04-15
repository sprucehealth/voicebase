alter table rx_refill_request add column comments varchar(500) not null;
alter table rx_refill_request add column denial_reason_id int unsigned;
alter table rx_refill_request add foreign key (denial_reason_id) references deny_refill_reason(id);
update rx_refill_request inner join rx_refill_status_events on rx_refill_request.id = rx_refill_status_events.rx_refill_request_id
	 set comments = rx_refill_status_events.notes, denial_reason_id = rx_refill_status_events.reason_id
	 where rx_refill_status_events.notes is not null and rx_refill_status_events.reason_id is not null;
alter table rx_refill_status_events drop column notes;
alter table rx_refill_status_events drop foreign key rx_refill_status_events_ibfk_2;
alter table rx_refill_status_events drop column reason_id;


