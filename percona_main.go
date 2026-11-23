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
	"log/slog"
	"net/http"
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/percona/mysqld_exporter/config"
)

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
		queryCollect := r.URL.Query().Get("collect[]")
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
