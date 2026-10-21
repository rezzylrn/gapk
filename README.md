# gapk v0.2.0

Go-style declarative language and compiler that transpiles `.gapk` source files into full **Flutter (`lib/main.dart`)** or **Kotlin Jetpack Compose (`MainActivity.kt`)** projects and compiles Android APKs.

## Features

- **Go-Like Syntax**: Clean blocks, `:=` reactive state inference (`string`, `int`, `float`, `bool`, `list`), and zero semicolon/comma clutter.
- **Inline UI Modifiers `[key: value]`**: Style any widget inline (`[padding: 16, bg: "#1E293B", radius: 12, size: 18, weight: "bold", color: "#10B981"]`) without deep Flutter wrapper trees.
- **Full Flutter Widget Parity**:
  - **Scaffold & Navigation**: `screen`, `appbar`, `drawer`, `fab`, `bottom_nav`, `navigate`, `navigate_back`
  - **Layout & Containers**: `column`, `row`, `stack`, `container`, `card`, `listview`, `list_for`, `gridview`, `scroll`, `center`, `expanded`, `padding`, `sizedbox`, `spacer`, `divider`
  - **Display & Media**: `text`, `icon`, `image`, `avatar`, `badge`, `listtile`, `progress`
  - **Forms & Inputs**: `input`, `button`, `icon_button`, `switch_toggle`, `checkbox`, `slider`, `dropdown`
  - **Reactive Actions**: State mutation (`=`), `list_add`, `list_remove`, `snackbar`, `toast`, `dialog`, `http_get`, `if / else`

## CLI Usage

```text
Usage:
  gapk <command> [flags] [arguments]

Commands:
  init <dir>                        Create a new gapk project
  transpile [--target=flutter] <f>  Print generated Dart/Kotlin code to stdout
  build [--target=flutter] <f> [d]  Scaffold full project and build APK
  version                           Print compiler version
  help                              Show this help message

Flags:
  --target=flutter                  Output Flutter project / lib/main.dart (default)
  --target=compose                  Output Android Kotlin Jetpack Compose project
```

## Flutter Target Dependencies

Generated Flutter projects require these pub dependencies (auto-added to `pubspec.yaml`):

| Package | Version | Used by |
|---------|---------|---------|
| `http` | `^1.2.0` | `http_get`, `http_post` actions |
| `shared_preferences` | `^2.2.0` | `storage_set`, `storage_get` actions |

### Android: INTERNET Permission (REQUIRED)

Apps using `http_get`/`http_post` **will crash on startup** without this. Add to `android/app/src/main/AndroidManifest.xml`:

```xml
<manifest xmlns:android="http://schemas.android.com/apk/res/android">
    <uses-permission android:name="android.permission.INTERNET"/>
    <application ...>
```

### Async HTTP (v0.3.0+)

`http_get`/`http_post` now use `await` with try-catch. Button `onPressed` handlers are generated as `async`. Failed requests set the target variable to `""` instead of crashing. `json_get` is null-safe and won't crash on invalid JSON.
