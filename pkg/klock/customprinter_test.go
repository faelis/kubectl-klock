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

	"github.com/applejag/kubectl-klock/pkg/table"
	tea "github.com/charmbracelet/bubbletea"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
)

var podGVK = schema.GroupVersionKind{Version: "v1", Kind: "Pod"}

// newPod builds a full Pod object; extra keys are merged at the top level.
func newPod(namespace, name, uid string, extra map[string]any) *unstructured.Unstructured {
	obj := map[string]any{
		"apiVersion": "v1",
		"kind":       "Pod",
		"metadata": map[string]any{
			"name":              name,
			"namespace":         namespace,
			"uid":               uid,
			"creationTimestamp": "2026-10-16T10:00:00Z",
		},
	}
	for k, v := range extra {
		obj[k] = v
	}
	return &unstructured.Unstructured{Object: obj}
}

// mustColumns parses a custom-columns spec or fails the test.
func mustColumns(t *testing.T, spec string) []CustomColumn {
	t.Helper()
	cols, err := ParseCustomColumns(spec)
	if err != nil {
		t.Fatalf("ParseCustomColumns(%q): %v", spec, err)
	}
	return cols
}

// customPrinter returns a Printer configured for custom columns.
func customPrinter(t *testing.T, spec string, printNamespace bool) *Printer {
	t.Helper()
	p := &Printer{Table: table.New(), CustomColumns: mustColumns(t, spec)}
	p.Configure(podGVK, printNamespace)
	return p
}

func TestEvalCell(t *testing.T) {
	tests := []struct {
		name       string
		spec       string
		obj        map[string]any
		want       string
		wantPrefix string
	}{
		{
			name: "absent field is <none> (Scenario: Champ absent)",
			spec: "custom-columns=NODE:.spec.nodeName",
			obj:  newPod("default", "nginx", "u1", map[string]any{"spec": map[string]any{}}).Object,
			want: "<none>",
		},
		{
			name: "multiple values joined by comma (Scenario: Valeurs multiples)",
			spec: "custom-columns=C:.spec.containers[*].name",
			obj: newPod("default", "nginx", "u1", map[string]any{"spec": map[string]any{
				"containers": []any{
					map[string]any{"name": "app"},
					map[string]any{"name": "sidecar"},
				},
			}}).Object,
			want: "app,sidecar",
		},
		{
			name: "field outside server table (Scenario: Champ hors de la table du serveur)",
			spec: "custom-columns=IMAGE:.spec.containers[0].image",
			obj: newPod("default", "nginx", "u1", map[string]any{"spec": map[string]any{
				"containers": []any{map[string]any{"name": "app", "image": "nginx:1.27"}},
			}}).Object,
			want: "nginx:1.27",
		},
		{
			name:       "evaluation error is reported in cell (Scenario: Erreur d'evaluation sur un objet)",
			spec:       "custom-columns=X:.metadata.name[0]",
			obj:        newPod("default", "nginx", "u1", nil).Object,
			wantPrefix: "<error:",
		},
		{
			name: "ANSI escape is neutralised (Scenario: Annotation avec sequence ANSI)",
			spec: "custom-columns=A:.metadata.annotations.x",
			obj: map[string]any{"metadata": map[string]any{
				"annotations": map[string]any{"x": "\x1b[31mrouge"},
			}},
			want: "^[[31mrouge",
		},
		{
			name: "newline and tab are escaped (Scenario: Saut de ligne et tabulation)",
			spec: "custom-columns=A:.metadata.annotations.x",
			obj: map[string]any{"metadata": map[string]any{
				"annotations": map[string]any{"x": "l1\nl2\tt"},
			}},
			want: `l1\nl2\tt`,
		},
		{
			name: "bell is escaped (Scenario: Autres caracteres de controle)",
			spec: "custom-columns=A:.metadata.annotations.x",
			obj: map[string]any{"metadata": map[string]any{
				"annotations": map[string]any{"x": "\a"},
			}},
			want: `\a`,
		},
		{
			name: "C1 CSI is escaped (Scenario: Autres caracteres de controle)",
			spec: "custom-columns=A:.metadata.annotations.x",
			obj: map[string]any{"metadata": map[string]any{
				"annotations": map[string]any{"x": "\u009b"},
			}},
			want: `\u009b`,
		},
		{
			name: "bidi override is escaped (Scenario: Autres caracteres de controle)",
			spec: "custom-columns=A:.metadata.annotations.x",
			obj: map[string]any{"metadata": map[string]any{
				"annotations": map[string]any{"x": "\u202e"},
			}},
			want: `\u202e`,
		},
		{
			name: "invisible U+200B is escaped (Scenario: Caracteres invisibles)",
			spec: "custom-columns=A:.metadata.annotations.x",
			obj: map[string]any{"metadata": map[string]any{
				"annotations": map[string]any{"x": "\u200b"},
			}},
			want: `\u200b`,
		},
		{
			name: "invisible U+2028 is escaped (Scenario: Caracteres invisibles)",
			spec: "custom-columns=A:.metadata.annotations.x",
			obj: map[string]any{"metadata": map[string]any{
				"annotations": map[string]any{"x": "\u2028"},
			}},
			want: `\u2028`,
		},
		{
			name: "invisible U+FEFF is escaped (Scenario: Caracteres invisibles)",
			spec: "custom-columns=A:.metadata.annotations.x",
			obj: map[string]any{"metadata": map[string]any{
				"annotations": map[string]any{"x": "\ufeff"},
			}},
			want: `\ufeff`,
		},
		{
			name: "error message is escaped (Scenario: Message d'erreur echappe)",
			spec: "custom-columns=X:.metadata.annotations.x[?(@.a)]",
			obj: map[string]any{"metadata": map[string]any{
				"annotations": map[string]any{"x": "\x1b[31m"},
			}},
			wantPrefix: "<error:",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cols := mustColumns(t, test.spec)
			got := evalCell(cols[0].Path, test.obj)
			if strings.Contains(got, "\x1b") {
				t.Errorf("evalCell contains raw ESC byte: %q", got)
			}
			if test.wantPrefix != "" {
				if !strings.HasPrefix(got, test.wantPrefix) {
					t.Errorf("evalCell = %q, want prefix %q", got, test.wantPrefix)
				}
				return
			}
			if got != test.want {
				t.Errorf("evalCell = %q, want %q", got, test.want)
			}
		})
	}
}

