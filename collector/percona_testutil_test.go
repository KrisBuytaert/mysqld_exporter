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
	"fmt"
	"strconv"
	"strings"
)

// newInstanceFromDB builds an instance around an already opened *sql.DB (for
// example one wrapped by the query recorder), detecting version and flavor
// the same way upstream's newInstance does.
func newInstanceFromDB(ctx context.Context, db *sql.DB) (*instance, error) {
	version, versionString, err := queryVersion(ctx, db)
	if err != nil {
		return nil, err
	}
	i := &instance{db: db, version: version, flavor: FlavorMySQL}
	if i.versionMajorMinor, err = strconv.ParseFloat(fmt.Sprintf("%d.%d", version.Major, version.Minor), 64); err != nil {
		return nil, err
	}
	if strings.Contains(strings.ToLower(versionString), "mariadb") {
		i.flavor = FlavorMariaDB
	}
	return i, nil
}
