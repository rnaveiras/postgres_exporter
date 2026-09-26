-- Least-privilege role for postgres_exporter, used by the local development stack.
-- Idempotent: safe to run on every stack start. Requires PG14+.
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'postgres_exporter') THEN
    CREATE ROLE postgres_exporter LOGIN;
  END IF;
END
$$;

-- Local development password only. Never reuse this outside the dev stack.
ALTER ROLE postgres_exporter PASSWORD 'postgres_exporter';

-- pg_monitor = pg_read_all_settings + pg_read_all_stats + pg_stat_scan_tables.
GRANT pg_monitor TO postgres_exporter;

-- Server-side guard rails.
ALTER ROLE postgres_exporter SET statement_timeout = '5s';
ALTER ROLE postgres_exporter SET lock_timeout = '1s';
ALTER ROLE postgres_exporter SET idle_in_transaction_session_timeout = '10s';
ALTER ROLE postgres_exporter SET default_transaction_read_only = on;
ALTER ROLE postgres_exporter SET application_name = 'postgres_exporter';
ALTER ROLE postgres_exporter CONNECTION LIMIT 5;

CREATE EXTENSION IF NOT EXISTS pg_stat_statements;
