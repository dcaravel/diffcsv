// compare_test.go
package main

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTestCSV(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestParseCSV_Basic(t *testing.T) {
	dir := t.TempDir()
	csv := "Name,Age,City\nAlice,30,NYC\nBob,25,LA\n"
	p := writeTestCSV(t, dir, "test.csv", csv)

	header, keyIdx, rows, err := parseCSV(p, []string{"Name"})
	if err != nil {
		t.Fatal(err)
	}
	if len(header) != 3 {
		t.Errorf("expected 3 header columns, got %d", len(header))
	}
	if len(keyIdx) != 1 || keyIdx[0] != 0 {
		t.Errorf("expected key index [0], got %v", keyIdx)
	}
	if len(rows) != 2 {
		t.Errorf("expected 2 rows, got %d", len(rows))
	}
	if rows[0].key != "Alice" {
		t.Errorf("expected key 'Alice', got %q", rows[0].key)
	}
}

func TestParseCSV_MissingKeyColumn(t *testing.T) {
	dir := t.TempDir()
	csv := "Name,Age\nAlice,30\n"
	p := writeTestCSV(t, dir, "test.csv", csv)

	_, _, _, err := parseCSV(p, []string{"Name", "Missing"})
	if err == nil {
		t.Fatal("expected error for missing key column")
	}
}

func TestParseCSV_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	p := writeTestCSV(t, dir, "empty.csv", "")

	_, _, _, err := parseCSV(p, []string{"Name"})
	if err == nil {
		t.Fatal("expected error for empty CSV")
	}
}

func TestParseCSV_CompositeKey(t *testing.T) {
	dir := t.TempDir()
	csv := "First,Last,Age\nAlice,Smith,30\nBob,Jones,25\n"
	p := writeTestCSV(t, dir, "test.csv", csv)

	_, keyIdx, rows, err := parseCSV(p, []string{"First", "Last"})
	if err != nil {
		t.Fatal(err)
	}
	if len(keyIdx) != 2 {
		t.Errorf("expected 2 key indices, got %d", len(keyIdx))
	}
	if rows[0].key != "Alice\x1fSmith" {
		t.Errorf("expected composite key 'Alice\\x1fSmith', got %q", rows[0].key)
	}
}

func TestDiffRows_Identical(t *testing.T) {
	dir := t.TempDir()
	csv := "Name,Age,City\nAlice,30,NYC\nBob,25,LA\n"
	pathA := writeTestCSV(t, dir, "a.csv", csv)
	pathB := writeTestCSV(t, dir, "b.csv", csv)

	keyCols := []string{"Name"}
	headerA, keyIdxA, rowsA, err := parseCSV(pathA, keyCols)
	if err != nil {
		t.Fatal(err)
	}
	headerB, keyIdxB, rowsB, err := parseCSV(pathB, keyCols)
	if err != nil {
		t.Fatal(err)
	}

	result := diffRows(headerA, headerB, keyIdxA, keyIdxB, rowsA, rowsB, nil)

	if len(result.added) != 0 {
		t.Errorf("expected 0 added, got %d", len(result.added))
	}
	if len(result.removed) != 0 {
		t.Errorf("expected 0 removed, got %d", len(result.removed))
	}
	if len(result.changed) != 0 {
		t.Errorf("expected 0 changed, got %d", len(result.changed))
	}
	if len(result.unchanged) != 2 {
		t.Errorf("expected 2 unchanged, got %d", len(result.unchanged))
	}
}