func TestCustomRowCells(t *testing.T) {
	p := customPrinter(t, "custom-columns=NAME:.metadata.name,NODE:.spec.nodeName", false)
	pod := newPod("default", "nginx", "u1", map[string]any{"spec": map[string]any{"nodeName": "node-1"}})

	row, err := p.customRow(pod, watch.Added)
	if err != nil {
		t.Fatalf("customRow: %v", err)
	}
	if row.ID != "u1" {
		t.Errorf("ID = %q, want %q", row.ID, "u1")
	}
	if len(row.Fields) != 2 {
		t.Fatalf("len(Fields) = %d, want 2: %v", len(row.Fields), row.Fields)
	}
	if row.Fields[0] != "nginx" || row.Fields[1] != "node-1" {
		t.Errorf("Fields = %v, want [nginx node-1] (Scenario: Deux colonnes inline)", row.Fields)
	}
	if row.HasLeadingNamespaceColumn {
		t.Error("HasLeadingNamespaceColumn = true, want false")
	}
}

func TestCustomRowEvalErrorKeepsRow(t *testing.T) {
	p := customPrinter(t, "custom-columns=X:.metadata.name[0],NAME:.metadata.name", false)
	row, err := p.customRow(newPod("default", "nginx", "u1", nil), watch.Added)
	if err != nil {
		t.Fatalf("customRow: %v (Scenario: Erreur d'evaluation, watch must continue)", err)
	}
	if got, ok := row.Fields[0].(string); !ok || !strings.HasPrefix(got, "<error:") {
		t.Errorf("Fields[0] = %v, want <error: prefix", row.Fields[0])
	}
	if row.Fields[1] != "nginx" {
		t.Errorf("Fields[1] = %v, want nginx", row.Fields[1])
	}
}

func TestCustomRowColoring(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		eventType watch.EventType
		wantStyle *styleRef // nil: must not be styled with a warning/error style
	}{
		{"incomplete fraction (Scenario: Fraction incomplete)", "0/1", watch.Added, &styleRef{"fraction warning", StyleFractionWarning.GetForeground()}},
		{"error status (Scenario: Statut d'erreur)", "CrashLoopBackOff", watch.Added, &styleRef{"status error", StyleStatusError.GetForeground()}},
		{"deleted row has no fraction style (Scenario: Ligne supprimee)", "0/1", watch.Deleted, nil},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			p := customPrinter(t, "custom-columns=V:.status.v", false)
			pod := newPod("default", "nginx", "u1", map[string]any{"status": map[string]any{"v": test.value}})

			row, err := p.customRow(pod, test.eventType)
			if err != nil {
				t.Fatalf("customRow: %v", err)
			}
			styled, isStyled := row.Fields[0].(table.StyledColumn)
			if test.wantStyle == nil {
				if isStyled && styled.Style.GetForeground() == StyleFractionWarning.GetForeground() {
					t.Errorf("deleted row cell has fraction warning style")
				}
				if row.Fields[0] != test.value {
					t.Errorf("Fields[0] = %#v, want raw string %q (Scenario: Ligne supprimee)", row.Fields[0], test.value)
				}
				if row.Status != table.StatusDeleted {
					t.Errorf("Status = %v, want StatusDeleted", row.Status)
				}
				return
			}
			if !isStyled {
				t.Fatalf("Fields[0] = %T(%v), want table.StyledColumn", row.Fields[0], row.Fields[0])
			}
			if got, want := styled.Style.GetForeground(), test.wantStyle.fg; got != want {
				t.Errorf("foreground = %v, want %s %v", got, test.wantStyle.name, want)
			}
		})
	}
}

