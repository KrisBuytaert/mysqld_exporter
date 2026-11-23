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

package main

// PERCONA: hooks the Percona build calls from mysqld_exporter.go. Each call
// site there is marked with a "PERCONA:" comment.

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/percona/mysqld_exporter/collector"
	"github.com/percona/mysqld_exporter/config"
)

var collectAll = kingpin.Flag(
	"collect.all",
	"Collect all metrics.",
).Default("false").Bool()

// perconaDefaultMyCnf is the default for --config.my-cnf: $HOME/.my.cnf rather
// than upstream's .my.cnf in the working directory.
func perconaDefaultMyCnf() string {
	return filepath.Join(os.Getenv("HOME"), ".my.cnf")
}

// perconaDisableCollectorsByDefault turns every collector off by default.
// pmm-agent enables exactly the collectors it wants with --collect.* flags.
// It only changes the generated flag defaults; the library defaults in
// config.DefaultCollectorConfig stay upstream's.
func perconaDisableCollectorsByDefault(cfg *config.Config) {
	for name := range cfg.Collectors {
		cfg.Collectors[name] = false
	}
}

// perconaConfigure applies Percona-only flags to the parsed configuration.
func perconaConfigure(cfg *config.Config, logger *slog.Logger) error {
	if *collectAll {
		for name := range cfg.Collectors {
			cfg.Collectors[name] = true
		}
	}

	// pmm-agent passes --exporter.max-open-conns (Percona, default 3). Upstream
	// v0.21 added --exporter.max_open_connections (default 2). The upstream flag
	// wins only when it is given explicitly.
	if !perconaFlagSetByUser("exporter.max_open_connections") {
		n := collector.PerconaMaxOpenConns()
		if n < 1 {
			return fmt.Errorf("invalid value for --exporter.max-open-conns, must be >= 1: %d", n)
		}
		cfg.ExporterMaxOpenConns = n
	}
	logger.Debug("Connection pool", "max_open_connections", cfg.ExporterMaxOpenConns)
	return nil
}

// perconaFlagSetByUser reports whether a flag was given on the command line.
func perconaFlagSetByUser(name string) bool {
	ctx, err := kingpin.CommandLine.ParseContext(os.Args[1:])
	if err != nil {
		return false
	}
	for _, el := range ctx.Elements {
		if f, ok := el.Clause.(*kingpin.FlagClause); ok && f.Model().Name == name {
			return true
		}
	}
	return false
}

type errLogger struct {
	logger *slog.Logger
}

func (el *errLogger) Println(v ...any) {
	el.logger.Error(fmt.Sprint(v...))
}

var _ promhttp.Logger = &errLogger{}

// perconaHandlerOpts keeps serving the metrics of the collectors that
// succeeded when one fails (for example on a duplicate series), instead of
// upstream's default of answering the whole scrape with HTTP 500.
func perconaHandlerOpts(logger *slog.Logger) promhttp.HandlerOpts {
	opts := promhttp.HandlerOpts{
		ErrorHandling: promhttp.ContinueOnError,
	}
	if logger.Enabled(context.Background(), slog.LevelInfo) {
		opts.ErrorLog = &errLogger{logger: logger}
	}
	return opts
}

// perconaGatherers returns the gatherers for one /metrics request. When the
// standard.go or standard.process scraper is selected it already exposes the
// exporter's runtime and process metrics, so the default gatherer is dropped
// to avoid duplicate series (PMM-14440).
func perconaGatherers(cfg config.Config, registry *prometheus.Registry) prometheus.Gatherers {
	if cfg.Collectors["standard.go"] || cfg.Collectors["standard.process"] {
		return prometheus.Gatherers{registry}
	}
	return prometheus.Gatherers{prometheus.DefaultGatherer, registry}
}

// perconaWrapHandler wraps the upstream /metrics handler.
//
// pmm-agent scrapes each custom_query resolution on its own request. A slow
// custom query must not pile up concurrent executions on the database, so a
// request for collect[]=custom_query.{hr,mr,lr} is refused with 429 while the
// previous request for the same resolution is still running.
func perconaWrapHandler(next http.HandlerFunc, logger *slog.Logger) http.HandlerFunc {
	inFlight := map[string]*atomic.Bool{
		"custom_query.hr": {},
		"custom_query.mr": {},
		"custom_query.lr": {},
	}
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		q := r.URL.Query()
		queryCollect := q.Get("collect[]")
		if collect := q["collect[]"]; len(collect) > 0 {
			logger.Info("collect[]", "params", strings.Join(collect, ","))
		}
		defer func() {
			logger.Debug("Request elapsed time", "sinceStart", time.Since(start), "query_collect", queryCollect)
		}()
		if busy, ok := inFlight[queryCollect]; ok {
			if !busy.CompareAndSwap(false, true) {
				logger.Warn("Received metrics request while previous still in progress: returning 429 Too Many Requests", "collect", queryCollect)
				http.Error(w, "429 Too Many Requests", http.StatusTooManyRequests)
				return
			}
			defer busy.Store(false)
		}
		next(w, r)
	}
}
