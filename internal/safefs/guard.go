package safefs

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type Guard struct {
	root    string
	allowed []string
}

func NewGuard(root string, allowed []string) (*Guard, error) {
	absolute, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	absolute, err = filepath.EvalSymlinks(absolute)
	if err != nil {
		return nil, fmt.Errorf("resolve filesystem root: %w", err)
	}
	if len(allowed) == 0 {
		return nil, fmt.Errorf("at least one allowed path is required")
	}
	cleaned := make([]string, 0, len(allowed))
	for _, item := range allowed {
		if !strings.HasPrefix(item, "/") || path.Clean(item) != item || item == "/" {
			return nil, fmt.Errorf("invalid allowed path %q", item)
		}
		cleaned = append(cleaned, item)
	}
	return &Guard{root: filepath.Clean(absolute), allowed: cleaned}, nil
}

func (g *Guard) Resolve(virtual string) (string, error) {
	if !strings.HasPrefix(virtual, "/") || path.Clean(virtual) != virtual || virtual == "/" {
		return "", fmt.Errorf("unsafe absolute path %q", virtual)
	}
	if !g.allows(virtual) {
		return "", fmt.Errorf("path %s is outside declared write paths", virtual)
	}
	host := filepath.Join(g.root, filepath.FromSlash(strings.TrimPrefix(virtual, "/")))
	if err := rejectSymlinkParents(g.root, host); err != nil {
		return "", err
	}
	return host, nil
}

func (g *Guard) Virtual(host string) (string, error) {
	relative, err := filepath.Rel(g.root, host)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("host path is outside root")
	}
	virtual := "/" + filepath.ToSlash(relative)
	if !g.allows(virtual) {
		return "", fmt.Errorf("path %s is outside declared write paths", virtual)
	}
	return virtual, nil
}

func (g *Guard) allows(virtual string) bool {
	for _, allowed := range g.allowed {
		if virtual == allowed || strings.HasPrefix(virtual, allowed+"/") {
			return true
		}
	}
	return false
}

func rejectSymlinkParents(root, target string) error {
	relative, err := filepath.Rel(root, target)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return fmt.Errorf("resolved path escapes filesystem root")
	}
	current := root
	parts := strings.Split(relative, string(filepath.Separator))
	for index, part := range parts {
		if part == "." || part == "" {
			continue
		}
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing symlink in write path at %s", strings.Join(parts[:index+1], "/"))
		}
	}
	return nil
}
