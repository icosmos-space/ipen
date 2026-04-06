package genres

import "embed"

// Files contains the built-in genre markdown assets embedded in the binary.
// The genres are stored under the "genres" subdirectory.
//
//go:embed genres/*.md
var Files embed.FS
