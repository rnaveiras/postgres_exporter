# Postgres exporter

Prometheus exporter for PostgreSQL server metrics.

## Collectors

- disk_usage
- stat_activity
- stat_archiver
- stat_bgwriter
- stat_checkpointer (PostgreSQL 17+)
- stat_database
- stat_progress_vacuum
- stat_replication
- stat_user_indexes
- stat_user_tables
- info
- locks

## Exported Metrics

| Metric | Meaning | Labels | Availability |
| ------ | ------- | ------ | ------------ |
| postgres_config_last_load_time_seconds | Timestamp of the last configuration reload |  |  |
| postgres_disk_usage_index_bytes | Bytes used on disk to store this index | datname, indexname, schemaname, tablename |  |
| postgres_disk_usage_table_bytes | Bytes used on disk to store this table | datname, schemaname, tablename |  |
| postgres_exporter_scraper_duration_seconds | Duration of a scrapers scrape. | datname, scraper |  |
| postgres_exporter_scraper_success | Whether a scraper succeeded. | datname, scraper |  |
| postgres_exporter_unsupported_version | 1 when the Postgres major is older than the oldest supported major (14). |  |  |
| postgres_exporter_untested_version | 1 when the Postgres major is newer than the newest tested major (18). |  |  |
| postgres_info | Postgres server information: server_version, server_version_num and the detected platform. | platform, version, version_num |  |
| postgres_is_in_backup | True if an on-line exclusive backup is still in progress. |  | PostgreSQL 14 |
| postgres_is_in_recovery | Postgres pg_is_in_recovery() True if recovery is still in progress. |  |  |
| postgres_locks_table | Number of locks by datname, locktype, mode and granted | datname, granted, locktype, mode |  |
| postgres_start_time_seconds | Postgres start time, in seconds since the unix epoch. |  |  |
| postgres_stat_activity_connections | Number of current connections in their current state | datname, state |  |
| postgres_stat_activity_oldest_backend_timestamp | The oldest backend started timestamp |  |  |
| postgres_stat_activity_oldest_backend_xmin | The lowest backend_xmin across all backends |  |  |
| postgres_stat_activity_oldest_query_active_seconds | The oldest query in running state (long query) |  |  |
| postgres_stat_activity_oldest_snapshot_seconds | The oldest snapshot |  |  |
| postgres_stat_activity_oldest_xact_seconds | The oldest transaction (active or idle in transaction) |  |  |
| postgres_stat_archiver_archived_total | Number of WAL files that have been successfully archived |  |  |
| postgres_stat_archiver_failed_total | Number of failed attempts for archiving WAL files |  |  |
| postgres_stat_archiver_stats_reset_timestamp | Time at which these statistics were last reset |  | once the statistics have been reset |
| postgres_stat_bgwriter_buffers_alloc_total | Number of buffers allocated |  |  |
| postgres_stat_bgwriter_buffers_backend_fsync_total | Number of times a backend had to execute its own fsync call |  | PostgreSQL 14–16 |
| postgres_stat_bgwriter_buffers_backend_total | Number of buffers written directly by a backend |  | PostgreSQL 14–16 |
| postgres_stat_bgwriter_buffers_checkpoint_total | Number of buffers written during checkpoints |  | PostgreSQL 14–16 |
| postgres_stat_bgwriter_buffers_clean_total | Number of buffers written by the background writer |  |  |
| postgres_stat_bgwriter_checkpoint_sync_time_seconds_total | Total amount of time that has been spent in the portion of checkpoint processing where files are synchronized to disk |  | PostgreSQL 14–16 |
| postgres_stat_bgwriter_checkpoint_write_time_seconds_total | Total amount of time that has been spent in the portion of checkpoint processing where files are written to disk |  | PostgreSQL 14–16 |
| postgres_stat_bgwriter_checkpoints_req_total | Number of requested checkpoints that have been performed |  | PostgreSQL 14–16 |
| postgres_stat_bgwriter_checkpoints_timed_total | Number of scheduled checkpoints that have been performed |  | PostgreSQL 14–16 |
| postgres_stat_bgwriter_maxwritten_clean_total | Number of times the background writer stopped a cleaning scan because it had written too many buffers |  |  |
| postgres_stat_bgwriter_stats_reset_timestamp | Time at which these statistics were last reset |  |  |
| postgres_stat_checkpointer_buffers_written_total | Number of shared buffers written during checkpoints and restartpoints |  | PostgreSQL 17+ |
| postgres_stat_checkpointer_num_done_total | Number of checkpoints that have been performed (PostgreSQL 18+) |  | PostgreSQL 18+ |
| postgres_stat_checkpointer_num_requested_total | Number of requested checkpoints |  | PostgreSQL 17+ |
| postgres_stat_checkpointer_num_timed_total | Number of scheduled checkpoints due to timeout |  | PostgreSQL 17+ |
| postgres_stat_checkpointer_restartpoints_done_total | Number of restartpoints that have been performed |  | PostgreSQL 17+ |
| postgres_stat_checkpointer_restartpoints_req_total | Number of requested restartpoints |  | PostgreSQL 17+ |
| postgres_stat_checkpointer_restartpoints_timed_total | Number of scheduled restartpoints due to timeout or after a failed attempt to perform it |  | PostgreSQL 17+ |
| postgres_stat_checkpointer_slru_written_total | Number of SLRU buffers written during checkpoints and restartpoints (PostgreSQL 18+) |  | PostgreSQL 18+ |
| postgres_stat_checkpointer_stats_reset_timestamp_seconds | Time at which these statistics were last reset |  | PostgreSQL 17+ |
| postgres_stat_checkpointer_sync_time_seconds_total | Total time spent in the portion of processing checkpoints and restartpoints where files are synchronized to disk |  | PostgreSQL 17+ |
| postgres_stat_checkpointer_write_time_seconds_total | Total time spent in the portion of processing checkpoints and restartpoints where files are written to disk |  | PostgreSQL 17+ |
| postgres_stat_database_blks_hit_total | Number of times disk blocks were found already in the buffer cache, so that a read was not necessary (this only includes hits in the PostgreSQL buffer cache, not the operating system's file system cache) | datname |  |
| postgres_stat_database_blks_read_total | Number of disk blocks read in this database | datname |  |
| postgres_stat_database_conflicts_total | Number of queries canceled due to conflicts with recovery in this database. (Conflicts occur only on standby servers; see pg_stat_database_conflicts for details.) | datname |  |
| postgres_stat_database_deadlocks_total | Number of deadlocks detected in this database | datname |  |
| postgres_stat_database_numbackends | Number of backends currently connected to this database. This is the only column in this view that returns a value reflecting current state; all other columns return the accumulated values since the last reset. | datname |  |
| postgres_stat_database_temp_bytes_total | Total amount of data written to temporary files by queries in this database. All temporary files are counted, regardless of why the temporary file was created, and regardless of the log_temp_files setting. | datname |  |
| postgres_stat_database_temp_files_total | Number of temporary files created by queries in this database. All temporary files are counted, regardless of why the temporary file was created (e.g., sorting or hashing), and regardless of  the log_temp_files setting. | datname |  |
| postgres_stat_database_tup_deleted_total | Number of rows deleted by queries in this database | datname |  |
| postgres_stat_database_tup_fetched_total | Number of rows fetched by queries in this database | datname |  |
| postgres_stat_database_tup_inserted_total | Number of rows inserted by queries in this database | datname |  |
| postgres_stat_database_tup_returned_total | Number of rows returned by queries in this database | datname |  |
| postgres_stat_database_tup_updated_total | Number of rows updated by queries in this database | datname |  |
| postgres_stat_database_xact_commit_total | Number of transactions in this database that have been committed | datname |  |
| postgres_stat_database_xact_rollback_total | Number of transactions in this database that have been rolled back | datname |  |
| postgres_stat_replication_lag_bytes | Replication lag in bytes; client_addr is empty for replicas connected over a Unix socket | application_name, client_addr, state, sync_state | on a primary with connected replicas |
| postgres_stat_user_indexes_scan_total | Number of times this index has been scanned | datname, indexname, relname, schemaname |  |
| postgres_stat_user_indexes_tuple_fetch_total | Number of live tuples fetched by scans on this index | datname, indexname, relname, schemaname |  |
| postgres_stat_user_indexes_tuple_read_total | Number of times tuples have been returned from scanning this index | datname, indexname, relname, schemaname |  |
| postgres_stat_user_tables_analyze_total | Number of times this table has been manually analyzed | datname, relname, schemaname |  |
| postgres_stat_user_tables_autoanalyze_total | Number of times this table has been analyzed by the autovacuum daemon | datname, relname, schemaname |  |
| postgres_stat_user_tables_autovacuum_total | Number of times this table has been vacuumed by the autovacuum daemon | datname, relname, schemaname |  |
| postgres_stat_user_tables_idx_scan_total | Number of index scans initiated on this table | datname, relname, schemaname |  |
| postgres_stat_user_tables_idx_tup_fetch_total | Number of live rows fetched by index scans | datname, relname, schemaname |  |
| postgres_stat_user_tables_last_analyze_timestamp | Last time at which this table was manually analyzed | datname, relname, schemaname |  |
| postgres_stat_user_tables_last_autoanalyze_timestamp | Last time at which this table was analyzed by the autovacuum daemon | datname, relname, schemaname |  |
| postgres_stat_user_tables_last_autovacuum_timestamp | Last time at which this table was vacuumed by the autovacuum daemon | datname, relname, schemaname |  |
| postgres_stat_user_tables_last_vacuum_timestamp | Last time at which this table was manually vacuumed (not counting VACUUM FULL) | datname, relname, schemaname |  |
| postgres_stat_user_tables_n_dead_tup | Estimated number of dead rows | datname, relname, schemaname |  |
| postgres_stat_user_tables_n_live_tup | Estimated number of live rows | datname, relname, schemaname |  |
| postgres_stat_user_tables_n_mod_since_analyze | Estimated number of rows modified since this table was last analyzed | datname, relname, schemaname |  |
| postgres_stat_user_tables_n_tup_del_total | Number of rows deleted | datname, relname, schemaname |  |
| postgres_stat_user_tables_n_tup_hot_upd | (DEPRECATED) Use postgres_stat_user_tables_n_tup_hot_upd_total. Number of rows HOT updated (i.e., with no separate index update required) | datname, relname, schemaname | deprecated; only with `--compat.legacy-names` (default on) |
| postgres_stat_user_tables_n_tup_hot_upd_total | Number of rows HOT updated (i.e., with no separate index update required) | datname, relname, schemaname |  |
| postgres_stat_user_tables_n_tup_ins_total | Number of rows inserted | datname, relname, schemaname |  |
| postgres_stat_user_tables_n_tup_upd_total | Number of rows updated | datname, relname, schemaname |  |
| postgres_stat_user_tables_seq_scan_total | Number of sequential scans initiated on this table | datname, relname, schemaname |  |
| postgres_stat_user_tables_seq_tup_read_total | Number of live rows fetched by sequential scans | datname, relname, schemaname |  |
| postgres_stat_user_tables_vacuum_total | Number of times this table has been manually vacuumed (not counting VACUUM FULL) | datname, relname, schemaname |  |
| postgres_stat_vacuum_progress_dead_tuple_bytes | Amount of dead tuple data collected since the last index vacuum cycle | datname, schemaname, relname | PostgreSQL 17+ |
| postgres_stat_vacuum_progress_heap_blks_scanned | Number of heap blocks scanned | datname, schemaname, relname | while a VACUUM runs |
| postgres_stat_vacuum_progress_heap_blks_total | Total number of heap blocks in the table | datname, schemaname, relname | while a VACUUM runs |
| postgres_stat_vacuum_progress_heap_blks_vacuumed | Number of heap blocks vacuumed | datname, schemaname, relname | while a VACUUM runs |
| postgres_stat_vacuum_progress_index_vacuum_count | Number of completed index vacuum cycles | datname, schemaname, relname | while a VACUUM runs |
| postgres_stat_vacuum_progress_max_dead_tuple_bytes | Amount of dead tuple data that we can store before needing to perform an index vacuum cycle | datname, schemaname, relname | PostgreSQL 17+ |
| postgres_stat_vacuum_progress_max_dead_tuples | Number of dead tuples that we can store before needing to perform an index vacuum cycle | datname, schemaname, relname | PostgreSQL 14–16 |
| postgres_stat_vacuum_progress_num_dead_item_ids | Number of dead item identifiers collected since the last index vacuum cycle | datname, schemaname, relname | PostgreSQL 17+ |
| postgres_stat_vacuum_progress_num_dead_tuples | Number of dead tuples collected since the last index vacuum cycle | datname, schemaname, relname | PostgreSQL 14–16 |
| postgres_stat_vacuum_progress_phase_cleaning_up_indexes | VACUUM is currently cleaning up indexes | datname, schemaname, relname | while a VACUUM runs |
| postgres_stat_vacuum_progress_phase_initializing | VACUUM is preparing to begin scanning the heap | datname, schemaname, relname | while a VACUUM runs |
| postgres_stat_vacuum_progress_phase_performing_final_cleanup | VACUUM is performing final cleanup | datname, schemaname, relname | while a VACUUM runs |
| postgres_stat_vacuum_progress_phase_scanning_heap | VACUUM is currently scanning the heap | datname, schemaname, relname | while a VACUUM runs |
| postgres_stat_vacuum_progress_phase_truncating_heap | VACUUM is currently truncating the heap | datname, schemaname, relname | while a VACUUM runs |
| postgres_stat_vacuum_progress_phase_vacuuming_heap | VACUUM is currently vacuuming the heap | datname, schemaname, relname | while a VACUUM runs |
| postgres_stat_vacuum_progress_phase_vacuuming_indexes | VACUUM is currently vacuuming the indexes | datname, schemaname, relname | while a VACUUM runs |
| postgres_stat_vacuum_progress_running | VACUUM is running | datname, schemaname, relname | while a VACUUM runs |
| postgres_up | Whether the Postgres server is up. |  |  |

PostgreSQL 14 to 18 are supported. The Availability column lists metrics that depend on the Postgres major or on
server activity. Older servers are scraped on a best-effort basis and reported by `postgres_exporter_unsupported_version`.

### Run

#### Passing in a libpq connection string

```
./postgres_exporter \
    --db.data-source="user=postgres host=/var/run/postgresql"
```

#### Using the PG* environment variables

- Set the [libpq PG* envvars](https://www.postgresql.org/docs/current/libpq-envars.html) like so:

```
export PGHOST=/var/run/postgresql
export PGUSER=postgres
```

- or in a [pgservicefile](https://www.postgresql.org/docs/current/libpq-pgservice.html)

```
export PGSERVICEFILE=/var/run/cloudsql/pg_service.conf
```

- then, invoke the `postgres_exporter` binary

```
./postgres_exporter
```
