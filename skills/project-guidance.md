# EviDock project guidance

Use this guide only when the user asks where to organize project documentation,
working notes, analysis materials, reports, or project history, or asks to
create or revise that structure. It helps choose and maintain a user-approved
layout; it does not define a required EviDock layout.

For questions about EviDock commands or evidence declarations, use
`onboarding.md` instead. Do not conduct a requirement assessment, choose an
assessment method, create report templates, or imply that a suggested folder
structure is needed for the CLI.

## Start with the user's workflow

1. Inspect the existing workspace and preserve its files. Use its `README.md`
   and the user's stated preferences as the current source of truth.
2. Ask only for missing decisions that affect the structure, such as whether
   products, releases, or requirements need separate locations. Do not repeat
   questions the user has already answered.
3. Propose the smallest layout that fits the stated workflow. Get the user's
   agreement before creating, moving, or renaming files. If they only ask for
   advice, do not make filesystem changes.
4. Keep the single root `evidence.yaml` as the evidence registry. Evidence
   paths are relative to the workspace root, even when a source is stored in a
   subdirectory or referenced outside the workspace.

## Optional practices

- The tool README and workspace README offer examples such as product
  documentation under `docs/<product>/`, main reports under `reports/`,
  detailed reports under `reports/<product>/<requirement>/`, and an optional
  `CHANGELOG.md`. Present these only as examples; follow the user's preferred
  names, grouping, and locations instead. Do not assign a fixed directory to
  analysis notes or materials unless the user chooses one.
- Reports can be stored wherever the user prefers. Help make their scope,
  evidence IDs, citations, and source provenance clear without requiring a
  particular report format.
- A concise changelog can summarize notable project changes, especially when
  version control is used. It is optional and complements rather than replaces
  the detailed commit history.
- Keep analysis notes, source material, and conclusions distinct where useful,
  but do not create extra directories or files unless the user wants them.
- Treat example projects as demonstrations to adapt, not structures to copy
  blindly. Never treat an example report or its conclusions as evidence.

The user chooses the project layout, assessment method, and report format.
This guide helps maintain the agreed organization; it does not assess
requirements or certify compliance.
