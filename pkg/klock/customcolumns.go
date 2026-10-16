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
	"errors"
	"fmt"
	"strings"

	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/util/jsonpath"
	"k8s.io/kubectl/pkg/cmd/get"
)

// CustomColumn is one column of a custom-columns output.
type CustomColumn struct {
	// Header is the column title.
	Header string
	// Path is the parsed JSONPath expression evaluated against each object.
	Path *jsonpath.JSONPath
}

// ParseCustomColumns parses an output value of the form
// "custom-columns=<spec>" or "custom-columns-file=<path>".
//
// It returns nil, nil when output is not a custom-columns format. It returns
// an error if the spec is empty or malformed, if the file cannot be read, or
// if a JSONPath expression does not parse; the latter names the column.
func ParseCustomColumns(output string) ([]CustomColumn, error) {
	// Reuse kubectl's own spec/file parsing so the syntax and messages match
	// "kubectl get -o custom-columns".
	printer, err := get.NewCustomColumnsPrintFlags().ToPrinter(output)
	if genericclioptions.IsNoCompatiblePrinterError(err) {
		return nil, nil
	}
	if err != nil {
		// kubectl's file-not-found message ends with a newline.
		return nil, errors.New(strings.TrimSpace(err.Error()))
	}
	cc, ok := printer.(*get.CustomColumnsPrinter)
	if !ok {
		return nil, fmt.Errorf("unexpected printer type %T for %q", printer, output)
	}
	columns := cc.Columns
	result := make([]CustomColumn, len(columns))
	for i, col := range columns {
		// Parse once at startup so a bad expression fails fast instead of on
		// the first watch event. Missing keys render empty, as in kubectl.
		// Sanitize once: the header is printed in the TUI and in errors, and
		// comes from the user's spec or file.
		header := sanitizeCell(col.Header)
		path := jsonpath.New(header).AllowMissingKeys(true)
		if err := path.Parse(col.FieldSpec); err != nil {
			return nil, fmt.Errorf("column %q: %w", header, err)
		}
		result[i] = CustomColumn{Header: header, Path: path}
	}
	return result, nil
}
