-- last_reserved_time tracks when the proxy number was last reserved
ALTER TABLE proxy_phone_number ADD COLUMN last_reserved_time TIMESTAMP;