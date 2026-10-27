# Changelog

All notable changes to gapk are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [0.3.0] — 2026-10-27

### Fixed

- Lexer no longer silently drops unknown characters: `>=`, `<=`, `&&`,
  `||`, `!`, parentheses, and `%` are now real tokens. Previously `!active`
  compiled to `active` and `>=` produced invalid Dart.
- String literals in generated Dart/Kotlin are escaped (quotes,
  backslashes, `$`, control characters). A quote inside a `.gapk` string
  used to break out of the generated string literal.
- Float modifiers (`[size: 16.5]`) no longer emit invalid `16.5.0`.
- `await` in every action host: `fab`, `icon_button`, and `listtile`
  handlers are `async` like `button` already was.
- `storage_get`/`storage_set` are awaited instead of fire-and-forget
  `.then()` calls, so a token read before an HTTP call is no longer stale.
- `http_post` now checks the 2xx status code like `http_get` does.
- Diamond imports (two files importing the same file) no longer report
  a false circular import. Real cycles still error.
- Fragment files that start with `import` lines now parse their screens
  and states instead of returning empty.
- `scroll`, `center`, `expanded`, and `padding` keep all of their
  children instead of only the first.
- `list_remove` is bounds-checked; `list_add`/`list_remove` accept full
  expressions.
- Colors accept `#RGB`, `#RRGGBB`, and `#AARRGGBB`, plus canonical
  Material names (`gray` → `grey`, `blue_grey` → `blueGrey`).
- Generated header maps are deterministic (keys sorted).
- `gapk build` runs a real `flutter create` scaffold (with `android/`
  and Gradle) instead of emitting only `pubspec.yaml` + `main.dart`;
  without Flutter installed it says so plainly instead of implying an
  APK was built.
- `gapk init` refuses to overwrite an existing `main.gapk` without
  `--force`; unknown `--target` values are errors instead of silently
  falling back to Flutter.

### Added

- Semantic validation on every parse, with `.gapk` line numbers:
  duplicate screens/states, unknown `navigate` targets and state
  references, wrong widget/state kinds, dropdown initial values outside
  their options, invalid colors/icons/numbers, unknown modifiers, and
  misplaced structural widgets.
- `gapk check` command: parse + validate without emitting code.
- Full expression parsing: parentheses, unary `-`/`!`, and all binary
  operators; negative state values (`x := -5`).
- Compose target: multi-screen navigation, drawer, FAB, bottom nav,
  more widgets; unsupported features (HTTP, storage, images) fail
  loudly instead of being dropped.
- Conditional Flutter dependencies: `http` and `shared_preferences`
  are only imported and added to `pubspec.yaml` when actually used.
- `gridview [cross: N]` accepted as an alias for `[cols: N]`.
- Tests: 30+ covering the lexer, parser, validation, codegen, and
  compiler, including negative cases for every fixed bug.
- CI (GitHub Actions): `gofmt`, `go vet`, `go test`.

### Changed

- `GenerateFlutterDart` / `GenerateKotlinCompose` now return
  `(string, error)` and run validation; `Tokenize` returns an error.
- `examples/smartmedika-test.gapk` now sends a real JSON body and a
  `Bearer <token>` expression instead of a raw identifier string and a
  literal `"Bearer token"` header.

## [0.2.0] — 2026-10-21

- Multi-file `import "path.gapk"` with circular-import detection.
- `http_get`/`http_post` with headers and Bearer auth, `json_get`,
  `storage_set`/`storage_get`.
- Lexer escaped quotes (`\"`, `\\`, `\n`, `\t`).
- Async HTTP with `await`; `json_get` null-safe.
- `shared_preferences` added to generated `pubspec.yaml`.
