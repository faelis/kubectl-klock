// SPDX-FileCopyrightText: 2026 faelis
//
// SPDX-License-Identifier: GPL-3.0-or-later

package table

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestViewLongCellInNonFinalColumn covers Scenario: Cellule très longue.
// A 300-character first column must not panic View() and the second column
// must still be rendered.
func TestViewLongCellInNonFinalColumn(t *testing.T) {
	m := New()
	m.Update(tea.WindowSizeMsg{Width: 400, Height: 50})
	m.SetHeaders([]string{"A", "B"})
	m.AddRow(Row{ID: "r1", Fields: []any{strings.Repeat("a", 300), "ok"}, SortKey: "r1"})

	var view string
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("View() panicked: %v", r)
			}
		}()
		view = m.View()
	}()
	if !strings.Contains(view, "ok") {
		t.Errorf("View() lacks second column %q", "ok")
	}
}
