-- add a column to track for which thread we want to presend the sender identity
-- to the message so that we know who in the organization is writing the message.
ALTER TABLE thread_links ADD COLUMN thread1_prepend_sender TINYINT(1) NOT NULL DEFAULT 0;
ALTER TABLE thread_links ADD COLUMN thread2_prepend_sender TINYINT(1) NOT NULL DEFAULT 0;
-- set the prepend sender flag for the second thread to true since that thread represents the thread on the spruce org side.
UPDATE thread_links SET thread2_prepend_sender = 1;