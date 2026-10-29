package evidock

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestInitCreatesEvidenceAndWorkspaceREADME(t *testing.T) {
	root := t.TempDir()
	root = filepath.Join(root, "new", "workspace")
	svc := NewWorkspaceService()

	if err := svc.Init(root); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || entries[0].Name() != "README.md" || entries[1].Name() != "evidence.yaml" ||
		entries[0].IsDir() || entries[1].IsDir() {
		t.Fatalf("init must create only README.md and evidence.yaml, got %v", entries)
	}
	readme, err := os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{
		"Created by evi init",
		"workspace is initialized",
		"check --workspace before running init again",
		"docs/<product>/",
		"reports/<product>/<requirement>/",
		"CHANGELOG.md",
		"main or summary reports",
		"This is a suggestion, not a required structure.",
	} {
		if !strings.Contains(string(readme), text) {
			t.Fatalf("workspace README must contain %q", text)
		}
	}
	if strings.Contains(string(readme), "assessments/") {
		t.Fatal("workspace README must not prescribe an assessments directory")
	}
}

func TestInitAndTidyPreserveFilesAndEnsureScaffolding(t *testing.T) {
	root := t.TempDir()
	svc := NewWorkspaceService()
	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatal(err)
	}
	existing := filepath.Join(root, "docs", "notes.md")
	if err := os.MkdirAll(filepath.Dir(existing), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(existing, []byte("keep me"), 0644); err != nil {
		t.Fatal(err)
	}
	for _, operation := range []func(string) error{svc.Init, svc.Tidy} {
		if err := operation(root); err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(existing)
		if err != nil || string(got) != "keep me" {
			t.Fatalf("existing files must survive init/tidy: %v", err)
		}
	}
	if err := os.Remove(filepath.Join(root, "evidence.yaml")); err != nil {
		t.Fatal(err)
	}
	if err := svc.Tidy(root); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("tidy must recreate evidence.yaml and README.md, preserving existing files; got %v", entries)
	}
	for _, name := range []string{"evidence.yaml", "README.md"} {
		if _, err := os.Stat(filepath.Join(root, name)); err != nil {
			t.Fatalf("tidy must recreate %s: %v", name, err)
		}
	}
}

func TestInitDoesNotOverwriteExistingEvidence(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "evidence.yaml")
	custom := "evidence: [] # user-owned\n"
	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(custom), 0644); err != nil {
		t.Fatal(err)
	}
	if err := NewWorkspaceService().Init(root); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != custom {
		t.Fatalf("existing evidence must remain untouched: %v", err)
	}
}

func TestInitDoesNotOverwriteWorkspaceREADME(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "README.md")
	custom := "# Project-specific instructions\n"
	if err := os.WriteFile(path, []byte(custom), 0644); err != nil {
		t.Fatal(err)
	}
	if err := NewWorkspaceService().Init(root); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != custom {
		t.Fatalf("existing workspace README must remain untouched: %v", err)
	}
}

func TestTidyIsIdempotent(t *testing.T) {
	root := t.TempDir()
	svc := NewWorkspaceService()
	if err := svc.Init(root); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	snapshot, err := readSnapshot(root)
	if err != nil {
		t.Fatalf("snapshot failed: %v", err)
	}

	if err := svc.Tidy(root); err != nil {
		t.Fatalf("first tidy failed: %v", err)
	}
	if err := svc.Tidy(root); err != nil {
		t.Fatalf("second tidy failed: %v", err)
	}

	snapshotAfter, err := readSnapshot(root)
	if err != nil {
		t.Fatalf("snapshot after failed: %v", err)
	}
	if snapshot != snapshotAfter {
		t.Fatalf("expected idempotent tidy")
	}
}

