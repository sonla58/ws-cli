package detect

import (
	"os"
	"path/filepath"
	"strings"
)

// Type is a short key identifying a project kind. It also determines icon rendering.
type Type string

const (
	TypeUnknown  Type = ""
	TypeNode     Type = "node"
	TypeNext     Type = "nextjs"
	TypeElectron Type = "electron"
	TypeTauri    Type = "tauri"
	TypeRust     Type = "rust"
	TypeGo       Type = "go"
	TypePython   Type = "python"
	TypeIOS      Type = "ios"
	TypeAndroid  Type = "android"
	TypeRuby     Type = "ruby"
	TypeJava     Type = "java"
	TypeDocker   Type = "docker"
	TypeDir      Type = "dir"
)

// Icon returns a Nerd Font glyph for the type. Falls back to a letter when NO_NERD_FONT=1.
func Icon(t Type) string {
	if os.Getenv("NO_NERD_FONT") == "1" {
		switch t {
		case TypeUnknown, TypeDir:
			return "·"
		default:
			return strings.ToUpper(string(t)[:1])
		}
	}
	switch t {
	case TypeNode:
		return "\uE718" // nf-mdi-nodejs
	case TypeNext:
		return "\uE781" // nf-dev-react (close enough)
	case TypeElectron:
		return "\uE62E"
	case TypeTauri:
		return "\U000F0816"
	case TypeRust:
		return "\uE7A8"
	case TypeGo:
		return "\uE627"
	case TypePython:
		return "\uE606"
	case TypeIOS:
		return "\uE711"
	case TypeAndroid:
		return "\uE70E"
	case TypeRuby:
		return "\uE739"
	case TypeJava:
		return "\uE738"
	case TypeDocker:
		return "\uE7B0"
	case TypeDir:
		return "\uF07B"
	default:
		return "\uF07B"
	}
}

// DetectType inspects signature files in dir and returns the best-guess type.
// Order matters: more specific signatures win over generic ones.
func DetectType(dir string) Type {
	has := func(name string) bool {
		_, err := os.Stat(filepath.Join(dir, name))
		return err == nil
	}
	hasGlob := func(pattern string) bool {
		m, _ := filepath.Glob(filepath.Join(dir, pattern))
		return len(m) > 0
	}

	// More specific JS/TS signatures first.
	if has("tauri.conf.json") || has("src-tauri") {
		return TypeTauri
	}
	if has("next.config.js") || has("next.config.mjs") || has("next.config.ts") {
		return TypeNext
	}
	if hasGlob("electron-builder.*") || has("electron.vite.config.ts") {
		return TypeElectron
	}
	// Native mobile.
	if has("Podfile") || hasGlob("*.xcodeproj") || hasGlob("*.xcworkspace") {
		return TypeIOS
	}
	if has("AndroidManifest.xml") || has("build.gradle") || has("build.gradle.kts") || has("settings.gradle") {
		return TypeAndroid
	}
	// Generic languages.
	if has("package.json") {
		return TypeNode
	}
	if has("Cargo.toml") {
		return TypeRust
	}
	if has("go.mod") {
		return TypeGo
	}
	if has("pyproject.toml") || has("requirements.txt") || has("setup.py") || has("Pipfile") {
		return TypePython
	}
	if has("Gemfile") {
		return TypeRuby
	}
	if has("pom.xml") {
		return TypeJava
	}
	if has("Dockerfile") || has("docker-compose.yml") || has("compose.yml") {
		return TypeDocker
	}
	if fi, err := os.Stat(dir); err == nil && fi.IsDir() {
		return TypeDir
	}
	return TypeUnknown
}
