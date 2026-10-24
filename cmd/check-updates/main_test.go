package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jellydn/knulli-app-store/internal/updatecheck"
)

// The weekly checker is the least-covered entry point, and it runs unattended,
// so its report is exercised end to end against a local stand-in for the GitHub
// releases API rather than against the network.
func TestWriteReportsWritesBothReportsFromTheLiveCatalogue(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		// No stable release for any package: the report says so for each one
		// instead of claiming an update it cannot support.
		writer.Header().Set("Content-Type", "application/json")
		writer.Write([]byte("[]"))
	}))
	defer server.Close()

	output := filepath.Join(t.TempDir(), "reports", "catalogue-update-report")
	checker := updatecheck.Checker{BaseURL: server.URL, Client: server.Client()}
	if err := writeReports(checker, filepath.Join("..", "..", "catalogue", "packages"), output, time.Now()); err != nil {
		t.Fatal(err)
	}

	jsonData, err := os.ReadFile(output + ".json")
	if err != nil {
		t.Fatalf("the JSON report was not written: %v", err)
	}
	markdown, err := os.ReadFile(output + ".md")
	if err != nil {
		t.Fatalf("the Markdown report was not written: %v", err)
	}
	if len(markdown) == 0 {
		t.Fatal("the Markdown report is empty")
	}

	var report updatecheck.Report
	if err := json.Unmarshal(jsonData, &report); err != nil {
		t.Fatalf("the JSON report does not decode: %v", err)
	}
	if len(report.Packages) == 0 {
		t.Fatal("the report covers no packages")
	}
	for _, item := range report.Packages {
		if item.Status != "no-stable-release" {
			t.Fatalf("%s reported %q with no upstream release; the report invented a finding", item.ID, item.Status)
		}
		if item.ID == "" || item.Repository == "" {
			t.Fatalf("a report entry lacks the package identity it reports on: %#v", item)
		}
	}
}

// A report is only worth reviewing if it names the packages it checked, so the
// count has to match the catalogue the run was pointed at.
func TestWriteReportsCoversEveryReviewedPackage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Write([]byte("[]"))
	}))
	defer server.Close()

	output := filepath.Join(t.TempDir(), "report")
	checker := updatecheck.Checker{BaseURL: server.URL, Client: server.Client()}
	if err := writeReports(checker, filepath.Join("..", "..", "catalogue", "packages"), output, time.Now()); err != nil {
		t.Fatal(err)
	}
	jsonData, err := os.ReadFile(output + ".json")
	if err != nil {
		t.Fatal(err)
	}
	var report updatecheck.Report
	if err := json.Unmarshal(jsonData, &report); err != nil {
		t.Fatal(err)
	}

	entries, err := os.ReadDir(filepath.Join("..", "..", "catalogue", "packages"))
	if err != nil {
		t.Fatal(err)
	}
	manifests := 0
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".json" {
			manifests++
		}
	}
	if manifests == 0 {
		t.Fatal("the catalogue holds no manifests, so this test proves nothing")
	}
	if len(report.Packages) != manifests {
		t.Fatalf("checked %d packages, want one entry per manifest (%d)", len(report.Packages), manifests)
	}
}

func TestWriteReportsRejectsAMissingCatalogueDirectory(t *testing.T) {
	output := filepath.Join(t.TempDir(), "report")
	if err := writeReports(updatecheck.Checker{}, filepath.Join(t.TempDir(), "absent"), output, time.Now()); err == nil {
		t.Fatal("a missing catalogue directory produced a report instead of an error")
	}
	if _, err := os.Stat(output + ".json"); err == nil {
		t.Fatal("a failed run left a report behind")
	}
}

func TestRunRejectsAnUnknownFlag(t *testing.T) {
	if err := run([]string{"-not-a-flag"}); err == nil {
		t.Fatal("an unknown flag was accepted")
	}
}