func TestDiffRows_AllCategories(t *testing.T) {
	dir := t.TempDir()
	csvA := "Name,Age,City\nAlice,30,NYC\nBob,25,LA\nCharlie,35,SF\n"
	csvB := "Name,Age,City\nAlice,31,NYC\nCharlie,35,SF\nDave,40,CHI\n"
	pathA := writeTestCSV(t, dir, "a.csv", csvA)
	pathB := writeTestCSV(t, dir, "b.csv", csvB)

	keyCols := []string{"Name"}
	headerA, keyIdxA, rowsA, _ := parseCSV(pathA, keyCols)
	headerB, keyIdxB, rowsB, _ := parseCSV(pathB, keyCols)

	result := diffRows(headerA, headerB, keyIdxA, keyIdxB, rowsA, rowsB, nil)

	if len(result.removed) != 1 {
		t.Errorf("expected 1 removed, got %d", len(result.removed))
	}
	if len(result.added) != 1 {
		t.Errorf("expected 1 added, got %d", len(result.added))
	}
	if len(result.changed) != 1 {
		t.Errorf("expected 1 changed, got %d", len(result.changed))
	}
	if len(result.unchanged) != 1 {
		t.Errorf("expected 1 unchanged, got %d", len(result.unchanged))
	}

	ch := result.changed[0]
	if len(ch.changes) != 1 {
		t.Fatalf("expected 1 change field, got %d", len(ch.changes))
	}
	if ch.changes[0].column != "Age" || ch.changes[0].oldVal != "30" || ch.changes[0].newVal != "31" {
		t.Errorf("unexpected change: %+v", ch.changes[0])
	}
}

func TestDiffRows_DifferentColumnOrder(t *testing.T) {
	dir := t.TempDir()
	csvA := "Name,Age,City\nAlice,30,NYC\n"
	csvB := "City,Name,Age\nLA,Alice,30\n"
	pathA := writeTestCSV(t, dir, "a.csv", csvA)
	pathB := writeTestCSV(t, dir, "b.csv", csvB)

	keyCols := []string{"Name"}
	headerA, keyIdxA, rowsA, _ := parseCSV(pathA, keyCols)
	headerB, keyIdxB, rowsB, _ := parseCSV(pathB, keyCols)

	result := diffRows(headerA, headerB, keyIdxA, keyIdxB, rowsA, rowsB, nil)

	if len(result.changed) != 1 {
		t.Fatalf("expected 1 changed, got %d", len(result.changed))
	}
	c := result.changed[0].changes[0]
	if c.column != "City" || c.oldVal != "NYC" || c.newVal != "LA" {
		t.Errorf("expected City NYC→LA, got %s %s→%s", c.column, c.oldVal, c.newVal)
	}
}

func TestDiffRows_DifferentColumnSets(t *testing.T) {
	dir := t.TempDir()
	csvA := "Name,Age,City\nAlice,30,NYC\n"
	csvB := "Name,Age,State\nAlice,30,NY\n"
	pathA := writeTestCSV(t, dir, "a.csv", csvA)
	pathB := writeTestCSV(t, dir, "b.csv", csvB)

	keyCols := []string{"Name"}
	headerA, keyIdxA, rowsA, _ := parseCSV(pathA, keyCols)
	headerB, keyIdxB, rowsB, _ := parseCSV(pathB, keyCols)

	result := diffRows(headerA, headerB, keyIdxA, keyIdxB, rowsA, rowsB, nil)

	if len(result.changed) != 1 {
		t.Fatalf("expected 1 changed, got %d", len(result.changed))
	}
	if len(result.changed[0].changes) != 2 {
		t.Fatalf("expected 2 changed fields, got %d", len(result.changed[0].changes))
	}

	expected := []string{"Name", "Age", "State", "City"}
	if len(result.unionHeader) != len(expected) {
		t.Fatalf("expected %d union columns, got %d: %v", len(expected), len(result.unionHeader), result.unionHeader)
	}
	for i, col := range expected {
		if result.unionHeader[i] != col {
			t.Errorf("union header[%d]: expected %q, got %q", i, col, result.unionHeader[i])
		}
	}
}

func TestDiffRows_DuplicateKeys(t *testing.T) {
	dir := t.TempDir()
	csvA := "Name,Age\nAlice,30\nAlice,30\nAlice,25\n"
	csvB := "Name,Age\nAlice,30\n"
	pathA := writeTestCSV(t, dir, "a.csv", csvA)
	pathB := writeTestCSV(t, dir, "b.csv", csvB)

	keyCols := []string{"Name"}
	headerA, keyIdxA, rowsA, _ := parseCSV(pathA, keyCols)
	headerB, keyIdxB, rowsB, _ := parseCSV(pathB, keyCols)

	result := diffRows(headerA, headerB, keyIdxA, keyIdxB, rowsA, rowsB, nil)

	if len(result.unchanged) != 1 {
		t.Errorf("expected 1 unchanged, got %d", len(result.unchanged))
	}
	if len(result.removed) != 2 {
		t.Errorf("expected 2 removed, got %d", len(result.removed))
	}
}

