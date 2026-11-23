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

package config

import "maps"

// PERCONA: collector names the Percona build adds to upstream's
// DefaultCollectorConfig, with their default state. Keep in sync with
// collector/percona_registry.go.
var perconaCollectors = map[string]bool{
	// Exporter's own Go runtime and process metrics (PMM-2038, PMM-14440).
	"standard.go":      false,
	"standard.process": false,
	// User-defined queries, one directory per resolution (PMM-2038).
	"custom_query.hr": false,
	"custom_query.mr": false,
	"custom_query.lr": false,
	// upstream global_status, taken over by the Percona implementation (PMM-2038).
	"global_status.prometheus": false,
	// upstream info_schema.processlist, taken over by the Percona implementation.
	"info_schema.processlist.prometheus": false,
	// upstream innodb_cmp/innodb_cmpmem, taken over by the Percona implementations.
	"info_schema.innodb_cmp.prometheus":    false,
	"info_schema.innodb_cmpmem.prometheus": false,
	// SHOW PLUGINS (PMM-11267).
	"plugins": false,
}

// perconaCollectorDefaults extends upstream's default collector map.
func perconaCollectorDefaults(upstream map[string]bool) map[string]bool {
	maps.Copy(upstream, perconaCollectors)
	return upstream
}
