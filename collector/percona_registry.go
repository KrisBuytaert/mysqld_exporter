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
	"slices"

	"github.com/percona/mysqld_exporter/config"
)

// PERCONA: this file is the single place where the Percona build changes the
// scraper set that upstream's AllScrapers returns.
//
//   - Percona re-implementations take over an upstream scraper's name (and so
//     its --collect.<name> flag and collect[]=<name> parameter). The upstream
//     scraper stays available under "<name>.prometheus".
//   - Percona-only scrapers are appended.
//
// Every name used here must also be listed in config/percona.go, which
// extends config.DefaultCollectorConfig.

// perconaScraperFactory builds a Percona scraper from the runtime config.
type perconaScraperFactory func(cfg config.Config) Scraper

// perconaReplacements maps an upstream scraper name to the Percona scraper
// that takes it over.
var perconaReplacements = map[string]perconaScraperFactory{}

// perconaExtraScrapers are Percona-only scrapers, in registration order.
var perconaExtraScrapers []perconaScraperFactory

// prometheusVariant exposes an upstream scraper under "<name>.prometheus"
// when a Percona scraper has taken over its original name.
type prometheusVariant struct {
	Scraper
}

// Name of the Scraper. Should be unique.
func (s prometheusVariant) Name() string {
	return s.Scraper.Name() + ".prometheus"
}

// Help describes the role of the Scraper.
func (s prometheusVariant) Help() string {
	return s.Scraper.Help() + " (upstream implementation)"
}

// perconaScrapers applies the Percona replacements and additions to the
// upstream scraper list.
func perconaScrapers(cfg config.Config, upstream []Scraper) []Scraper {
	res := make([]Scraper, 0, len(upstream)+len(perconaReplacements)+len(perconaExtraScrapers))
	for _, s := range upstream {
		if factory, ok := perconaReplacements[s.Name()]; ok {
			res = append(res, factory(cfg), perconaUpstreamVariant(cfg, s))
			continue
		}
		res = append(res, s)
	}
	for _, factory := range perconaExtraScrapers {
		res = append(res, factory(cfg))
	}
	return slices.Clip(res)
}

// perconaUpstreamVariant returns the ".prometheus" variant of a replaced
// upstream scraper. Scrapers that need their own settings are rebuilt here.
func perconaUpstreamVariant(cfg config.Config, s Scraper) Scraper {
	if f, ok := perconaUpstreamVariants[s.Name()]; ok {
		return prometheusVariant{f(cfg)}
	}
	return prometheusVariant{s}
}

// perconaUpstreamVariants rebuilds the upstream scraper for its ".prometheus"
// variant when it needs settings that differ from the Percona scraper's.
var perconaUpstreamVariants = map[string]perconaScraperFactory{}
