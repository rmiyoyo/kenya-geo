package api

import _ "embed"

// Docs is the API reference in Markdown, the same file as api/README.md.
//
//go:embed README.md
var Docs string
