-- Create language ID column and FK
ALTER TABLE photo_slot 
ADD COLUMN language_id INT(10) UNSIGNED DEFAULT 1,
ADD CONSTRAINT fk_photo_slot_languages_supported_id 
FOREIGN KEY(language_id)
REFERENCES languages_supported(id);

-- Create photo_slot_text column and populate from existing data
ALTER TABLE photo_slot ADD COLUMN name_text VARCHAR(600);
UPDATE photo_slot ps
LEFT JOIN localized_text lt ON
  ps.slot_name_app_text_id = app_text_id
SET
  name_text = ltext;

  -- Create photo_slot_type column and pooulate from existing data
ALTER TABLE photo_slot ADD COLUMN photo_slot_type VARCHAR(60);
UPDATE photo_slot ps
LEFT JOIN photo_slot_type pst on
  ps.slot_type_id = pst.id
SET
  photo_slot_type = pst.slot_type;
ALTER TABLE photo_slot MODIFY photo_slot_type VARCHAR(60) NOT NULL;

-- Create the new client data blob
ALTER TABLE photo_slot ADD COLUMN client_data BLOB;

-- Drop the FK
ALTER TABLE photo_slot DROP FOREIGN KEY photo_slot_ibfk_3;
ALTER TABLE photo_slot DROP COLUMN slot_type_id;

-- Drop the FK on the app_text table and remove the column
ALTER TABLE photo_slot DROP FOREIGN KEY photo_slot_ibfk_2;
ALTER TABLE photo_slot DROP COLUMN slot_name_app_text_id;