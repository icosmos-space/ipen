package agents

import (
	"os"
	"path/filepath"
	"testing"
)

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

func TestReadGenreProfile_UsesEmbeddedFS(t *testing.T) {
	projectRoot := t.TempDir()

	// Test reading builtin profile from embedded filesystem
	profile, err := ReadGenreProfile(projectRoot, "other")
	if err != nil {
		t.Fatalf("ReadGenreProfile failed for \"other\": %v", err)
	}
	if profile.Profile.Name == "" {
		t.Fatalf("expected non-empty name from embedded profile")
	}
}

func TestReadGenreProfile_ProjectOverride(t *testing.T) {
	projectRoot := t.TempDir()

	// Create project-level genre override
	genresDir := filepath.Join(projectRoot, "genres")
	if err := os.MkdirAll(genresDir, 0755); err != nil {
		t.Fatalf("failed to create genres dir: %v", err)
	}

	customContent := `---
name: Custom Genre
language: en
description: A custom genre
---
Custom rules here.`

	if err := os.WriteFile(filepath.Join(genresDir, "custom.md"), []byte(customContent), 0644); err != nil {
		t.Fatalf("failed to write custom genre: %v", err)
	}

	// Test project override
	profile, err := ReadGenreProfile(projectRoot, "custom")
	if err != nil {
		t.Fatalf("ReadGenreProfile failed for \"custom\": %v", err)
	}
	if profile.Profile.Name != "Custom Genre" {
		t.Fatalf("expected \"Custom Genre\", got %q", profile.Profile.Name)
	}
}

func TestListAvailableGenres_SortedOrder(t *testing.T) {
	projectRoot := t.TempDir()
	genres, err := ListAvailableGenres(projectRoot)
	if err != nil {
		t.Fatalf("ListAvailableGenres failed: %v", err)
	}

	// Verify sorted order
	for i := 1; i < len(genres); i++ {
		if genres[i].ID < genres[i-1].ID {
			t.Fatalf("genres not sorted: %q should come before %q", genres[i-1].ID, genres[i].ID)
		}
	}
}

func TestListAvailableGenres_ProjectOverridesBuiltin(t *testing.T) {
	projectRoot := t.TempDir()

	// Create project-level genre that overrides builtin
	genresDir := filepath.Join(projectRoot, "genres")
	if err := os.MkdirAll(genresDir, 0755); err != nil {
		t.Fatalf("failed to create genres dir: %v", err)
	}

	customContent := `---
name: Overridden Other
language: en
description: Overridden other genre
---
Overridden rules here.`

	if err := os.WriteFile(filepath.Join(genresDir, "other.md"), []byte(customContent), 0644); err != nil {
		t.Fatalf("failed to write custom other: %v", err)
	}

	genres, err := ListAvailableGenres(projectRoot)
	if err != nil {
		t.Fatalf("ListAvailableGenres failed: %v", err)
	}

	// Find "other" - should be project source
	found := false
	for _, genre := range genres {
		if genre.ID == "other" {
			found = true
			if genre.Source != "project" {
				t.Fatalf("expected other source=project when overridden, got %q", genre.Source)
			}
			if genre.Name != "Overridden Other" {
				t.Fatalf("expected overridden name \"Overridden Other\", got %q", genre.Name)
			}
		}
	}
	if !found {
		t.Fatalf("expected to find overridden \"other\" genre")
	}
}
