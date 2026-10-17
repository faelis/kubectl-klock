// SPDX-FileCopyrightText: 2023 Kalle Fagerberg
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
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fsnotify/fsnotify"
	"github.com/gookit/color"
	"github.com/kubecolor/kubecolor/config"
	kubecolor "github.com/kubecolor/kubecolor/config/color"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	metav1beta1 "k8s.io/apimachinery/pkg/apis/meta/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/jsonpath"

	"github.com/applejag/kubectl-klock/internal/util"
	"github.com/applejag/kubectl-klock/pkg/table"
	"github.com/applejag/kubectl-klock/pkg/types"
)

type Options struct {
	ConfigFlags *genericclioptions.ConfigFlags `koanf:"-"`
	Kubecolor   *config.Config                 `koanf:"-"`

	AllNamespaces   bool                   `koanf:"all-namespaces"`
	FieldSelector   string                 `koanf:"field-selector"`
	LabelColumns    []string               `koanf:"label-columns"`
	LabelSelector   string                 `koanf:"label-selector"`
	HideDeleted     types.OptionalDuration `koanf:"hide-deleted"`
	Output          string                 `koanf:"output"`
	WatchKubeconfig bool                   `koanf:"watch-kubeconfig"`
}

// Validate checks the options and returns the custom columns requested by
// Output, or nil when Output is not a custom-columns format.
//
// It returns an error if Output is unsupported or unknown, if the
// custom-columns spec is invalid, or if LabelColumns is combined with a
// custom-columns format.
func (o Options) Validate() ([]CustomColumn, error) {
	const allowedFormats = "wide, custom-columns=..., custom-columns-file=..."
	cols, err := ParseCustomColumns(o.Output)
	if err != nil {
		return nil, err
	}
	if cols != nil {
		if len(o.LabelColumns) > 0 {
			return nil, fmt.Errorf("--label-columns option cannot be used with %s printer", sanitizeCell(o.Output))
		}
		return cols, nil
	}
	switch o.Output {
	case "", "wide":
		// Valid
		return nil, nil
	case "go-template",
		"go-template-file", "json", "jsonpath", "jsonpath-as-json",
		"jsonpath-file", "name", "template", "templatefile", "yaml":
		return nil, fmt.Errorf("unsupported output format: %q, allowed formats are: %s", o.Output, allowedFormats)

	default:
		return nil, fmt.Errorf("unknown output format: %q, allowed formats are: %s", o.Output, allowedFormats)
	}
}

// validateArgs returns an error if there's something wrong with the provided args.
//
// The "args" is assumed to not be an empty slice
func validateArgs(args []string) error {
	if strings.Contains(args[0], ",") {
		// same error as in "kubectl get pod,svc --watch"
		return errors.New("you may only specify a single resource type")
	}
	return nil
}

