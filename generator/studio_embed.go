//go:build studio

package generator

import "embed"

// studioPythonFiles holds the AI training / learning scripts that are not
// part of the customer (release) build. See docs/EDITIONS.md.
//go:embed studio/*
var studioPythonFiles embed.FS