func TestDiffRows_IgnoreColumns(t *testing.T) {
	dir := t.TempDir()
	csvA := "Name,Age,City\nAlice,30,NYC\n"
	csvB := "Name,Age,City\nAlice,31,LA\n"
	pathA := writeTestCSV(t, dir, "a.csv", csvA)
	pathB := writeTestCSV(t, dir, "b.csv", csvB)

	keyCols := []string{"Name"}
	headerA, keyIdxA, rowsA, _ := parseCSV(pathA, keyCols)
	headerB, keyIdxB, rowsB, _ := parseCSV(pathB, keyCols)

	ignore := map[string]bool{"City": true}
	result := diffRows(headerA, headerB, keyIdxA, keyIdxB, rowsA, rowsB, ignore)

	// Age changed (not ignored) → changed
	if len(result.changed) != 1 {
		t.Fatalf("expected 1 changed, got %d", len(result.changed))
	}
	// The changed row should have Age as a change and City as an ignored change
	ch := result.changed[0]
	if len(ch.changes) != 1 || ch.changes[0].column != "Age" {
		t.Errorf("expected Age change, got %+v", ch.changes)
	}
	if len(ch.ignoredChanges) != 1 || ch.ignoredChanges[0].column != "City" {
		t.Errorf("expected City ignored change, got %+v", ch.ignoredChanges)
	}
}

func TestDiffRows_IgnoreColumnsOnly(t *testing.T) {
	dir := t.TempDir()
	csvA := "Name,Age,City\nAlice,30,NYC\n"
	csvB := "Name,Age,City\nAlice,30,LA\n"
	pathA := writeTestCSV(t, dir, "a.csv", csvA)
	pathB := writeTestCSV(t, dir, "b.csv", csvB)

	keyCols := []string{"Name"}
	headerA, keyIdxA, rowsA, _ := parseCSV(pathA, keyCols)
	headerB, keyIdxB, rowsB, _ := parseCSV(pathB, keyCols)

	ignore := map[string]bool{"City": true}
	result := diffRows(headerA, headerB, keyIdxA, keyIdxB, rowsA, rowsB, ignore)

	// Only City changed (ignored) → goes to ignored, not changed
	if len(result.changed) != 0 {
		t.Errorf("expected 0 changed, got %d", len(result.changed))
	}
	if len(result.ignored) != 1 {
		t.Errorf("expected 1 ignored, got %d", len(result.ignored))
	}
	if len(result.unchanged) != 0 {
		t.Errorf("expected 0 unchanged, got %d", len(result.unchanged))
	}
}

func TestDiffRows_IgnoreColumnsNoChange(t *testing.T) {
	dir := t.TempDir()
	csv := "Name,Age,City\nAlice,30,NYC\n"
	pathA := writeTestCSV(t, dir, "a.csv", csv)
	pathB := writeTestCSV(t, dir, "b.csv", csv)

	keyCols := []string{"Name"}
	headerA, keyIdxA, rowsA, _ := parseCSV(pathA, keyCols)
	headerB, keyIdxB, rowsB, _ := parseCSV(pathB, keyCols)

	ignore := map[string]bool{"City": true}
	result := diffRows(headerA, headerB, keyIdxA, keyIdxB, rowsA, rowsB, ignore)

	// Nothing changed at all → unchanged
	if len(result.unchanged) != 1 {
		t.Errorf("expected 1 unchanged, got %d", len(result.unchanged))
	}
	if len(result.ignored) != 0 {
		t.Errorf("expected 0 ignored, got %d", len(result.ignored))
	}
}

