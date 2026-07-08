# diffcsv

Compare two CSV files and produce categorized output files showing what was added, removed, changed, and unchanged.

## Install

```
go install github.com/dcaravel/diffcsv@latest
```

## Usage

```
diffcsv <before.csv> <after.csv> --key-cols <columns> [flags]
```

### Flags

| Flag | Description |
|------|-------------|
| `--key-cols` | Comma-separated columns that form the row identity (required) |
| `--ignore-cols` | Comma-separated columns to exclude from change detection |
| `--output-dir` | Directory for output files (default `.`) |
| `--show-ignored` | Write `ignored.csv` for rows changed only in ignored columns |
| `--show-duplicates` | Write `duplicates.csv` for rows with duplicate keys |

### Output files

diffcsv always produces four files:

| File | Contents |
|------|----------|
| `added.csv` | Rows present in the after file but not before |
| `removed.csv` | Rows present in the before file but not after |
| `changed.csv` | Rows present in both files with differing values (includes a `Changes` column) |
| `unchanged.csv` | Rows identical in both files |

Optional files:

| File | Produced when | Contents |
|------|---------------|----------|
| `ignored.csv` | `--show-ignored` | Rows that differ only in ignored columns (includes a `Changes` column) |
| `duplicates.csv` | `--show-duplicates` | All rows with duplicate keys, tagged with their source (`A` or `B`) |

Without `--show-ignored`, rows that differ only in ignored columns are included in `unchanged.csv`.

### Extra columns

Some output files include columns that are not present in the input data:

| Column | Appears in | Description |
|--------|------------|-------------|
| `Changes` | `changed.csv`, `ignored.csv` | A human-readable summary of which fields differ between the before and after rows, formatted as `column: old → new` pairs separated by semicolons. Empty values are shown as `<empty>`. |
| `Source` | `duplicates.csv` | Indicates which input file the row came from: `A` (before file) or `B` (after file). |

### Summary

A summary is printed to stderr:

```
Rows in A:       1000
Rows in B:       1050
Added:           75
Removed:         25
Changed:         30
Unchanged:       945
```

## Examples

The `testdata/` directory contains `before.csv` and `after.csv` that exercise every feature. Try them with the examples below.

Compare using a composite key:

```
diffcsv testdata/before.csv testdata/after.csv --key-cols "ID,Region"
```

Ignore a column and write the ignored changes to a separate file:

```
diffcsv testdata/before.csv testdata/after.csv --key-cols "ID,Region" --ignore-cols Notes --show-ignored
```

Write output to a specific directory and include duplicate key info:

```
diffcsv testdata/before.csv testdata/after.csv --key-cols "ID,Region" --ignore-cols Notes --output-dir ./diff-results --show-ignored --show-duplicates
```

## How it works

1. Both files are parsed and rows are grouped by their key column values.
2. Rows with matching keys are compared field-by-field across the union of both files' columns (so columns can differ between files).
3. When duplicate keys exist within a file, rows are paired positionally — first occurrence to first occurrence, etc. Surplus rows are treated as added or removed.
4. Results are written to separate CSV files, all normalized to the union header.
