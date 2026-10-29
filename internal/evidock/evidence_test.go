package evidock

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func writeEvidenceTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func evidenceWorkspace(t *testing.T) string {
	t.Helper()
	base := t.TempDir()
	root := filepath.Join(base, "workspace")
	if err := NewWorkspaceService().Init(root); err != nil {
		t.Fatal(err)
	}
	writeEvidenceTestFile(t, filepath.Join(base, "source.txt"), "one\r\ntwo\nthree")
	writeEvidenceTestFile(t, filepath.Join(base, "other.txt"), "alpha\nbeta")
	writeEvidenceTestFile(t, filepath.Join(root, "evidence.yaml"), `evidence:
  - id: ev-first
    product: rauc
    requirement: cra-2d
    path: ../source.txt
  - id: ev-second
    product: rauc
    requirement: cra-2d
    path: ../other.txt
`)
	if _, err := NewWorkspaceService().Lock(root); err != nil {
		t.Fatal(err)
	}
	return root
}

func readEvidenceTestFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestEvidenceLockRoundTrip(t *testing.T) {
	root := evidenceWorkspace(t)
	svc := NewWorkspaceService()
	before := readEvidenceTestFile(t, filepath.Join(root, "evidence.yaml"))
	count, err := svc.Validate(root)
	if err != nil || count != 2 {
		t.Fatalf("validate: count=%d err=%v", count, err)
	}
	lock, err := svc.Lock(root)
	if err != nil {
		t.Fatal(err)
	}
	if lock.SchemaVersion != 2 || len(lock.Declarations) != 2 || len(lock.Files) != 2 {
		t.Fatalf("unexpected lock: %+v", lock)
	}
	if lock.Files[0].SHA256 != digest([]byte("alpha\nbeta")) ||
		lock.Files[1].SHA256 != digest([]byte("one\r\ntwo\nthree")) {
		t.Fatalf("incorrect raw-byte hashes: %+v", lock)
	}
	if _, err := time.Parse(time.RFC3339Nano, lock.LockedAt); err != nil {
		t.Fatal(err)
	}
	saved := readEvidenceTestFile(t, filepath.Join(root, "evidence.lock.yaml"))
	if _, err := svc.Verify(root); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(saved, readEvidenceTestFile(t, filepath.Join(root, "evidence.lock.yaml"))) ||
		!bytes.Equal(before, readEvidenceTestFile(t, filepath.Join(root, "evidence.yaml"))) {
		t.Fatal("validate/verify/lock must preserve evidence; verify must preserve lock")
	}
	entries, _, err := collectEvidence(root)
	if err != nil {
		t.Fatal(err)
	}
	entries[0], entries[1] = entries[1], entries[0]
	reordered, err := yaml.Marshal(struct {
		Evidence []Evidence `yaml:"evidence"`
	}{entries})
	if err != nil {
		t.Fatal(err)
	}
	writeEvidenceTestFile(t, filepath.Join(root, "evidence.yaml"), "# formatting changes do not invalidate lock\n"+string(reordered))
	if _, err := svc.Verify(root); err != nil {
		t.Fatalf("reordered declarations must verify: %v", err)
	}
}

func TestValidateRejectsInvalidEvidence(t *testing.T) {
	cases := map[string]string{
		"missing list":       "{}",
		"null list":          "evidence: null",
		"wrong list type":    "evidence: text",
		"multiple documents": "evidence: []\n---\nevidence: []",
		"unknown field":      "evidence: []\nunknown: true",
		"duplicate key":      "evidence: []\nevidence: []",
		"empty entry":        "evidence: [{}]",
		"unknown entry key":  "evidence: [{id: ev, product: p, requirement: r, path: ../source.txt, typo: 1}]",
		"blank label":        "evidence: [{id: ev, product: ' ', requirement: r, path: ../source.txt}]",
		"absolute path":      "evidence: [{id: ev, product: p, requirement: r, path: /tmp/source.txt}]",
		"missing file":       "evidence: [{id: ev, product: p, requirement: r, path: missing.txt}]",
		"directory":          "evidence: [{id: ev, product: p, requirement: r, path: shared}]",
		"self lock":          "evidence: [{id: ev, product: p, requirement: r, path: evidence.lock.yaml}]",
		"line fields":        "evidence: [{id: ev, product: p, requirement: r, path: ../source.txt, start_line: 1, end_line: 2}]",
		"duplicate ID": `evidence:
  - {id: ev, product: p, requirement: r, path: ../source.txt}
  - {id: ev, product: p, requirement: r, path: ../source.txt}
`,
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			root := evidenceWorkspace(t)
			writeEvidenceTestFile(t, filepath.Join(root, "evidence.yaml"), input)
			before := readEvidenceTestFile(t, filepath.Join(root, "evidence.lock.yaml"))
			if _, err := NewWorkspaceService().Validate(root); err == nil {
				t.Fatal("expected validation failure")
			}
			if _, err := NewWorkspaceService().Lock(root); err == nil {
				t.Fatal("expected lock failure")
			}
			if !bytes.Equal(before, readEvidenceTestFile(t, filepath.Join(root, "evidence.lock.yaml"))) {
				t.Fatal("failed lock must preserve previous file")
			}
		})
	}
}

