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
	"unicode/utf8"
)

// Scenarios: Cellule de 2000 / 1024 caracteres, Borne apres echappement
// (table-output), at the boundCell level.
func TestBoundCell(t *testing.T) {
	a := func(n int) string { return strings.Repeat("a", n) }
	tests := []struct{ name, in, want string }{
		{"2000 runes (Cellule de 2000 caracteres)", a(2000), a(1024) + "…"},
		{"1024 runes (Cellule de 1024 caracteres)", a(1024), a(1024)},
		{"1025 runes", a(1025), a(1024) + "…"},
		{"multibyte counted as code points", strings.Repeat("é", 2000), strings.Repeat("é", 1024) + "…"},
		{"short", "abc", "abc"},
		{"empty", "", ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := boundCell(test.in); got != test.want {
				t.Errorf("boundCell: rune count = %d, want %d (equal = false)",
					utf8.RuneCountInString(got), utf8.RuneCountInString(test.want))
			}
		})
	}
}
