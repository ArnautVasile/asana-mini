package write

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
)

var (
	// allow letters, digits, dot, underscore, space, dash
	invalidDirChars = regexp.MustCompile(`[^a-zA-Z0-9._ -]+`)
	multiSpace      = regexp.MustCompile(`\s+`)
)

func SafeDirName(name, fallback string) string {
	trim := invalidDirChars.ReplaceAllString(name, "_")
	trim = multiSpace.ReplaceAllString(trim, " ")
	if trim == "" {
		return fallback
	}
	return trim
}

func JSON(dir, name string, v any) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	f, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