func Execute(o Options, args []string) error {
	if err := validateArgs(args); err != nil {
		return err
	}
	customColumns, err := o.Validate()
	if err != nil {
		return err
	}
	var fileEvents chan fsnotify.Event
	if o.WatchKubeconfig {
		if fileWatcher, err := fsnotify.NewWatcher(); err == nil {
			configLoader := o.ConfigFlags.ToRawKubeConfigLoader()
			kubeconfigFiles := configLoader.ConfigAccess().GetLoadingPrecedence()
			for _, file := range kubeconfigFiles {
				fileWatcher.Add(file)
			}
			fileEvents = fileWatcher.Events
			defer fileWatcher.Close()
		}
	}

	t := table.New()
	t.HideDeletedAfter = o.HideDeleted

	if o.Kubecolor != nil {
		overrideLipglossWithKubecolor(&t.Styles.Header, o.Kubecolor.Theme.Table.Header)
		overrideLipglossWithKubecolor(&t.Styles.Row.Deleted, o.Kubecolor.Theme.Base.Muted)
		overrideLipglossWithKubecolor(&t.Styles.Row.Error, o.Kubecolor.Theme.Base.Danger)
		overrideLipglossWithKubecolor(&t.Styles.NoneFound, o.Kubecolor.Theme.Base.Muted)
		overrideLipglossWithKubecolor(&t.Styles.FilterInfo, o.Kubecolor.Theme.Base.Muted)
		overrideLipglossWithKubecolor(&t.Styles.FilterPrompt, o.Kubecolor.Theme.Base.Secondary)
		overrideLipglossWithKubecolor(&t.Styles.FilterNoneVisible, o.Kubecolor.Theme.Base.Warning)
		overrideLipglossWithKubecolor(&t.Styles.Toggles, o.Kubecolor.Theme.Base.Muted)

		overrideLipglossWithKubecolor(&StyleFractionOK, o.Kubecolor.Theme.Data.Ratio.Equal)
		overrideLipglossWithKubecolor(&StyleFractionWarning, o.Kubecolor.Theme.Data.Ratio.Unequal)

		StyleStatusDefault = lipgloss.NewStyle()
		overrideLipglossWithKubecolor(&StyleStatusOK, o.Kubecolor.Theme.Status.Success)
		overrideLipglossWithKubecolor(&StyleStatusError, o.Kubecolor.Theme.Status.Error)
		overrideLipglossWithKubecolor(&StyleStatusWarning, o.Kubecolor.Theme.Status.Warning)

		overrideLipglossWithKubecolor(&StyleStatusNull, o.Kubecolor.Theme.Data.Null)
		overrideLipglossWithKubecolor(&StyleStatusTrue, o.Kubecolor.Theme.Data.True)
		overrideLipglossWithKubecolor(&StyleStatusFalse, o.Kubecolor.Theme.Data.False)
	}

	printer := Printer{
		Kubecolor:        o.Kubecolor,
		Table:            t,
		HideDeletedAfter: o.HideDeleted,
		WideOutput:       o.Output == "wide",
		LabelCols:        o.LabelColumns,
		CustomColumns:    customColumns,
	}
	p := tea.NewProgram(t)
	w := NewWatcher(o, p, printer, args)
	t.StartSpinner()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	restartChan := make(chan struct{})
	defer close(restartChan)

	go func() {
		for {
			select {
			case event, ok := <-fileEvents:
				if !ok {
					fileEvents = nil
					continue
				}
				if event.Op != fsnotify.Write {
					continue
				}
				restartChan <- struct{}{}

			case err := <-w.ErrorChan():
				t.SetError(err)
				p.Send(nil)
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		w.WatchLoop(ctx, restartChan)
	}()

	_, err = p.Run()
	return err
}

func NewWatcher(options Options, program *tea.Program, printer Printer, args []string) *Watcher {
	return &Watcher{
		Options: options,
		Program: program,
		Printer: printer,
		Args:    args,

		errorChan: make(chan error, 3),
	}
}

type Watcher struct {
	Options
	Program *tea.Program
	Printer Printer
	Args    []string

	errorChan chan error
}

func (w *Watcher) ErrorChan() <-chan error {
	return w.errorChan
}

func (w *Watcher) WatchLoop(ctx context.Context, restartChan <-chan struct{}) error {
	clearBeforePrinting := false
	watchErrChan := make(chan error, 1)
	for {
		var wg sync.WaitGroup
		wg.Add(1)
		watchCtx, cancel := context.WithCancel(ctx)
		go func(clearBeforePrinting bool) {
			defer wg.Done()
			err := w.watch(watchCtx, clearBeforePrinting)
			if watchCtx.Err() == nil {
				watchErrChan <- err
			}
		}(clearBeforePrinting)
		clearBeforePrinting = true
		select {
		case err := <-watchErrChan:
			if cmd := w.Printer.Table.StartSpinner(); cmd != nil {
				w.Program.Send(cmd())
			}
			w.errorChan <- fmt.Errorf("restart in 5s: %w", err)
			time.Sleep(5 * time.Second)
			w.Printer.Table.StopSpinner()
			w.Printer.Table.SetError(nil)
		case <-restartChan:
			if cmd := w.Printer.Table.StartSpinner(); cmd != nil {
				w.Program.Send(cmd())
			}
			// Prevent it from restarting too eagerly when we're told to restart
			// so the filesystem has time to flush, such as in case of
			// "kubectx" on bigger kubeconfigs.
			// https://github.com/applejag/kubectl-klock/issues/62
			slidingSleep(150*time.Millisecond, restartChan)
			cancel()
		case <-ctx.Done():
			cancel()
			return ctx.Err()
		}
		wg.Wait()
	}
}

func slidingSleep(dur time.Duration, ch <-chan struct{}) {
	timer := time.NewTimer(dur)
	for {
		select {
		case <-timer.C:
			return
		case <-ch:
			if !timer.Stop() {
				<-timer.C
			}
			timer.Reset(dur)
		}
	}
}

func (w *Watcher) Watch(ctx context.Context) error {
	return w.watch(ctx, false)
}

func (w *Watcher) watch(ctx context.Context, clearBeforePrinting bool) error {
	ns, _, err := w.ConfigFlags.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return fmt.Errorf("read namespace: %w", err)
	}
	if ns == "" {
		return fmt.Errorf("no namespace selected")
	}

	r := resource.NewBuilder(w.ConfigFlags).
		Unstructured().
		NamespaceParam(ns).DefaultNamespace().AllNamespaces(w.AllNamespaces).
		// FilenameParam(o.ExplicitNamespace, &o.FilenameOptions).
		LabelSelectorParam(w.LabelSelector).
		FieldSelectorParam(w.FieldSelector).
		// RequestChunksOf(o.ChunkSize).
		ResourceTypeOrNameArgs(true, w.Args...).
		SingleResourceType().
		Latest().
		TransformRequests(func(req *rest.Request) {
			// Custom columns need full objects, not the server-side Table.
			if accept := acceptHeader(len(w.Printer.CustomColumns) > 0); accept != "" {
				req.SetHeader("Accept", accept)
			}
		}).
		Do()
	if err := r.Err(); err != nil {
		return err
	}

	infos, err := r.Infos()
	if err != nil {
		return err
	}

	if len(infos) != 1 {
		return fmt.Errorf("expected a single resource info, but got %d", len(infos))
	}
	info := infos[0]
	obj := info.Object
	mapping := info.Mapping

	printNamespace := w.Options.AllNamespaces
	if mapping != nil && mapping.Scope.Name() == meta.RESTScopeNameRoot {
		// Resource isn't namespaced
		printNamespace = false
	}
	w.Printer.Configure(mapping.GroupVersionKind, printNamespace)

	// watching from resourceVersion 0, starts the watch at ~now and
	// will return an initial watch event.  Starting form ~now, rather
	// the resVersion of the object will insure that we start the watch from
	// inside the watch window, which the resVersion of the object might not be.
	resVersion := "0"
	isList := meta.IsListType(obj)
	var objsToPrint []runtime.Object
	if isList {
		// the resourceVersion of list objects is ~now but won't return
		// an initial watch event
		resVersion, err = meta.NewAccessor().ResourceVersion(obj)
		if err != nil {
			return err
		}
		objsToPrint, _ = meta.ExtractList(obj)
	} else {
		objsToPrint = []runtime.Object{obj}
	}

	if clearBeforePrinting {
		w.Printer.Clear()
	}

	for _, objToPrint := range objsToPrint {
		if _, err := w.Printer.PrintObj(objToPrint, watch.Added); err != nil {
			return err
		}
	}

	w.Printer.Table.StopSpinner()

	return w.pipeEvents(ctx, r, resVersion)
}

func (w *Watcher) pipeEvents(ctx context.Context, r *resource.Result, resVersion string) error {
	watch, err := r.Watch(resVersion)
	if err != nil {
		return err
	}
	defer watch.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case event, ok := <-watch.ResultChan():
			if !ok {
				return fmt.Errorf("watch channel closed")
			}
			cmd, err := w.Printer.PrintObj(event.Object, event.Type)
			if err != nil {
				return err
			}
			w.Program.Send(cmd())
		}
	}
}

