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
	"k8s.io/apimachinery/pkg/watch"
)

func TestParseCustomColumnsHeaderEscapesANSI(t *testing.T) {
	// Scenario: En-tete avec sequence ANSI.
	cols, err := ParseCustomColumns("custom-columns=X\x1b[31m:.metadata.name")
	if err != nil {
		t.Fatalf("ParseCustomColumns: %v", err)
	}
	if len(cols) != 1 {
		t.Fatalf("len(cols) = %d, want 1", len(cols))
	}
	if got, want := cols[0].Header, "X^[[31m"; got != want {
		t.Errorf("Header = %q, want %q", got, want)
	}
	if strings.Contains(cols[0].Header, "\x1b") {
		t.Errorf("Header contains raw ESC byte: %q", cols[0].Header)
	}
}

// serverTable builds a meta.k8s.io/v1 Table with one pod row.
func serverTable(namespace, name, uid string) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "meta.k8s.io/v1",
		"kind":       "Table",
		"columnDefinitions": []any{
			map[string]any{"name": "Name", "type": "string", "priority": int64(0)},
			map[string]any{"name": "Status", "type": "string", "priority": int64(0)},
		},
		"rows": []any{
			map[string]any{
				"cells":  []any{name, "Running"},
				"object": newPod(namespace, name, uid, nil).Object,
			},
		},
	}}
}

// TestPrintObjTableMode characterises the default (server table) mode; it
// guards the baseRow extraction and is expected to pass already.
func TestPrintObjTableMode(t *testing.T) {
	tests := []struct {
		name           string
		printNamespace bool
		wantNamespace  bool
	}{
		{"without namespace column", false, false},
		{"with namespace column", true, true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			p := &Printer{Table: table.New()}
			p.Configure(podGVK, test.printNamespace)
			p.Table.Update(tea.WindowSizeMsg{Width: 200, Height: 50})

			if _, err := p.PrintObj(serverTable("myns", "nginx", "u1"), watch.Added); err != nil {
				t.Fatalf("PrintObj added: %v", err)
			}
			if idx := p.Table.RowIndex("u1"); idx < 0 {
				t.Errorf("RowIndex(u1) = %d after ADDED, want >= 0", idx)
			}
			view := p.Table.View()
			for _, want := range []string{"nginx", "Running"} {
				if !strings.Contains(view, want) {
					t.Errorf("View() lacks %q:\n%s", want, view)
				}
			}
			if got := strings.Contains(view, "NAMESPACE"); got != test.wantNamespace {
				t.Errorf("View() has NAMESPACE = %v, want %v:\n%s", got, test.wantNamespace, view)
			}
			if got := strings.Contains(view, "myns"); got != test.wantNamespace {
				t.Errorf("View() has namespace cell = %v, want %v:\n%s", got, test.wantNamespace, view)
			}

			if _, err := p.PrintObj(serverTable("myns", "nginx", "u1"), watch.Deleted); err != nil {
				t.Fatalf("PrintObj deleted: %v", err)
			}
			if idx := p.Table.RowIndex("u1"); idx < 0 {
				t.Errorf("RowIndex(u1) = %d after DELETED, want row kept", idx)
			}
		})
	}
}
