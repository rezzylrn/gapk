package compiler

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gapk-lang/gapk/internal/ast"
	"github.com/gapk-lang/gapk/internal/codegen"
	"github.com/gapk-lang/gapk/internal/lexer"
	"github.com/gapk-lang/gapk/internal/parser"
	"github.com/gapk-lang/gapk/internal/validate"
)

func ParseFile(path string) (*ast.AppNode, error) {
	app, err := parseFileRecursive(path, make(map[string]bool), make(map[string]bool))
	if err != nil {
		return nil, err
	}
	if err := validate.ValidateApp(app); err != nil {
		return nil, err
	}
	return app, nil
}

func parseFileRecursive(path string, visiting, done map[string]bool) (*ast.AppNode, error) {
	// Resolve absolute path to avoid duplicates.
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if visiting[absPath] {
		return nil, fmt.Errorf("circular import detected: %s", path)
	}
	if done[absPath] {
		// Diamond import: the file was already parsed through another
		// branch and its states/screens are merged by that branch.
		// Returning an empty app here is correct; treating it as a
		// cycle was the old (wrong) behavior.
		return &ast.AppNode{PackageName: "com.gapk.app", Props: make(ast.Modifiers)}, nil
	}
	visiting[absPath] = true
	defer func() {
		delete(visiting, absPath)
		done[absPath] = true
	}()

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	tokens, err := lexer.Tokenize(string(data))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	p := parser.New(tokens)

	// ParseApp handles files with an app block. Files without one
	// (fragments: only imports/screens/states) must fall back to
	// ParseFragment — including files that START with imports, where
	// ParseApp returns early with an empty app instead of an error.
	app, err := p.ParseApp()
	if err != nil || app.AppName == "" {
		p2 := parser.New(tokens)
		frag, ferr := p2.ParseFragment()
		if ferr != nil {
			return nil, fmt.Errorf("%s: %w", path, ferr)
		}
		app, err = frag, nil
	}

	// Recursively parse imports and merge
	baseDir := filepath.Dir(absPath)
	for _, imp := range app.Imports {
		impPath := imp
		if !filepath.IsAbs(impPath) {
			impPath = filepath.Join(baseDir, impPath)
		}
		imported, err := parseFileRecursive(impPath, visiting, done)
		if err != nil {
			return nil, err
		}
		// Merge states (avoid duplicates by name)
		existingStates := make(map[string]bool)
		for _, s := range app.States {
			existingStates[s.Name] = true
		}
		for _, s := range imported.States {
			if !existingStates[s.Name] {
				app.States = append(app.States, s)
				existingStates[s.Name] = true
			}
		}
		// Merge screens (avoid duplicates by title)
		existingScreens := make(map[string]bool)
		for _, s := range app.Screens {
			existingScreens[s.Title] = true
		}
		for _, s := range imported.Screens {
			if !existingScreens[s.Title] {
				app.Screens = append(app.Screens, s)
				existingScreens[s.Title] = true
			}
		}
	}
	// Clear imports after merging (they're resolved)
	app.Imports = nil

	return app, nil
}

// BuildOptions controls `gapk build` behavior.
type BuildOptions struct {
	// Release builds a release APK instead of a debug APK.
	Release bool
	// Force allows `flutter create` into a non-empty output directory.
	Force bool
}

func BuildProject(inputFile, outDir, target string) error {
	return BuildProjectOptions(inputFile, outDir, target, BuildOptions{})
}

