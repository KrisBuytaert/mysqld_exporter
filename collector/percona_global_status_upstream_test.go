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
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/promslog"
)

// The upstream global_status scraper (global_status.prometheus) carries the
// same redo log aggregation through the PERCONA: call sites in global_status.go.

// The redo log aggregation runs on keys that have already been through
// validPrometheusName, so it is driven here with the capitalisation a real
// server reports. Without that lowercasing every case in observe() would miss
// and all canonical and compatibility redo metrics would silently disappear.
func TestScrapeGlobalStatusInnodbRedoAliases(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("error opening a stub database connection: %s", err)
	}
	defer db.Close()
	inst := &instance{db: db}

	rows := sqlmock.NewRows([]string{"Variable_name", "Value"}).
		AddRow("Innodb_redo_log_current_lsn", "1500").
		AddRow("Innodb_redo_log_checkpoint_lsn", "1000").
		AddRow("Innodb_redo_log_flushed_to_disk_lsn", "1400").
		AddRow("Innodb_redo_log_capacity_resized", "2000").
		AddRow("Innodb_os_log_written", "4096")
	mock.ExpectQuery(sanitizeQuery(globalStatusQuery)).WillReturnRows(rows)

	ch := make(chan prometheus.Metric)
	go func() {
		defer close(ch)
		if scrapeErr := (ScrapeGlobalStatus{}).Scrape(context.Background(), inst, ch, promslog.NewNopLogger()); scrapeErr != nil {
			t.Errorf("error calling function on test: %s", scrapeErr)
		}
	}()

	collected := metricsByName(t, ch)

	assertMetric(t, collected, "mysql_innodb_redo_log_current_lsn", 1500, dto.MetricType_GAUGE)
	assertMetric(t, collected, "mysql_innodb_redo_log_checkpoint_lsn", 1000, dto.MetricType_GAUGE)
	assertMetric(t, collected, "mysql_innodb_redo_log_checkpoint_age_bytes", 500, dto.MetricType_GAUGE)
	assertMetric(t, collected, "mysql_innodb_redo_log_flushed_lsn", 1400, dto.MetricType_GAUGE)
	assertMetric(t, collected, "mysql_innodb_redo_log_checkpoint_age_ratio", 0.25, dto.MetricType_GAUGE)
	assertMetric(t, collected, "mysql_innodb_redo_log_capacity_bytes", 2000, dto.MetricType_GAUGE)
	assertMetric(t, collected, "mysql_innodb_redo_log_written_bytes_total", 4096, dto.MetricType_COUNTER)

	assertMetric(t, collected, "mysql_global_status_innodb_lsn_current", 1500, dto.MetricType_GAUGE)
	assertMetric(t, collected, "mysql_global_status_innodb_lsn_last_checkpoint", 1000, dto.MetricType_GAUGE)
	assertMetric(t, collected, "mysql_global_status_innodb_lsn_flushed", 1400, dto.MetricType_GAUGE)
	assertMetric(t, collected, "mysql_global_status_innodb_checkpoint_age", 500, dto.MetricType_GAUGE)

	// The rows themselves keep their generic series alongside the aliases.
	assertMetric(t, collected, "mysql_global_status_innodb_redo_log_current_lsn", 1500, dto.MetricType_UNTYPED)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled exceptions: %s", err)
	}
}

// Percona Server 8.0.30 and newer publish the historical Innodb_lsn_* and
// Innodb_checkpoint_* status variables alongside the Oracle Innodb_redo_log_*
// ones. The generic global status path already exports the historical names, so
// the compatibility aliases have to stay silent here: two metrics sharing one
// fqName make Gather() drop the whole family, which blanks every panel querying
// it instead of surfacing an error. metricsByName fails if a name is collected
// twice, so this pins the guard through Scrape rather than through collect()
// alone.
func TestScrapeGlobalStatusInnodbRedoNativeNamesNotAliased(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("error opening a stub database connection: %s", err)
	}
	defer db.Close()
	inst := &instance{db: db}

	rows := sqlmock.NewRows([]string{"Variable_name", "Value"}).
		AddRow("Innodb_lsn_current", "1500").
		AddRow("Innodb_lsn_last_checkpoint", "1000").
		AddRow("Innodb_lsn_flushed", "1400").
		AddRow("Innodb_checkpoint_age", "500").
		AddRow("Innodb_checkpoint_max_age", "1600").
		AddRow("Innodb_redo_log_current_lsn", "9501").
		AddRow("Innodb_redo_log_checkpoint_lsn", "9001").
		AddRow("Innodb_redo_log_flushed_to_disk_lsn", "9401").
		AddRow("Innodb_redo_log_capacity_resized", "2000").
		AddRow("Innodb_os_log_written", "4096")
	mock.ExpectQuery(sanitizeQuery(globalStatusQuery)).WillReturnRows(rows)

	ch := make(chan prometheus.Metric)
	go func() {
		defer close(ch)
		if scrapeErr := (ScrapeGlobalStatus{}).Scrape(context.Background(), inst, ch, promslog.NewNopLogger()); scrapeErr != nil {
			t.Errorf("error calling function on test: %s", scrapeErr)
		}
	}()

	collected := metricsByName(t, ch)

	// The canonical series carry the native values, not the Oracle ones.
	assertMetric(t, collected, "mysql_innodb_redo_log_current_lsn", 1500, dto.MetricType_GAUGE)
	assertMetric(t, collected, "mysql_innodb_redo_log_checkpoint_lsn", 1000, dto.MetricType_GAUGE)
	assertMetric(t, collected, "mysql_innodb_redo_log_flushed_lsn", 1400, dto.MetricType_GAUGE)
	assertMetric(t, collected, "mysql_innodb_redo_log_checkpoint_age_bytes", 500, dto.MetricType_GAUGE)
	assertMetric(t, collected, "mysql_innodb_redo_log_checkpoint_age_ratio", 0.25, dto.MetricType_GAUGE)
	assertMetric(t, collected, "mysql_innodb_redo_log_capacity_bytes", 2000, dto.MetricType_GAUGE)

	// The historical names come from the generic path, which emits them untyped.
	// An alias would have published the same fqName as a gauge.
	assertMetric(t, collected, "mysql_global_status_innodb_lsn_current", 1500, dto.MetricType_UNTYPED)
	assertMetric(t, collected, "mysql_global_status_innodb_lsn_last_checkpoint", 1000, dto.MetricType_UNTYPED)
	assertMetric(t, collected, "mysql_global_status_innodb_lsn_flushed", 1400, dto.MetricType_UNTYPED)
	assertMetric(t, collected, "mysql_global_status_innodb_checkpoint_age", 500, dto.MetricType_UNTYPED)
	assertMetric(t, collected, "mysql_global_status_innodb_checkpoint_max_age", 1600, dto.MetricType_UNTYPED)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled exceptions: %s", err)
	}
}