func TestWriteSplitOutput(t *testing.T) {
	dir := t.TempDir()
	csvA := "Name,Age,City\nAlice,30,NYC\nBob,25,LA\nCharlie,35,SF\n"
	csvB := "Name,Age,City\nAlice,31,NYC\nCharlie,35,SF\nDave,40,CHI\n"
	pathA := writeTestCSV(t, dir, "a.csv", csvA)
	pathB := writeTestCSV(t, dir, "b.csv", csvB)

	keyCols := []string{"Name"}
	headerA, keyIdxA, rowsA, _ := parseCSV(pathA, keyCols)
	headerB, keyIdxB, rowsB, _ := parseCSV(pathB, keyCols)
	result := diffRows(headerA, headerB, keyIdxA, keyIdxB, rowsA, rowsB, nil)

	splitDir := filepath.Join(dir, "output")
	if err := writeSplitOutput(splitDir, result, headerA, headerB, rowsA, rowsB, false, false); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name         string
		expectedRows int
		hasChanges   bool
	}{
		{"added.csv", 2, false},
		{"removed.csv", 2, false},
		{"changed.csv", 2, true},
		{"unchanged.csv", 2, false},
	} {
		p := filepath.Join(splitDir, tc.name)
		data, err := os.ReadFile(p)
		if err != nil {
			t.Errorf("missing %s: %v", tc.name, err)
			continue
		}
		r := csv.NewReader(strings.NewReader(string(data)))
		rows, err := r.ReadAll()
		if err != nil {
			t.Errorf("parsing %s: %v", tc.name, err)
			continue
		}
		if len(rows) != tc.expectedRows {
			t.Errorf("%s: expected %d rows, got %d", tc.name, tc.expectedRows, len(rows))
		}
		header := rows[0]
		lastCol := header[len(header)-1]
		if tc.hasChanges && lastCol != "Changes" {
			t.Errorf("%s: expected last column 'Changes', got %q", tc.name, lastCol)
		}
		if !tc.hasChanges {
			for _, col := range header {
				if col == "Changes" {
					t.Errorf("%s: should not have Changes column", tc.name)
				}
			}
		}
	}

	// ignored.csv should NOT exist when showIgnored is false
	if _, err := os.Stat(filepath.Join(splitDir, "ignored.csv")); err == nil {
		t.Error("ignored.csv should not exist when showIgnored is false")
	}
}

func TestWriteSplitOutput_ShowIgnored(t *testing.T) {
	dir := t.TempDir()
	csvA := "Name,Age,City\nAlice,30,NYC\n"
	csvB := "Name,Age,City\nAlice,30,LA\n"
	pathA := writeTestCSV(t, dir, "a.csv", csvA)
	pathB := writeTestCSV(t, dir, "b.csv", csvB)

	keyCols := []string{"Name"}
	ignore := map[string]bool{"City": true}
	headerA, keyIdxA, rowsA, _ := parseCSV(pathA, keyCols)
	headerB, keyIdxB, rowsB, _ := parseCSV(pathB, keyCols)
	result := diffRows(headerA, headerB, keyIdxA, keyIdxB, rowsA, rowsB, ignore)

	splitDir := filepath.Join(dir, "output")
	if err := writeSplitOutput(splitDir, result, headerA, headerB, rowsA, rowsB, true, false); err != nil {
		t.Fatal(err)
	}

	// ignored.csv should exist with 2 rows (header + 1 data row)
	data, err := os.ReadFile(filepath.Join(splitDir, "ignored.csv"))
	if err != nil {
		t.Fatal("ignored.csv should exist when showIgnored is true")
	}
	r := csv.NewReader(strings.NewReader(string(data)))
	rows, err := r.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Errorf("ignored.csv: expected 2 rows, got %d", len(rows))
	}
	lastCol := rows[0][len(rows[0])-1]
	if lastCol != "Changes" {
		t.Errorf("ignored.csv: expected last column 'Changes', got %q", lastCol)
	}

	// unchanged.csv should be header-only (no data rows — the ignored row went to ignored.csv)
	data2, _ := os.ReadFile(filepath.Join(splitDir, "unchanged.csv"))
	r2 := csv.NewReader(strings.NewReader(string(data2)))
	uRows, _ := r2.ReadAll()
	if len(uRows) != 1 {
		t.Errorf("unchanged.csv: expected 1 row (header only), got %d", len(uRows))
	}
}

