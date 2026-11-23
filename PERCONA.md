# Percona divergences from prometheus/mysqld_exporter

This fork is based on upstream **v0.21.0** with real git ancestry, so the next
sync is `git merge v0.2x.0`. This file lists every intentional divergence. It is
the checklist for that merge: an edit to an upstream file needs a `// PERCONA:`
marker and an entry here.

## Where Percona code lives

| What | Where |
|---|---|
| Scraper replacements and Percona-only scrapers | `collector/percona_registry.go` (+ `init()` in each Percona scraper file) |
| Extra collector names and defaults | `config/percona.go` |
| Main-package hooks (flags, handler, gatherers) | `percona_main.go` |
| Percona flags used by the collector package | `collector/percona_options.go` |
| Percona tests | `collector/percona_*_test.go`, `config/percona_config_test.go`, `percona_main_test.go`, `percona/tests/` |

## Marked call sites in upstream files

| File | Call site | Ticket |
|---|---|---|
| `collector/runtime.go` | `AllScrapers` returns `perconaScrapers(...)` | PMM-2038 |
| `config/config.go` | `DefaultCollectorConfig` returns `perconaCollectorDefaults(...)` | PMM-2038 |
| `mysqld_exporter.go` | `--config.my-cnf` default, flag defaults off, `perconaConfigure`, `perconaGatherers`, `perconaHandlerOpts`, `perconaWrapHandler`, pprof import | PMM-2038, PMM-14440, PMM-14686 |
| `collector/exporter.go` | `dsnHasParams`, `logScraperError`, `perconaPprofLabels` | PMM-13962, PMM-10049, PMM-14686 |
| `collector/instance.go` | `perconaTunePool` | PMM-2038 |
| `collector/global_status.go` | `innodbRedoStatus` observe/collect | PMM-15195 |
| `collector/info_schema_innodb_metrics.go` | stable aliases and gauge overrides | PMM-15195 |
| `collector/info_schema_auto_increment.go` | `'%unsigned%'` | PMM-15161 |
| `collector/info_schema_tables.go` | `information_schema_stats_expiry=0` | PMM-5684 |
| `collector/slave_status.go` | `querySlaveStatus` | PMM-7 (#429) |

Plus the module path `github.com/percona/mysqld_exporter` in every import.

## Behaviour that differs from upstream

| Topic | Percona | Upstream | Ticket |
|---|---|---|---|
| `--collect.global_status` | Percona implementation (`PScrapeGlobalStatus`); upstream is `global_status.prometheus` | upstream | PMM-2038 |
| `--collect.info_schema.processlist` | Percona (normalised states, performance_schema path on 8.0.22+/5.7.39+); upstream is `.prometheus` with `--collect.info_schema.processlist.min_time.prometheus` | upstream | PMM-2038, PMM-12392 |
| `--collect.info_schema.innodb_cmp`, `innodb_cmpmem` | Percona; upstream is `.prometheus` | upstream | PMM-2038 |
| `--collect.plugins` | Percona-only | absent | PMM-11267 |
| `--collect.custom_query.{hr,mr,lr}` (+ `.directory`) | Percona-only, 429 on overlapping request per resolution | absent | PMM-2038, PMM-14958 |
| `--collect.standard.{go,process}` | Percona-only; default gatherer dropped when selected | absent | PMM-14440 |
| InnoDB redo log `mysql_innodb_redo_log_*` and `Innodb_lsn_*` aliases | emitted by both global_status scrapers | absent | PMM-15195 |
| INNODB_METRICS stable `_total` aliases | emitted | absent | PMM-15195 |
| Collector defaults | all off; `--collect.all` | small default set on | PMM-2038 |
| Scraper error log level | debug; `--collector.failures.error` for error | error | PMM-10049 |
| Partial failure | `ContinueOnError` | HTTP 500 | PMM-2038 |
| `--config.my-cnf` default | `$HOME/.my.cnf` | `.my.cnf` | PMM-2038 |
| Connections | `--exporter.max-open-conns` (3, used unless `--exporter.max_open_connections` is given), `--exporter.max-idle-conns` (3), `--exporter.conn-max-lifetime` (1m) | `max_open_connections` 2, 1 idle | PMM-2038 |
| Replica status statement | chosen by flavor/version | probed every scrape | PMM-7 |
| `/debug/pprof` | served, scraper goroutine labels | not served | PMM-14686 |

## Candidates to send upstream

plugins collector, processlist from performance_schema, InnoDB redo log
unification, INNODB_METRICS stable names, auto_increment unsigned fix,
info_schema.tables stats expiry, replica status statement by flavor, DSN `?`
detection, pprof goroutine labels.
