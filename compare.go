// compare.go
package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type fieldChange struct {
	column string
	oldVal string
	newVal string
}

type compareRow struct {
	key    string
	values []string
}

type diffRow struct {
	status         string
	rowA           []string
	rowB           []string
	changes        []fieldChange
	ignoredChanges []fieldChange
}

type duplicateKeyInfo struct {
	key    string
	countA int
	countB int
}

type compareResult struct {
	added         []diffRow
	removed       []diffRow
	changed       []diffRow
	unchanged     []diffRow
	ignored       []diffRow
	unionHeader   []string
	duplicateKeys  []duplicateKeyInfo
	duplicateRowsA int
	duplicateRowsB int
}

func parseCSV(path string, keyColumns []string) (header []string, keyIndices []int, rows []compareRow, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("opening %s: %w", path, err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	allRows, err := reader.ReadAll()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("reading %s: %w", path, err)
	}

	if len(allRows) == 0 {
		return nil, nil, nil, fmt.Errorf("%s: empty CSV", path)
	}

	header = allRows[0]

	colIdx := make(map[string]int, len(header))
	for i, col := range header {
		colIdx[col] = i
	}

	keyIndices = make([]int, len(keyColumns))
	var missing []string
	for i, kc := range keyColumns {
		idx, ok := colIdx[kc]
		if !ok {
			missing = append(missing, kc)
			continue
		}
		keyIndices[i] = idx
	}
	if len(missing) > 0 {
		return nil, nil, nil, fmt.Errorf("%s: missing key column(s): %s", path, strings.Join(missing, ", "))
	}

	rows = make([]compareRow, 0, len(allRows)-1)
	for _, row := range allRows[1:] {
		parts := make([]string, len(keyColumns))
		for i, ki := range keyIndices {
			if ki < len(row) {
				parts[i] = row[ki]
			}
		}
		rows = append(rows, compareRow{
			key:    strings.Join(parts, "\x1f"),
			values: row,
		})
	}

	return header, keyIndices, rows, nil
}

func buildUnionHeader(headerA, headerB []string) []string {
	seen := make(map[string]bool, len(headerB))
	union := make([]string, len(headerB))
	copy(union, headerB)
	for _, col := range headerB {
		seen[col] = true
	}
	for _, col := range headerA {
		if !seen[col] {
			union = append(union, col)
			seen[col] = true
		}
	}
	return union
}

func compareOneRow(rA, rB compareRow, unionHeader []string, colIdxA, colIdxB map[string]int, keySet map[string]bool, ignoreColumns map[string]bool) diffRow {
	var changes []fieldChange
	var ignoredChanges []fieldChange
	for _, col := range unionHeader {
		if keySet[col] {
			continue
		}
		valA := ""
		if idx, ok := colIdxA[col]; ok && idx < len(rA.values) {
			valA = rA.values[idx]
		}
		valB := ""
		if idx, ok := colIdxB[col]; ok && idx < len(rB.values) {
			valB = rB.values[idx]
		}
		if valA != valB {
			fc := fieldChange{column: col, oldVal: valA, newVal: valB}
			if ignoreColumns[col] {
				ignoredChanges = append(ignoredChanges, fc)
			} else {
				changes = append(changes, fc)
			}
		}
	}

	if len(changes) > 0 {
		return diffRow{status: "CHANGED", rowA: rA.values, rowB: rB.values, changes: changes, ignoredChanges: ignoredChanges}
	}
	if len(ignoredChanges) > 0 {
		return diffRow{status: "IGNORED", rowA: rA.values, rowB: rB.values, ignoredChanges: ignoredChanges}
	}
	return diffRow{status: "UNCHANGED", rowB: rB.values}
}

func appendDiffRow(result *compareResult, dr diffRow) {
	switch dr.status {
	case "CHANGED":
		result.changed = append(result.changed, dr)
	case "IGNORED":
		result.ignored = append(result.ignored, dr)
	case "UNCHANGED":
		result.unchanged = append(result.unchanged, dr)
	}
}

