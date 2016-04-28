CREATE TABLE externalmsg (
	data blob NOT NULL,
	type varchar(255) NOT NULL,
	from varchar(255) NOT NULL,
	to varchar(255) NOT NULL,
	status varchar(255) NOT NULL,
	created timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
	KEY (from, to)
);
