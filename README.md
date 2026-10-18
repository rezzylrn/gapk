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
