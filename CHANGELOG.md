## unreleased

PostgreSQL 14 to 18 are supported. Every collector now works on PostgreSQL 15, 16, 17 and 18.

* [CHANGE] `postgres_stat_vacuum_progress_*` lose the `pid` and `query_start` labels; series are identified by
  `datname`, `schemaname` and `relname`. Remove `pid`/`query_start` from selectors and `by (...)` clauses.
* [CHANGE] On PostgreSQL 17+ `postgres_stat_vacuum_progress_max_dead_tuples` and `..._num_dead_tuples` are replaced by
  `..._max_dead_tuple_bytes`, `..._dead_tuple_bytes` and `..._num_dead_item_ids`, following the view.
* [CHANGE] On PostgreSQL 17+ `postgres_stat_bgwriter_checkpoints_*`, `..._checkpoint_*_time_seconds_total`,
  `..._buffers_checkpoint_total`, `..._buffers_backend_total` and `..._buffers_backend_fsync_total` are not emitted.
  Checkpoint metrics come from the new `postgres_stat_checkpointer_*`, e.g.
  `rate(postgres_stat_bgwriter_checkpoints_timed_total[5m])` becomes
  `rate(postgres_stat_checkpointer_num_timed_total[5m])`.
* [CHANGE] `postgres_is_in_backup` is not emitted on PostgreSQL 15+, where `pg_is_in_backup()` no longer exists.
* [CHANGE] `postgres_stat_user_tables_n_tup_hot_upd` is deprecated in favour of
  `postgres_stat_user_tables_n_tup_hot_upd_total`. Both are emitted while `--compat.legacy-names` is on (the
  default); the old name will be removed.
* [CHANGE] `--db.excluded-databases` defaults to `cloudsqladmin`, `rdsadmin`, `azure_maintenance` and `azure_sys`.
  The previous default misspelled `cloudsqladmin`.
* [CHANGE] `--web.enabled-pprof` is renamed to `--web.enable-pprof`. The old name was never in a release, so it is not
  kept as an alias.
* [CHANGE] The command moved to `cmd/postgres_exporter`; install it with
  `go install github.com/rnaveiras/postgres_exporter/cmd/postgres_exporter@latest`.
* [CHANGE] The unauthenticated `/admin/loglevel` endpoint is removed. Set the level with `--log.level`, or at
  runtime with `/-/log-level` behind `--web.enable-admin-api`.
* [CHANGE] PostgreSQL 9.x and 10–13 code paths are removed. PostgreSQL 13 and older are scraped on a best-effort basis
  and log a warning.
* [FEATURE] `postgres_stat_checkpointer_*` collector for PostgreSQL 17+, with `num_done` and `slru_written` on 18+.
* [FEATURE] `/-/healthy` and `/-/ready` endpoints for liveness and readiness probes. They never query Postgres.
* [FEATURE] `--web.enable-admin-api` flag (default `false`) serves `/-/log-level`: `GET` returns the log level,
  `PUT`/`POST` `{"level":"debug","for":"15m"}` changes it, reverting after `for` when set. It has no authentication.
* [FEATURE] `--compat.legacy-names` flag (default `true`) to keep emitting deprecated metric names.
* [ENHANCEMENT] `postgres_info` gains the `version_num` (`server_version_num`) and `platform` (`community`, `rds`,
  `aurora`, `cloudsql`, `azure_flexible`) labels; `version` is the server's `server_version` without the packager
  suffix, e.g. `17.6` for `17.6 (Debian 17.6-1.pgdg12+1)`.
* [ENHANCEMENT] `postgres_exporter_unsupported_version` and `postgres_exporter_untested_version` gauges.
* [ENHANCEMENT] A database the exporter cannot connect to is reported as
  `postgres_exporter_scraper_success{scraper="connect",datname="..."} 0` and no longer stops the other databases.
* [ENHANCEMENT] A failed database list is reported as `postgres_exporter_scraper_success{scraper="list_databases"} 0`;
  before, every per-database series vanished with only a log line.
* [BUGFIX] The info, bgwriter and vacuum-progress collectors failed on PostgreSQL 15+ / 17+.
* [BUGFIX] `postgres_stat_database_tup_deleted_total` reported the updated-rows count.
* [BUGFIX] `postgres_info{version}` truncated minor versions of 10 or more (`16.10` was reported as `16.1`) and
  was `0` on beta releases.
