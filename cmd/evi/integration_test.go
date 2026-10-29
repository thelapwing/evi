package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func runEvi(t *testing.T, args ...string) (string, error) {
	t.Helper()
	out := new(bytes.Buffer)
	rootCmd.SetOut(out)
	rootCmd.SetErr(out)
	rootCmd.SetArgs(args)
	defer func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
		workspaceDir = "."
	}()
	err := rootCmd.Execute()
	return out.String(), err
}

func writeCLIFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestCLIIntegration(t *testing.T) {
	scenarios := []struct {
		name    string
		sources map[string]string
		extra   map[string]string
		mutate  func(t *testing.T, dir string)
		wantErr string
		notWant string
	}{
		{
			name:    "source change detected",
			sources: map[string]string{"main.go": "package main\n", "util.go": "package main\n"},
			mutate: func(t *testing.T, dir string) {
				writeCLIFile(t, filepath.Join(dir, "util.go"), "package main // changed\n")
			},
			wantErr: "util.go [evidence: ev-002]: file SHA-256 changed",
			notWant: "ev-001",
		},
		{
			name:    "unregistered file change ignored",
			sources: map[string]string{"main.go": "package main\n"},
			extra:   map[string]string{"notes.md": "draft\n"},
			mutate: func(t *testing.T, dir string) {
				writeCLIFile(t, filepath.Join(dir, "notes.md"), "edited\n")
			},
		},
		{
			name:    "deleted source detected",
			sources: map[string]string{"main.go": "package main\n"},
			mutate: func(t *testing.T, dir string) {
				if err := os.Remove(filepath.Join(dir, "main.go")); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: "ev-001: stat main.go",
		},
		{
			name:    "lock tampering detected",
			sources: map[string]string{"main.go": "package main\n"},
			mutate: func(t *testing.T, dir string) {
				path := filepath.Join(dir, "evidence.lock.yaml")
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				i := bytes.LastIndex(data, []byte("sha256: "))
				if i < 0 {
					t.Fatal("lock must contain a file sha256 entry")
				}
				i += len("sha256: ")
				if data[i] == 'a' {
					data[i] = 'b'
				} else {
					data[i] = 'a'
				}
				if err := os.WriteFile(path, data, 0644); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: "differ from lock",
		},
	}

	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			dir := t.TempDir()
			names := make([]string, 0, len(sc.sources))
			for name := range sc.sources {
				names = append(names, name)
			}
			sort.Strings(names)
			var registry strings.Builder
			registry.WriteString("evidence:\n")
			for i, name := range names {
				writeCLIFile(t, filepath.Join(dir, name), sc.sources[name])
				fmt.Fprintf(&registry, "  - id: ev-%03d\n    product: p\n    requirement: r\n    path: %s\n", i+1, name)
			}
			for name, content := range sc.extra {
				writeCLIFile(t, filepath.Join(dir, name), content)
			}
			writeCLIFile(t, filepath.Join(dir, "evidence.yaml"), registry.String())

			for _, command := range []string{"validate", "lock", "verify"} {
				if out, err := runEvi(t, "--workspace", dir, command); err != nil {
					t.Fatalf("%s failed: %v\n%s", command, err, out)
				}
			}
			if _, err := os.Stat(filepath.Join(dir, "evidence.lock.yaml")); err != nil {
				t.Fatalf("lock must exist: %v", err)
			}

			sc.mutate(t, dir)

			out, err := runEvi(t, "--workspace", dir, "verify")
			if sc.wantErr == "" {
				if err != nil {
					t.Fatalf("verify must pass: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), sc.wantErr) {
				t.Fatalf("expected error containing %q, got %v\n%s", sc.wantErr, err, out)
			}
			if sc.notWant != "" && strings.Contains(err.Error(), sc.notWant) {
				t.Fatalf("error must not contain %q: %v", sc.notWant, err)
			}
		})
	}
}
