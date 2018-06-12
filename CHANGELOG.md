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
