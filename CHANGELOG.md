## Version 0.3.0 / 2019-05-12

[full changelog](https://github.com/rnaveiras/postgres_exporter/compare/v0.3.0...v0.2.5)

* Change connection model ([#24](https://github.com/rnaveiras/postgres_exporter/pull/24))

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
