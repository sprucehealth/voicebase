START TRANSACTION;

-- threads in the onboarding table that have no type associated with them are SETUP threads
UPDATE threads as t
INNER JOIN onboarding_threads AS ot ON t.id = ot.thread_id
SET t.type='SETUP'
WHERE t.type = '';

-- threads in the thread_links table that have no type associated with them are SUPPORT threads
UPDATE threads as t
INNER JOIN thread_links AS tl on tl.thread1_id = t.id
SET t.type = 'SUPPORT'
WHERE t.system_title is null;

UPDATE threads as t
INNER JOIN thread_links AS tl on tl.thread2_id = t.id
SET t.type = 'SUPPORT'
WHERE t.type = '';

-- threads that contain the empty state message for a legacy team thread but don't have a type should be marked as such
UPDATE threads as t
SET t.type = 'LEGACY_TEAM'
WHERE t.type = ''
AND t.id in (select thread_id from thread_items where data like '%Invite some colleagues to join and then send%');

-- Only EXTERNAL threads can be deleted so lets go ahead and mark those threads as EXTERNAL
UPDATE threads 
SET t.type = 'EXTERNAL'
WHERE t.type = ''
AND t.deleted = true;

-- The reamining threads that are none of the above types are essentially external threads
UPDATE threads
SET t.type='EXTERNAL'
WHERE t.type = '';

COMMIT;