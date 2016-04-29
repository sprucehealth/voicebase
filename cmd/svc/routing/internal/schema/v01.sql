CREATE TABLE externalmsg (
	data blob NOT NULL,
	type varchar(255) NOT NULL,
	from_endpoint_id varchar(255) NOT NULL,
	to_endpoint_id varchar(255) NOT NULL,
	status varchar(255) NOT NULL,
	created timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
	KEY (from_endpoint_id, to_endpoint_id)
);
