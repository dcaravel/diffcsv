// main.go
package main

import (
	"os"

	"github.com/spf13/cobra"
)

func main() {
	var keyColumns, ignoreColumns, outputDir string
	var showIgnored, showDuplicates bool

	rootCmd := &cobra.Command{
		Use:   "diffcsv <before.csv> <after.csv>",
		Short: "Compare two CSV files and produce categorized output files",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCompare(args[0], args[1], keyColumns, ignoreColumns, outputDir, showIgnored, showDuplicates)
		},
		SilenceUsage: true,
	}

	rootCmd.Flags().StringVarP(&keyColumns, "key-cols", "", "", "comma-separated list of columns that form the row identity (required)")
	rootCmd.Flags().StringVar(&ignoreColumns, "ignore-cols", "", "comma-separated list of columns to exclude from change detection")
	rootCmd.Flags().StringVar(&outputDir, "output-dir", ".", "directory for output files")
	rootCmd.Flags().BoolVar(&showIgnored, "show-ignored", false, "write ignored.csv for rows changed only in ignored columns")
	rootCmd.Flags().BoolVar(&showDuplicates, "show-duplicates", false, "write duplicates.csv for rows with duplicate keys")
	rootCmd.MarkFlagRequired("key-cols")
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
