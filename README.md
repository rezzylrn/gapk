# gapk

A small declarative language for building Android apps. Write UI in a
Go-flavored syntax, get a real Flutter project (or a Jetpack Compose
preview) out.

```gapk
app "Counter" [theme: "light", primary: "#0E6B5C"] {
    count := 0

    screen "Home" {
        appbar "Counter"
        column [padding: 16, spacing: 12] {
            text "Count: " + count [size: 24, weight: "bold"]
            button "Tap me" {
                count = count + 1
            }
        }
    }
}
```

## Install & quickstart

Requires Go 1.22+ to build the compiler. To build an APK you also need
the Flutter SDK on your `PATH`.

```sh
git clone https://github.com/rezzylrn/gapk.git
cd gapk
go build -o gapk ./cmd/gapk

gapk init myapp          # writes myapp/main.gapk
gapk check myapp/main.gapk
gapk transpile myapp/main.gapk          # Dart to stdout
gapk build myapp/main.gapk ./myapp-out  # scaffold + flutter build apk
```

## Commands

| Command | What it does |
|---|---|
| `gapk init <dir> [--force]` | Scaffold a starter `main.gapk` (refuses to overwrite unless `--force`) |
| `gapk check <file.gapk>` | Parse + validate, no code emitted |
| `gapk transpile [--target flutter\|compose] <file.gapk>` | Print generated code to stdout |
| `gapk build [--target flutter] [--release] [--force] <file.gapk> [outDir]` | Scaffold a project; with Flutter installed, runs `flutter build apk` (`--debug` by default) |
| `gapk version`, `gapk help` | Version / usage |

## Language

**State.** `name := value` declares reactive state. The type is inferred:
string, int, float, bool, or list (`["a", "b"]`). Reassign with `=`.

**Expressions.** Strings concatenate with `+` and interpolate state
(`"Count: " + count`). Arithmetic and comparisons work as expected:
`(count + 1) * 2`, `count >= 5 && !active`, `total % 2 == 0`.

**Widgets (33 kinds).** `screen`, `appbar`, `drawer`, `fab`,
`bottom_nav`, `column`, `row`, `stack`, `container`, `card`, `listview`,
`gridview`, `list_for`, `scroll`, `center`, `expanded`, `padding`,
`sizedbox`, `spacer`, `divider`, `text`, `icon`, `image`, `avatar`,
`badge`, `listtile`, `progress`, `input`, `button`, `icon_button`,
`switch_toggle`, `checkbox`, `slider`, `dropdown`.

**Actions (14 kinds).** Assignment, `list_add` / `list_remove`,
`snackbar` / `toast` / `dialog`, `navigate` / `navigate_back`,
`http_get` / `http_post`, `json_get`, `storage_set` / `storage_get`,
`if` / `else`.

**Modifiers.** Inline styling: `[padding: 16, bg: "#1E293B", radius: 12,
size: 18, weight: "bold", color: "#10B981"]`. Colors accept names
(`"blue_grey"`, `"red"`), `#RGB`, `#RRGGBB`, or `#AARRGGBB`.
`gridview` takes `[cols: 2]` (alias `[cross: 2]`).

**Multi-file.** Split an app across files with imports, like `#include`:

```gapk
import "state/auth.gapk"
import "screens/login.gapk"

app "My App" {
    screen "Home" { text "Hello" }
}
```

Circular imports are rejected; importing the same file from two places
(diamond imports) works. See `examples/smartmedika-test.gapk` for a
two-screen app using HTTP and storage.

**Validation.** Every parse runs a semantic check and fails fast with a
`.gapk` line number: unknown screens in `navigate`, widgets bound to the
wrong state kind, a dropdown whose initial value is not among its
options, invalid colors/icons, duplicate screens/states, unknown state
references, and structural widgets (`fab`, `drawer`, `appbar`,
`bottom_nav`) nested inside other widgets. Nothing is silently dropped.

## Targets

**Flutter (default, full support).** Generates `lib/main.dart` plus a
`pubspec.yaml` that only includes the dependencies the app actually
uses (`http` for `http_get`/`http_post`, `shared_preferences` for
`storage_*`). All action hosts are `async`; HTTP checks status codes and
storage calls are awaited. `gapk build` runs `flutter create` to get a
real Android scaffold, patches in the generated code, adds the INTERNET
permission when HTTP is used, and builds the APK. Without Flutter
installed it still writes the scaffold files and tells you the exact
commands to finish the build.

**Compose (preview).** Generates a single `MainActivity.kt` covering
most widgets, multi-screen navigation, drawer, FAB, and bottom nav.
HTTP, storage, and image loading are rejected with a clear error instead
of being silently dropped — this target is not yet at parity. It needs
`material-icons-extended` (noted in the generated file header).

## Examples

- `examples/ecommerce.gapk` — shop with drawer, bottom nav, cart state.
- `examples/smartmedika-test.gapk` — login + API fetch with Bearer
  token, JSON body, and persistent storage.

## Developing

```sh
go test ./...   # 35 tests: lexer, parser, validation, codegen, compiler
go vet ./...
```

The compiler has zero runtime dependencies — just the Go standard
library. See [CONTRIBUTING.md](CONTRIBUTING.md).