type Printer struct {
	Kubecolor        *config.Config
	Table            *table.Model
	HideDeletedAfter types.OptionalDuration
	WideOutput       bool
	colDefs          []metav1.TableColumnDefinition
	LabelCols        []string
	// CustomColumns, when non-empty, switches the printer to custom-columns mode.
	CustomColumns []CustomColumn

	info           schema.GroupVersionKind
	apiVersion     string
	kind           string
	printNamespace bool
}

func (p *Printer) Configure(info schema.GroupVersionKind, printNamespace bool) {
	p.info = info
	p.apiVersion, p.kind = info.ToAPIVersionAndKind()
	p.printNamespace = printNamespace
}

func (p *Printer) Clear() {
	p.Table.SetRows(nil)
}

func (p *Printer) PrintObj(obj runtime.Object, eventType watch.EventType) (tea.Cmd, error) {
	if len(p.CustomColumns) > 0 {
		return p.printCustom(obj, eventType)
	}
	objTable, err := decodeIntoTable(obj)
	if err != nil {
		return nil, err
	}
	p.updateColDefHeaders(objTable)
	return p.addObjectToTable(objTable, eventType)
}

// printCustom adds obj to the table in custom-columns mode. Objects are full
// resources here, since no Table representation is requested.
func (p *Printer) printCustom(obj runtime.Object, eventType watch.EventType) (tea.Cmd, error) {
	u, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return nil, fmt.Errorf("want *unstructured.Unstructured, got %T", obj)
	}
	headers := make([]string, len(p.CustomColumns))
	for i, col := range p.CustomColumns {
		headers[i] = col.Header
	}
	p.Table.SetHeaders(headers)
	row, err := p.customRow(u, eventType)
	if err != nil {
		return nil, err
	}
	return p.Table.AddRow(row), nil
}

