package archive

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
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
			_, err := Extract(context.Background(), archivePath, "zip", t.TempDir(), 0, 1024)
			if err == nil {
				t.Fatalf("expected %q to be rejected", name)
			}
		})
	}
}

func TestExtractZIPStripsOneDirectoryAndChecksExpandedSize(t *testing.T) {
	archivePath := makeZIP(t, "release/launch.sh", 0755, "#!/bin/sh\n")
	files, err := Extract(context.Background(), archivePath, "zip", t.TempDir(), 1, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].Relative != "launch.sh" || files[0].Mode.Perm() != 0755 {
		t.Fatalf("unexpected extracted files: %#v", files)
	}
	if _, err := Extract(context.Background(), archivePath, "zip", t.TempDir(), 1, 2); err == nil {
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
	files, err := Extract(context.Background(), archivePath, "zip", t.TempDir(), 1, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].Relative != "launch.sh" {
		t.Fatalf("unexpected extracted files: %#v", files)
	}
}

func TestExtractTarGZRejectsLinksAndExtractsRegularFiles(t *testing.T) {
	regular := makeTarGZ(t, &tar.Header{Name: "release/launch.sh", Mode: 0755, Size: 3, Typeflag: tar.TypeReg}, "run")
	files, err := Extract(context.Background(), regular, "tar.gz", t.TempDir(), 1, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].Relative != "launch.sh" {
		t.Fatalf("unexpected extracted files: %#v", files)
	}
	link := makeTarGZ(t, &tar.Header{Name: "link", Linkname: "../outside", Mode: 0777, Typeflag: tar.TypeSymlink}, "")
	if _, err := Extract(context.Background(), link, "tar.gz", t.TempDir(), 0, 100); err == nil {
		t.Fatal("expected tar symlink to be rejected")
	}
	traversalDirectory := makeTarGZ(t, &tar.Header{Name: "../outside/", Mode: 0755, Typeflag: tar.TypeDir}, "")
	if _, err := Extract(context.Background(), traversalDirectory, "tar.gz", t.TempDir(), 0, 100); err == nil {
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
	if _, err := Extract(context.Background(), archivePath, "tar.gz", t.TempDir(), 1, 1024); err == nil {
		t.Fatal("expected a non-gzip stream to be rejected")
	}
}

// A well-formed gzip wrapper around a damaged tar stream has to be rejected
// too, so the failure is reported from the decompressed bytes rather than being
// mistaken for a truncated download.
func TestExtractTarGZRejectsACorruptTarStream(t *testing.T) {
	archivePath := writeCorruptTarGZ(t)
	if _, err := Extract(context.Background(), archivePath, "tar.gz", t.TempDir(), 1, 1024); err == nil {
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

// A context that is already cancelled has to be refused before the archive is
// touched at all. An implementation that only checked inside the entry loop
// would write the first entry into the destination before it noticed.
func TestExtractRefusesAnAlreadyCancelledContext(t *testing.T) {
	for _, format := range []string{"zip", "tar.gz"} {
		t.Run(format, func(t *testing.T) {
			archivePath := makeArchive(t, format, []archiveEntry{{name: "launch.sh", body: "#!/bin/sh\n"}})
			destination := t.TempDir()
			cancelled, cancel := context.WithCancel(context.Background())
			cancel()

			_, err := Extract(cancelled, archivePath, format, destination, 0, 1024)
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("expected the cancelled context to be reported, got %v", err)
			}
			written, readErr := os.ReadDir(destination)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if len(written) != 0 {
				t.Fatalf("a cancelled extraction wrote %d entries into the destination", len(written))
			}
		})
	}
}

// Cancellation has to reach a running extraction at both points where work
// happens: partway through one entry's transfer, and between one entry and the
// next. One entry can run to hundreds of megabytes, so an implementation that
// checked only between entries would keep reading long after the user asked it
// to stop, and finish data.bin instead of leaving it short — which is the case
// the first scenario fails an unsafe implementation on. The first entry is far
// larger than one transfer buffer, so the mid-transfer case always has bytes on
// disk when the cancellation takes hold. The trigger is the extraction's own
// output rather than a timer or a call count, so the interruption lands at the
// same point on every run.
func TestExtractStopsAnExtractionThatIsAlreadyRunning(t *testing.T) {
	const entryBytes = 4 << 20
	body := strings.Repeat("x", entryBytes)
	for _, format := range []string{"zip", "tar.gz"} {
		for _, scenario := range []struct {
			name string
			// threshold is the size data.bin reaches before cancellation takes
			// hold, so each scenario stops at a different point in the run.
			threshold int64
			// finished is whether the first entry must be complete when it stops.
			finished bool
		}{
			{name: "partway through a transfer", threshold: 1 << 20},
			{name: "between two entries", threshold: entryBytes, finished: true},
		} {
			t.Run(format+" "+scenario.name, func(t *testing.T) {
				archivePath := makeArchive(t, format, []archiveEntry{
					{name: "data.bin", body: body},
					{name: "second.bin", body: "second"},
				})
				destination := t.TempDir()
				target := filepath.Join(destination, "data.bin")
				watching := &watchingContext{
					Context: context.Background(),
					watch: func() bool {
						info, err := os.Stat(target)
						return err == nil && info.Size() >= scenario.threshold
					},
				}

				_, err := Extract(watching, archivePath, format, destination, 0, entryBytes+1024)
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("expected the interrupted extraction to be reported as cancelled, got %v", err)
				}
				interrupted, statErr := os.Stat(target)
				if statErr != nil {
					t.Fatalf("expected the interrupted entry to be on disk: %v", statErr)
				}
				if scenario.finished {
					if interrupted.Size() != entryBytes {
						t.Fatalf("expected the completed entry to be %d bytes, it is %d", entryBytes, interrupted.Size())
					}
				} else if interrupted.Size() == 0 || interrupted.Size() >= entryBytes {
					t.Fatalf("expected the transfer to stop partway to %d bytes, it wrote %d", entryBytes, interrupted.Size())
				}
				if _, statErr := os.Stat(filepath.Join(destination, "second.bin")); !os.IsNotExist(statErr) {
					t.Fatal("expected the entry after the interrupted one never to be started")
				}
			})
		}
	}
}

// The wrapper is what makes a running transfer stoppable, so its contract is
// pinned directly: cancellation replaces the next read, and the underlying
// reader is never consulted again once the context is done.
func TestInterruptibleReportsCancellationInPlaceOfTheNextRead(t *testing.T) {
	reads := 0
	source := readerFunc(func([]byte) (int, error) {
		reads++
		return 1, nil
	})
	cancelled, cancel := context.WithCancel(context.Background())
	guarded := interruptible{ctx: cancelled, reader: source}
	buffer := make([]byte, 8)

	if _, err := guarded.Read(buffer); err != nil {
		t.Fatalf("expected a read before cancellation to succeed, got %v", err)
	}
	cancel()
	if _, err := guarded.Read(buffer); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected the cancelled context in place of the next read, got %v", err)
	}
	if reads != 1 {
		t.Fatalf("expected the source to be read once, it was read %d times", reads)
	}
}

// watchingContext reports cancellation once watch says the extraction has
// reached the point under test, which lets a test stop a run that is already
// under way at a fixed, observable moment.
type watchingContext struct {
	context.Context
	watch func() bool
}

func (c *watchingContext) Err() error {
	if c.watch() {
		return context.Canceled
	}
	return nil
}

type readerFunc func([]byte) (int, error)

func (f readerFunc) Read(buffer []byte) (int, error) { return f(buffer) }

// archiveEntry is one member of a fixture archive. Fixtures are ordered slices
// rather than maps so an extraction always meets the same member first.
type archiveEntry struct {
	name string
	body string
}

// makeArchive builds a multi-member fixture in either supported format, so a
// cancellation test can assert that a member was interrupted partway and that
// the member after it was never started.
func makeArchive(t *testing.T, format string, entries []archiveEntry) string {
	t.Helper()
	switch format {
	case "zip":
		archivePath := filepath.Join(t.TempDir(), "fixture.zip")
		file, err := os.Create(archivePath)
		if err != nil {
			t.Fatal(err)
		}
		writer := zip.NewWriter(file)
		for _, item := range entries {
			header := &zip.FileHeader{Name: item.name, Method: zip.Store}
			header.SetMode(0644)
			member, err := writer.CreateHeader(header)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := member.Write([]byte(item.body)); err != nil {
				t.Fatal(err)
			}
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
		return archivePath
	case "tar.gz":
		archivePath := filepath.Join(t.TempDir(), "fixture.tar.gz")
		file, err := os.Create(archivePath)
		if err != nil {
			t.Fatal(err)
		}
		compressed := gzip.NewWriter(file)
		writer := tar.NewWriter(compressed)
		for _, item := range entries {
			header := &tar.Header{Name: item.name, Mode: 0644, Size: int64(len(item.body)), Typeflag: tar.TypeReg}
			if err := writer.WriteHeader(header); err != nil {
				t.Fatal(err)
			}
			if _, err := writer.Write([]byte(item.body)); err != nil {
				t.Fatal(err)
			}
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
	default:
		t.Fatalf("unsupported fixture format %q", format)
		return ""
	}
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
