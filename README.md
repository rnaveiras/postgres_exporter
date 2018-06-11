# Postgres exporter

Prometheus exporter for PostgreSQL server metrics.

## Collectors

- stat_activity
- stat_archiver
- stat_bgwriter
- stat_database
- stat_replication
- info
- locks

## Exported Metrics

| Metric | Meaning | Labels |
| ------ | ------- | ------ |
| postgres_info| Postgres version | version |
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
| postgres_stat_activity_connections | Number of current connections in their current state | datname, state |
| postgres_up | Whether the Postgres server is up | |
| postgres_in_recovery | Whether Postgres is in recovery | |
| postgres_stat_activity_oldest_backend_timestamp| Oldest backend timestamp (epoch) | |
| postgres_stat_activity_oldest_xact_seconds | Oldest transaction | |
| postgres_stat_activity_oldest_query_active_seconds| Oldest query in running state | |
| postgres_stat_activity_oldest_snapshot_seconds | Oldest Snapshot | |
| postgres_stat_replication_lag_bytes | Replication Lag in bytes | application_name, client_addr, state, sync_state |
| postgres_stat_bgwriter_checkpoints_timed_total | Number of scheduled
checkpoints that have been performed | |
| postgres_stat_bgwriter_checkpoints_req_total | Number of requested checkpoints
that have been performed | |
| postgres_stat_bgwriter_checkpoint_write_time_seconds_total | Total amount of time that has been spent in the portion of checkpoint processing where files are written to disk | |
| postgres_stat_bgwriter_checkpoint_sync_time_seconds_total | Total amount of time that has been spent in the portion of checkpoint processing where files are synchronized to disk | |
| postgres_stat_bgwriter_buffers_checkpoint_total | Number of buffers written during checkpoints | |
| postgres_stat_bgwriter_buffers_clean_total | Number of buffers written by the background writer | |
| postgres_stat_bgwriter_maxwritten_clean_total | Number of times the background writter stopped a cleaning scan because it had written too many buffers | |
| postgres_stat_bgwriter_buffers_backend_total | Number of buffers written directly  by a backend | |
| postgres_stat_bgwriter_buffers_backend_fsync_total | Number of times a backend had to execute its own fsync call | |
| postgres_stat_bgwriter_buffers_allow_total | Number of buffers allocated | |
| postgres_stat_bgwriter_stats_reset_timestamp | Time at wich these statistics were last reset | |
| postgres_stat_archiver_archived_total | Number of WAL files that have been successfully archived | |
| postgres_stat_archiver_failed_total   | Number of failed attempts for archiving WAL files | |
| postgres_stat_archiveR_stats_reset_timestamp | Time at which these statistics were last reset | |

### Run

```
./postgres_exporter \
    --db.data-source="user=postgres host=/var/run/postgresql"
```
