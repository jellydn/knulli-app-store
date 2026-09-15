package input

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestRuntimeCodeDoesNotBypassSemanticControllerActions(t *testing.T) {
	root := filepath.Clean("../..")
	prohibited := []*regexp.Regexp{
		regexp.MustCompile("SDL_CONTROLLER_BUTTON_" + "[AB]"),
		regexp.MustCompile(`(?i)\b[AB]\s+(confirm|cancel|back)\b`),
		regexp.MustCompile(`HandleButton\(\s*[0-9]+\s*\)`),
		regexp.MustCompile(`event_key\(event\) == C\.SDLK_y`),
	}
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if path == filepath.Join(root, "internal", "input") || path == filepath.Join(root, ".git") || path == filepath.Join(root, "build") || path == filepath.Join(root, "dist") {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, pattern := range prohibited {
			if pattern.Match(data) {
				t.Errorf("runtime file %s bypasses semantic controller actions with %q", path, pattern)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
