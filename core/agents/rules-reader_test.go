package agents

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetBuiltinGenresDir_ResolvesDirectory(t *testing.T) {
	dir := GetBuiltinGenresDir()
	if strings.TrimSpace(dir) == "" {
		t.Fatalf("expected builtin genres dir, got empty string")
	}
	if _, err := os.Stat(filepath.Join(dir, "other.md")); err != nil {
		t.Fatalf("expected other.md in builtin dir %q: %v", dir, err)
	}
}

func TestListAvailableGenres_IncludesBuiltinProfiles(t *testing.T) {
	projectRoot := t.TempDir()
	genres, err := ListAvailableGenres(projectRoot)
	if err != nil {
		t.Fatalf("ListAvailableGenres failed: %v", err)
	}
	if len(genres) == 0 {
		t.Fatalf("expected at least one builtin genre")
	}

	foundOther := false
	for _, genre := range genres {
		if genre.ID == "other" {
			foundOther = true
			if genre.Source != "builtin" {
				t.Fatalf("expected other source=builtin, got %q", genre.Source)
			}
		}
	}
	if !foundOther {
		t.Fatalf("expected builtin genre \"other\" to be listed")
	}
}
