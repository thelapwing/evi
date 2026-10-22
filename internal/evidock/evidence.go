package evidock

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Evidence struct {
	Registry    string `yaml:"registry,omitempty" json:"Registry,omitempty"`
	ID          string `yaml:"id"`
	Product     string `yaml:"product"`
	Requirement string `yaml:"requirement"`
	Path        string `yaml:"path"`
	StartLine   *int   `yaml:"start_line,omitempty"`
	EndLine     *int   `yaml:"end_line,omitempty"`
}

type LockedFile struct {
	Path   string `yaml:"path"`
	SHA256 string `yaml:"sha256"`
}

type LockedFragment struct {
	Registry  string `yaml:"registry,omitempty"`
	ID        string `yaml:"id"`
	Path      string `yaml:"path"`
	StartLine int    `yaml:"start_line"`
	EndLine   int    `yaml:"end_line"`
	SHA256    string `yaml:"sha256"`
}

type EvidenceLock struct {
	SchemaVersion  int              `yaml:"schema_version"`
	LockedAt       string           `yaml:"locked_at"`
	EvidenceSHA256 string           `yaml:"evidence_sha256"`
	Declarations   []Evidence       `yaml:"declarations,omitempty"`
	Registries     []string         `yaml:"registries,omitempty"`
	Files          []LockedFile     `yaml:"files"`
	Fragments      []LockedFragment `yaml:"fragments"`
}

type evidenceKey struct {
	registry string
	id       string
}

func (e Evidence) key() evidenceKey {
	return evidenceKey{e.Registry, e.ID}
}

func (e Evidence) label() string {
	if e.Registry != "" {
		return e.Registry + "#" + e.ID
	}
	return e.ID
}

func evidenceRegistries(root string) ([]string, bool, error) {
	indexPath := filepath.Join(root, "evi.yaml")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		return []string{"evidence.yaml"}, false, nil
	} else if err != nil {
		return nil, false, fmt.Errorf("stat workspace index: %w", err)
	}
	var index struct {
		EvidenceFiles []string `yaml:"evidence_files"`
	}
	if err := decodeYAML(indexPath, &index); err != nil {
		return nil, true, err
	}
	if len(index.EvidenceFiles) == 0 {
		return nil, true, fmt.Errorf("evi.yaml: evidence_files must be a non-empty list")
	}
	seen := make(map[string]bool)
	for i, path := range index.EvidenceFiles {
		clean := filepath.Clean(path)
		if strings.TrimSpace(path) == "" || filepath.IsAbs(path) ||
			filepath.VolumeName(path) != "" || clean == "." || clean == ".." ||
			strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			return nil, true, fmt.Errorf("evi.yaml: registry %q must be a file path within the workspace", path)
		}
		clean = filepath.ToSlash(clean)
		if seen[clean] {
			return nil, true, fmt.Errorf("evi.yaml: duplicate registry %q", clean)
		}
		seen[clean] = true
		index.EvidenceFiles[i] = clean
	}
	sort.Strings(index.EvidenceFiles)
	return index.EvidenceFiles, true, nil
}

func decodeYAML(path string, target interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	var extra interface{}
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("%s: expected exactly one YAML document", path)
		}
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}

func digest(data []byte) string {
	return fmt.Sprintf("%x", sha256.Sum256(data))
}

func (s *WorkspaceService) Validate(root string) (int, error) {
	entries, _, err := collectEvidence(root)
	return len(entries), err
}

