// Copyright 2026 Percona LLC
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package collector

import (
	"database/sql"
	"strings"

	"github.com/alecthomas/kingpin/v2"
)

// PERCONA: flags used by the marked call sites in exporter.go and
// instance.go. They are defined in this package, as on Percona main, so
// pmm-agent's command line keeps working unchanged.
var (
	collectorFailuresAsError = kingpin.Flag(
		"collector.failures.error",
		"Log collector failures as errors (Debug by default)").Bool()

	exporterMaxOpenConnsPercona = kingpin.Flag(
		"exporter.max-open-conns",
		"Maximum number of open connections to the database per scrape. Used unless --exporter.max_open_connections is given. https://golang.org/pkg/database/sql/#DB.SetMaxOpenConns",
	).Default("3").Int()
	exporterMaxIdleConns = kingpin.Flag(
		"exporter.max-idle-conns",
		"Maximum number of connections in the idle connection pool. https://golang.org/pkg/database/sql/#DB.SetMaxIdleConns",
	).Default("3").Int()
	exporterConnMaxLifetime = kingpin.Flag(
		"exporter.conn-max-lifetime",
		"Maximum amount of time a connection may be reused. https://golang.org/pkg/database/sql/#DB.SetConnMaxLifetime",
	).Default("1m").Duration()
)

// PerconaMaxOpenConns returns the value of --exporter.max-open-conns.
func PerconaMaxOpenConns() int {
	return *exporterMaxOpenConnsPercona
}

// perconaTunePool applies --exporter.max-idle-conns and
// --exporter.conn-max-lifetime. Upstream keeps a single idle connection, which
// makes parallel scrapers reopen connections within one scrape. Unparsed flags
// (unit tests) leave upstream's settings in place.
func perconaTunePool(db *sql.DB) {
	if *exporterMaxIdleConns > 0 {
		db.SetMaxIdleConns(*exporterMaxIdleConns)
	}
	if *exporterConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(*exporterConnMaxLifetime)
	}
}

// logScraperError logs a scraper failure at debug level unless
// --collector.failures.error is set (PMM-10049): PMM enables collectors that
// do not apply to every server, which would otherwise flood the log.
func (e *Exporter) logScraperError(scraper Scraper, err error) {
	log := e.logger.Debug
	if *collectorFailuresAsError {
		log = e.logger.Error
	}
	log("Error from scraper", "scraper", scraper.Name(), "target", e.getTargetFromDsn(), "err", err)
}

// dsnHasParams reports whether the DSN already has a parameter list. A plain
// strings.Contains(dsn, "?") is fooled by a '?' in the password. Like
// go-sql-driver's ParseDSN, the parameters are looked for after the last '/'.
// Percona main used a regexp that required an explicit tcp(...)/unix(...)
// address and so missed DSNs such as "user@/db?parseTime=true".
func dsnHasParams(dsn string) bool {
	i := strings.LastIndex(dsn, "/")
	return i >= 0 && strings.Contains(dsn[i:], "?")
}
