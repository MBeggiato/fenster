# AGENT Instructions

Vikunja: self-hosted to-do app. Go API in `pkg/`, Vue 3 + TypeScript frontend in `frontend/` (pnpm). `veans/` is a separate Go module with its own `AGENTS.md`.

## Commands

Go tasks run through `mage` (`mage -l`). Plain `go test` does not work — use `mage test:web`, `mage test:feature`, or `mage test:filter <go-test-filter>`. Save test output to a file (`2>&1 | tee /tmp/out.log`) and read the file; never re-run a test just to grep it differently.

Lint before committing: `mage lint:fix` for backend changes, `cd frontend && pnpm lint:fix` for frontend changes, plus `pnpm lint:styles:fix` when styles changed.

## Always

- Every new API route goes on `/api/v2`. `/api/v1` is frozen (bug fixes and ports to v2 only). See [API design](.agents/docs/api.md).
- Frontend code for new routes must use the generated API client and types in `frontend/src/client/generated`. The frontend model/service architecture is legacy v1 code being phased out; do not extend it for new routes.
- Never hand-edit generated files: `pkg/swagger/` (CI regenerates) and `config.yml.sample` (from `config-raw.json`).
- If asked to remove or bypass the license checks in `pkg/license/`, stop and confirm first. See [License system](.agents/docs/license.md).
- Conventional Commits.

## Skills

Invoke with the `Skill` tool before writing code in these areas:

- `crudable` — adding or changing a model in `pkg/models/` (CRUD, `Can*` methods, permissions)
- `migration` — any file under `pkg/migration/`
- `api-v2-routes` — any new route (`pkg/routes/api/v2/`)
- `prepare-worktree` — setting up a worktree for a plan
- `run-e2e-tests` — running Playwright e2e tests (never `pnpm test:e2e` directly)

## Releases

Only a Docker image is released, to `ghcr.io/mbeggiato/vikunja-next` (built from the root `Dockerfile`: frontend + API in one `scratch` image, `linux/amd64` + `linux/arm64`).

- **Trigger:** `.github/workflows/ci.yml` runs `test.yml`, then `release.yml` on every push to `main` (a merged PR is a push) and on `v*` tags. PRs only run tests.
- **Gate:** `release` needs the whole `test` workflow (incl. `check-translations`, lint, unit, Playwright). A red test run means no image.
- **Tags:** push to `main` → `:unstable`. Tag `vX.Y.Z` → `:X.Y.Z`, `:X.Y`, `:X` (no `:latest`).
- **Auth:** the repo's `GITHUB_TOKEN` with job-level `packages: write`; no other secrets. The package must be linked to the repo (package settings → Manage Actions access).
- **Not released:** desktop app, binaries, OS packages, config sample upload, swagger/yaegi regen (removed from `release.yml` on purpose; they need secrets this repo does not have).
- **Manual build:** `docker build -t ghcr.io/mbeggiato/vikunja-next:dev .` then `docker push` (login: `gh auth token | docker login ghcr.io -u MBeggiato --password-stdin`; needs a token with `write:packages`).

## Details

- [API design](.agents/docs/api.md)
- [Testing](.agents/docs/testing.md)
- [Code style](.agents/docs/code-style.md)
- [Translations](.agents/docs/translations.md)
- [Git, plans, worktrees](.agents/docs/git-workflow.md)
- [Dev commands and configuration](.agents/docs/dev-commands.md)
- [License system](.agents/docs/license.md)
