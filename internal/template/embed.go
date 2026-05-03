package template

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"io/fs"
)

//go:embed assets/*.vdf
var assetsFS embed.FS

// Template is a single Valve-format controller config file we manage.
type Template struct {
	// Filename is the basename Steam expects in controller_base/templates/.
	Filename string
	// Content is the embedded canonical bytes.
	Content []byte
	// Hash is the lowercase hex sha256 of Content.
	Hash string
}

// All returns every template embedded in the binary.
func All() ([]Template, error) {
	entries, err := assetsFS.ReadDir("assets")
	if err != nil {
		return nil, fmt.Errorf("read embedded assets dir: %w", err)
	}

	out := make([]Template, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		t, err := loadTemplate(entry)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, nil
}

func loadTemplate(entry fs.DirEntry) (Template, error) {
	path := "assets/" + entry.Name()
	content, err := assetsFS.ReadFile(path)
	if err != nil {
		return Template{}, fmt.Errorf("read embedded %s: %w", path, err)
	}
	sum := sha256.Sum256(content)
	return Template{
		Filename: entry.Name(),
		Content:  content,
		Hash:     hex.EncodeToString(sum[:]),
	}, nil
}

// HashBytes returns the lowercase hex sha256 of b.
func HashBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