func TestWriteSplitOutput_IgnoredToUnchanged(t *testing.T) {
	dir := t.TempDir()
	csvA := "Name,Age,City\nAlice,30,NYC\n"
	csvB := "Name,Age,City\nAlice,30,LA\n"
	pathA := writeTestCSV(t, dir, "a.csv", csvA)
	pathB := writeTestCSV(t, dir, "b.csv", csvB)

	keyCols := []string{"Name"}
	ignore := map[string]bool{"City": true}
	headerA, keyIdxA, rowsA, _ := parseCSV(pathA, keyCols)
	headerB, keyIdxB, rowsB, _ := parseCSV(pathB, keyCols)
	result := diffRows(headerA, headerB, keyIdxA, keyIdxB, rowsA, rowsB, ignore)

	splitDir := filepath.Join(dir, "output")
	// showIgnored=false → ignored rows go to unchanged.csv
	if err := writeSplitOutput(splitDir, result, headerA, headerB, rowsA, rowsB, false, false); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(filepath.Join(splitDir, "unchanged.csv"))
	r := csv.NewReader(strings.NewReader(string(data)))
	rows, _ := r.ReadAll()
	if len(rows) != 2 {
		t.Errorf("unchanged.csv: expected 2 rows (header + ignored row), got %d", len(rows))
	}
}

func TestRunCompare_Basic(t *testing.T) {
	dir := t.TempDir()
	csvA := "Name,Age,City\nAlice,30,NYC\nBob,25,LA\n"
	csvB := "Name,Age,City\nAlice,31,NYC\nCharlie,35,SF\n"
	pathA := writeTestCSV(t, dir, "a.csv", csvA)
	pathB := writeTestCSV(t, dir, "b.csv", csvB)
	outDir := filepath.Join(dir, "out")

	err := runCompare(pathA, pathB, "Name", "", outDir, false, false)
	if err != nil {
		t.Fatal(err)
	}

	// Verify output files exist
	for _, name := range []string{"added.csv", "removed.csv", "changed.csv", "unchanged.csv"} {
		if _, err := os.Stat(filepath.Join(outDir, name)); err != nil {
			t.Errorf("missing %s: %v", name, err)
		}
	}

	// Verify changed.csv content
	data, _ := os.ReadFile(filepath.Join(outDir, "changed.csv"))
	r := csv.NewReader(strings.NewReader(string(data)))
	rows, _ := r.ReadAll()
	if len(rows) != 2 {
		t.Errorf("changed.csv: expected 2 rows, got %d", len(rows))
	}
}

func TestRunCompare_WithIgnoreAndShowIgnored(t *testing.T) {
	dir := t.TempDir()
	csvA := "Name,Age,City\nAlice,30,NYC\n"
	csvB := "Name,Age,City\nAlice,30,LA\n"
	pathA := writeTestCSV(t, dir, "a.csv", csvA)
	pathB := writeTestCSV(t, dir, "b.csv", csvB)
	outDir := filepath.Join(dir, "out")

	err := runCompare(pathA, pathB, "Name", "City", outDir, true, false)
	if err != nil {
		t.Fatal(err)
	}

	// ignored.csv should exist
	data, err := os.ReadFile(filepath.Join(outDir, "ignored.csv"))
	if err != nil {
		t.Fatal("ignored.csv should exist with --show-ignored")
	}
	r := csv.NewReader(strings.NewReader(string(data)))
	rows, _ := r.ReadAll()
	if len(rows) != 2 {
		t.Errorf("ignored.csv: expected 2 rows, got %d", len(rows))
	}
}

