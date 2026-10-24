package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jellydn/knulli-app-store/internal/catalog"
	"github.com/jellydn/knulli-app-store/internal/manifest"
	"github.com/jellydn/knulli-app-store/internal/updatecheck"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(arguments []string) error {
	flags := flag.NewFlagSet("check-updates", flag.ContinueOnError)
	directory := flags.String("catalogue", "catalogue/packages", "package manifest directory")
	output := flags.String("output", "build/update-report", "report path without extension")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	return writeReports(updatecheck.Checker{Token: os.Getenv("GITHUB_TOKEN")}, *directory, *output, time.Now())
}

// writeReports builds the index, compares every reviewed package against its
// upstream releases, and writes the JSON and Markdown reports. It is separate
// from run so a report can be produced without the caller's flags, which is what
// lets the weekly checker be exercised offline.
//
// The checker never downloads an asset and never edits a manifest: it reports
// metadata for a human to review.
func writeReports(checker updatecheck.Checker, directory, output string, now time.Time) error {
	catalogueIndex, err := catalog.Build(directory)
	if err != nil {
		return err
	}
	packages := make([]manifest.Package, len(catalogueIndex.Packages))
	for index := range catalogueIndex.Packages {
		packages[index] = catalogueIndex.Packages[index].Package
	}
	report := checker.Check(context.Background(), packages, now)
	jsonData, err := updatecheck.JSON(report)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(output+".json", jsonData, 0644); err != nil {
		return err
	}
	if err := os.WriteFile(output+".md", updatecheck.Markdown(report), 0644); err != nil {
		return err
	}
	fmt.Printf("wrote update reports for %d packages\n", len(packages))
	return nil
}
