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

// Scenario: Message d'erreur borné (custom-columns-output). A JSONPath
// filter on a string yields an error cell that must stay bounded.
func TestEvalCellErrorBounded(t *testing.T) {
	cols := mustColumns(t, "custom-columns=X:.metadata.annotations.x[?(@.a)]")
	obj := map[string]any{"metadata": map[string]any{
		"annotations": map[string]any{"x": strings.Repeat("a", 5000)},
	}}
	got := evalCell(cols[0].Path, obj)
	if !strings.HasPrefix(got, "<error:") {
		t.Errorf("evalCell = %.40q..., want prefix %q", got, "<error:")
	}
	if n := utf8.RuneCountInString(got); n != 1025 {
		t.Errorf("rune count = %d, want 1025", n)
	}
	if !strings.HasSuffix(got, "…") {
		t.Errorf("suffix ellipsis = false, want true")
	}
}

// Scenario: Format refusé avec caractère ignorable (custom-columns-output).
func TestValidateRefusedFormatIgnorable(t *testing.T) {
	_, err := Options{Output: "json" + "\u3164"}.Validate()
	if err == nil {
		t.Fatal("Validate error = nil, want error")
	}
	if !strings.Contains(err.Error(), `json\u3164`) {
		t.Errorf("error = %q, want it to contain literal %q", err, `json\u3164`)
	}
	if strings.Contains(err.Error(), "\u3164") {
		t.Errorf("error = %q, must not contain U+3164", err)
	}
}

// Scenario: En-tête échappé une seule fois dans les erreurs
// (custom-columns-output).
func TestParseCustomColumnsHeaderEscapedOnce(t *testing.T) {
	_, err := ParseCustomColumns("custom-columns=A" + "\n" + ":.a[")
	if err == nil {
		t.Fatal("ParseCustomColumns error = nil, want error")
	}
	if want := `column "A\n": `; !strings.HasPrefix(err.Error(), want) {
		t.Errorf("error = %q, want prefix %q", err, want)
	}
}
