package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gapk-lang/gapk/internal/codegen"
	"github.com/gapk-lang/gapk/internal/compiler"
)

const version = "0.2.0"

const helpText = `gapk - Go-style Flutter & Android DSL Compiler

Usage:
  gapk <command> [flags] [arguments]

Commands:
  init <dir>                        Initialize a new .gapk project
  transpile [--target=flutter] <f>  Print generated Flutter Dart or Kotlin Compose code
  build [--target=flutter] <f> [d]  Generate project scaffold and compile APK
  version                           Print compiler version
  help                              Show this help message

Flags:
  --target=flutter                  Generate Flutter (lib/main.dart) [default]
  --target=compose                  Generate Android Kotlin Jetpack Compose
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
	var args []string
	for _, a := range os.Args[2:] {
		if strings.HasPrefix(a, "--target=") {
			target = strings.TrimPrefix(a, "--target=")
		} else {
			args = append(args, a)
		}
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
		if err := os.WriteFile(mainFile, []byte(starterTemplate), 0644); err != nil {
			fatal(err)
		}
		fmt.Printf("[gapk] initialized %s\n", mainFile)

	case "transpile":
		if len(args) < 1 {
			fatal(fmt.Errorf("missing input file: gapk transpile <file.gapk>"))
		}
		app, err := compiler.ParseFile(args[0])
		if err != nil {
			fatal(err)
		}
		if target == "compose" {
			fmt.Print(codegen.GenerateKotlinCompose(app))
		} else {
			fmt.Print(codegen.GenerateFlutterDart(app))
		}

	case "build":
		if len(args) < 1 {
			fatal(fmt.Errorf("missing input file: gapk build <file.gapk> [out_dir]"))
		}
		outDir := "./build"
		if len(args) >= 2 {
			outDir = args[1]
		}
		if err := compiler.BuildProject(args[0], outDir, target); err != nil {
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