func collectEvidence(root string) ([]Evidence, EvidenceLock, error) {
	lock := EvidenceLock{
		SchemaVersion: 2,
		Files:         []LockedFile{},
		Fragments:     []LockedFragment{},
	}
	registries, indexed, err := evidenceRegistries(root)
	if err != nil {
		return nil, lock, err
	}
	if indexed {
		lock.SchemaVersion = 3
		lock.Registries = registries
	}
	entries := []Evidence{}
	for _, registry := range registries {
		var declaration struct {
			Evidence *[]Evidence `yaml:"evidence"`
		}
		if err := decodeYAML(filepath.Join(root, registry), &declaration); err != nil {
			return nil, lock, err
		}
		if declaration.Evidence == nil {
			return nil, lock, fmt.Errorf("%s: evidence must be a list (use evidence: [] for an empty registry)", registry)
		}
		for _, entry := range *declaration.Evidence {
			if entry.Registry != "" {
				return nil, lock, fmt.Errorf("%s: evidence %q: registry is assigned by evi, not by declarations", registry, entry.ID)
			}
			if indexed {
				entry.Registry = registry
			}
			entries = append(entries, entry)
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Registry != entries[j].Registry {
			return entries[i].Registry < entries[j].Registry
		}
		return entries[i].ID < entries[j].ID
	})
	ids := make(map[evidenceKey]bool)
	files := make(map[string][]byte)
	lockPath := filepath.Join(root, "evidence.lock.yaml")
	lockInfo, err := os.Stat(lockPath)
	if err != nil && !os.IsNotExist(err) {
		return nil, lock, fmt.Errorf("stat lock: %w", err)
	}
	for _, entry := range entries {
		if strings.TrimSpace(entry.ID) == "" || strings.TrimSpace(entry.Product) == "" ||
			strings.TrimSpace(entry.Requirement) == "" || strings.TrimSpace(entry.Path) == "" {
			return nil, lock, fmt.Errorf("evidence %q: id, product, requirement and path must be non-empty", entry.label())
		}
		if ids[entry.key()] {
			return nil, lock, fmt.Errorf("duplicate evidence id %q", entry.label())
		}
		ids[entry.key()] = true
		if filepath.IsAbs(entry.Path) || filepath.VolumeName(entry.Path) != "" {
			return nil, lock, fmt.Errorf("evidence %s: path must be workspace-relative", entry.label())
		}
		if (entry.StartLine == nil) != (entry.EndLine == nil) {
			return nil, lock, fmt.Errorf("evidence %s: start_line and end_line must be supplied together", entry.label())
		}
		path := filepath.Join(root, entry.Path)
		absolute, err := filepath.Abs(path)
		if err != nil {
			return nil, lock, fmt.Errorf("evidence %s: resolve path: %w", entry.label(), err)
		}
		absoluteLock, err := filepath.Abs(lockPath)
		if err != nil {
			return nil, lock, fmt.Errorf("resolve lock path: %w", err)
		}
		if absolute == absoluteLock {
			return nil, lock, fmt.Errorf("evidence %s: cannot reference the generated lock file", entry.label())
		}
		info, err := os.Stat(path)
		if err != nil {
			return nil, lock, fmt.Errorf("evidence %s: stat %s: %w", entry.label(), entry.Path, err)
		}
		if !info.Mode().IsRegular() {
			return nil, lock, fmt.Errorf("evidence %s: %s is not a regular file", entry.label(), entry.Path)
		}
		if lockInfo != nil && os.SameFile(info, lockInfo) {
			return nil, lock, fmt.Errorf("evidence %s: cannot reference the generated lock file", entry.label())
		}
		content, ok := files[entry.Path]
		if !ok {
			content, err = os.ReadFile(path)
			if err != nil {
				return nil, lock, fmt.Errorf("evidence %s: read %s: %w", entry.label(), entry.Path, err)
			}
			files[entry.Path] = content
		}
		if entry.StartLine != nil {
			// Retain exact bytes, including CRLF and the final newline if present.
			lines := bytes.SplitAfter(content, []byte("\n"))
			if len(lines[len(lines)-1]) == 0 {
				lines = lines[:len(lines)-1]
			}
			first, last := *entry.StartLine, *entry.EndLine
			if first < 1 || last < first || last > len(lines) {
				return nil, lock, fmt.Errorf("evidence %s: invalid line range %d-%d (file has %d lines)", entry.label(), first, last, len(lines))
			}
			lock.Fragments = append(lock.Fragments, LockedFragment{
				Registry: entry.Registry, ID: entry.ID, Path: entry.Path, StartLine: first, EndLine: last,
				SHA256: digest(bytes.Join(lines[first-1:last], nil)),
			})
		}
	}
	canonical, err := json.Marshal(entries)
	if err != nil {
		return nil, lock, fmt.Errorf("encode evidence declarations: %w", err)
	}
	lock.EvidenceSHA256 = digest(canonical)
	lock.Declarations = entries
	for path, content := range files {
		lock.Files = append(lock.Files, LockedFile{Path: path, SHA256: digest(content)})
	}
	sort.Slice(lock.Files, func(i, j int) bool { return lock.Files[i].Path < lock.Files[j].Path })
	return entries, lock, nil
}

func (s *WorkspaceService) Lock(root string) (*EvidenceLock, error) {
	entries, lock, err := collectEvidence(root)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("cannot lock empty evidence")
	}
	lock.LockedAt = time.Now().UTC().Format(time.RFC3339Nano)
	content, err := yaml.Marshal(lock)
	if err != nil {
		return nil, fmt.Errorf("encode lock: %w", err)
	}
	if err := writeLock(root, content); err != nil {
		return nil, err
	}
	return &lock, nil
}

func writeLock(root string, content []byte) error {
	file, err := os.CreateTemp(root, ".evidence-lock-*")
	if err != nil {
		return fmt.Errorf("create temporary lock: %w", err)
	}
	name := file.Name()
	defer os.Remove(name)
	defer file.Close()
	if err := file.Chmod(0644); err != nil {
		return fmt.Errorf("set lock permissions: %w", err)
	}
	if _, err := file.Write(content); err != nil {
		return fmt.Errorf("write lock: %w", err)
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("sync lock: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close lock: %w", err)
	}
	if err := os.Rename(name, filepath.Join(root, "evidence.lock.yaml")); err != nil {
		return fmt.Errorf("replace lock: %w", err)
	}
	return nil
}