func (p *Printer) updateColDefHeaders(objTable *metav1.Table) {
	if len(objTable.ColumnDefinitions) == 0 {
		return
	}

	numColumns := len(objTable.ColumnDefinitions)
	if p.printNamespace {
		numColumns++
	}

	headers := make([]string, 0, numColumns)

	if p.printNamespace {
		headers = append(headers, "NAMESPACE")
	}
	for _, colDef := range objTable.ColumnDefinitions {
		if colDef.Priority == 0 || p.WideOutput {
			// Escape after upper-casing: "\u202e" must not become "\U202E".
			headers = append(headers, sanitizeCell(strings.ToUpper(colDef.Name)))
		}
	}
	for _, label := range p.LabelCols {
		headers = append(headers, labelColumnHeader(label))
	}
	p.Table.SetHeaders(headers)
	p.colDefs = objTable.ColumnDefinitions
}

func labelColumnHeader(label string) string {
	label = sanitizeCell(strings.ToUpper(label))
	index := strings.LastIndexByte(label, '/')
	if index == -1 {
		return label
	}
	return label[index+1:]
}

func (p *Printer) addObjectToTable(objTable *metav1.Table, eventType watch.EventType) (tea.Cmd, error) {
	var cmd tea.Cmd
	for _, row := range objTable.Rows {
		unstrucObj, ok := row.Object.Object.(*unstructured.Unstructured)
		if !ok {
			return nil, fmt.Errorf("want *unstructured.Unstructured, got %T", row.Object.Object)
		}
		tableRow, creationTime, err := p.baseRow(unstrucObj, eventType)
		if err != nil {
			return nil, err
		}
		tableRow.HasLeadingNamespaceColumn = p.printNamespace
		if p.printNamespace {
			tableRow.Fields = append(tableRow.Fields, unstrucObj.GetNamespace())
		}
		for i, cell := range row.Cells {
			if i >= len(p.colDefs) {
				return nil, fmt.Errorf("cant find index %d (%v) in column defs: %v", i, cell, p.colDefs)
			}
			colDef := p.colDefs[i]
			if colDef.Priority != 0 && !p.WideOutput {
				continue
			}
			// Escape once here so every parseCell branch gets safe text,
			// including the ones that return the cell as is. Numbers and
			// booleans cannot carry control characters; nil stays nil.
			switch cell.(type) {
			case nil, bool, int, int32, int64, float32, float64:
			default:
				cell = boundCell(sanitizeCell(fmt.Sprint(cell)))
			}
			tableRow.Fields = append(tableRow.Fields, p.parseCell(cell, row, eventType, unstrucObj.Object, colDef, creationTime))
		}
		objLabels := unstrucObj.GetLabels()
		for _, label := range p.LabelCols {
			tableRow.Fields = append(tableRow.Fields, sanitizeCell(objLabels[label]))
		}
		// it's fine to only use the latest returned cmd, because of how
		// [table.Model.AddRow] is implemented

		cmd = p.Table.AddRow(tableRow)
	}
	return cmd, nil
}

