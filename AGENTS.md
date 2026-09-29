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

Only a Docker image is released, to `ghcr.io/mbeggiato/vikunja-next` (built from the root `Dockerfile`: frontend + API in one `scratch` image). There are exactly two workflows in `.github/workflows/`:

- **`release.yml`:** runs on every push to `main` (a merged PR is a push). Builds `linux/amd64` + `linux/arm64` and pushes `:latest` and `:<8-char sha>`. No tests run first.
- **`preview.yml`:** manual only (Actions → Preview image → Run workflow, choose the branch in "Use workflow from"; the branch must contain the file). Builds `linux/amd64` and pushes `:preview-<branch>` (slashes become `-`).
- **Auth:** the repo's `GITHUB_TOKEN` with job-level `packages: write`; no other secrets. The package must be linked to the repo (package settings → Manage Actions access) and Actions must be enabled.
- **No versions/tags:** there are no git tags, so the version baked into the binary is the short commit SHA.
- **No CI tests:** PRs run no checks. Run lint, unit tests and the e2e suite locally before merging.
- **Manual local build:** `docker build -t ghcr.io/mbeggiato/vikunja-next:dev .` then `docker push` (login: `gh auth token | docker login ghcr.io -u MBeggiato --password-stdin`; needs a token with `write:packages`).

## Details

- [API design](.agents/docs/api.md)
- [Testing](.agents/docs/testing.md)
- [Code style](.agents/docs/code-style.md)
- [Translations](.agents/docs/translations.md)
- [Git, plans, worktrees](.agents/docs/git-workflow.md)
- [Dev commands and configuration](.agents/docs/dev-commands.md)
- [License system](.agents/docs/license.md)