// styleRef names an expected foreground for failure messages.
type styleRef struct {
	name string
	fg   any
}

func TestCustomRowAllNamespaces(t *testing.T) {
	p := customPrinter(t, "custom-columns=NAME:.metadata.name", true)

	rowB, err := p.customRow(newPod("b", "x", "ub", nil), watch.Added)
	if err != nil {
		t.Fatalf("customRow b/x: %v", err)
	}
	rowA, err := p.customRow(newPod("a", "y", "ua", nil), watch.Added)
	if err != nil {
		t.Fatalf("customRow a/y: %v", err)
	}

	if rowA.SortKey != "a/y" || rowB.SortKey != "b/x" {
		t.Errorf("SortKeys = %q, %q; want a/y, b/x (Scenario: Toutes les namespaces)", rowA.SortKey, rowB.SortKey)
	}
	if rowA.SortKey >= rowB.SortKey {
		t.Errorf("row y must sort before row x")
	}
	for _, r := range []table.Row{rowA, rowB} {
		if r.HasLeadingNamespaceColumn {
			t.Error("HasLeadingNamespaceColumn = true, want false")
		}
		if len(r.Fields) != 1 {
			t.Errorf("len(Fields) = %d, want 1 (no NAMESPACE cell)", len(r.Fields))
		}
	}
}

func TestCustomRowSortKeyWithoutNamespace(t *testing.T) {
	p := customPrinter(t, "custom-columns=NAME:.metadata.name", false)
	row, err := p.customRow(newPod("default", "nginx", "u1", nil), watch.Added)
	if err != nil {
		t.Fatalf("customRow: %v", err)
	}
	if row.SortKey != "nginx" {
		t.Errorf("SortKey = %q, want %q", row.SortKey, "nginx")
	}
}

func TestPrintObjCustomColumnsShowsCells(t *testing.T) {
	p := customPrinter(t, "custom-columns=NAME:.metadata.name,NODE:.spec.nodeName", false)
	p.Table.Update(tea.WindowSizeMsg{Width: 200, Height: 50})
	pod := newPod("default", "nginx", "u1", map[string]any{"spec": map[string]any{"nodeName": "node-1"}})

	if _, err := p.PrintObj(pod, watch.Added); err != nil {
		t.Fatalf("PrintObj: %v", err)
	}

	view := p.Table.View()
	for _, want := range []string{"NAME", "NODE", "nginx", "node-1"} {
		if !strings.Contains(view, want) {
			t.Errorf("View() lacks %q (Scenario: Deux colonnes inline):\n%s", want, view)
		}
	}
	if strings.Contains(view, "NAMESPACE") {
		t.Errorf("View() contains NAMESPACE column:\n%s", view)
	}
}

func TestPrintObjCustomColumnsModified(t *testing.T) {
	p := customPrinter(t, "custom-columns=NAME:.metadata.name,NODE:.spec.nodeName", false)
	p.Table.Update(tea.WindowSizeMsg{Width: 200, Height: 50})

	pending := newPod("default", "nginx", "u1", map[string]any{"spec": map[string]any{}})
	if _, err := p.PrintObj(pending, watch.Added); err != nil {
		t.Fatalf("PrintObj added: %v", err)
	}
	if view := p.Table.View(); !strings.Contains(view, "<none>") {
		t.Fatalf("precondition: View() lacks <none>:\n%s", view)
	}

	scheduled := newPod("default", "nginx", "u1", map[string]any{"spec": map[string]any{"nodeName": "node-1"}})
	if _, err := p.PrintObj(scheduled, watch.Modified); err != nil {
		t.Fatalf("PrintObj modified: %v", err)
	}

	view := p.Table.View()
	if !strings.Contains(view, "node-1") || strings.Contains(view, "<none>") {
		t.Errorf("View() after MODIFIED must show node-1 and no <none> (Scenario: Modification):\n%s", view)
	}
	if got := strings.Count(view, "nginx"); got != 1 {
		t.Errorf("nginx appears %d times, want 1 (no duplicate row):\n%s", got, view)
	}
	if idx := p.Table.RowIndex("u1"); idx < 0 {
		t.Errorf("RowIndex(u1) = %d, want >= 0", idx)
	}
}

func TestPrintObjCustomColumnsDeleted(t *testing.T) {
	p := customPrinter(t, "custom-columns=NAME:.metadata.name", false)
	p.Table.Update(tea.WindowSizeMsg{Width: 200, Height: 50})
	pod := newPod("default", "nginx", "u1", nil)

	if _, err := p.PrintObj(pod, watch.Added); err != nil {
		t.Fatalf("PrintObj added: %v", err)
	}
	if _, err := p.PrintObj(pod, watch.Deleted); err != nil {
		t.Fatalf("PrintObj deleted: %v", err)
	}

	// Row stays (greyed out) until the hide-deleted delay elapses.
	if idx := p.Table.RowIndex("u1"); idx < 0 {
		t.Errorf("RowIndex(u1) = %d after DELETED, want row kept (Scenario: Suppression)", idx)
	}
}
