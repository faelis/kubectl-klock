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
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/applejag/kubectl-klock/pkg/table"
	tea "github.com/charmbracelet/bubbletea"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/watch"
)

// colDef is a (name, type, priority 0) server column definition.
type colDef struct{ name, typ string }

// tableOf builds a server Table with one row whose cells match defs; obj is
// the embedded row object (a Pod when nil).
func tableOf(defs []colDef, cells []any, obj map[string]any) *unstructured.Unstructured {
	if obj == nil {
		obj = newPod("default", "nginx", "u1", nil).Object
	}
	cd := make([]any, len(defs))
	for i, d := range defs {
		cd[i] = map[string]any{"name": d.name, "type": d.typ, "priority": int64(0)}
	}
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion":        "meta.k8s.io/v1",
		"kind":              "Table",
		"columnDefinitions": cd,
		"rows":              []any{map[string]any{"cells": cells, "object": obj}},
	}}
}

// viewOf prints tbl through a table-mode Printer (labels = -L keys) and
// returns the rendered table, wide enough to never clip a bounded cell.
func viewOf(t *testing.T, tbl *unstructured.Unstructured, labels ...string) string {
	t.Helper()
	p := &Printer{Table: table.New()}
	p.Configure(podGVK, false)
	p.LabelCols = labels
	p.Table.Update(tea.WindowSizeMsg{Width: 5000, Height: 50})
	if _, err := p.PrintObj(tbl, watch.Added); err != nil {
		t.Fatalf("PrintObj: %v", err)
	}
	return p.Table.View()
}

// oneCol prints a single string column "Msg" holding cell.
func oneCol(t *testing.T, cell any) string {
	t.Helper()
	return viewOf(t, tableOf([]colDef{{"Msg", "string"}}, []any{cell}, nil))
}

func checkNoRaw(t *testing.T, got string) {
	t.Helper()
	if strings.ContainsAny(got, "\x1b\x07") {
		t.Errorf("output contains raw control byte: %q", got)
	}
}

// Scenario: Caracteres ignorables par defaut (custom-columns-output).
func TestSanitizeCellDefaultIgnorable(t *testing.T) {
	tests := []struct{ in, want string }{
		{"\u3164", `\u3164`},
		{"\u115f", `\u115f`},
		{"\u034f", `\u034f`},
	}
	for _, test := range tests {
		t.Run(test.want, func(t *testing.T) {
			got := sanitizeCell(test.in)
			if got != test.want {
				t.Errorf("sanitizeCell(%q) = %q, want %q", test.in, got, test.want)
			}
			for _, r := range got {
				if r > 0x7f {
					t.Errorf("sanitizeCell(%q) = %q, non-ASCII rune %U", test.in, got, r)
				}
			}
		})
	}
}

// Scenario: Caracteres ignorables par defaut, through evalCell.
func TestEvalCellDefaultIgnorable(t *testing.T) {
	cols := mustColumns(t, "custom-columns=A:.metadata.annotations.x")
	obj := map[string]any{"metadata": map[string]any{
		"annotations": map[string]any{"x": "a\u3164b"},
	}}
	if got, want := evalCell(cols[0].Path, obj), `a\u3164b`; got != want {
		t.Errorf("evalCell = %q, want %q", got, want)
	}
}

// Scenario: Message d'Event avec OSC 52 (table-output).
func TestTableModeCellOSC52(t *testing.T) {
	view := oneCol(t, "\x1b]52;c;aGVsbG8=\x07")
	if want := `^[]52;c;aGVsbG8=\a`; !strings.Contains(view, want) {
		t.Errorf("View lacks %q:\n%q", want, view)
	}
	checkNoRaw(t, view)
}

// Scenario: Saut de ligne dans une colonne de CR (table-output).
func TestTableModeCellNewline(t *testing.T) {
	view := oneCol(t, "l1\nl2")
	if want := `l1\nl2`; !strings.Contains(view, want) {
		t.Errorf("View lacks literal %q:\n%q", want, view)
	}
}

// Scenario: Caractere invisible en mode table (table-output).
func TestTableModeCellInvisible(t *testing.T) {
	view := oneCol(t, "a\u200bb")
	if want := `a\u200bb`; !strings.Contains(view, want) {
		t.Errorf("View lacks %q:\n%q", want, view)
	}
}

// Scenario: Cellule liste (table-output).
func TestTableModeCellList(t *testing.T) {
	view := oneCol(t, []any{"\x1bx"})
	if want := "^[x"; !strings.Contains(view, want) {
		t.Errorf("View lacks %q:\n%q", want, view)
	}
	checkNoRaw(t, view)
}

