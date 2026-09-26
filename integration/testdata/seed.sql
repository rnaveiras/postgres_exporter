-- Objects the integration tests assert on. Applied once to the pgfresh template database.
CREATE TABLE app_events (
  id      bigserial PRIMARY KEY,
  payload text NOT NULL
);

-- A table without indexes: its idx_* statistics are NULL.
CREATE TABLE app_no_index (
  id   bigint,
  note text
);

-- Identifiers that break naive string concatenation.
CREATE SCHEMA "Mixed Case";
CREATE TABLE "Mixed Case"."Weird.Name" (id int PRIMARY KEY);