func TestVerifyDetectsChanges(t *testing.T) {
	for _, change := range []string{"source", "label", "id", "path", "remove entry", "file hash", "schema", "timestamp", "unknown lock field", "missing source"} {
		t.Run(change, func(t *testing.T) {
			root := evidenceWorkspace(t)
			svc := NewWorkspaceService()
			if _, err := svc.Lock(root); err != nil {
				t.Fatal(err)
			}
			evidencePath := filepath.Join(root, "evidence.yaml")
			lockPath := filepath.Join(root, "evidence.lock.yaml")
			text := string(readEvidenceTestFile(t, evidencePath))
			switch change {
			case "source":
				writeEvidenceTestFile(t, filepath.Join(root, "../source.txt"), "one\r\nchanged\nthree")
			case "label":
				text = strings.ReplaceAll(text, "rauc", "other")
			case "id":
				text = strings.ReplaceAll(text, "ev-first", "new-id")
			case "path":
				text = strings.ReplaceAll(text, "../source.txt", ".././source.txt")
			case "remove entry":
				text = strings.Split(text, "  - id: ev-second")[0]
			case "file hash", "schema", "timestamp":
				var lock EvidenceLock
				if err := decodeYAML(lockPath, &lock); err != nil {
					t.Fatal(err)
				}
				switch change {
				case "file hash":
					lock.Files[0].SHA256 = strings.Repeat("0", 64)
				case "schema":
					lock.SchemaVersion = 99
				case "timestamp":
					lock.LockedAt = ""
				}
				data, err := yaml.Marshal(lock)
				if err != nil {
					t.Fatal(err)
				}
				writeEvidenceTestFile(t, lockPath, string(data))
			case "unknown lock field":
				writeEvidenceTestFile(t, lockPath, string(readEvidenceTestFile(t, lockPath))+"unknown: true\n")
			case "missing source":
				if err := os.Remove(filepath.Join(root, "../source.txt")); err != nil {
					t.Fatal(err)
				}
			}
			writeEvidenceTestFile(t, evidencePath, text)
			before := readEvidenceTestFile(t, lockPath)
			if _, err := svc.Verify(root); err == nil {
				t.Fatal("expected verification failure")
			}
			if !bytes.Equal(before, readEvidenceTestFile(t, lockPath)) {
				t.Fatal("verify must not change lock even on failure")
			}
		})
	}
}

func TestEmptyAndMissingEvidence(t *testing.T) {
	root := t.TempDir()
	svc := NewWorkspaceService()
	if _, err := svc.Validate(root); err == nil {
		t.Fatal("missing evidence must fail")
	}
	if _, err := svc.Verify(root); err == nil {
		t.Fatal("missing lock must fail")
	}
	if err := svc.Init(root); err != nil {
		t.Fatal(err)
	}
	if count, err := svc.Validate(root); err != nil || count != 0 {
		t.Fatalf("empty scaffold should validate: %d, %v", count, err)
	}
	if _, err := svc.Lock(root); err == nil {
		t.Fatal("empty evidence must not produce a successful lock")
	}
	if _, err := svc.Verify(root); err == nil {
		t.Fatal("empty scaffold lock must not verify")
	}
}

func TestEvidenceCannotAliasLock(t *testing.T) {
	root := evidenceWorkspace(t)
	if err := os.Symlink("evidence.lock.yaml", filepath.Join(root, "alias.yaml")); err != nil {
		t.Fatal(err)
	}
	writeEvidenceTestFile(t, filepath.Join(root, "evidence.yaml"),
		"evidence: [{id: ev, product: p, requirement: r, path: alias.yaml}]\n")
	if _, err := NewWorkspaceService().Validate(root); err == nil {
		t.Fatal("lock aliases must be rejected")
	}
}

func TestVerifyChangeDiagnostics(t *testing.T) {
	for _, tc := range []struct {
		name    string
		path    string
		content string
		want    string
		notWant string
	}{
		{"first source", "../source.txt", "one\r\nchanged\nthree", "../source.txt [evidence: ev-first]: file SHA-256 changed", "ev-second"},
		{"second source", "../other.txt", "alpha\nchanged", "../other.txt [evidence: ev-second]: file SHA-256 changed", "ev-first"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := evidenceWorkspace(t)
			svc := NewWorkspaceService()
			if _, err := svc.Lock(root); err != nil {
				t.Fatal(err)
			}

			before := readEvidenceTestFile(t, filepath.Join(root, "evidence.lock.yaml"))
			writeEvidenceTestFile(t, filepath.Join(root, tc.path), tc.content)
			_, err := svc.Verify(root)
			if err == nil {
				t.Fatal("expected verification failure")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected %q in %q", tc.want, err.Error())
			}
			if strings.Contains(err.Error(), tc.notWant) {
				t.Fatalf("unchanged evidence must not be reported in %q", err.Error())
			}
			if !bytes.Equal(before, readEvidenceTestFile(t, filepath.Join(root, "evidence.lock.yaml"))) {
				t.Fatal("verify must preserve lock")
			}
		})
	}
}

