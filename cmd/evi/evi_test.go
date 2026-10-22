package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEvidenceCommands(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "source.txt"), []byte("source\n"), 0644); err != nil {
		t.Fatal(err)
	}
	evidence := "evidence: [{id: ev, product: p, requirement: r, path: source.txt}]\n"
	if err := os.WriteFile(filepath.Join(root, "evidence.yaml"), []byte(evidence), 0644); err != nil {
		t.Fatal(err)
	}
	defer func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
		workspaceDir = "."
	}()
	for i, command := range []string{"validate", "lock", "verify"} {
		out := new(bytes.Buffer)
		rootCmd.SetOut(out)
		rootCmd.SetErr(out)
		args := []string{"--workspace", root, command}
		if i == 1 {
			args = []string{command, "-w", root}
		}
		rootCmd.SetArgs(args)
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("%s failed: %v", command, err)
		}
		if out.Len() == 0 {
			t.Fatalf("%s did not report success", command)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "source.txt"), []byte("changed\n"), 0644); err != nil {
		t.Fatal(err)
	}
	out := new(bytes.Buffer)
	rootCmd.SetOut(out)
	rootCmd.SetErr(out)
	rootCmd.SetArgs([]string{"--workspace", root, "verify"})
	if err := rootCmd.Execute(); err == nil || !strings.Contains(err.Error(), "source.txt [evidence: ev]: whole-file SHA-256 changed") {
		t.Fatalf("expected verification error, got %v", err)
	}
	if out.Len() != 0 {
		t.Fatalf("command must return errors without duplicate error/usage output: %s", out.String())
	}
	for _, command := range []string{"init", "lock", "verify", "validate", "tidy"} {
		rootCmd.SetArgs([]string{"--workspace", root, command, "unexpected"})
		if err := rootCmd.Execute(); err == nil {
			t.Fatalf("%s must reject positional arguments", command)
		}
	}
}

func TestRootHelp(t *testing.T) {
	defer func() {
		rootCmd.SetOut(nil)
		rootCmd.SetArgs(nil)
	}()
	b := bytes.NewBufferString("")
	rootCmd.SetOut(b)
	rootCmd.SetArgs([]string{"--help"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("root --help failed: %v", err)
	}

	if b.Len() == 0 {
		t.Fatalf("expected non-empty help output")
	}
	for _, text := range []string{"evi [command]", "init", "lock", "verify", "--workspace", "-w"} {
		if !strings.Contains(b.String(), text) {
			t.Fatalf("missing %q in help: %s", text, b.String())
		}
	}
}

func TestInitCreatesEvidenceAndREADME(t *testing.T) {
	root := filepath.Join(t.TempDir(), "workspace")
	out := new(bytes.Buffer)
	rootCmd.SetOut(out)
	rootCmd.SetErr(out)
	rootCmd.SetArgs([]string{"--workspace", root, "init"})
	defer func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
		workspaceDir = "."
	}()
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"evidence.yaml", "README.md"} {
		if !strings.Contains(out.String(), filepath.Join(root, name)) {
			t.Fatalf("init must announce %s: %s", name, out.String())
		}
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || entries[0].Name() != "README.md" || entries[1].Name() != "evidence.yaml" {
		t.Fatalf("init must create only README.md and evidence.yaml, got %v", entries)
	}
}

func TestInitHelp(t *testing.T) {
	defer func() {
		rootCmd.SetOut(nil)
		rootCmd.SetArgs(nil)
	}()
	b := bytes.NewBufferString("")
	rootCmd.SetOut(b)
	rootCmd.SetArgs([]string{"init", "--help"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("init --help failed: %v", err)
	}
}

func TestTidyHelp(t *testing.T) {
	defer func() {
		rootCmd.SetOut(nil)
		rootCmd.SetArgs(nil)
	}()
	b := bytes.NewBufferString("")
	rootCmd.SetOut(b)
	rootCmd.SetArgs([]string{"tidy", "--help"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("tidy --help failed: %v", err)
	}
}
