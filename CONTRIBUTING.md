# Contributing to gapk

Keep it small and boring: the compiler is pure Go stdlib, no runtime
dependencies, and every behavior change should come with a test.

## Workflow

1. Fork, branch off `main`, make focused commits.
2. `go test ./...` and `go vet ./...` must pass; `gofmt -l .` must be
   empty.
3. Add or update tests for the behavior you change — including the
   failure case, not just the happy path.
4. Update `CHANGELOG.md` under a new `## [Unreleased]` section and
   update `README.md` if user-visible behavior changed.

## Ground rules

- **Fail loudly.** Never silently drop a widget, action, or modifier
  the user wrote. If the compiler cannot express something in the
  target language, return an error that names the `.gapk` line.
- **Validate before generating.** New checks belong in
  `internal/validate`, so `check`, `transpile`, and `build` all get
  them for free.
- **No new dependencies** without a strong reason — the stdlib has
  covered everything so far.
- **Deterministic output.** Generated code must be byte-identical
  across runs (sort map iteration, no timestamps).
- **Keep the language small.** A new widget or action needs a reason to
  exist in `.gapk` rather than in hand-written Dart/Kotlin, and it
  needs to work (or fail loudly) on both targets.

## Repo layout

| Path | What lives there |
|---|---|
| `cmd/gapk` | CLI: flags, commands, exit codes |
| `internal/token`, `internal/lexer` | Tokens and scanning |
| `internal/ast` | AST types, identifier/color helpers |
| `internal/parser` | `.gapk` → AST |
| `internal/strlit` | String-literal escaping for generated code |
| `internal/validate` | Semantic checks over the merged AST |
| `internal/codegen` | AST → Dart (`flutter.go`) / Kotlin (`compose.go`) |
| `internal/compiler` | Multi-file loading, `flutter create` scaffolding, APK builds |
| `examples/` | `.gapk` programs that must transpile cleanly for both targets |
