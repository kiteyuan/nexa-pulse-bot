ALTER TABLE themes ADD COLUMN sort_order INTEGER NOT NULL DEFAULT 0;

WITH ranked AS (
    SELECT id, (ROW_NUMBER() OVER (ORDER BY id))::integer AS n
    FROM themes
)
UPDATE themes AS t
SET sort_order = ranked.n
FROM ranked
WHERE t.id = ranked.id;
