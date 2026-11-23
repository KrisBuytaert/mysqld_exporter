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

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/promslog"
)

// PMM-13962, PMM-14066: passwords written by pmm-agent into .my.cnf may
// contain '#', '?', '@' and a '$' that is not followed by a variable name.
func TestPerconaSpecialCharsInPassword(t *testing.T) {
	c, err := NewAuthConfigHandler(prometheus.NewRegistry())
	if err != nil {
		t.Fatal(err)
	}
	if err := c.ReloadConfig("testdata/percona_special_chars.cnf", "localhost:3306", "", false, promslog.NewNopLogger()); err != nil {
		t.Fatal(err)
	}
	for section, want := range map[string]string{
		"client":     "test:UfY9s73Gx`~!?@#$%^&*(){}[]<>|/:;,.-_+=@tcp(server2:3306)/",
		"client.pmm": "pmm:Exp#1$%x@tcp(server2:3306)/",
	} {
		dsn, err := c.GetConfig().Sections[section].FormDSN("", section)
		if err != nil {
			t.Fatal(err)
		}
		if dsn != want {
			t.Errorf("[%s] dsn = %q, want %q", section, dsn, want)
		}
	}
}
