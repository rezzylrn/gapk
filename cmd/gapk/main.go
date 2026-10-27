package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gapk-lang/gapk/internal/codegen"
	"github.com/gapk-lang/gapk/internal/compiler"
)

const version = "0.3.0"

const helpText = `gapk - Go-style Flutter & Android DSL Compiler

Usage:
  gapk <command> [flags] [arguments]

Commands:
  init [--force] <dir>              Initialize a new .gapk project
  check <file.gapk>                 Parse and validate a .gapk file without generating code
  transpile [--target=flutter] <f>  Print generated Flutter Dart or Kotlin Compose code
  build [--target=flutter] [--release] [--force] <f> [d]
                                    Generate a project scaffold and compile an APK
  version                           Print compiler version
  help                              Show this help message

Flags:
  --target=flutter                  Generate Flutter (lib/main.dart) [default]
  --target=compose                  Generate Android Kotlin Jetpack Compose (preview)
  --release                         Build a release APK instead of a debug APK
  --force                           Overwrite an existing main.gapk (init) or
                                    a non-empty output directory (build)
`

const starterTemplate = `package main

app "StarterApp" [theme: "dark", primary: "#10B981"] {
    username := "Developer"
    count := 0
    notifications := true

    screen "Dashboard" [padding: 16] {
        appbar "StarterApp" [center: true]

        column [spacing: 12] {
            card [padding: 16, radius: 12, elevation: 3] {
                text "Welcome, " + username [size: 20, weight: "bold", color: "#10B981"]
                text "Total actions: " + count [size: 14]
            }

            input username "Update Username" [icon: "person"]
            switch_toggle notifications "Enable Push Alerts"

            row [spacing: 10] {
                button "Increment" [bg: "#10B981"] {
                    count = count + 1
                    snackbar "Count updated to " + count
                }
                button "Details" [variant: "outlined"] {
                    dialog "Summary" "User " + username + " triggered " + count + " actions."
                }
            }
        }

        fab "add" {
            count = count + 1
            toast "Quick add: " + count
        }
    }
}
`

func main() {
	if len(os.Args) < 2 {
		fmt.Print(helpText)
		os.Exit(0)
	}

	cmd := os.Args[1]
	target := "flutter"
	release := false
	force := false
	var args []string
	rest := os.Args[2:]
	for i := 0; i < len(rest); i++ {
		a := rest[i]
		switch {
		case strings.HasPrefix(a, "--target="):
			target = strings.TrimPrefix(a, "--target=")
		case a == "--target":
			if i+1 >= len(rest) {
				fatal(fmt.Errorf("--target requires a value: flutter or compose"))
			}
			i++
			target = rest[i]
		case a == "--release":
			release = true
		case a == "--force":
			force = true
		case strings.HasPrefix(a, "-"):
			fatal(fmt.Errorf("unknown flag %q", a))
		default:
			args = append(args, a)
		}
	}
	if target != "flutter" && target != "compose" {
		fatal(fmt.Errorf("unknown target %q (valid targets: flutter, compose)", target))
	}

	switch cmd {
	case "help", "-h", "--help":
		fmt.Print(helpText)

	case "version", "-v", "--version":
		fmt.Printf("gapk v%s\n", version)

	case "init":
		dir := "."
		if len(args) >= 1 {
			dir = args[0]
		}
		if err := os.MkdirAll(dir, 0755); err != nil {
			fatal(err)
		}
		mainFile := filepath.Join(dir, "main.gapk")
		if _, err := os.Stat(mainFile); err == nil && !force {
			fatal(fmt.Errorf("%s already exists (use --force to overwrite)", mainFile))
		}
		if err := os.WriteFile(mainFile, []byte(starterTemplate), 0644); err != nil {
			fatal(err)
		}
		fmt.Printf("[gapk] initialized %s\n", mainFile)

	case "check":
		if len(args) < 1 {
			fatal(fmt.Errorf("missing input file: gapk check <file.gapk>"))
		}
		app, err := compiler.ParseFile(args[0])
		if err != nil {
			fatal(err)
		}
		fmt.Printf("[gapk] %s OK: %d screen(s), %d state(s)\n", args[0], len(app.Screens), len(app.States))

	case "transpile":
		if len(args) < 1 {
			fatal(fmt.Errorf("missing input file: gapk transpile <file.gapk>"))
		}
		app, err := compiler.ParseFile(args[0])
		if err != nil {
			fatal(err)
		}
		var out string
		if target == "compose" {
			out, err = codegen.GenerateKotlinCompose(app)
		} else {
			out, err = codegen.GenerateFlutterDart(app)
		}
		if err != nil {
			fatal(err)
		}
		fmt.Print(out)

	case "build":
		if len(args) < 1 {
			fatal(fmt.Errorf("missing input file: gapk build <file.gapk> [out_dir]"))
		}
		outDir := "./build"
		if len(args) >= 2 {
			outDir = args[1]
		}
		if err := compiler.BuildProjectOptions(args[0], outDir, target, compiler.BuildOptions{Release: release, Force: force}); err != nil {
			fatal(err)
		}

	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n%s", cmd, helpText)
		os.Exit(1)
	}
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}
