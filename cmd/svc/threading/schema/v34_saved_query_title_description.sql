ALTER TABLE saved_queries ADD COLUMN long_title varchar(2048) NOT NULL DEFAULT 'All';
ALTER TABLE saved_queries ADD COLUMN description varchar(2048) NOT NULL DEFAULT 'All';

UPDATE saved_queries
	SET long_title = 'All Conversations',
	description = 'Any new activity in any conversation' 
	WHERE title = 'All';

UPDATE saved_queries
	SET long_title = 'All Patient Conversations',
	description = 'Any new activity in a patient conversation' 
	WHERE title = 'Patient';


UPDATE saved_queries
	SET long_title = '@ Pages',
	description = 'When you\'re @ paged in a message' 
	WHERE title = '@Pages';

UPDATE saved_queries
	SET long_title = 'Patient Conversations You Follow',
	description = 'New activity in patient conversations you are currently following' 
	WHERE title = 'Following';

UPDATE saved_queries
	SET long_title = 'Spruce Support',
	description = 'New messages from the Spruce Team' 
	WHERE title = 'Support';

UPDATE saved_queries
	SET long_title = 'Notifications',
	description = 'Hidden query to populate an accurate count of notifications' 
	WHERE title = 'Notifications';

UPDATE saved_queries
	SET long_title = 'Team Conversations',
	description = 'New messages in team conversations'
	WHERE title = 'Team';


