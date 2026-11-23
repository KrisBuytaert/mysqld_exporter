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

	"github.com/prometheus/common/promslog"
)

// PMM-13962, PMM-14066: a '?' or '@' in the password must not be mistaken for
// the start of the DSN parameter list.
func TestExporterDSNSpecialCharacters(t *testing.T) {
	const pw = "test:UfY9s73Gx`~!?@#$%^&*(){}[]<>|/:;,.-_+="
	for _, tc := range []struct{ dsn, want string }{
		{pw + "@tcp(localhost:3306)/", pw + "@tcp(localhost:3306)/?lock_wait_timeout=0"},
		{pw + "@tcp(localhost:3306)/mysql", pw + "@tcp(localhost:3306)/mysql?lock_wait_timeout=0"},
		{pw + "@tcp(localhost:3306)/?tls=true", pw + "@tcp(localhost:3306)/?tls=true&lock_wait_timeout=0"},
		{pw + "@tcp(localhost:3306)/test?tls=skip-verify", pw + "@tcp(localhost:3306)/test?tls=skip-verify&lock_wait_timeout=0"},
		{pw + "@unix(/run/mysqld/mysqld.sock)/", pw + "@unix(/run/mysqld/mysqld.sock)/?lock_wait_timeout=0"},
	} {
		e := New(t.Context(), tc.dsn, []Scraper{ScrapeGlobalStatus{}}, promslog.NewNopLogger(),
			EnableLockWaitTimeout(true), SetLockWaitTimeout(0))
		if e.dsn != tc.want {
			t.Errorf("New(%q).dsn = %q, want %q", tc.dsn, e.dsn, tc.want)
		}
	}
}

func TestDSNHasParams(t *testing.T) {
	for dsn, want := range map[string]bool{
		"root@/mysql":                                false,
		"root@/mysql?parseTime=true":                 true,
		"u:p?w/d@tcp(h:3306)/":                       false,
		"u:p?w/d@tcp(h:3306)/db?tls=true":            true,
		"u:p?w@unix(/run/mysqld/mysqld.sock)/":       false,
		"u:p?w@unix(/run/mysqld/mysqld.sock)/?tls=1": true,
	} {
		if got := dsnHasParams(dsn); got != want {
			t.Errorf("dsnHasParams(%q) = %v, want %v", dsn, got, want)
		}
	}
}
