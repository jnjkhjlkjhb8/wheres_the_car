-- PowerSync requires a single text 'id' column per table.
-- search_vector uses a composite PK (type, uid, city); add a generated id column.

ALTER TABLE search_vector
  ADD COLUMN IF NOT EXISTS id TEXT GENERATED ALWAYS AS (type || '_' || uid || '_' || city) STORED;

CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS idx_search_vector_id ON search_vector (id);