func TestRunCompare_WithIgnoreNoShowIgnored(t *testing.T) {
	dir := t.TempDir()
	csvA := "Name,Age,City\nAlice,30,NYC\n"
	csvB := "Name,Age,City\nAlice,30,LA\n"
	pathA := writeTestCSV(t, dir, "a.csv", csvA)
	pathB := writeTestCSV(t, dir, "b.csv", csvB)
	outDir := filepath.Join(dir, "out")

	err := runCompare(pathA, pathB, "Name", "City", outDir, false, false)
	if err != nil {
		t.Fatal(err)
	}

	// ignored.csv should NOT exist
	if _, err := os.Stat(filepath.Join(outDir, "ignored.csv")); err == nil {
		t.Error("ignored.csv should not exist without --show-ignored")
	}

	// Row should be in unchanged.csv instead
	data, _ := os.ReadFile(filepath.Join(outDir, "unchanged.csv"))
	r := csv.NewReader(strings.NewReader(string(data)))
	rows, _ := r.ReadAll()
	if len(rows) != 2 {
		t.Errorf("unchanged.csv: expected 2 rows, got %d", len(rows))
	}
}

func TestDiffRows_DuplicateKeyInfo(t *testing.T) {
	dir := t.TempDir()
	csvA := "Name,Age\nAlice,30\nAlice,25\nBob,20\n"
	csvB := "Name,Age\nAlice,30\nBob,20\nBob,22\n"
	pathA := writeTestCSV(t, dir, "a.csv", csvA)
	pathB := writeTestCSV(t, dir, "b.csv", csvB)

	keyCols := []string{"Name"}
	headerA, keyIdxA, rowsA, _ := parseCSV(pathA, keyCols)
	headerB, keyIdxB, rowsB, _ := parseCSV(pathB, keyCols)

	result := diffRows(headerA, headerB, keyIdxA, keyIdxB, rowsA, rowsB, nil)

	if len(result.duplicateKeys) != 2 {
		t.Fatalf("expected 2 duplicate keys, got %d", len(result.duplicateKeys))
	}

	dupeMap := make(map[string]duplicateKeyInfo)
	for _, d := range result.duplicateKeys {
		dupeMap[d.key] = d
	}

	alice, ok := dupeMap["Alice"]
	if !ok {
		t.Fatal("expected Alice in duplicate keys")
	}
	if alice.countA != 2 || alice.countB != 1 {
		t.Errorf("Alice: expected countA=2, countB=1, got countA=%d, countB=%d", alice.countA, alice.countB)
	}

	bob, ok := dupeMap["Bob"]
	if !ok {
		t.Fatal("expected Bob in duplicate keys")
	}
	if bob.countA != 1 || bob.countB != 2 {
		t.Errorf("Bob: expected countA=1, countB=2, got countA=%d, countB=%d", bob.countA, bob.countB)
	}

	// Alice: 2 in A (1 extra), Bob: 2 in B (1 extra)
	if result.duplicateRowsA != 1 {
		t.Errorf("expected 1 duplicate row in A, got %d", result.duplicateRowsA)
	}
	if result.duplicateRowsB != 1 {
		t.Errorf("expected 1 duplicate row in B, got %d", result.duplicateRowsB)
	}
}

func TestDiffRows_NoDuplicateKeyInfo(t *testing.T) {
	dir := t.TempDir()
	csv := "Name,Age\nAlice,30\nBob,25\n"
	pathA := writeTestCSV(t, dir, "a.csv", csv)
	pathB := writeTestCSV(t, dir, "b.csv", csv)

	keyCols := []string{"Name"}
	headerA, keyIdxA, rowsA, _ := parseCSV(pathA, keyCols)
	headerB, keyIdxB, rowsB, _ := parseCSV(pathB, keyCols)

	result := diffRows(headerA, headerB, keyIdxA, keyIdxB, rowsA, rowsB, nil)

	if len(result.duplicateKeys) != 0 {
		t.Errorf("expected 0 duplicate keys, got %d", len(result.duplicateKeys))
	}
	if result.duplicateRowsA != 0 {
		t.Errorf("expected 0 duplicate rows in A, got %d", result.duplicateRowsA)
	}
	if result.duplicateRowsB != 0 {
		t.Errorf("expected 0 duplicate rows in B, got %d", result.duplicateRowsB)
	}
}