// Scenario: Valeur numerique inchangee (table-output); already green.
func TestTableModeCellInteger(t *testing.T) {
	view := viewOf(t, tableOf([]colDef{{"Count", "integer"}}, []any{int64(3)}, nil))
	if !regexp.MustCompile(`(^|\s)3(\s|$)`).MatchString(view) {
		t.Errorf("View lacks cell 3:\n%q", view)
	}
}

// Scenario: Statut toujours colore (table-output); already green. The
// colour is carried by parseCell's result, which the escaped value feeds.
func TestTableModeStatusStillStyled(t *testing.T) {
	p := &Printer{}
	got := p.parseCell(sanitizeCell("Running"), metav1.TableRow{}, watch.Added, nil,
		metav1.TableColumnDefinition{Name: "Status"}, time.Now())
	styled, ok := got.(table.StyledColumn)
	if !ok {
		t.Fatalf("parseCell = %T(%v), want table.StyledColumn", got, got)
	}
	if got, want := styled.Style.GetForeground(), StyleStatusOK.GetForeground(); got != want {
		t.Errorf("foreground = %v, want status OK %v", got, want)
	}
}

// Scenarios: Nom de colonne avec sequence ANSI, Echappement apres les
// majuscules (table-output).
func TestTableModeHeaders(t *testing.T) {
	tests := []struct{ name, col, want string }{
		{"ANSI (Nom de colonne avec sequence ANSI)", "x\x1b[31m", "X^[[31M"},
		{"after upper-casing (Echappement apres les majuscules)", "x\u202e", `X\u202e`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			view := viewOf(t, tableOf([]colDef{{test.col, "string"}}, []any{"v"}, nil))
			if !strings.Contains(view, test.want) {
				t.Errorf("View lacks header %q:\n%q", test.want, view)
			}
			checkNoRaw(t, view)
		})
	}
}

// Scenario: Cle de label avec sequence ANSI (table-output).
func TestLabelColumnHeaderANSI(t *testing.T) {
	if got, want := labelColumnHeader("app\x1b"), "APP^["; got != want {
		t.Errorf("labelColumnHeader = %q, want %q", got, want)
	}
	view := viewOf(t, tableOf([]colDef{{"Msg", "string"}}, []any{"v"}, nil), "app\x1b")
	if !strings.Contains(view, "APP^[") {
		t.Errorf("View lacks header APP^[:\n%q", view)
	}
	checkNoRaw(t, view)
}

// Scenario: Valeur de label avec sequence ANSI (table-output).
func TestTableModeLabelValueANSI(t *testing.T) {
	pod := newPod("default", "nginx", "u1", nil).Object
	pod["metadata"].(map[string]any)["labels"] = map[string]any{"app": "\x1b[31m"}
	view := viewOf(t, tableOf([]colDef{{"Msg", "string"}}, []any{"v"}, pod), "app")
	if want := "^[[31m"; !strings.Contains(view, want) {
		t.Errorf("View lacks %q:\n%q", want, view)
	}
	checkNoRaw(t, view)
}

// Scenarios: Spec sans deux-points echappee, Expression invalide echappee
// (custom-columns-output).
func TestParseCustomColumnsErrorsEscaped(t *testing.T) {
	tests := []struct{ name, spec, want string }{
		{"no colon (Spec sans deux-points echappee)", "custom-columns=X\x1b", "X^["},
		{"bad expression (Expression invalide echappee)", "custom-columns=X:.a[\x1bc]", "^[c"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := ParseCustomColumns(test.spec)
			if err == nil {
				t.Fatal("ParseCustomColumns error = nil, want error")
			}
			if !strings.Contains(err.Error(), test.want) {
				t.Errorf("error %q lacks %q", err, test.want)
			}
			checkNoRaw(t, err.Error())
		})
	}
}

// Scenario: Combinaison avec -L echappee (custom-columns-output).
func TestValidateLabelColumnsErrorEscaped(t *testing.T) {
	opts := Options{
		Output:       "custom-columns=A\x1b:.metadata.name",
		LabelColumns: []string{"app"},
	}
	_, err := opts.Validate()
	if err == nil {
		t.Fatal("Validate error = nil, want error")
	}
	checkNoRaw(t, err.Error())
	if !strings.Contains(err.Error(), "A^[") {
		t.Errorf("error %q lacks escaped spec A^[", err)
	}
}

