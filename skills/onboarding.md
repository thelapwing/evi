# EviDock onboarding

Use this guide when the user asks what EviDock does, how to start or inspect a
workspace, how to register evidence, or how to run `validate`, `lock`, or
`verify`. Start with the tool's purpose, then inspect the target workspace.

Do not use this guide to design a project's directory structure or report
organization; use `project-guidance.md` for that. EviDock does not perform
requirement assessments, determine compliance, or prescribe an assessment
method.

## Explain

- EviDock is a local CLI for validating evidence declarations and checking
  whether registered source files still match a reviewed lock.
- It does not perform an assessment, decide compliance, or prescribe where
  documentation, analyses or reports must be stored.
- `evi init` creates an empty root `evidence.yaml` and a short workspace
   `README.md` if they are missing. It does not create a lock or project
   subdirectories.
   `evi lock` creates or intentionally replaces `evidence.lock.yaml`.
- Evidence paths are relative to the workspace root; relative paths may also
   refer to sources outside it. Use only paths and sources the user trusts.

## Walk through evidence

1. Identify the target workspace. If it is unclear, ask before running commands.
   Inspect its files and declarations without changing them. Read its
   `README.md`; use `evi --help` and `evi <command> --help` rather than guessing
   command syntax. If the README is missing, explain that `evi tidy` can
   restore the starter README without overwriting existing files. Use the
   installed `evi` command or the invocation documented in the tool README;
   do not assume the tool repository is the current directory.
2. Explain each field: stable unique `id`, `product`, `requirement`, and local
   source `path`.
3. Add only real sources selected with the user. Never invent IDs, paths,
   citations, hashes or test results. Preserve existing entries. An empty
   registry is valid scaffolding.
4. Run `evi --workspace <project> validate` to check declarations and sources.
5. After reviewing and accepting the evidence, run `lock`, then `verify`.
   Explain that `lock` updates the baseline, while `verify` only checks it.
6. If `verify` fails unexpectedly, investigate the listed changes first.
   Never run `lock` just to make the error disappear.

## Be clear about limits

Hashes detect changes; they do not prove source authenticity, evidence quality
or regulatory compliance. An empty registry validates but cannot be locked or
verified successfully. Do not claim a successful lock or verification unless
the command actually completed successfully. Refer users to the tool README
for CLI usage and format details. Use `project-guidance.md` only when they ask
for help organizing a project.