func TestTidyPreservesLegacyManifestAndFiles(t *testing.T) {
	root := t.TempDir()
	manifest := `schema_version: 1
products:
  - id: product-a
    assessments:
      - type: cra-gap
        requirement: cra-001
`
	if err := os.WriteFile(filepath.Join(root, "evidock.yaml"), []byte(manifest), 0644); err != nil {
		t.Fatalf("write legacy manifest failed: %v", err)
	}

	legacyFile := filepath.Join(root, "products", "product-a", "shared", "CONVERSATION.md")
	if err := os.MkdirAll(filepath.Dir(legacyFile), 0755); err != nil {
		t.Fatalf("create legacy directory failed: %v", err)
	}
	if err := os.WriteFile(legacyFile, []byte("keep this"), 0644); err != nil {
		t.Fatalf("write legacy file failed: %v", err)
	}
	legacyPaths := []string{"docs/cra.md", "output/run/report.md", "TASKS.md", "skills/supporting-v1.md"}
	for _, name := range legacyPaths {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("legacy "+name), 0644); err != nil {
			t.Fatal(err)
		}
	}

	if err := NewWorkspaceService().Tidy(root); err != nil {
		t.Fatalf("legacy manifest should not prevent tidy: %v", err)
	}

	got, err := os.ReadFile(legacyFile)
	if err != nil {
		t.Fatalf("legacy file should remain: %v", err)
	}
	if string(got) != "keep this" {
		t.Fatalf("legacy file content changed: %q", got)
	}
	gotManifest, err := os.ReadFile(filepath.Join(root, "evidock.yaml"))
	if err != nil || string(gotManifest) != manifest {
		t.Fatalf("legacy manifest should remain unchanged: %v", err)
	}
	for _, name := range []string{"evidence.yaml", "README.md"} {
		if _, err := os.Stat(filepath.Join(root, name)); err != nil {
			t.Fatalf("tidy should create %s: %v", name, err)
		}
	}
	for _, name := range legacyPaths {
		data, err := os.ReadFile(filepath.Join(root, name))
		if err != nil || string(data) != "legacy "+name {
			t.Fatalf("legacy path %s must remain unchanged: %v", name, err)
		}
	}
}

func TestTidyWithoutManifestRestoresMissingFilesAndPreservesEdits(t *testing.T) {
	root := filepath.Join(t.TempDir(), "workspace")
	svc := NewWorkspaceService()
	if err := svc.Tidy(root); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "evidence.yaml"), []byte("evidence: [] # custom\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("# Custom README\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "user.md"), []byte("user docs"), 0644); err != nil {
		t.Fatal(err)
	}
	before, err := readSnapshot(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Init(root); err != nil {
		t.Fatal(err)
	}
	after, err := readSnapshot(root)
	if err != nil {
		t.Fatal(err)
	}
	if before != after {
		t.Fatal("init must preserve all existing content")
	}
	if err := os.Remove(filepath.Join(root, "evidence.yaml")); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, "README.md")); err != nil {
		t.Fatal(err)
	}
	if err := svc.Tidy(root); err != nil {
		t.Fatal(err)
	}
	evidence, err := os.ReadFile(filepath.Join(root, "evidence.yaml"))
	if err != nil || string(evidence) != defaultEvidenceYAML {
		t.Fatalf("tidy should restore missing evidence: %v", err)
	}
	readme, err := os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil || string(readme) != defaultWorkspaceREADME {
		t.Fatalf("tidy should restore missing workspace README: %v", err)
	}
}

func TestEvidenceTemplateIsEmptyAndExampleIsValid(t *testing.T) {
	var declaration struct {
		Evidence []struct {
			ID          string `yaml:"id"`
			Product     string `yaml:"product"`
			Requirement string `yaml:"requirement"`
			Path        string `yaml:"path"`
		} `yaml:"evidence"`
	}
	if err := yaml.Unmarshal([]byte(defaultEvidenceYAML), &declaration); err != nil {
		t.Fatal(err)
	}
	if len(declaration.Evidence) != 0 {
		t.Fatal("init must not invent evidence")
	}
	var example strings.Builder
	example.WriteString("evidence:\n")
	for _, line := range strings.Split(defaultEvidenceYAML, "\n") {
		if strings.HasPrefix(line, "# - ") || strings.HasPrefix(line, "#   ") {
			example.WriteString("  " + strings.TrimPrefix(line, "# ") + "\n")
		}
	}
	if err := yaml.Unmarshal([]byte(example.String()), &declaration); err != nil {
		t.Fatal(err)
	}
	if len(declaration.Evidence) != 1 {
		t.Fatalf("expected one example entry: %s", example.String())
	}
	entry := declaration.Evidence[0]
	if entry.ID != "ev-001" || entry.Product != "product-a" || entry.Requirement != "update-integrity" || entry.Path != "sources/update.go" {
		t.Fatalf("unexpected evidence example: %+v", entry)
	}
}

func readSnapshot(root string) (string, error) {
	var b strings.Builder
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		b.WriteString(rel)
		b.WriteString("\n")
		b.Write(content)
		b.WriteString("\n---\n")
		return nil
	})
	if err != nil {
		return "", err
	}
	return b.String(), nil
}
