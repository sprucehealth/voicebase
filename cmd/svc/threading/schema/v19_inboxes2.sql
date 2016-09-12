DELIMITER $$

ALTER TABLE saved_queries DROP COLUMN organization_id$$

CREATE TABLE saved_query_thread (
    saved_query_id BIGINT UNSIGNED NOT NULL,
    thread_id BIGINT UNSIGNED NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    unread BOOL NOT NULL, -- unread is included so the triggers know how to update the stats
    PRIMARY KEY (thread_id, saved_query_id),
    KEY saved_query_last_timestamp (saved_query_id, timestamp),
    CONSTRAINT saved_query_thread_saved_query FOREIGN KEY (saved_query_id) REFERENCES saved_queries (id),
    CONSTRAINT saved_query_thread_thread FOREIGN KEY (thread_id) REFERENCES threads (id)
)$$

CREATE TRIGGER saved_query_thread_update
    AFTER UPDATE ON saved_query_thread
    FOR EACH ROW
    BEGIN
        IF OLD.unread = true AND NEW.unread = false THEN
            UPDATE saved_queries SET unread = unread - 1 WHERE id = NEW.saved_query_id;
        ELSEIF OLD.unread = false AND new.unread = true THEN
            UPDATE saved_queries SET unread = unread + 1 WHERE id = NEW.saved_query_id;
        END IF;
    END$$

CREATE TRIGGER saved_query_thread_insert
    AFTER INSERT ON saved_query_thread
    FOR EACH ROW
    BEGIN
        IF NEW.unread = true THEN
            UPDATE saved_queries SET total = total + 1, unread = unread + 1 WHERE id = NEW.saved_query_id;
        ELSE
            UPDATE saved_queries SET total = total + 1 WHERE id = NEW.saved_query_id;
        END IF;
    END$$

CREATE TRIGGER saved_query_thread_delete
    AFTER DELETE ON saved_query_thread
    FOR EACH ROW
    BEGIN
        IF OLD.unread = true THEN
            UPDATE saved_queries SET total = total - 1, unread = unread - 1 WHERE id = OLD.saved_query_id;
        ELSE
            UPDATE saved_queries SET total = total - 1 WHERE id = OLD.saved_query_id;
        END IF;
    END$$