func TestVerifyDeclarationDiagnostics(t *testing.T) {
	for _, tc := range []struct {
		name string
		edit func(string) string
		want []string
	}{
		{"labels", func(s string) string { return strings.ReplaceAll(s, "rauc", "updated") },
			[]string{"ev-first]: declaration changed", "ev-second]: declaration changed"}},
		{"path", func(s string) string { return strings.ReplaceAll(s, "../source.txt", ".././source.txt") },
			[]string{".././source.txt [evidence: ev-first]: declaration changed", "previous path: ../source.txt"}},
		{"rename", func(s string) string { return strings.ReplaceAll(s, "ev-first", "ev-new") },
			[]string{"ev-new]: declaration added", "ev-first]: declaration removed"}},
		{"remove", func(s string) string { return strings.Split(s, "  - id: ev-second")[0] },
			[]string{"ev-second]: declaration removed"}},
		{"remove all", func(string) string { return "evidence: []\n" },
			[]string{"ev-first]: declaration removed", "ev-second]: declaration removed"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := evidenceWorkspace(t)
			svc := NewWorkspaceService()
			if _, err := svc.Lock(root); err != nil {
				t.Fatal(err)
			}
			lockPath := filepath.Join(root, "evidence.lock.yaml")
			before := readEvidenceTestFile(t, lockPath)
			path := filepath.Join(root, "evidence.yaml")
			writeEvidenceTestFile(t, path, tc.edit(string(readEvidenceTestFile(t, path))))
			_, err := svc.Verify(root)
			if err == nil {
				t.Fatal("expected declaration mismatch")
			}
			for _, want := range tc.want {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("expected %q in %v", want, err)
				}
			}
			if !bytes.Equal(before, readEvidenceTestFile(t, lockPath)) {
				t.Fatal("verify changed lock")
			}
		})
	}
}

func TestVerifyLegacyLock(t *testing.T) {
	root := evidenceWorkspace(t)
	svc := NewWorkspaceService()
	lock, err := svc.Lock(root)
	if err != nil {
		t.Fatal(err)
	}
	lock.SchemaVersion = 1
	lock.Declarations = nil
	data, err := yaml.Marshal(lock)
	if err != nil {
		t.Fatal(err)
	}
	lockPath := filepath.Join(root, "evidence.lock.yaml")
	writeEvidenceTestFile(t, lockPath, string(data))
	if _, err := svc.Verify(root); err != nil {
		t.Fatalf("unchanged schema 1 must still verify: %v", err)
	}
	writeEvidenceTestFile(t, filepath.Join(root, "../source.txt"), "one\r\nchanged\nthree")
	if _, err := svc.Verify(root); err == nil || !strings.Contains(err.Error(), "ev-first") {
		t.Fatalf("schema 1 must still diagnose source changes: %v", err)
	}
	path := filepath.Join(root, "evidence.yaml")
	text := string(readEvidenceTestFile(t, path))
	writeEvidenceTestFile(t, path, strings.ReplaceAll(text, "rauc", "updated"))
	if _, err := svc.Verify(root); err == nil || !strings.Contains(err.Error(), "schema 1 has no declaration details") {
		t.Fatalf("expected explicit legacy limitation: %v", err)
	}
	if !bytes.Equal(data, readEvidenceTestFile(t, lockPath)) {
		t.Fatal("verify must not upgrade a legacy lock")
	}
	upgraded, err := svc.Lock(root)
	if err != nil || upgraded.SchemaVersion != 2 {
		t.Fatalf("explicit lock must upgrade: %v", err)
	}
	if _, err := svc.Verify(root); err != nil {
		t.Fatal(err)
	}
}

func TestVerifyRejectsInconsistentLockedDeclarations(t *testing.T) {
	root := evidenceWorkspace(t)
	svc := NewWorkspaceService()
	lock, err := svc.Lock(root)
	if err != nil {
		t.Fatal(err)
	}
	lock.Declarations[0].Product = "tampered"
	data, err := yaml.Marshal(lock)
	if err != nil {
		t.Fatal(err)
	}
	writeEvidenceTestFile(t, filepath.Join(root, "evidence.lock.yaml"), string(data))
	if _, err := svc.Verify(root); err == nil || !strings.Contains(err.Error(), "lock declarations do not match") {
		t.Fatalf("inconsistent lock must not pass: %v", err)
	}
}
