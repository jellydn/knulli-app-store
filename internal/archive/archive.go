package archive

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type File struct {
	Relative string
	Path     string
	Mode     os.FileMode
	Size     int64
	SHA256   string
}

// Extract writes every regular file the archive holds below destination and
// describes what it wrote. Cancelling ctx stops an extraction that is already
// running, both between entries and partway through a transfer. Nothing outside
// destination is written, so a cancelled extraction needs no rollback: the
// caller discards the directory it passed in.
func Extract(ctx context.Context, archivePath, format, destination string, stripComponents int, maximumBytes int64) ([]File, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(destination, 0700); err != nil {
		return nil, err
	}
	switch format {
	case "zip":
		return extractZIP(ctx, archivePath, destination, stripComponents, maximumBytes)
	case "tar.gz":
		return extractTarGZ(ctx, archivePath, destination, stripComponents, maximumBytes)
	default:
		return nil, fmt.Errorf("unsupported archive format %q", format)
	}
}

func extractZIP(ctx context.Context, archivePath, destination string, stripComponents int, maximumBytes int64) ([]File, error) {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	var files []File
	var total int64
	seen := make(map[string]bool)
	for _, item := range reader.File {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if item.Mode()&os.ModeSymlink != 0 || (!item.Mode().IsRegular() && !item.FileInfo().IsDir()) {
			return nil, fmt.Errorf("archive contains unsupported entry %q", item.Name)
		}
		name := item.Name
		if item.FileInfo().IsDir() {
			name = strings.TrimSuffix(name, "/")
		}
		relative, skip, err := safeName(name, stripComponents)
		if err != nil {
			return nil, err
		}
		if skip || item.FileInfo().IsDir() {
			continue
		}
		if seen[relative] {
			return nil, fmt.Errorf("archive contains duplicate path %q", relative)
		}
		seen[relative] = true
		if item.UncompressedSize64 > uint64(maximumBytes) || total > maximumBytes-int64(item.UncompressedSize64) {
			return nil, fmt.Errorf("archive expands beyond %d bytes", maximumBytes)
		}
		total += int64(item.UncompressedSize64)
		input, err := item.Open()
		if err != nil {
			return nil, err
		}
		file, err := extractFile(ctx, input, destination, relative, item.Mode(), int64(item.UncompressedSize64))
		input.Close()
		if err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	return files, nil
}

func extractTarGZ(ctx context.Context, archivePath, destination string, stripComponents int, maximumBytes int64) ([]File, error) {
	input, err := os.Open(archivePath)
	if err != nil {
		return nil, err
	}
	defer input.Close()
	gzipReader, err := gzip.NewReader(input)
	if err != nil {
		return nil, err
	}
	defer gzipReader.Close()
	reader := tar.NewReader(gzipReader)
	var files []File
	var total int64
	seen := make(map[string]bool)
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if header.Typeflag == tar.TypeDir {
			if _, _, err := safeName(strings.TrimSuffix(header.Name, "/"), stripComponents); err != nil {
				return nil, err
			}
			continue
		}
		if header.Typeflag != tar.TypeReg {
			return nil, fmt.Errorf("archive contains unsupported entry %q", header.Name)
		}
		relative, skip, err := safeName(header.Name, stripComponents)
		if err != nil {
			return nil, err
		}
		if skip {
			continue
		}
		if seen[relative] {
			return nil, fmt.Errorf("archive contains duplicate path %q", relative)
		}
		seen[relative] = true
		if header.Size < 0 || header.Size > maximumBytes || total > maximumBytes-header.Size {
			return nil, fmt.Errorf("archive expands beyond %d bytes", maximumBytes)
		}
		total += header.Size
		file, err := extractFile(ctx, reader, destination, relative, os.FileMode(header.Mode), header.Size)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	return files, nil
}

func extractFile(ctx context.Context, reader io.Reader, destination, relative string, mode os.FileMode, size int64) (File, error) {
	if err := ctx.Err(); err != nil {
		return File{}, err
	}
	target := filepath.Join(destination, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
		return File{}, err
	}
	permissions := mode.Perm() & 0755
	if permissions == 0 {
		permissions = 0644
	}
	output, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, permissions)
	if err != nil {
		return File{}, err
	}
	hash := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(output, hash), interruptible{ctx: ctx, reader: io.LimitReader(reader, size+1)})
	closeErr := output.Close()
	if copyErr != nil {
		return File{}, copyErr
	}
	if closeErr != nil {
		return File{}, closeErr
	}
	if written != size {
		return File{}, fmt.Errorf("archive entry %s has size %d, expected %d", relative, written, size)
	}
	return File{Relative: relative, Path: target, Mode: permissions, Size: size, SHA256: hex.EncodeToString(hash.Sum(nil))}, nil
}

// interruptible stops a transfer that is already in flight. One archive entry
// may be hundreds of megabytes, which is the longest stretch of work an install
// does without returning to its caller, so a check between entries alone would
// leave a cancelled operation reading for minutes before it noticed. The
// wrapper implements Read and nothing else, which keeps io.Copy on its generic
// path: a destination that could take over the transfer would otherwise bypass
// every check here.
type interruptible struct {
	ctx    context.Context
	reader io.Reader
}

func (r interruptible) Read(buffer []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(buffer)
}

func safeName(name string, stripComponents int) (string, bool, error) {
	normalized := strings.ReplaceAll(name, "\\", "/")
	if strings.HasPrefix(normalized, "/") || path.Clean(normalized) != normalized || normalized == "." || normalized == ".." || strings.HasPrefix(normalized, "../") {
		return "", false, fmt.Errorf("unsafe archive path %q", name)
	}
	parts := strings.Split(normalized, "/")
	if len(parts) <= stripComponents {
		return "", true, nil
	}
	relative := path.Join(parts[stripComponents:]...)
	if relative == "." || strings.HasPrefix(relative, "../") {
		return "", false, fmt.Errorf("unsafe archive path %q", name)
	}
	return relative, false, nil
}