// baseRow builds the part of a table row shared by table and custom-columns
// modes: identity, sort key, suggestion, theme and event status. Fields is
// left empty and HasLeadingNamespaceColumn false; callers fill both.
//
// It returns the row and the object's creation time. It returns an error if
// metadata.uid, metadata.name or metadata.creationTimestamp is missing or
// malformed.
func (p *Printer) baseRow(obj *unstructured.Unstructured, eventType watch.EventType) (table.Row, time.Time, error) {
	metadata, ok := obj.Object["metadata"].(map[string]any)
	if !ok {
		return table.Row{}, time.Time{}, fmt.Errorf("metadata: want map[string]any, got %T", obj.Object["metadata"])
	}
	uid, ok := metadata["uid"].(string)
	if !ok {
		return table.Row{}, time.Time{}, fmt.Errorf("metadata.uid: want string, got %T", metadata["uid"])
	}
	name, ok := metadata["name"].(string)
	if !ok {
		return table.Row{}, time.Time{}, fmt.Errorf("metadata.name: want string, got %T", metadata["name"])
	}
	creationTimestamp, ok := metadata["creationTimestamp"].(string)
	if !ok {
		return table.Row{}, time.Time{}, fmt.Errorf("metadata.creationTimestamp: want string, got %T", metadata["creationTimestamp"])
	}
	creationTime, err := time.Parse(time.RFC3339, creationTimestamp)
	if err != nil {
		return table.Row{}, time.Time{}, fmt.Errorf("metadata.creationTimestamp: %w", err)
	}
	row := table.Row{
		ID:         uid,
		SortKey:    name,
		Suggestion: name,
		Kubecolor:  p.Kubecolor,
	}
	if p.apiVersion == "v1" && p.kind == "Event" {
		row.SortKey = creationTimestamp
	}
	if p.printNamespace {
		// The namespace keeps the sort order stable in all-namespaces mode
		// even when no NAMESPACE cell is shown.
		row.SortKey = fmt.Sprintf("%s/%s", metadata["namespace"], row.SortKey)
	}
	switch eventType {
	case watch.Error:
		row.Status = table.StatusError
	case watch.Deleted:
		row.MarkDeleted()
	}
	return row, creationTime, nil
}

// customRow builds the table row for obj in custom-columns mode: one cell
// per configured column and no NAMESPACE cell.
//
// It returns an error if the object's metadata is malformed (see baseRow).
// A failing JSONPath expression does not fail the row; it shows in its cell.
func (p *Printer) customRow(obj *unstructured.Unstructured, eventType watch.EventType) (table.Row, error) {
	row, _, err := p.baseRow(obj, eventType)
	if err != nil {
		return table.Row{}, err
	}
	row.Fields = make([]any, 0, len(p.CustomColumns))
	for _, col := range p.CustomColumns {
		cell := colorCell(evalCell(col.Path, obj.Object), eventType)
		// Keep uncolored cells as plain strings: wrapping every value in a
		// no-op style only adds work for the table renderer.
		if styled, ok := cell.(table.StyledColumn); ok && styled.Style.GetForeground() == (lipgloss.NoColor{}) {
			cell = styled.Value
		}
		row.Fields = append(row.Fields, cell)
	}
	return row, nil
}

