// SPDX-FileCopyrightText: 2026 faelis
//
// SPDX-License-Identifier: GPL-3.0-or-later
//
// This program is free software: you can redistribute it and/or modify it
// under the terms of the GNU General Public License as published by the
// Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful, but WITHOUT
// ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or
// FITNESS FOR A PARTICULAR PURPOSE.  See the GNU General Public License for
// more details.
//
// You should have received a copy of the GNU General Public License along
// with this program.  If not, see <http://www.gnu.org/licenses/>.

package cmd

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/kubecolor/kubecolor/config"
)

// TestOutputFlagCompletion covers the "Complétion" scenario: completing the
// value of -o proposes exactly wide, custom-columns= and custom-columns-file=.
func TestOutputFlagCompletion(t *testing.T) {
	root := RootCmd(&config.Config{})
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"__complete", "pods", "-o", ""})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute(__complete): %v", err)
	}

	var got []string
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		// Skip the directive line (":6") and cobra's stderr hint.
		if strings.HasPrefix(line, ":") {
			if line != ":6" {
				t.Errorf("completion directive = %q, want %q", line, ":6")
			}
			continue
		}
		if strings.HasPrefix(line, "Completion ended") {
			continue
		}
		got = append(got, line)
	}
	want := []string{"wide", "custom-columns=", "custom-columns-file="}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("completion of -o = %q, want %q", got, want)
	}
}