func TestDiffRows_PositionalPairing(t *testing.T) {
	dir := t.TempDir()
	// A has Alice rows in order: (30,NYC), (25,LA)
	// B has them reversed: (25,LA), (30,NYC)
	// Positional pairing pairs (30,NYC)↔(25,LA) and (25,LA)↔(30,NYC) → 2 changed
	csvA := "Name,Age,City\nAlice,30,NYC\nAlice,25,LA\n"
	csvB := "Name,Age,City\nAlice,25,LA\nAlice,30,NYC\n"
	pathA := writeTestCSV(t, dir, "a.csv", csvA)
	pathB := writeTestCSV(t, dir, "b.csv", csvB)

	keyCols := []string{"Name"}
	headerA, keyIdxA, rowsA, _ := parseCSV(pathA, keyCols)
	headerB, keyIdxB, rowsB, _ := parseCSV(pathB, keyCols)

	result := diffRows(headerA, headerB, keyIdxA, keyIdxB, rowsA, rowsB, nil)

	if len(result.changed) != 2 {
		t.Errorf("expected 2 changed with positional pairing, got %d", len(result.changed))
	}
	if len(result.unchanged) != 0 {
		t.Errorf("expected 0 unchanged with positional pairing, got %d", len(result.unchanged))
	}
}

func TestDiffRows_PositionalPairingUnequal(t *testing.T) {
	dir := t.TempDir()
	// A has 3 Alice rows, B has 1 — first A pairs with first B
	csvA := "Name,Age\nAlice,30\nAlice,25\nAlice,30\n"
	csvB := "Name,Age\nAlice,30\n"
	pathA := writeTestCSV(t, dir, "a.csv", csvA)
	pathB := writeTestCSV(t, dir, "b.csv", csvB)

	keyCols := []string{"Name"}
	headerA, keyIdxA, rowsA, _ := parseCSV(pathA, keyCols)
	headerB, keyIdxB, rowsB, _ := parseCSV(pathB, keyCols)

	result := diffRows(headerA, headerB, keyIdxA, keyIdxB, rowsA, rowsB, nil)

	// First A row (Alice,30) pairs with B row (Alice,30) → unchanged
	// Remaining 2 A rows → removed
	if len(result.unchanged) != 1 {
		t.Errorf("expected 1 unchanged, got %d", len(result.unchanged))
	}
	if len(result.removed) != 2 {
		t.Errorf("expected 2 removed, got %d", len(result.removed))
	}
	if len(result.changed) != 0 {
		t.Errorf("expected 0 changed, got %d", len(result.changed))
	}
}

func TestWriteSplitOutput_ShowDuplicates(t *testing.T) {
	dir := t.TempDir()
	csvA := "Name,Age\nAlice,30\nAlice,25\n"
	csvB := "Name,Age\nAlice,30\nBob,20\n"
	pathA := writeTestCSV(t, dir, "a.csv", csvA)
	pathB := writeTestCSV(t, dir, "b.csv", csvB)

	keyCols := []string{"Name"}
	headerA, keyIdxA, rowsA, _ := parseCSV(pathA, keyCols)
	headerB, keyIdxB, rowsB, _ := parseCSV(pathB, keyCols)
	result := diffRows(headerA, headerB, keyIdxA, keyIdxB, rowsA, rowsB, nil)

	splitDir := filepath.Join(dir, "output")
	if err := writeSplitOutput(splitDir, result, headerA, headerB, rowsA, rowsB, false, true); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(splitDir, "duplicates.csv"))
	if err != nil {
		t.Fatal("duplicates.csv should exist when showDuplicates is true")
	}
	r := csv.NewReader(strings.NewReader(string(data)))
	rows, err := r.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	// Header + 2 Alice rows from A only (Alice is not duplicate in B)
	if len(rows) != 3 {
		t.Errorf("duplicates.csv: expected 3 rows (header + 2 data), got %d", len(rows))
	}
	// First column should be "Source"
	if rows[0][0] != "Source" {
		t.Errorf("expected first column 'Source', got %q", rows[0][0])
	}
	// All rows should be from A
	for _, row := range rows[1:] {
		if row[0] != "A" {
			t.Errorf("expected all duplicate rows from A, got source %q", row[0])
		}
	}
}

