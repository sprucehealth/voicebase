ALTER TABLE threads ADD COLUMN system_title VARCHAR(4096) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;
ALTER TABLE threads ADD COLUMN user_title VARCHAR(4096) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;
ALTER TABLE threads ADD COLUMN type VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin;
ALTER TABLE thread_members RENAME thread_entities;
ALTER TABLE thread_entities DROP COLUMN following;
ALTER TABLE thread_entities ADD COLUMN member BOOLEAN NOT NULL DEFAULT false;
