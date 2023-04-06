# Postgres exporter

Prometheus exporter for PostgreSQL server metrics.

## Collectors

- disk_usage
- stat_activity
- stat_archiver
- stat_bgwriter
- stat_database
- stat_progress_vacuum
- stat_replication
- stat_user_indexes
- stat_user_tables
- info
- locks

## Exported Metrics

| Metric | Meaning | Labels |
| ------ | ------- | ------ |
| postgres_disk_usage_index_bytes| Number of bytes used on disk to store this index | datname, schemaname, relname, indexname |
| postgres_disk_usage_table_bytes| Number of bytes used on disk to store this table | datname, schemaname, relname |
| postgres_in_recovery | Whether Postgres is in recovery | |
| postgres_info| Postgres version | version |
| postgres_stat_activity_connections | Number of current connections in their current state | datname, state |
| postgres_stat_activity_oldest_backend_timestamp| Oldest backend timestamp (epoch) | |
| postgres_stat_activity_oldest_query_active_seconds| Oldest query in running state | |
| postgres_stat_activity_oldest_snapshot_seconds | Oldest Snapshot | |
| postgres_stat_activity_oldest_xact_seconds | Oldest transaction | |
| postgres_stat_archiver_archived_total | Number of WAL files that have been successfully archived | |
| postgres_stat_archiver_failed_total   | Number of failed attempts for archiving WAL files | |
| postgres_stat_archiver_stats_reset_timestamp | Time at which these statistics were last reset | |
| postgres_stat_bgwriter_buffers_allow_total | Number of buffers allocated | |
| postgres_stat_bgwriter_buffers_backend_fsync_total | Number of times a backend had to execute its own fsync call | |
| postgres_stat_bgwriter_buffers_backend_total | Number of buffers written directly  by a backend | |
| postgres_stat_bgwriter_buffers_checkpoint_total | Number of buffers written during checkpoints | |
| postgres_stat_bgwriter_buffers_clean_total | Number of buffers written by the background writer | |
| postgres_stat_bgwriter_checkpoint_sync_time_seconds_total | Total amount of time that has been spent in the portion of checkpoint processing where files are synchronized to disk | |
| postgres_stat_bgwriter_checkpoint_write_time_seconds_total | Total amount of time that has been spent in the portion of checkpoint processing where files are written to disk | |
| postgres_stat_bgwriter_checkpoints_req_total | Number of requested checkpoints that have been performed | |
| postgres_stat_bgwriter_checkpoints_timed_total | Number of scheduled checkpoints that have been performed | |
| postgres_stat_bgwriter_maxwritten_clean_total | Number of times the background writter stopped a cleaning scan because it had written too many buffers | |
| postgres_stat_bgwriter_stats_reset_timestamp | Time at wich these statistics were last reset | |
| postgres_stat_database_blks_hit_total | Number of times disk blocks were found already in the buffer cache, so that a read was not necessary (this only includes hits in the PostgreSQL buffer cache, not the operating system's file system cache) | datname |
| postgres_stat_database_blks_read_total | Number of disk blocks read in this database | datname |
| postgres_stat_database_conflicts_total | Number of queries canceled due to conflicts with recovery in this database | datname |
| postgres_stat_database_deadlocks_total | Number of deadlocks detected in this database | datname |
| postgres_stat_database_numbackends | Number of backends currently connected to this database | datname |
| postgres_stat_database_temp_bytes_total | Total amount of data written to temporary files by queries in this database | datname |
| postgres_stat_database_temp_files_total | Number of temporary files created by queries in this database | datname |
| postgres_stat_database_tup_deleted_total | Number of rows deleted by queries in this database | datname |
| postgres_stat_database_tup_fetched_total | Number of rows fetched by queries in this database | datname |
| postgres_stat_database_tup_inserted_total | Number of rows inserted by queries in this database | datname |
| postgres_stat_database_tup_returned_total | Number of rows returned by queries in this database | datname |
| postgres_stat_database_tup_updated_total | Number of rows updated by queries in this database | datname |
| postgres_stat_database_xact_commit_total | Number of transactions in this database that have been committed | datname |
| postgres_stat_database_xact_rollback_total | Number of transactions in this database that have been rolled back | datname |
| postgres_stat_replication_lag_bytes | Replication Lag in bytes | application_name, client_addr, state, sync_state |
| postgres_stat_replication_flush_lag_seconds | Elapsed time during committed WALs from primary to the standby (WAL's has already been flushed but not yet applied). Reported from the primary node. *Only available on Posgres versions > 9x*. | application_name, client_addr, state, sync_state |
| postgres_stat_replication_replay_lag_seconds | Elapsed time during committed WALs from primary to the standby (fully committed in standby node). Reported from the primary node. *Only available on Posgres versions > 9x*. | application_name, client_addr, state, sync_state |
| postgres_stat_replication_write_lag_seconds | Elapsed time during committed WALs from primary to the standby (but not yet committed in the standby). Reported from the primary node. *Only available on Posgres versions > 9x*. | application_name, client_addr, state, sync_state |
| postgres_stat_vacuum_progress_heap_blks_scanned | Number of heap blocks scanned | pid, query_start, schemaname, datname, relname |
| postgres_stat_vacuum_progress_heap_blks_total | Total number of heap blocks in the table | pid, query_start, schemaname, datname, relname |
| postgres_stat_vacuum_progress_heap_blks_vacuumed | Number of heap blocks vacuumed | pid, query_start, schemaname, datname, relname |
| postgres_stat_vacuum_progress_index_vacuum_count | Number of completed index vacuum cycles | pid, query_start, schemaname, datname, relname |
| postgres_stat_vacuum_progress_max_dead_tuples | Number of dead tuples that we can store before needing to perform an index vacuum cycle | pid, query_start, schemaname, datname, relname |
| postgres_stat_vacuum_progress_num_dead_tuples | Number of dead tuples collected since the last index vacuum cycle | pid, query_start, schemaname, datname, relname |
| postgres_stat_vacuum_progress_phase_cleaning_up_indexes | VACUUM is currently cleaning up indexes | pid, query_start, schemaname, datname, relname |
| postgres_stat_vacuum_progress_phase_initializing | VACUUM is preparing to begin scanning the heap | pid, query_start, schemaname, datname, relnam
| postgres_stat_vacuum_progress_phase_performing_final_cleanup | VACUUM is performing final cleanup | pid, query_start, schemaname, datname, relname |
| postgres_stat_vacuum_progress_phase_scanning_heap | VACUUM is currently scanning the heap | pid, query_start, schemaname, datname, relname |
| postgres_stat_vacuum_progress_phase_truncating_heap | VACUUM is currently truncating the heap | pid, query_start, schemaname, datname, relname |
| postgres_stat_vacuum_progress_phase_vacuuming_heap | VACUUM is currently vacuuming the heap | pid, query_start, schemaname, datname, relname |
| postgres_stat_vacuum_progress_phase_vacuuming_indexes | VACUUM is currently vacuuming the indexes | pid, query_start, schemaname, datname, relname |
| postgres_stat_vacuum_progress_running | VACUUM is running | pid, query_start, schemaname, datname, relname |
| postgres_stat_user_indexes_scan_total | Number of times this index has been scanned | datname, schemaname, tablename, indexname |
| postgres_stat_user_indexes_tuple_read_total | Number of times tuples have been returned from scanning this index | datname, schemaname, tablename, indexname |
| postgres_stat_user_indexes_tuple_fetch_total | Number of live tuples fetched by scans on this index | datname, schemaname, tablename, indexname |
| postgres_wal_receiver_replay_lag_seconds | Replication lag measured in seconds on the standby. Measured as `EXTRACT (EPOCH FROM now()) - pg_last_xact_replay_timestamp()`  | status |
| postgres_wal_receiver_replay_lag_bytes | Replication lag measured in bytes on the standby. Measured as `pg_wal_lsn_diff(pg_last_wal_receive_lsn(), pg_last_wal_replay_lsn())::float`  | status |
| postgres_up | Whether the Postgres server is up | |


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
