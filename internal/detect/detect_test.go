package detect

import (
	"os"
	"path/filepath"
	"testing"
)

func touch(t *testing.T, p string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
}

func TestDetectType(t *testing.T) {
	cases := []struct {
		name  string
		files []string
		want  Type
	}{
		{"node", []string{"package.json"}, TypeNode},
		{"next", []string{"package.json", "next.config.js"}, TypeNext},
		{"tauri", []string{"package.json", "tauri.conf.json"}, TypeTauri},
		{"rust", []string{"Cargo.toml"}, TypeRust},
		{"go", []string{"go.mod"}, TypeGo},
		{"python", []string{"pyproject.toml"}, TypePython},
		{"ios", []string{"Podfile"}, TypeIOS},
		{"android", []string{"build.gradle"}, TypeAndroid},
		{"plain dir", []string{}, TypeDir},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, f := range tc.files {
				touch(t, filepath.Join(dir, f))
			}
			if got := DetectType(dir); got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}
