package diagnostics

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestRedactRemovesURLCredentialsAndSecrets(t *testing.T) {
	input := `download https://name:pass@example.com/file.zip?token=abc#part password=hunter2 authorization:Bearer-token`
	got := Redact(input)
	for _, secret := range []string{"name:pass", "token=abc", "#part", "hunter2", "Bearer-token"} {
		if strings.Contains(got, secret) {
			t.Fatalf("redaction retained %q in %q", secret, got)
		}
	}
	if !strings.Contains(got, "https://example.com/file.zip") || !strings.Contains(got, "[REDACTED]") {
		t.Fatalf("redaction removed useful context: %q", got)
	}
}

func TestBoundedWriterRotatesAndCapsCurrentLog(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.log")
	writer := &boundedWriter{path: path, limit: 24}
	if _, err := writer.Write([]byte("first diagnostic line\n")); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write([]byte("second diagnostic line\n")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path + ".1"); err != nil {
		t.Fatalf("rotated log missing: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "second diagnostic line\n" {
		t.Fatalf("unexpected current log: %q, %v", data, err)
	}
}

func TestExportContainsMetadataAndRedactedLogsOnly(t *testing.T) {
	root := t.TempDir()
	logger, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	logger.Event("download_error", "url", "https://user:pass@example.com/release.zip?token=abc", "error", "password=hunter2")
	virtual, err := logger.Export([]string{"Platform: firmware=knulli", "Package: app.romm.grout experimental"})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(strings.TrimPrefix(virtual, "/"))))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !regexp.MustCompile(`\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2} event=download_error`).MatchString(text) {
		t.Fatalf("export lacks timestamped structured event: %s", text)
	}
	for _, wanted := range []string{"Platform: firmware=knulli", "Package: app.romm.grout experimental", "event=download_error", "https://example.com/release.zip", "[REDACTED]"} {
		if !strings.Contains(text, wanted) {
			t.Fatalf("export lacks %q: %s", wanted, text)
		}
	}
	for _, secret := range []string{"user:pass", "token=abc", "hunter2"} {
		if strings.Contains(text, secret) {
			t.Fatalf("export retained %q", secret)
		}
	}
}
