package data

import (
	"embed"
	"io/fs"
)

// Files contains the built-in genre markdown assets embedded in the binary.
// The genres are stored under the "genres" subdirectory.
//
//go:embed genres/*.md
var GenresFiles embed.FS

// Genres is a fs.FS rooted at the "genres" directory.
// Use directly: fs.ReadFile(Genres, "fiction.md")
var Genres, _ = fs.Sub(GenresFiles, "genres")

//go:embed prompts/*.md
var PromptsFiles embed.FS

// Prompts is a fs.FS rooted at the "prompts" directory.
// Use directly: fs.ReadFile(Prompts, "system.md")
var Prompts, _ = fs.Sub(PromptsFiles, "prompts")

//go:embed foundation/*.md
var FoundationFiles embed.FS

// Foundation is a fs.FS rooted at the "foundation" directory.
// Use directly: fs.ReadFile(Foundation, "base.md")
var Foundation, _ = fs.Sub(FoundationFiles, "foundation")
