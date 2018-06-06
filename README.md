# Postgres exporter

Prometheus exporter for PostgreSQL server metrics.

## Collectors

- TODO

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
| postgres_stat_activity_oldest_backend_timestamp| Oldest backend timestamp (epoch) |
| postgres_stat_activity_oldest_xact_seconds | Oldest transaction | |
| postgres_stat_activity_oldest_query_active_seconds| Oldest query in running
| postgres_stat_activity_oldest_snapshot_seconds| Oldest Snapshot
state |

### Run

```
./postgres_exporter \
    --db.data-source="application_name=postgres_exporter user=postgres host=/var/run/postgresql"
```
