package diagnostics

import (
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	Path          = "/userdata/system/logs/knulli-app-store.log"
	maximumBytes  = 512 << 10
	diagnosticDir = "/userdata/system/knulli-app-store/diagnostics"
)

type Log struct {
	root   string
	path   string
	writer *boundedWriter
	logger *log.Logger
}

func Open(root string) (*Log, error) {
	if root == "" {
		root = "/"
	}
	path := filepath.Join(root, filepath.FromSlash(strings.TrimPrefix(Path, "/")))
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}
	writer := &boundedWriter{path: path, limit: maximumBytes}
	return &Log{root: root, path: path, writer: writer, logger: log.New(writer, "", log.Ldate|log.Ltime|log.LUTC)}, nil
}

func (l *Log) Event(name string, fields ...string) {
	if l == nil {
		return
	}
	var line strings.Builder
	line.WriteString("event=")
	line.WriteString(safeField(name))
	for index := 0; index+1 < len(fields); index += 2 {
		line.WriteByte(' ')
		line.WriteString(safeField(fields[index]))
		line.WriteByte('=')
		line.WriteString(fmt.Sprintf("%q", Redact(fields[index+1])))
	}
	l.logger.Print(line.String())
}

func (l *Log) Export(lines []string) (string, error) {
	if l == nil {
		return "", fmt.Errorf("diagnostics logging is unavailable")
	}
	directory := filepath.Join(l.root, filepath.FromSlash(strings.TrimPrefix(diagnosticDir, "/")))
	if err := os.MkdirAll(directory, 0700); err != nil {
		return "", err
	}
	name := "knulli-app-store-diagnostics-" + time.Now().UTC().Format("20060102T150405Z") + ".txt"
	host := filepath.Join(directory, name)
	var output strings.Builder
	output.WriteString("Knulli App Store diagnostics\n")
	output.WriteString("Generated: " + time.Now().UTC().Format(time.RFC3339) + "\n")
	for _, line := range lines {
		output.WriteString(Redact(line))
		output.WriteByte('\n')
	}
	for _, logPath := range []string{l.path + ".1", l.path} {
		if data, err := os.ReadFile(logPath); err == nil {
			output.WriteString("\n--- " + filepath.Base(logPath) + " ---\n")
			output.WriteString(Redact(string(data)))
		}
	}
	if err := os.WriteFile(host, []byte(output.String()), 0600); err != nil {
		return "", err
	}
	return diagnosticDir + "/" + name, nil
}

var sensitiveValue = regexp.MustCompile(`(?i)(token|password|passwd|secret|api[_-]?key|authorization)(\s*[=:]\s*)([^\s&]+)`)
var webURL = regexp.MustCompile(`https?://[^\s"'<>]+`)

func Redact(value string) string {
	value = sensitiveValue.ReplaceAllString(value, `$1$2[REDACTED]`)
	return webURL.ReplaceAllStringFunc(value, func(candidate string) string {
		suffix := ""
		for strings.ContainsRune("),.;", rune(candidate[len(candidate)-1])) {
			suffix = candidate[len(candidate)-1:] + suffix
			candidate = candidate[:len(candidate)-1]
		}
		parsed, err := url.Parse(candidate)
		if err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != "" && (parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "") {
			parsed.User = nil
			parsed.RawQuery = ""
			parsed.Fragment = ""
			return parsed.String() + suffix
		}
		return candidate + suffix
	})
}

func safeField(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.Map(func(character rune) rune {
		if character >= 'a' && character <= 'z' || character >= '0' && character <= '9' || character == '_' || character == '-' {
			return character
		}
		return '_'
	}, value)
	if value == "" {
		return "unknown"
	}
	return value
}

type boundedWriter struct {
	mu    sync.Mutex
	path  string
	limit int64
}

func (w *boundedWriter) Write(data []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	originalLength := len(data)
	data = []byte(Redact(string(data)))
	if int64(len(data)) > w.limit {
		data = data[int64(len(data))-w.limit:]
	}
	if info, err := os.Stat(w.path); err == nil && info.Size()+int64(len(data)) > w.limit {
		if err := w.rotate(info.Size()); err != nil {
			return 0, err
		}
	}
	file, err := os.OpenFile(w.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return 0, err
	}
	written, writeErr := file.Write(data)
	closeErr := file.Close()
	if writeErr != nil {
		return written, writeErr
	}
	if closeErr != nil {
		return written, closeErr
	}
	return originalLength, nil
}

func (w *boundedWriter) rotate(size int64) error {
	file, err := os.Open(w.path)
	if err != nil {
		return err
	}
	if size > w.limit {
		if _, err := file.Seek(size-w.limit, 0); err != nil {
			file.Close()
			return err
		}
	}
	data, readErr := io.ReadAll(io.LimitReader(file, w.limit))
	closeErr := file.Close()
	if readErr != nil {
		return readErr
	}
	if closeErr != nil {
		return closeErr
	}
	if err := os.WriteFile(w.path+".1", data, 0600); err != nil {
		return err
	}
	return os.Remove(w.path)
}
