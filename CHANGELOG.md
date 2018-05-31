## Unreleased

[full changelog](https://github.com/rnaveiras/postgres_exporter/compare/v0.1.4...master)

* Add `oldest_query_active_seconds` metric ([#10](https://github.com/rnaveiras/postgres_exporter/pull/10))

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