* [BUGFIX] The disk-usage collector failed for the whole database when a table had a mixed-case, quoted or dotted
  name.
* [BUGFIX] The disk-usage collector failed for the whole database when a table or index was dropped during the scrape,
  e.g. another session's temporary table.
* [BUGFIX] `postgres_up` was `1` for a server that accepted the connection but could not run queries; it is now `0`.
* [BUGFIX] Tables without indexes were missing from `postgres_stat_user_tables_*`.
* [BUGFIX] The archiver collector failed on clusters whose archiver statistics were never reset.
* [BUGFIX] `postgres_stat_replication_lag_bytes` reported `client_addr="<nil>"` for replicas connected over a Unix
  socket; the label is now empty.
* [BUGFIX] After scraping a database that was later dropped, the exporter reported `postgres_up 0` until restarted,
  and the global collectors ran against the last scraped database.
* [BUGFIX] With an empty `--db.excluded-databases` list no database was scraped.
* [BUGFIX] `pprof` endpoints under `/debug/pprof/` were unreachable.
* [BUGFIX] The exporter kept running without serving when `--web.listen-address` could not be bound, or when the HTTP
  server stopped; it now exits with an error.

## Version 0.10.0 / 2022-02-23

[full changelog](https://github.com/rnaveiras/postgres_exporter/compare/v0.9.0...v0.10.0)

* Add metrics from pg_stat_user_indexes ([#88](https://github.com/rnaveiras/postgres_exporter/pull/88))
* Replace Replace go-kit/kit with go-kit/log ([#94](https://github.com/rnaveiras/postgres_exporter/pull/95))
* Support postgresql 12/13 ([#85](https://github.com/rnaveiras/postgres_exporter/pull/85))

## Version 0.9.0 / 2021-04-13

[full changelog](https://github.com/rnaveiras/postgres_exporter/compare/v0.8.0...v0.9.0)

* Add disk usage metrics([#83](https://github.com/rnaveiras/postgres_exporter/pull/83))
* Remove vendor depedencies ([#75](https://github.com/rnaveiras/postgres_exporter/pull/75))

## Version 0.8.0 / 2020-02-27

[full changelog](https://github.com/rnaveiras/postgres_exporter/compare/v0.7.0...v0.8.0)

* Add stat_activity_oldest_backend_xmin ([#38](https://github.com/rnaveiras/postgres_exporter/pull/38))
* Bump github.com/prometheus/client_golang from 1.3.0 to 1.4.1 ([#40](https://github.com/rnaveiras/postgres_exporter/pull/40))
* Bump github.com/go-kit/kit from 0.9.0 to 0.10.0 ([#42](https://github.com/rnaveiras/postgres_exporter/pull/42))

## Version 0.7.0 / 2020-01-16

[full changelog](https://github.com/rnaveiras/postgres_exporter/compare/v0.6.0...v0.7.0)

* Amend oldest query active and oldest snapshot ([#32](https://github.com/rnaveiras/postgres_exporter/pull/32))

## Version 0.6.0 / 2020-01-13

[full changelog](https://github.com/rnaveiras/postgres_exporter/compare/v0.5.0...v0.6.0)

* Add multi-db support ([#31](https://github.com/rnaveiras/postgres_exporter/pull/31))
* Update go-kit ([#29](https://github.com/rnaveiras/postgres_exporter/pull/29))
* go mod ([#28](https://github.com/rnaveiras/postgres_exporter/pull/28))

## Version 0.5.0 / 2020-01-06

[full changelog](https://github.com/rnaveiras/postgres_exporter/compare/v0.4.0...v0.5.0)

* Add vacuum metrics ([#27](https://github.com/rnaveiras/postgres_exporter/pull/27))

## Version 0.4.0 / 2019-05-19

[full changelog](https://github.com/rnaveiras/postgres_exporter/compare/v0.3.0...v0.4.0)

* Add additional info metrics ([#26](https://github.com/rnaveiras/postgres_exporter/pull/26))

## Version 0.3.0 / 2019-05-12

[full changelog](https://github.com/rnaveiras/postgres_exporter/compare/v0.2.5...v0.3.0)

* Change connection model ([#24](https://github.com/rnaveiras/postgres_exporter/pull/24))
* Add version support ([#25](https://github.com/rnaveiras/postgres_exporter/pull/25))

## Version 0.2.5 / 2019-03-07

[full changelog](https://github.com/rnaveiras/postgres_exporter/compare/v0.2.4...v0.2.5)

* Mutex to avoid issues with concurrent scrapes ([#23](https://github.com/rnaveiras/postgres_exporter/pull/23))

## Version 0.2.4 / 2018-09-20

[full changelog](https://github.com/rnaveiras/postgres_exporter/compare/v0.2.3...v0.2.4)

* Ignore vacuums in snapshot metric ([#22](https://github.com/rnaveiras/postgres_exporter/pull/22))

## Version 0.2.3 / 2018-08-29

[full changelog](https://github.com/rnaveiras/postgres_exporter/compare/v0.2.2...v0.2.3)

* Amend replication metric ([#21](https://github.com/rnaveiras/postgres_exporter/pull/21))

## Version 0.2.2 / 2018-08-27

[full changelog](https://github.com/rnaveiras/postgres_exporter/compare/v0.2.1...v0.2.2)

* Improve replication metrics
    ([#20](https://github.com/rnaveiras/postgres_exporter/pull/20))
* Use go-kit/log
    ([#19](https://github.com/rnaveiras/postgres_exporter/pull/19))
* Add support cascade replication
    ([#18](https://github.com/rnaveiras/postgres_exporter/pull/18))

## Version 0.2.1 / 2018-06-12

[full changelog](https://github.com/rnaveiras/postgres_exporter/compare/v0.2.0...v0.2.1)

* Don't exit during startup if database is unavailable
    ([#16](https://github.com/rnaveiras/postgres_exporter/pull/16))

## Version 0.2.0 / 2018-06-11

[full changelog](https://github.com/rnaveiras/postgres_exporter/compare/v0.1.4...v0.2.0)

* Add collector `stat_archiver`
    ([#15](https://github.com/rnaveiras/postgres_exporter/pull/15))
* Add collector `stat_bgwriter`
    ([#14](https://github.com/rnaveiras/postgres_exporter/pull/14))
* Use package `pgx` directly
    ([#13](https://github.com/rnaveiras/postgres_exporter/pull/13))
* Add collector `stat_replication`
    ([#12](https://github.com/rnaveiras/postgres_exporter/pull/12))
* Add metrics oldest query and oldest snapshot ([#10](https://github.com/rnaveiras/postgres_exporter/pull/10))

## Version 0.1.4 / 2018-03-20

[full changelog](https://github.com/rnaveiras/postgres_exporter/compare/v0.1.3...v0.1.4)

* Improve `postgres_stat_activity_oldest_xact_timestamp` ([#9](https://github.com/rnaveiras/postgres_exporter/pull/9))

## Version 0.1.3 / 2018-03-01

[full changelog](https://github.com/rnaveiras/postgres_exporter/compare/v0.1.2...v0.1.3)

* Metrics about the oldest transaction and backend ([#8](https://github.com/rnaveiras/postgres_exporter/pull/8))
* Added `postgres_in_recovery` metric ([#7](https://github.com/rnaveiras/postgres_exporter/pull/7))
* Replaced lib/pq with jackc/pgx ([#6](https://github.com/rnaveiras/postgres_exporter/pull/6))
* Expose locks from `pg_locks` ([#5](https://github.com/rnaveiras/postgres_exporter/pull/5))

## Version 0.1.2 / 2018-01-18

[full changelog](https://github.com/rnaveiras/postgres_exporter/compare/v0.1.1...v0.1.2)

* Added goreleaser.yml

## Version 0.1.1 / 2018-01-18

[full changelog](https://github.com/rnaveiras/postgres_exporter/compare/v0.1.0...v0.1.1)

* Add flag for data source ([#4](https://github.com/rnaveiras/postgres_exporter/pull/4))

## Version 0.1.0 / 2018-01-18

* Update README.md
* Add StatActivityCollector ([#1](https://github.com/rnaveiras/postgres_exporter/pull/1))
* Initial version