func BuildProjectOptions(inputFile, outDir, target string, opts BuildOptions) error {
	app, err := ParseFile(inputFile)
	if err != nil {
		return err
	}

	if target == "compose" {
		if err := writeAndroidComposeScaffold(app, outDir); err != nil {
			return err
		}
		fmt.Printf("[gapk] generated Kotlin Compose project in %s\n", outDir)
		fmt.Println("[gapk] compose target is a preview: create an Android Studio \"Empty Compose Activity\" project, replace MainActivity.kt with the generated file, and build from Android Studio or Gradle.")
		return nil
	}

	flutterBin, flutterErr := exec.LookPath("flutter")
	if flutterErr != nil {
		// No Flutter toolchain: emit the files we can and say plainly
		// what is missing instead of pretending an APK was built.
		if err := writeFlutterScaffold(app, outDir); err != nil {
			return err
		}
		fmt.Printf("[gapk] generated Flutter project files in %s\n", outDir)
		fmt.Println("[gapk] 'flutter' CLI not found in PATH, so no APK was built.")
		fmt.Println("[gapk] To build: install Flutter, then run:")
		fmt.Println("[gapk]   flutter create --platforms=android <newdir>")
		fmt.Printf("[gapk]   cp %s/lib/main.dart <newdir>/lib/main.dart && cp %s/pubspec.yaml <newdir>/pubspec.yaml\n", outDir, outDir)
		fmt.Println("[gapk]   cd <newdir> && flutter build apk")
		return nil
	}

	// With a Flutter toolchain, scaffold a real project with
	// `flutter create` (android/ + gradle wrapper included), then
	// replace the generated main.dart/pubspec with gapk output. The
	// old scaffold wrote only pubspec.yaml + main.dart, so
	// `flutter build apk` could never succeed in that directory.
	if entries, err := os.ReadDir(outDir); err == nil && len(entries) > 0 && !opts.Force {
		return fmt.Errorf("output directory %s is not empty (use --force to overwrite, or pick another directory)", outDir)
	}
	pkgSlug := flutterProjectName(app.AppName)
	createArgs := []string{"create", "--project-name", pkgSlug, "--org", app.PackageName, "--platforms", "android", outDir}
	if opts.Force {
		createArgs = append(createArgs, "--overwrite")
	}
	fmt.Printf("[gapk] scaffolding Flutter project with %s...\n", flutterBin)
	createCmd := exec.Command(flutterBin, createArgs...)
	createCmd.Stdout = os.Stdout
	createCmd.Stderr = os.Stderr
	if err := createCmd.Run(); err != nil {
		return fmt.Errorf("flutter create failed: %w", err)
	}
	if err := writeFlutterScaffold(app, outDir); err != nil {
		return err
	}
	needsHTTP, _ := codegen.FlutterDeps(app)
	if needsHTTP {
		if err := ensureInternetPermission(filepath.Join(outDir, "android", "app", "src", "main", "AndroidManifest.xml")); err != nil {
			return err
		}
	}

	mode := "--debug"
	if opts.Release {
		mode = "--release"
	}
	fmt.Printf("[gapk] compiling APK with %s build apk %s...\n", flutterBin, mode)
	cmd := exec.Command(flutterBin, "build", "apk", mode)
	cmd.Dir = outDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// flutterProjectName derives a valid Dart package name from the app name.
func flutterProjectName(appName string) string {
	return strings.ToLower(codegen.SanitizeIdent(appName))
}

// ensureInternetPermission adds the INTERNET permission to an Android
// manifest if it is not already there. Apps using http_get/http_post
// fail network requests without it.
func ensureInternetPermission(manifestPath string) error {
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return err
	}
	if strings.Contains(string(data), "android.permission.INTERNET") {
		return nil
	}
	const marker = "<application"
	idx := strings.Index(string(data), marker)
	if idx < 0 {
		return fmt.Errorf("AndroidManifest.xml has no <application> element, cannot add INTERNET permission")
	}
	patched := string(data)[:idx] + "    <uses-permission android:name=\"android.permission.INTERNET\"/>\n    " + string(data)[idx:]
	return os.WriteFile(manifestPath, []byte(patched), 0644)
}

func writeFlutterScaffold(app *ast.AppNode, outDir string) error {
	libDir := filepath.Join(outDir, "lib")
	if err := os.MkdirAll(libDir, 0755); err != nil {
		return err
	}

	pkgSlug := flutterProjectName(app.AppName)
	needsHTTP, needsStorage := codegen.FlutterDeps(app)
	var deps strings.Builder
	if needsHTTP {
		deps.WriteString("  http: ^1.2.0\n")
	}
	if needsStorage {
		deps.WriteString("  shared_preferences: ^2.2.0\n")
	}
	pubspec := fmt.Sprintf(`name: %s
description: Generated by gapk compiler.
publish_to: 'none'
version: 1.0.0+1

environment:
  sdk: '>=3.2.0 <4.0.0'

dependencies:
  flutter:
    sdk: flutter
%s
flutter:
  uses-material-design: true
`, pkgSlug, deps.String())

	if err := os.WriteFile(filepath.Join(outDir, "pubspec.yaml"), []byte(pubspec), 0644); err != nil {
		return err
	}

	dartCode, err := codegen.GenerateFlutterDart(app)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(libDir, "main.dart"), []byte(dartCode), 0644)
}

func writeAndroidComposeScaffold(app *ast.AppNode, outDir string) error {
	pkgPath := strings.ReplaceAll(app.PackageName, ".", "/")
	javaDir := filepath.Join(outDir, "app", "src", "main", "java", pkgPath)
	if err := os.MkdirAll(javaDir, 0755); err != nil {
		return err
	}
	kotlinCode, err := codegen.GenerateKotlinCompose(app)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(javaDir, "MainActivity.kt"), []byte(kotlinCode), 0644)
}
