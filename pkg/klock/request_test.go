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

package klock

import (
	"strings"
	"testing"
)

func TestAcceptHeader(t *testing.T) {
	tests := []struct {
		name          string
		customColumns bool
		check         func(got string) bool
		want          string
	}{
		{
			// Scenario: En-tête Accept en mode custom-columns.
			name:          "custom-columns mode sets no Accept header",
			customColumns: true,
			check:         func(got string) bool { return got == "" },
			want:          `"" (no as=Table)`,
		},
		{
			// Scenario: En-tête Accept en mode tableau.
			name:          "table mode asks for the Table representation",
			customColumns: false,
			check: func(got string) bool {
				return strings.Contains(got, "application/json;as=Table;v=v1;g=meta.k8s.io")
			},
			want: "value containing application/json;as=Table;v=v1;g=meta.k8s.io",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := acceptHeader(tc.customColumns); !tc.check(got) {
				t.Errorf("acceptHeader(%v) = %q, want %s", tc.customColumns, got, tc.want)
			}
		})
	}
}