// evalCell evaluates path against obj and renders the result like
// "kubectl get -o custom-columns": values joined by ",", "<none>" when empty,
// control characters escaped (see sanitizeCell). An evaluation error is
// returned as "<error: ...>" so one bad object does not stop the watch.
func evalCell(path *jsonpath.JSONPath, obj map[string]any) string {
	values, err := path.FindResults(obj)
	if err != nil {
		return sanitizeCell(fmt.Sprintf("<error: %v>", err))
	}
	if len(values) == 0 || len(values[0]) == 0 {
		return "<none>"
	}
	var parts []string
	for _, vs := range values {
		for _, v := range vs {
			parts = append(parts, sanitizeCell(fmt.Sprint(v.Interface())))
		}
	}
	return boundCell(strings.Join(parts, ","))
}

// maxCellRunes is the longest cell, in runes, that boundCell keeps.
const maxCellRunes = 1024

// boundCell truncates s to maxCellRunes runes and appends "…" when it cut
// something. Call it after sanitizeCell so escapes are counted, not split.
func boundCell(s string) string {
	r := []rune(s)
	if len(r) <= maxCellRunes {
		return s
	}
	return string(r[:maxCellRunes]) + "…"
}

// sanitizeCell makes s safe to show in a single table cell.
//
// It first applies kubectl's printers.EscapeTerminal (ESC becomes "^[", "\r"
// is handled there), then replaces every remaining control or bidi control
// rune, line or paragraph separator (Zl, Zp), invisible format rune (Cf,
// such as zero-width or soft hyphen) and other default-ignorable rune (such
// as U+3164 or U+034F) with its ASCII-quoted form without quotes ("\n",
// "\t", "\u009b", "\u200b"). ASCII quoting matters: strconv.QuoteRune
// leaves printable ignorables like U+3164 raw.
// This deliberately goes further than kubectl: a raw newline would break the
// TUI layout, and C1, bidi or invisible runes can still alter the terminal or
// hide text.
func sanitizeCell(s string) string {
	s = printers.EscapeTerminal(s)
	var b strings.Builder
	for _, r := range s {
		if unicode.IsControl(r) || unicode.Is(unicode.Bidi_Control, r) ||
			unicode.In(r, unicode.Zl, unicode.Zp, unicode.Cf) ||
			unicode.Is(unicode.Other_Default_Ignorable_Code_Point, r) {
			q := strconv.QuoteRuneToASCII(r)
			b.WriteString(q[1 : len(q)-1]) // drop the surrounding apostrophes
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// colorCell styles a cell as a fraction ("1/2") or a status word. Deleted
// rows stay unstyled so the grayed-out row is not colored.
func colorCell(cell any, eventType watch.EventType) any {
	cellStr := fmt.Sprint(cell)
	if eventType == watch.Deleted {
		return cellStr
	}
	if fractionStyle, ok := FractionStyle(cellStr); ok {
		return table.StyledColumn{Value: cell, Style: fractionStyle}
	}
	return StatusColumn(cellStr)
}

func (p *Printer) parseCell(cell any, row metav1.TableRow, eventType watch.EventType, object map[string]any, colDef metav1.TableColumnDefinition, creationTime time.Time) any {
	cellStr := fmt.Sprint(cell)
	columnNameLower := strings.ToLower(colDef.Name)
	switch {
	case columnNameLower == "age",
		// some non-namespaced resources (e.g Role) gives timestamp instead of age
		columnNameLower == "created at":
		return creationTime
	case p.apiVersion == "v1" && p.kind == "Event" && columnNameLower == "last seen",
		p.apiVersion == "batch/v1" && p.kind == "CronJob" && columnNameLower == "last schedule":

		dur, ok := util.ParseHumanDuration(cellStr)
		if !ok {
			return cell
		}
		return time.Now().Add(-dur)
	case p.apiVersion == "batch/v1" && p.kind == "Job" && columnNameLower == "duration":
		var completionsCell any
		for i, otherCell := range row.Cells {
			if i >= len(p.colDefs) {
				continue
			}
			def := p.colDefs[i]
			if strings.EqualFold(def.Name, "completions") {
				completionsCell = otherCell
				break
			}
		}
		if completionsCell == nil {
			return cell
		}
		f, ok := ParseFraction(fmt.Sprint(completionsCell))
		if !ok {
			return cell
		}
		if f.Count >= f.Total {
			return cell
		}
		dur, ok := util.ParseHumanDuration(cellStr)
		if !ok {
			return cell
		}
		return time.Now().Add(-dur)
	case p.apiVersion == "v1" && p.kind == "Pod" && columnNameLower == "restarts":
		// 0, the most common case
		if cellStr == "0" {
			return cell
		}
		countStr, dur, ok := parsePodRestarts(cellStr)
		if ok {
			cell = table.AgoColumn{
				Value: countStr,
				Time:  time.Now().Add(-dur),
			}
		}
		// Only add styling if not deleted, to not add excess coloring
		if eventType != watch.Deleted {
			cell = table.StyledColumn{
				Value: cell,
				Style: StyleFractionWarning,
			}
		}
		return cell
	case columnNameLower == "status":
		if eventType == watch.Deleted {
			return table.AgoColumn{
				Value: "Deleted",
				Time:  time.Now(),
			}
		}
		return StatusColumn(cellStr)
	default:
		return colorCell(cell, eventType)
	}
}

// acceptHeader returns the Accept header value to send to the API server.
//
// In table mode it asks for the server-side Table representation, falling
// back to plain JSON. In custom-columns mode it returns "": the JSONPath
// expressions apply to full objects, so the Table form must not be requested
// (kubectl's get does the same outside human-readable printing).
func acceptHeader(customColumns bool) string {
	if customColumns {
		return ""
	}
	return strings.Join([]string{
		fmt.Sprintf("application/json;as=Table;v=%s;g=%s", metav1.SchemeGroupVersion.Version, metav1.GroupName),
		fmt.Sprintf("application/json;as=Table;v=%s;g=%s", metav1beta1.SchemeGroupVersion.Version, metav1beta1.GroupName),
		"application/json",
	}, ",")
}

func overrideLipglossWithKubecolor(style *lipgloss.Style, c kubecolor.Color) {
	*style = kubecolorColorToLipgloss(c).SetString(style.Value())
}

func kubecolorColorToLipgloss(c kubecolor.Color) lipgloss.Style {
	style := lipgloss.NewStyle()
	for _, col := range c.Parsed {
		switch col := col.(type) {
		case color.Color256:
			if col.IsFg() {
				style = style.Foreground(lipgloss.Color("#" + col.RGB().Hex()))
			} else {
				style = style.Background(lipgloss.Color("#" + col.RGB().Hex()))
			}
		case color.RGBColor:
			if col.Color().IsFg() {
				style = style.Foreground(lipgloss.Color("#" + col.Hex()))
			} else {
				style = style.Background(lipgloss.Color("#" + col.Hex()))
			}
		case color.Color:
			switch {
			case col.IsFg():
				style = style.Foreground(lipgloss.ANSIColor(uint(col) - uint(color.FgBase)))
			case col.IsBg():
				style = style.Background(lipgloss.ANSIColor(uint(col) - uint(color.BgBase)))
			case col.IsOption():
				switch col {
				case color.OpBold:
					style = style.Bold(true)
				case color.OpItalic:
					style = style.Italic(true)
				case color.OpUnderscore:
					style = style.Underline(true)
				case color.OpFuzzy:
					style = style.Faint(true)
				case color.OpStrikethrough:
					style = style.Strikethrough(true)
				}
			}
		default:
			fmt.Printf("unknown: %T\n", col)
		}
	}
	return style
}
