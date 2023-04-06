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