func diffRows(headerA, headerB []string, keyIdxA, keyIdxB []int, rowsA, rowsB []compareRow, ignoreColumns map[string]bool) compareResult {
	unionHeader := buildUnionHeader(headerA, headerB)

	colIdxA := make(map[string]int, len(headerA))
	for i, col := range headerA {
		colIdxA[col] = i
	}
	colIdxB := make(map[string]int, len(headerB))
	for i, col := range headerB {
		colIdxB[col] = i
	}

	keySet := make(map[string]bool)
	for _, ki := range keyIdxA {
		if ki < len(headerA) {
			keySet[headerA[ki]] = true
		}
	}

	type rowGroup struct {
		rowsA []int
		rowsB []int
	}
	groups := make(map[string]*rowGroup)
	var keyOrder []string

	for i, r := range rowsA {
		g, ok := groups[r.key]
		if !ok {
			g = &rowGroup{}
			groups[r.key] = g
			keyOrder = append(keyOrder, r.key)
		}
		g.rowsA = append(g.rowsA, i)
	}
	for i, r := range rowsB {
		g, ok := groups[r.key]
		if !ok {
			g = &rowGroup{}
			groups[r.key] = g
			keyOrder = append(keyOrder, r.key)
		}
		g.rowsB = append(g.rowsB, i)
	}

	var result compareResult
	result.unionHeader = unionHeader

	for _, key := range keyOrder {
		g := groups[key]
		if len(g.rowsA) > 1 || len(g.rowsB) > 1 {
			result.duplicateKeys = append(result.duplicateKeys, duplicateKeyInfo{
				key:    key,
				countA: len(g.rowsA),
				countB: len(g.rowsB),
			})
			if len(g.rowsA) > 1 {
				result.duplicateRowsA += len(g.rowsA) - 1
			}
			if len(g.rowsB) > 1 {
				result.duplicateRowsB += len(g.rowsB) - 1
			}
		}
	}

	for _, key := range keyOrder {
		g := groups[key]

		paired := min(len(g.rowsA), len(g.rowsB))
		for i := range paired {
			rA := rowsA[g.rowsA[i]]
			rB := rowsB[g.rowsB[i]]
			dr := compareOneRow(rA, rB, unionHeader, colIdxA, colIdxB, keySet, ignoreColumns)
			appendDiffRow(&result, dr)
		}
		for i := paired; i < len(g.rowsA); i++ {
			result.removed = append(result.removed, diffRow{
				status: "REMOVED",
				rowA:   rowsA[g.rowsA[i]].values,
			})
		}
		for i := paired; i < len(g.rowsB); i++ {
			result.added = append(result.added, diffRow{
				status: "ADDED",
				rowB:   rowsB[g.rowsB[i]].values,
			})
		}
	}

	return result
}

func formatChanges(changes []fieldChange) string {
	parts := make([]string, len(changes))
	for i, c := range changes {
		old, new := c.oldVal, c.newVal
		if old == "" {
			old = "<empty>"
		}
		if new == "" {
			new = "<empty>"
		}
		parts[i] = fmt.Sprintf("%s: %s → %s", c.column, old, new)
	}
	return strings.Join(parts, "; ")
}

func rowToUnion(row []string, sourceHeader []string, unionHeader []string) []string {
	colIdx := make(map[string]int, len(sourceHeader))
	for i, col := range sourceHeader {
		colIdx[col] = i
	}
	out := make([]string, len(unionHeader))
	for i, col := range unionHeader {
		if idx, ok := colIdx[col]; ok && idx < len(row) {
			out[i] = row[idx]
		}
	}
	return out
}

