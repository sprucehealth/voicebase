-- Track the last time that we notified a user about the unread status of this thread
ALTER TABLE thread_members ADD COLUMN last_unread_notify TIMESTAMP;