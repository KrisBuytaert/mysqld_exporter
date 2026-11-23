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
	"testing"

	"github.com/percona/mysqld_exporter/config"
)

// Every scraper must have a unique name that config.DefaultCollectorConfig
// knows, otherwise config.Validate rejects it and no flag default exists.
func TestPerconaScraperNamesKnownToConfig(t *testing.T) {
	defaults := config.DefaultCollectorConfig()
	seen := map[string]bool{}
	for _, s := range AllScrapers(config.NewConfigWithDefaults()) {
		name := s.Name()
		if seen[name] {
			t.Errorf("duplicate scraper name %q", name)
		}
		seen[name] = true
		if _, ok := defaults[name]; !ok {
			t.Errorf("scraper %q missing from config.DefaultCollectorConfig", name)
		}
	}
	for name := range defaults {
		if !seen[name] {
			t.Errorf("collector %q in config.DefaultCollectorConfig has no scraper", name)
		}
	}
}

func TestPrometheusVariantName(t *testing.T) {
	s := prometheusVariant{ScrapeGlobalStatus{}}
	if got, want := s.Name(), "global_status.prometheus"; got != want {
		t.Errorf("Name() = %q, want %q", got, want)
	}
}
