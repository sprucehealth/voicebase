-- Update the night regimen section title to nighttime because thats what older doctor clients expect
UPDATE regimen_section SET
	title = 'Nighttime'
	WHERE title = 'Night';

UPDATE dr_favorite_regimen_section SET
	title = 'Nighttime'
	WHERE title = 'Night';

