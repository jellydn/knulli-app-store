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
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	directory := flag.String("catalogue", "catalogue/packages", "package manifest directory")
	output := flag.String("output", "build/update-report", "report path without extension")
	flag.Parse()
	catalogueIndex, err := catalog.Build(*directory)
	if err != nil {
		return err
	}
	packages := make([]manifest.Package, len(catalogueIndex.Packages))
	for index := range catalogueIndex.Packages {
		packages[index] = catalogueIndex.Packages[index].Package
	}
	report := (updatecheck.Checker{Token: os.Getenv("GITHUB_TOKEN")}).Check(context.Background(), packages, time.Now())
	jsonData, err := updatecheck.JSON(report)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(*output), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(*output+".json", jsonData, 0644); err != nil {
		return err
	}
	if err := os.WriteFile(*output+".md", updatecheck.Markdown(report), 0644); err != nil {
		return err
	}
	fmt.Printf("wrote update reports for %d packages\n", len(packages))
	return nil
}