func (s *WorkspaceService) Verify(root string) (*EvidenceLock, error) {
	var stored EvidenceLock
	if err := decodeYAML(filepath.Join(root, "evidence.lock.yaml"), &stored); err != nil {
		return nil, err
	}
	if stored.SchemaVersion != 1 && stored.SchemaVersion != 2 && stored.SchemaVersion != 3 {
		return nil, fmt.Errorf("unsupported lock schema_version %d", stored.SchemaVersion)
	}
	if _, err := time.Parse(time.RFC3339Nano, stored.LockedAt); err != nil {
		return nil, fmt.Errorf("invalid locked_at; generate a lock first: %w", err)
	}
	if stored.SchemaVersion >= 2 {
		canonical, err := json.Marshal(stored.Declarations)
		if err != nil {
			return nil, fmt.Errorf("encode locked declarations: %w", err)
		}
		if digest(canonical) != stored.EvidenceSHA256 {
			return nil, fmt.Errorf("lock declarations do not match evidence_sha256; inspect evidence.lock.yaml")
		}
	}
	entries, current, err := collectEvidence(root)
	if err != nil {
		return nil, err
	}
	var changes strings.Builder
	if !reflect.DeepEqual(stored.Registries, current.Registries) {
		fmt.Fprintf(&changes, "\nworkspace registry selection changed: locked %v; current %v; review evi.yaml and explicitly run evi lock after accepting changes", stored.Registries, current.Registries)
	}
	if stored.EvidenceSHA256 != current.EvidenceSHA256 {
		if stored.SchemaVersion == 1 {
			return nil, fmt.Errorf("evidence declarations changed since lock; schema 1 has no declaration details. Review evidence.yaml changes, then explicitly run evi lock to upgrade to schema 2")
		}
		changes.WriteString("\nevidence declarations changed since lock:")
		changes.WriteString(declarationChanges(stored.Declarations, entries))
	}
	if !reflect.DeepEqual(stored.Files, current.Files) {
		changes.WriteString("\nsource files or SHA-256 hashes differ from lock")
		changes.WriteString(evidenceChanges(entries, stored, current))
	} else if !reflect.DeepEqual(stored.Fragments, current.Fragments) {
		changes.WriteString("\nsource fragments or SHA-256 hashes differ from lock")
		changes.WriteString(evidenceChanges(entries, stored, current))
	}
	if changes.Len() > 0 {
		return nil, fmt.Errorf("evidence verification failed:%s", changes.String())
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("cannot verify empty evidence")
	}
	return &stored, nil
}

func declarationChanges(stored, current []Evidence) string {
	old := make(map[evidenceKey]Evidence, len(stored))
	for _, entry := range stored {
		old[entry.key()] = entry
	}
	var details strings.Builder
	for _, entry := range current {
		previous, exists := old[entry.key()]
		if !exists {
			fmt.Fprintf(&details, "\n- %s [evidence: %s]: declaration added", entry.Path, entry.label())
		} else if !reflect.DeepEqual(previous, entry) {
			fmt.Fprintf(&details, "\n- %s [evidence: %s]: declaration changed (previous path: %s; check labels, path and range)", entry.Path, entry.label(), previous.Path)
		}
		delete(old, entry.key())
	}
	for _, entry := range stored {
		if _, exists := old[entry.key()]; exists {
			fmt.Fprintf(&details, "\n- %s [evidence: %s]: declaration removed", entry.Path, entry.label())
		}
	}
	return details.String()
}

func evidenceChanges(entries []Evidence, stored, current EvidenceLock) string {
	files := make(map[string]string, len(stored.Files))
	for _, file := range stored.Files {
		files[file.Path] = file.SHA256
	}
	fragments := make(map[evidenceKey]LockedFragment, len(stored.Fragments))
	for _, fragment := range stored.Fragments {
		fragments[evidenceKey{fragment.Registry, fragment.ID}] = fragment
	}
	currentFragments := make(map[evidenceKey]LockedFragment, len(current.Fragments))
	for _, fragment := range current.Fragments {
		currentFragments[evidenceKey{fragment.Registry, fragment.ID}] = fragment
	}
	changedFiles := make(map[string]bool)
	for _, file := range current.Files {
		changedFiles[file.Path] = files[file.Path] != file.SHA256
	}

	var details strings.Builder
	for _, entry := range entries {
		fragment, ranged := currentFragments[entry.key()]
		fragmentChanged := ranged && fragments[entry.key()] != fragment
		if !changedFiles[entry.Path] && !fragmentChanged {
			continue
		}
		fmt.Fprintf(&details, "\n- %s [evidence: %s]", entry.Path, entry.label())
		if !ranged {
			details.WriteString(": whole-file SHA-256 changed")
		} else if fragmentChanged {
			fmt.Fprintf(&details, ": fragment at lines %d-%d differs from lock; check content and possible line shifts", fragment.StartLine, fragment.EndLine)
		} else {
			fmt.Fprintf(&details, ": file SHA-256 changed; fragment at lines %d-%d unchanged (change outside selected range)", fragment.StartLine, fragment.EndLine)
		}
	}
	if details.Len() == 0 {
		details.WriteString("\n- lock file/fragment records differ; inspect evidence.lock.yaml")
	}
	return details.String()
}
