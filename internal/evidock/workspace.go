package evidock

import (
	"fmt"
	"os"
	"path/filepath"
)

type WorkspaceService struct{}

func NewWorkspaceService() *WorkspaceService {
	return &WorkspaceService{}
}

func (s *WorkspaceService) Init(root string) error {
	return s.Tidy(root)
}

func (s *WorkspaceService) Tidy(root string) error {
	if err := os.MkdirAll(root, 0755); err != nil {
		return fmt.Errorf("create workspace dir %s: %w", root, err)
	}
	if err := writeFileIfMissing(filepath.Join(root, "evidence.yaml"), []byte(defaultEvidenceYAML)); err != nil {
		return err
	}
	if err := writeFileIfMissing(filepath.Join(root, "README.md"), []byte(defaultWorkspaceREADME)); err != nil {
		return err
	}
	return nil
}

func writeFileIfMissing(path string, content []byte) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat %s: %w", path, err)
	}

	if err := os.WriteFile(path, content, 0644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

const defaultEvidenceYAML = `evidence: []

# Add real evidence entries when needed; init does not create evidence.
# Each entry identifies one file. Paths are relative to this workspace.
# - id: ev-001
#   product: product-a
#   requirement: update-integrity
#   path: sources/update.go
`

const defaultWorkspaceREADME = `# Workspace

Created by evi init (restored by evi tidy). Its presence means this
workspace is initialized; check --workspace before running init again.

This directory contains a project used with EviDock. The tool validates
declarations in evidence.yaml and checks the integrity of the referenced local
sources; it does not perform an assessment or determine compliance.

## Evidence

Add only real sources to evidence.yaml. Paths are relative to the workspace
root. After changing declarations or sources, validate them:

    evi --workspace . validate

After reviewing and accepting the changes, create or update the lock:

    evi --workspace . lock

Check the current state against the saved lock:

    evi --workspace . verify

The lock command records a new baseline; the verify command only checks it.
Do not run lock automatically after an unexpected verify failure. For command
details, use evi --help and evi <command> --help.

## Optional organization

The CLI requires only evidence.yaml and the lock created by evi lock.
Organize product documentation and reports as needed. For example:

    docs/
      <product>/
    reports/
      <main-report>.md
      <product>/
        <requirement>/
    CHANGELOG.md

docs/<product>/ holds product documentation and supporting materials used
across requirements.

reports/ holds main or summary reports, such as an overall product or release
report. Detailed reports for a specific requirement can go in
reports/<product>/<requirement>/. Use the requirement identifier as the
directory name, and preserve the evidence IDs and source provenance cited by
each report.

CHANGELOG.md summarizes notable changes to the workspace. When using version
control, it gives readers a concise overview while the commit history provides
the detailed record.

This is a suggestion, not a required structure. Evidence paths still refer to
the workspace root.
`