// Scenarios: Valeur de 2000 caracteres, Valeurs jointes trop longues
// (custom-columns-output).
func TestEvalCellBounded(t *testing.T) {
	long := map[string]any{"metadata": map[string]any{
		"annotations": map[string]any{"x": strings.Repeat("a", 2000)},
	}}
	cols := mustColumns(t, "custom-columns=A:.metadata.annotations.x")
	if got, want := evalCell(cols[0].Path, long), strings.Repeat("a", 1024)+"…"; got != want {
		t.Errorf("2000 a: rune count = %d, want 1025 ending in ellipsis", utf8.RuneCountInString(got))
	}

	img := strings.Repeat("i", 600)
	pod := newPod("default", "nginx", "u1", map[string]any{"spec": map[string]any{
		"containers": []any{
			map[string]any{"name": "a", "image": img},
			map[string]any{"name": "b", "image": img},
		},
	}}).Object
	cols = mustColumns(t, "custom-columns=C:.spec.containers[*].image")
	got := evalCell(cols[0].Path, pod)
	if n := utf8.RuneCountInString(got); n != 1025 || !strings.HasSuffix(got, "…") {
		t.Errorf("joined: rune count = %d, suffix ellipsis = %v, want 1025 and true",
			n, strings.HasSuffix(got, "…"))
	}
}

// Scenarios: Cellule de 2000 caracteres, Cellule de 1024 caracteres, Borne
// apres echappement (table-output), end to end in table mode.
func TestTableModeCellBounded(t *testing.T) {
	a := func(n int) string { return strings.Repeat("a", n) }
	tests := []struct {
		name, in, want, notWant string
	}{
		{"2000 chars (Cellule de 2000 caracteres)", a(2000), a(1024) + "…", a(1025)},
		{"1024 chars (Cellule de 1024 caracteres)", a(1024), a(1024), "…"},
		{
			"bound after escaping (Borne apres echappement)",
			a(1000) + strings.Repeat("\n", 100),
			a(1000) + strings.Repeat(`\n`, 12) + "…",
			`\n\n\n\n\n\n\n\n\n\n\n\n\n`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			view := oneCol(t, test.in)
			if !strings.Contains(view, test.want) {
				t.Errorf("View lacks bounded cell (len %d), view len %d", len(test.want), len(view))
			}
			if strings.Contains(view, test.notWant) {
				t.Errorf("View contains %s", fmt.Sprintf("%.40q...", test.notWant))
			}
		})
	}
}

// roleObj builds a Role object with the given namespace and name.
func roleObj(namespace, name string) *unstructured.Unstructured {
	obj := newPod(namespace, name, "u-role", nil)
	obj.Object["apiVersion"] = "rbac.authorization.k8s.io/v1"
	obj.Object["kind"] = "Role"
	return obj
}

// filterView opens the filter of p's table, types prefix and returns the
// view; the completion suggestion is rendered after the typed text.
func filterView(t *testing.T, p *Printer, prefix string) string {
	t.Helper()
	p.Table.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	p.Table.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(prefix)})
	return p.Table.View()
}

// Scenario: Role au nom piege (table-output).
func TestTableModeSuggestionEscaped(t *testing.T) {
	name := "r\x1b]52;c;eA==\x07"
	p := &Printer{Table: table.New()}
	p.Configure(podGVK, false)
	p.Table.Update(tea.WindowSizeMsg{Width: 5000, Height: 50})
	tbl := tableOf([]colDef{{"Name", "string"}}, []any{name}, roleObj("default", name).Object)
	if _, err := p.PrintObj(tbl, watch.Added); err != nil {
		t.Fatalf("PrintObj: %v", err)
	}

	view := filterView(t, p, "r")
	checkNoRaw(t, view)
	// The first line is the filter input: typed "r" plus the suggestion rest.
	filterLine, _, _ := strings.Cut(view, "\n")
	if want := `r^[]52;c;eA==\a`; !strings.Contains(filterLine, want) {
		t.Errorf("filter line lacks escaped suggestion %q:\n%q", want, filterLine)
	}
}

// Scenario: Suggestion en mode custom-columns (table-output).
func TestCustomRowSuggestionEscaped(t *testing.T) {
	p := customPrinter(t, "custom-columns=NAME:.metadata.name", false)
	row, err := p.customRow(roleObj("default", "r\x1b"), watch.Added)
	if err != nil {
		t.Fatalf("customRow: %v", err)
	}
	if got, want := row.Suggestion, "r^["; got != want {
		t.Errorf("Suggestion = %q, want %q", got, want)
	}
}

// Scenario: Namespace piege avec -A (table-output).
func TestTableModeNamespaceCellEscaped(t *testing.T) {
	p := &Printer{Table: table.New()}
	p.Configure(podGVK, true)
	p.Table.Update(tea.WindowSizeMsg{Width: 5000, Height: 50})
	tbl := tableOf([]colDef{{"Name", "string"}}, []any{"ok"}, roleObj("n\x1b", "nginx").Object)
	if _, err := p.PrintObj(tbl, watch.Added); err != nil {
		t.Fatalf("PrintObj: %v", err)
	}

	view := p.Table.View()
	checkNoRaw(t, view)
	if !strings.Contains(view, "n^[") {
		t.Errorf("View lacks escaped namespace cell %q:\n%q", "n^[", view)
	}
}
