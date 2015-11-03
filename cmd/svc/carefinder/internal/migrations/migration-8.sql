-- BUILD an id out of state names
ALTER TABLE state ADD COLUMN key TEXT;

UPDATE state
SET key = translate(lower(state.full_name),' ', '-');

ALTER TABLE state ALTER COLUMN key SET NOT NULL;

CREATE UNIQUE INDEX states_key ON state USING btree (key);

ALTER TABLE city_shortlist ADD COLUMN featured BOOL NOT NULL default false;