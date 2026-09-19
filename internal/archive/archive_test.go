package archive

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractZIPRejectsTraversalAndLinks(t *testing.T) {
	for name, mode := range map[string]os.FileMode{
		"../outside":       0644,
		"folder/../../bad": 0644,
		"folder\\..\\bad":  0644,
		"link":             os.ModeSymlink | 0777,
	} {
		t.Run(strings.ReplaceAll(name, "/", "_"), func(t *testing.T) {
			archivePath := makeZIP(t, name, mode, "unsafe")
			_, err := Extract(archivePath, "zip", t.TempDir(), 0, 1024)
			if err == nil {
				t.Fatalf("expected %q to be rejected", name)
			}
		})
	}
}

func TestExtractZIPStripsOneDirectoryAndChecksExpandedSize(t *testing.T) {
	archivePath := makeZIP(t, "release/launch.sh", 0755, "#!/bin/sh\n")
	files, err := Extract(archivePath, "zip", t.TempDir(), 1, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].Relative != "launch.sh" || files[0].Mode.Perm() != 0755 {
		t.Fatalf("unexpected extracted files: %#v", files)
	}
	if _, err := Extract(archivePath, "zip", t.TempDir(), 1, 2); err == nil {
		t.Fatal("expected expanded-size limit to reject archive")
	}
}

func TestExtractZIPAcceptsValidatedDirectoryEntries(t *testing.T) {
	archivePath := filepath.Join(t.TempDir(), "fixture.zip")
	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	directory := &zip.FileHeader{Name: "release/"}
	directory.SetMode(os.ModeDir | 0755)
	if _, err := writer.CreateHeader(directory); err != nil {
		t.Fatal(err)
	}
	header := &zip.FileHeader{Name: "release/launch.sh", Method: zip.Store}
	header.SetMode(0644)
	entry, err := writer.CreateHeader(header)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write([]byte("run")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	files, err := Extract(archivePath, "zip", t.TempDir(), 1, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].Relative != "launch.sh" {
		t.Fatalf("unexpected extracted files: %#v", files)
	}
}

func TestExtractTarGZRejectsLinksAndExtractsRegularFiles(t *testing.T) {
	regular := makeTarGZ(t, &tar.Header{Name: "release/launch.sh", Mode: 0755, Size: 3, Typeflag: tar.TypeReg}, "run")
	files, err := Extract(regular, "tar.gz", t.TempDir(), 1, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].Relative != "launch.sh" {
		t.Fatalf("unexpected extracted files: %#v", files)
	}
	link := makeTarGZ(t, &tar.Header{Name: "link", Linkname: "../outside", Mode: 0777, Typeflag: tar.TypeSymlink}, "")
	if _, err := Extract(link, "tar.gz", t.TempDir(), 0, 100); err == nil {
		t.Fatal("expected tar symlink to be rejected")
	}
	traversalDirectory := makeTarGZ(t, &tar.Header{Name: "../outside/", Mode: 0755, Typeflag: tar.TypeDir}, "")
	if _, err := Extract(traversalDirectory, "tar.gz", t.TempDir(), 0, 100); err == nil {
		t.Fatal("expected tar traversal directory to be rejected")
	}
}

// The tar.gz arm decompresses before it trusts anything, so an input that is
// not gzip at all has to be rejected before a tar header is ever parsed.
func TestExtractTarGZRejectsInputThatIsNotGzip(t *testing.T) {
	archivePath := filepath.Join(t.TempDir(), "fixture.tar.gz")
	if err := os.WriteFile(archivePath, []byte("not a gzip stream"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Extract(archivePath, "tar.gz", t.TempDir(), 1, 1024); err == nil {
		t.Fatal("expected a non-gzip stream to be rejected")
	}
}

// A well-formed gzip wrapper around a damaged tar stream has to be rejected
// too, so the failure is reported from the decompressed bytes rather than being
// mistaken for a truncated download.
func TestExtractTarGZRejectsACorruptTarStream(t *testing.T) {
	archivePath := writeCorruptTarGZ(t)
	if _, err := Extract(archivePath, "tar.gz", t.TempDir(), 1, 1024); err == nil {
		t.Fatal("expected a corrupt tar stream to be rejected")
	}
}

// writeCorruptTarGZ gzips a tar header whose bytes no longer match its own
// checksum, so the damage survives compression and only shows up once the tar
// reader parses it.
func writeCorruptTarGZ(t *testing.T) string {
	t.Helper()
	var raw bytes.Buffer
	writer := tar.NewWriter(&raw)
	if err := writer.WriteHeader(&tar.Header{Name: "release/launch.sh", Mode: 0755, Size: 3, Typeflag: tar.TypeReg}); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write([]byte("run")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	corrupt := raw.Bytes()
	corrupt[0] ^= 0xff
	archivePath := filepath.Join(t.TempDir(), "fixture.tar.gz")
	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	compressed := gzip.NewWriter(file)
	if _, err := compressed.Write(corrupt); err != nil {
		t.Fatal(err)
	}
	if err := compressed.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return archivePath
}

func makeZIP(t *testing.T, name string, mode os.FileMode, body string) string {
	t.Helper()
	archivePath := filepath.Join(t.TempDir(), "fixture.zip")
	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	header := &zip.FileHeader{Name: name, Method: zip.Store}
	header.SetMode(mode)
	entry, err := writer.CreateHeader(header)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write([]byte(body)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return archivePath
}

func makeTarGZ(t *testing.T, header *tar.Header, body string) string {
	t.Helper()
	archivePath := filepath.Join(t.TempDir(), "fixture.tar.gz")
	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	compressed := gzip.NewWriter(file)
	writer := tar.NewWriter(compressed)
	if err := writer.WriteHeader(header); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write([]byte(body)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := compressed.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return archivePath
}
