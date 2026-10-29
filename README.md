# EviDock

[![CI](https://github.com/thelapwing/evi/actions/workflows/ci.yml/badge.svg)](https://github.com/thelapwing/evi/actions/workflows/ci.yml)

EviDock is a local tool for managing evidence in audit and compliance work.
The `evi` CLI maintains an evidence registry and checks whether its sources
still match a reviewed integrity lock. Users choose requirements and sources,
conduct and review analysis, and decide whether to use AI. EviDock does not
run AI, assess compliance, or manage analysis progress or results.

## Create a project

Build EviDock and initialize a separate project directory:

```sh
go build -o bin/evi ./cmd/evi
./bin/evi init --workspace ../my-audit
```

`evi init` creates `evidence.yaml` and a project `README.md` if missing. That
README is for project-specific instructions. The CLI does not create a Git
repository or prescribe a directory layout.

## Evidence workflow

Register real sources in the project's root `evidence.yaml`:

```yaml
evidence:
  - id: ev-001
    product: product-a
    requirement: update-integrity
    path: sources/update.go
```

Each entry identifies a source file. IDs must be unique and stable; paths are
relative to the project root. Use IDs to cite sources in your own analysis
records.

```sh
./bin/evi --workspace ../my-audit validate
./bin/evi --workspace ../my-audit lock
./bin/evi --workspace ../my-audit verify
```

`validate` checks declarations and references. After review, `lock` records a
baseline; `verify` checks it without changing the lock. Investigate unexpected
verification failures before updating the baseline. A lock detects changes; it
does not prove source authenticity, evidence quality, or compliance.

## Agent guides

The optional guides in [`skills/`](./skills/) cover:

- [`onboarding`](./skills/onboarding.md): EviDock, evidence, and CLI commands.
- [`project-guidance`](./skills/project-guidance.md): project layout and
  working practices.

They are not loaded automatically or copied into projects; provide a guide to
your agent when needed. Project organization, analysis methods, and report
formats are up to you.

## Tests

Run `go test -race -cover ./...` locally. GitHub Actions runs tests, formatting
checks, static analysis, and a build on pushes to `main` and pull requests.
Each successful run shows coverage in its summary and provides downloadable
text and HTML coverage reports.

## License

The code and supplied guides are licensed under [Apache-2.0](./LICENSE).
This does not change the license or rights for user materials or external
sources.
