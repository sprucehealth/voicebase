ALTER TABLE threads MODIFY COLUMN system_title VARCHAR(4096) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;

-- set the character set for the mysql client
SET NAMES 'utf8mb4' COLLATE 'utf8mb4_general_ci';

UPDATE threads
set system_title = 'Spruce Support 🌲'
WHERE system_title='Spruce Support'
AND type='SUPPORT';


UPDATE threads
set system_title = 'Setup 🚀'
WHERE system_title='Spruce Assistant'
AND type='SETUP';
