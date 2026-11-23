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
	"context"
	"database/sql"
	"errors"

	"github.com/blang/semver/v4"
	"github.com/go-sql-driver/mysql"
)

// PERCONA: replica status statement selection by server flavor (PMM-7,
// percona/mysqld_exporter#429), used by the marked call site in
// slave_status.go. Upstream probes every statement and suffix on every scrape,
// which sends statements the server is known to reject (and logs syntax
// errors on the server). Upstream v0.21.0 (#958) only added MariaDB GTID
// parsing on top of that loop; the selection below is not covered upstream.

// erParseError is the MySQL/MariaDB error number for a syntax error (ER_PARSE_ERROR).
const erParseError = 1064

// MySQL 8.4 removed SHOW SLAVE STATUS. SHOW REPLICA STATUS exists since 8.0.22, but it
// renames columns (Slave_IO_Running -> Replica_IO_Running, Seconds_Behind_Master ->
// Seconds_Behind_Source), and metric names are derived from column names, so it is only
// used where SHOW SLAVE STATUS is no longer available.
var replicaStatusMinVersion = semver.MustParse("8.4.0")

// slaveStatusQuery returns the statement the server is known to support, based on the
// flavor and version detected for the instance. It returns "" when the flavor is unknown.
func slaveStatusQuery(instance *instance) string {
	switch {
	case instance.flavor == FlavorMariaDB:
		// Only the ALL form returns every named connection of a multi-source replica.
		return "SHOW ALL SLAVES STATUS"
	case instance.flavor == FlavorMySQL && instance.version.GTE(replicaStatusMinVersion):
		return "SHOW REPLICA STATUS"
	case instance.flavor == FlavorMySQL:
		return "SHOW SLAVE STATUS"
	default:
		return ""
	}
}

func isParseError(err error) bool {
	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == erParseError
}

// slaveStatusProbeSuffixes tries the plain statement first, then upstream's
// lock-free suffixes.
func slaveStatusProbeSuffixes() []string {
	suffixes := []string{""}
	for _, s := range slaveStatusQuerySuffixes {
		if s != "" {
			suffixes = append(suffixes, s)
		}
	}
	return suffixes
}

// querySlaveStatus runs the statement matching the detected flavor and version, and only
// probes the other syntaxes (MariaDB, MySQL, lock-free variants) when that statement is
// rejected as a syntax error or the flavor is unknown.
func querySlaveStatus(ctx context.Context, instance *instance) (*sql.Rows, error) {
	db := instance.getDB()
	preferred := slaveStatusQuery(instance)
	if preferred != "" {
		rows, err := db.QueryContext(ctx, preferred)
		if err == nil || !isParseError(err) {
			return rows, err
		}
	}

	var err error
	for _, query := range slaveStatusQueries {
		for _, suffix := range slaveStatusProbeSuffixes() {
			statement := query + suffix
			if statement == preferred {
				continue
			}
			var rows *sql.Rows
			if rows, err = db.QueryContext(ctx, statement); err == nil {
				return rows, nil
			}
		}
	}
	return nil, err
}
