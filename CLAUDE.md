# Rules for working in this repo

- Never credit Claude in commits or pull requests: no `Co-Authored-By` trailers, no session links, no "Generated with Claude Code" lines, and never "Claude" as the author. Commit as `rmiyoyo <120647236+rmiyoyo@users.noreply.github.com>`.
- Don't write comments in Go code. Put explanations in the README. The only exceptions are lines the toolchain needs: `//go:embed` directives and `// Output:` blocks in examples.
- `data/counties.json` and `data/wards.json` are generated. Change `internal/gendata` or `data/source/`, then run `go run ./internal/gendata`.
- Run `gofmt -l .`, `go vet ./...` and `go test ./...` before committing.
