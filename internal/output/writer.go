// Package output builds the output file path from run metadata and writes command results to disk,
// creating the directory tree automatically.
package output

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var nonAlnumRe = regexp.MustCompile(`[^a-z0-9]+`)

func SanitizeCommand(command string) string {
	slug := strings.ToLower(command)
	slug = nonAlnumRe.ReplaceAllString(slug, "_")
	slug = strings.Trim(slug, "_")
	if slug == "" {
		slug = "command"
	}
	return slug
}

// Path builds the output file path for a single command's result:
// <baseDir>/<runStamp>/<datacenter>/<room>/<rack>/<hostname>[/specialized]/<command>.txt
func Path(baseDir, runStamp, datacenter, room, rack, hostname, category, command string) string {
	parts := []string{baseDir, runStamp, datacenter, room, rack, hostname}
	if category == "specialized" {
		parts = append(parts, "specialized")
	}
	parts = append(parts, SanitizeCommand(command)+".txt")
	return filepath.Join(parts...)
}

func Write(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}
