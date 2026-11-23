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

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/promslog"

	"github.com/percona/mysqld_exporter/config"
)

func TestPerconaGatherers(t *testing.T) {
	registry := prometheus.NewRegistry()
	for _, tc := range []struct {
		collectors map[string]bool
		want       int
	}{
		{map[string]bool{"global_status": true}, 2},
		{map[string]bool{"standard.go": true}, 1},
		{map[string]bool{"standard.process": true}, 1},
	} {
		cfg := config.NewConfigWithDefaults()
		cfg.Collectors = tc.collectors
		if got := len(perconaGatherers(cfg, registry)); got != tc.want {
			t.Errorf("collectors %v: got %d gatherers, want %d", tc.collectors, got, tc.want)
		}
	}
}

func TestPerconaWrapHandlerCustomQueryGuard(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	h := perconaWrapHandler(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("collect[]") == "custom_query.hr" {
			close(entered)
			<-release
		}
	}, promslog.NewNopLogger())

	done := make(chan struct{})
	go func() {
		h(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/metrics?collect[]=custom_query.hr", nil))
		close(done)
	}()
	<-entered

	rec := httptest.NewRecorder()
	h(rec, httptest.NewRequest(http.MethodGet, "/metrics?collect[]=custom_query.hr", nil))
	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("overlapping custom_query.hr request: got %d, want 429", rec.Code)
	}

	// Other resolutions are not blocked.
	rec = httptest.NewRecorder()
	h(rec, httptest.NewRequest(http.MethodGet, "/metrics?collect[]=custom_query.mr", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("custom_query.mr request: got %d, want 200", rec.Code)
	}

	close(release)
	<-done
	// Once the first request finished, the resolution is free again.
	rec = httptest.NewRecorder()
	h(rec, httptest.NewRequest(http.MethodGet, "/metrics?collect[]=global_status", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("global_status request: got %d, want 200", rec.Code)
	}
}