func TestWriteSplitOutput_NoDuplicatesWithoutFlag(t *testing.T) {
	dir := t.TempDir()
	csvA := "Name,Age\nAlice,30\nAlice,25\n"
	csvB := "Name,Age\nAlice,30\n"
	pathA := writeTestCSV(t, dir, "a.csv", csvA)
	pathB := writeTestCSV(t, dir, "b.csv", csvB)

	keyCols := []string{"Name"}
	headerA, keyIdxA, rowsA, _ := parseCSV(pathA, keyCols)
	headerB, keyIdxB, rowsB, _ := parseCSV(pathB, keyCols)
	result := diffRows(headerA, headerB, keyIdxA, keyIdxB, rowsA, rowsB, nil)

	splitDir := filepath.Join(dir, "output")
	if err := writeSplitOutput(splitDir, result, headerA, headerB, rowsA, rowsB, false, false); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(splitDir, "duplicates.csv")); err == nil {
		t.Error("duplicates.csv should not exist when showDuplicates is false")
	}
}

func TestRunCompare_ShowDuplicates(t *testing.T) {
	dir := t.TempDir()
	csvA := "Name,Age\nAlice,30\nAlice,25\n"
	csvB := "Name,Age\nAlice,30\nBob,20\n"
	pathA := writeTestCSV(t, dir, "a.csv", csvA)
	pathB := writeTestCSV(t, dir, "b.csv", csvB)
	outDir := filepath.Join(dir, "out")

	err := runCompare(pathA, pathB, "Name", "", outDir, false, true)
	if err != nil {
		t.Fatal(err)
	}

	// duplicates.csv should exist
	if _, err := os.Stat(filepath.Join(outDir, "duplicates.csv")); err != nil {
		t.Error("duplicates.csv should exist with --show-duplicates")
	}
}

func TestRunCompare_NoDuplicatesFlag(t *testing.T) {
	dir := t.TempDir()
	csvA := "Name,Age\nAlice,30\nAlice,25\n"
	csvB := "Name,Age\nAlice,30\n"
	pathA := writeTestCSV(t, dir, "a.csv", csvA)
	pathB := writeTestCSV(t, dir, "b.csv", csvB)
	outDir := filepath.Join(dir, "out")

	err := runCompare(pathA, pathB, "Name", "", outDir, false, false)
	if err != nil {
		t.Fatal(err)
	}

	// duplicates.csv should NOT exist
	if _, err := os.Stat(filepath.Join(outDir, "duplicates.csv")); err == nil {
		t.Error("duplicates.csv should not exist without --show-duplicates")
	}
}

func TestParseCSV_DuplicateColumns(t *testing.T) {
	dir := t.TempDir()
	csv := "Name,Age,Name\nAlice,30,Bob\n"
	p := writeTestCSV(t, dir, "test.csv", csv)

	_, _, _, err := parseCSV(p, []string{"Age"})
	if err == nil {
		t.Fatal("expected error for duplicate column names")
	}
	if !strings.Contains(err.Error(), "duplicate column") {
		t.Errorf("expected 'duplicate column' in error, got: %s", err)
	}
	if !strings.Contains(err.Error(), `"Name"`) {
		t.Errorf("expected '\"Name\"' in error, got: %s", err)
	}
	if !strings.Contains(err.Error(), "1") || !strings.Contains(err.Error(), "3") {
		t.Errorf("expected column positions in error, got: %s", err)
	}
}

func TestParseCSV_MultipleDuplicateColumns(t *testing.T) {
	dir := t.TempDir()
	csv := "Name,Age,Name,Age\nAlice,30,Bob,25\n"
	p := writeTestCSV(t, dir, "test.csv", csv)

	_, _, _, err := parseCSV(p, []string{"Name"})
	if err == nil {
		t.Fatal("expected error for duplicate column names")
	}
	if !strings.Contains(err.Error(), `"Name"`) {
		t.Errorf("expected '\"Name\"' in error, got: %s", err)
	}
	if !strings.Contains(err.Error(), `"Age"`) {
		t.Errorf("expected '\"Age\"' in error, got: %s", err)
	}
}