func writeSplitOutput(dir string, result compareResult, headerA, headerB []string, rowsA, rowsB []compareRow, showIgnored, showDuplicates bool) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	type splitFile struct {
		name       string
		rows       []diffRow
		useA       bool
		addChanges bool
	}

	// When showIgnored is false, ignored rows merge into unchanged.
	unchangedRows := result.unchanged
	if !showIgnored {
		unchangedRows = append(unchangedRows, result.ignored...)
	}

	files := []splitFile{
		{"added.csv", result.added, false, false},
		{"removed.csv", result.removed, true, false},
		{"changed.csv", result.changed, false, true},
		{"unchanged.csv", unchangedRows, false, false},
	}
	if showIgnored {
		files = append(files, splitFile{"ignored.csv", result.ignored, false, true})
	}

	for _, sf := range files {
		f, err := os.Create(filepath.Join(dir, sf.name))
		if err != nil {
			return fmt.Errorf("creating %s: %w", sf.name, err)
		}

		writer := csv.NewWriter(f)
		header := make([]string, len(result.unionHeader))
		copy(header, result.unionHeader)
		if sf.addChanges {
			header = append(header, "Changes")
		}
		if err := writer.Write(header); err != nil {
			f.Close()
			return err
		}

		for _, dr := range sf.rows {
			src := dr.rowB
			srcHeader := headerB
			if sf.useA {
				src = dr.rowA
				srcHeader = headerA
			}
			mapped := rowToUnion(src, srcHeader, result.unionHeader)
			if sf.addChanges {
				allChanges := dr.changes
				if sf.name == "ignored.csv" {
					allChanges = dr.ignoredChanges
				}
				mapped = append(mapped, formatChanges(allChanges))
			}
			if err := writer.Write(mapped); err != nil {
				f.Close()
				return err
			}
		}

		writer.Flush()
		if err := writer.Error(); err != nil {
			f.Close()
			return fmt.Errorf("flushing %s: %w", sf.name, err)
		}
		if err := f.Close(); err != nil {
			return fmt.Errorf("closing %s: %w", sf.name, err)
		}
	}

	if showDuplicates && len(result.duplicateKeys) > 0 {
		dupeKeysA := make(map[string]bool, len(result.duplicateKeys))
		dupeKeysB := make(map[string]bool, len(result.duplicateKeys))
		for _, d := range result.duplicateKeys {
			if d.countA > 1 {
				dupeKeysA[d.key] = true
			}
			if d.countB > 1 {
				dupeKeysB[d.key] = true
			}
		}

		f, err := os.Create(filepath.Join(dir, "duplicates.csv"))
		if err != nil {
			return fmt.Errorf("creating duplicates.csv: %w", err)
		}

		writer := csv.NewWriter(f)
		dupeHeader := append([]string{"Source"}, result.unionHeader...)
		if err := writer.Write(dupeHeader); err != nil {
			f.Close()
			return err
		}

		for _, r := range rowsA {
			if !dupeKeysA[r.key] {
				continue
			}
			mapped := rowToUnion(r.values, headerA, result.unionHeader)
			row := append([]string{"A"}, mapped...)
			if err := writer.Write(row); err != nil {
				f.Close()
				return err
			}
		}
		for _, r := range rowsB {
			if !dupeKeysB[r.key] {
				continue
			}
			mapped := rowToUnion(r.values, headerB, result.unionHeader)
			row := append([]string{"B"}, mapped...)
			if err := writer.Write(row); err != nil {
				f.Close()
				return err
			}
		}

		writer.Flush()
		if err := writer.Error(); err != nil {
			f.Close()
			return fmt.Errorf("flushing duplicates.csv: %w", err)
		}
		if err := f.Close(); err != nil {
			return fmt.Errorf("closing duplicates.csv: %w", err)
		}
	}

	return nil
}

func runCompare(csvA, csvB, keyColumnsStr, ignoreColumnsStr, outputDir string, showIgnored, showDuplicates bool) error {
	keyColumns := strings.Split(keyColumnsStr, ",")
	for i := range keyColumns {
		keyColumns[i] = strings.TrimSpace(keyColumns[i])
	}

	var ignoreColumns map[string]bool
	if ignoreColumnsStr != "" {
		ignoreColumns = make(map[string]bool)
		for _, col := range strings.Split(ignoreColumnsStr, ",") {
			ignoreColumns[strings.TrimSpace(col)] = true
		}
	}

	headerA, keyIdxA, rowsA, err := parseCSV(csvA, keyColumns)
	if err != nil {
		return err
	}
	headerB, keyIdxB, rowsB, err := parseCSV(csvB, keyColumns)
	if err != nil {
		return err
	}

	result := diffRows(headerA, headerB, keyIdxA, keyIdxB, rowsA, rowsB, ignoreColumns)

	unchangedCount := len(result.unchanged)
	if !showIgnored {
		unchangedCount += len(result.ignored)
	}

	if showDuplicates && result.duplicateRowsA > 0 {
		fmt.Fprintf(os.Stderr, "Rows in A:       %d (%d duplicate keys)\n", len(rowsA), result.duplicateRowsA)
	} else {
		fmt.Fprintf(os.Stderr, "Rows in A:       %d\n", len(rowsA))
	}
	if showDuplicates && result.duplicateRowsB > 0 {
		fmt.Fprintf(os.Stderr, "Rows in B:       %d (%d duplicate keys)\n", len(rowsB), result.duplicateRowsB)
	} else {
		fmt.Fprintf(os.Stderr, "Rows in B:       %d\n", len(rowsB))
	}
	fmt.Fprintf(os.Stderr, "Added:           %d\n", len(result.added))
	fmt.Fprintf(os.Stderr, "Removed:         %d\n", len(result.removed))
	fmt.Fprintf(os.Stderr, "Changed:         %d\n", len(result.changed))
	fmt.Fprintf(os.Stderr, "Unchanged:       %d\n", unchangedCount)
	if showIgnored && len(result.ignored) > 0 {
		fmt.Fprintf(os.Stderr, "Ignored:         %d\n", len(result.ignored))
	}

	return writeSplitOutput(outputDir, result, headerA, headerB, rowsA, rowsB, showIgnored, showDuplicates)
}
