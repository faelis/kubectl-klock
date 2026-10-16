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
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// nginxPod is the object the column paths are evaluated against.
var nginxPod = map[string]any{
	"metadata": map[string]any{"name": "nginx"},
	"spec":     map[string]any{"nodeName": "node-1"},
}

// evalColumn returns the first result of the column's path on obj.
func evalColumn(t *testing.T, col CustomColumn, obj map[string]any) any {
	t.Helper()
	results, err := col.Path.FindResults(obj)
	if err != nil {
		t.Fatalf("FindResults(%q): %v", col.Header, err)
	}
	if len(results) != 1 || len(results[0]) != 1 {
		t.Fatalf("FindResults(%q) = %v, want exactly one value", col.Header, results)
	}
	return results[0][0].Interface()
}

func writeColumnsFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "columns.txt")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestParseCustomColumns_NotCustomFormat(t *testing.T) {
	for _, output := range []string{"", "wide"} {
		cols, err := ParseCustomColumns(output)
		if err != nil || cols != nil {
			t.Errorf("ParseCustomColumns(%q) = %v, %v; want nil, nil", output, cols, err)
		}
	}
}

// Scenarios "Deux colonnes inline", "Expression entre accolades", "Fichier valide".
func TestParseCustomColumns_Valid(t *testing.T) {
	tests := []struct {
		name        string
		output      func(t *testing.T) string
		wantHeaders []string
		wantValues  []string
	}{
		{
			name:        "inline two columns (Deux colonnes inline)",
			output:      func(*testing.T) string { return "custom-columns=NAME:.metadata.name,NODE:.spec.nodeName" },
			wantHeaders: []string{"NAME", "NODE"},
			wantValues:  []string{"nginx", "node-1"},
		},
		{
			name:        "braced expression (Expression entre accolades)",
			output:      func(*testing.T) string { return "custom-columns=NAME:{.metadata.name}" },
			wantHeaders: []string{"NAME"},
			wantValues:  []string{"nginx"},
		},
		{
			name: "file (Fichier valide)",
			output: func(t *testing.T) string {
				return "custom-columns-file=" + writeColumnsFile(t, "NAME   NODE\n.metadata.name   .spec.nodeName\n")
			},
			wantHeaders: []string{"NAME", "NODE"},
			wantValues:  []string{"nginx", "node-1"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cols, err := ParseCustomColumns(test.output(t))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			var headers []string
			var values []string
			for _, col := range cols {
				headers = append(headers, col.Header)
				values = append(values, evalColumn(t, col, nginxPod).(string))
			}
			if !reflect.DeepEqual(headers, test.wantHeaders) {
				t.Errorf("headers = %v, want %v", headers, test.wantHeaders)
			}
			if !reflect.DeepEqual(values, test.wantValues) {
				t.Errorf("values = %v, want %v", values, test.wantValues)
			}
		})
	}
}

// Scenarios "Spec vide", "Colonne sans expression", "Expression JSONPath
// invalide", "Fichier introuvable", "En-têtes et expressions en nombre différent".
func TestParseCustomColumns_Errors(t *testing.T) {
	tests := []struct {
		name       string
		output     func(t *testing.T) string
		wantExact  string
		wantPrefix string
		wantSubstr string
	}{
		{
			name:      "empty spec with equals (Spec vide)",
			output:    func(*testing.T) string { return "custom-columns=" },
			wantExact: "custom-columns format specified but no custom columns given",
		},
		{
			name:      "empty spec without equals (Spec vide)",
			output:    func(*testing.T) string { return "custom-columns" },
			wantExact: "custom-columns format specified but no custom columns given",
		},
		{
			name:      "column without expression (Colonne sans expression)",
			output:    func(*testing.T) string { return "custom-columns=NAME" },
			wantExact: "unexpected custom-columns spec: NAME, expected <header>:<json-path-expr>",
		},
		{
			name:       "invalid JSONPath (Expression JSONPath invalide)",
			output:     func(*testing.T) string { return "custom-columns=NAME:.metadata.name[" },
			wantSubstr: "NAME",
		},
		{
			name:       "file not found (Fichier introuvable)",
			output:     func(*testing.T) string { return "custom-columns-file=/nexiste/pas" },
			wantPrefix: "error reading template /nexiste/pas, open /nexiste/pas:",
		},
		{
			name: "header/expression count mismatch (nombre différent)",
			output: func(t *testing.T) string {
				return "custom-columns-file=" + writeColumnsFile(t, "NAME NODE\n.metadata.name\n")
			},
			wantExact: "number of headers (2) and field specifications (1) don't match",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cols, err := ParseCustomColumns(test.output(t))
			if err == nil {
				t.Fatalf("expected error, got columns %v", cols)
			}
			msg := err.Error()
			if test.wantExact != "" && msg != test.wantExact {
				t.Errorf("message = %q, want %q", msg, test.wantExact)
			}
			if test.wantPrefix != "" && !strings.HasPrefix(msg, test.wantPrefix) {
				t.Errorf("message = %q, want prefix %q", msg, test.wantPrefix)
			}
			if test.wantSubstr != "" && !strings.Contains(msg, test.wantSubstr) {
				t.Errorf("message = %q, want substring %q", msg, test.wantSubstr)
			}
		})
	}
}

// Scenarios "Combinaison avec -L" and "Format toujours refusé", plus the
// happy paths of Validate.
func TestOptionsValidate(t *testing.T) {
	tests := []struct {
		name     string
		opts     Options
		wantCols int
		wantErr  string
	}{
		{name: "empty output", opts: Options{}},
		{name: "wide output", opts: Options{Output: "wide"}},
		{name: "wide with -L", opts: Options{Output: "wide", LabelColumns: []string{"app"}}},
		{
			name:     "custom columns returned",
			opts:     Options{Output: "custom-columns=NAME:.metadata.name,NODE:.spec.nodeName"},
			wantCols: 2,
		},
		{
			name: "custom columns with -L (Combinaison avec -L)",
			opts: Options{
				Output:       "custom-columns=NAME:.metadata.name",
				LabelColumns: []string{"app"},
			},
			wantErr: "--label-columns option cannot be used with custom-columns=NAME:.metadata.name printer",
		},
		{
			name:    "json refused (Format toujours refusé)",
			opts:    Options{Output: "json"},
			wantErr: `unsupported output format: "json", allowed formats are: wide, custom-columns=..., custom-columns-file=...`,
		},
		{
			name:    "invalid custom columns propagate",
			opts:    Options{Output: "custom-columns=NAME"},
			wantErr: "unexpected custom-columns spec: NAME, expected <header>:<json-path-expr>",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cols, err := test.opts.Validate()
			if test.wantErr != "" {
				if err == nil || err.Error() != test.wantErr {
					t.Fatalf("error = %v, want %q", err, test.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(cols) != test.wantCols {
				t.Errorf("got %d columns, want %d", len(cols), test.wantCols)
			}
		})
	}
}
