# Security audit report: Fenster (run-1)

## 1. Run scope and limits — this is a PARTIAL pass
- **Profile:** `quick`, no user-set budget. Source ref `699ee52965eba6e134ce93374e21df297af05399` (clean worktree). Scope: `pkg/`, `frontend/src/`, `veans/` (veans was seeded but not reviewed; see deferred units).
- **Execution:** sandboxed-source-and-local-only. **No target code was executed.** The Go module cache and `frontend/node_modules` are empty, so nothing could be built or tested offline. Every check is a source check, so no record can meet the "bounded observed local result" bar for `confirmed`.
- **Agents:** 4 reconnaissance, 11 hunters, 1 coverage critic, 7 candidate verifiers = 23. One quick-profile verifier per candidate served as Phase 3 and Phase 5; a fresh second pass was not run.
- **Prior runs:** none existed, so there is no carried coverage.
- **Coverage:** 20 ledger units: 7 covered, 4 candidate, 9 deferred. The 9 deferred units are the critic's accepted gaps (reason `quick_profile_final_critic`). There is no clean-coverage claim; a `standard` or `deep` run is needed to close them.
- **Process deviations (disclosed):** hunter and verifier prompts were assembled into files (verbatim method, class blocks, promotion procedure and schema) and agents were told to read them, rather than pasted inline. The hunter for `unit files` returned a JSON object with a stray `" uncovered"` key (treated as an empty `uncovered`). Hunter checks were summarized when transcribed into the ledger.

## 2. Posture summary
No confirmed vulnerability was found. Source review of authorization (`Can*` checks, link shares, API-token route scoping, admin gate, MCP), SQL/filter handling, migration/zip handling, outbound-request SSRF controls, CalDAV/feeds/mail escaping, rate limiting and frontend HTML rendering found controls that hold. Six source-grounded leads remain unresolved because they need execution or a product/deployment decision. The strongest is the WebSocket revocation gap.

## 3. Confirmed findings
None. (No record could meet the confirmed evidence bar in this environment.)

## 4. Needs validation (no severity assigned)
| Lead | Boundary | Blocker |
|---|---|---|
| WebSocket stays open after token expiry, logout, password change or disable (`websocket-auth-not-revalidated-after-revocation`) | session revocation → push events | no execution possible; verifier judged source trace complete (likely low) |
| OAuth2 authorize auto-approves any `vikunja-*`/loopback redirect, no consent or client registry (`oauth2-authorize-silent-consent-no-client-registry`) | logged-in user → code to local app | needs hostile local listener; owner decision on RFC 8252 trade-off |
| Unsplash empty-search global map race (`unsplash-empty-search-cache-concurrent-map-fatal`) | authenticated user → process crash | needs race run; provider default-off |
| Unsplash `doGet` caches error bodies with no TTL (`unsplash-doget-no-status-check-negative-cache-unbounded`) | authenticated user → memory growth / cache poisoning | Unsplash behaviour external; provider default-off; low impact |
| 50MP image decode with no concurrency bound (`imageutils-50mp-decode-unbounded-concurrency-oom`) | authenticated user → OOM | peak allocation and memory limit unobservable |
| CalDAV displayname from titles may be unescaped by the caldav-go fork (`caldav/go-caldav-displayname-xml-unescaped-title`) | project collaborator → CalDAV client XML | fork source unreadable offline |

Details, traces and plans are in `NEEDS-VALIDATION.md`. No plan involves probing a live target.

## 5. Rejected during validation
`webhooks.readall.target_url.exposed.to.read-only.members`: read-only members seeing webhook `target_url` is deliberate and tested (`pkg/webtests/huma_webhook_test.go:93`). Listed as hardening only.

## 6. Hardening notes (not findings)
- Metrics/pprof are unauthenticated if only one of username/password is set (`pkg/routes/metrics.go`). Fail closed.
- Testing token compared with `!=` (`pkg/routes/api/v1/testing.go`, `v2/testing.go`); use constant-time compare.
- `GuardLastAdmin` relies on serializable isolation; Postgres default is read committed (`pkg/user/user.go`).
- OIDC callback has no server-bound state/nonce/PKCE and takes a client-supplied `redirect_url`; `getOrCreateUser` accepts an unverified email claim for creation; cross-issuer duplicate emails are ambiguous.
- Access JWT has no `iss`/`aud`/`nbf` and no `sid`/user-status check for 10 minutes; refresh-token reuse rejects only the replay.
- `config-raw.json` ships the literal `<a-secret>` placeholder with no startup guard; OIDC id_token and TOTP secrets stored in plaintext.
- Rate-limit keys use raw IPv6 (a /64 can rotate); default `ipextractionmethod=direct` behind a proxy shares one budget; `ratelimit.enabled` defaults to false.
- `migration.DownloadFile*` have no size cap; Trello Authorization header sent with default redirect policy.
- `guardProxiedDials` skips the SSRF guard for proxy-matched dials (`pkg/utils/httpclient.go:81-104`).
- Dockerfile: `go install mage@latest` and `npm install -g corepack` are unpinned; release has no provenance/signing; `release.yml` pushes with no test gate.
- `static.go` renders config into JS with `text/template`; `Content-Disposition` in `user_export.go` is hand-built; task `CoverImageAttachmentID` accepted on create without a task check.
- Positive patterns: SSRF dialer control on all user-influenced fetches; zip traversal and byte budgets on imports; PKCE S256 + hashed single-use codes; hashed refresh/reset tokens with rotation; admin gate re-reads `is_admin`; exact route matching for API tokens; DOMPurify on the only `v-html`; SHA-pinned actions and `pull_request` (not `_target`).

## 7. Coverage detail
Covered (no candidates): supply chain/CI/plugin loading, frontend rendering and navigation, SQL/filter/search injection, rate limits and pre-auth work, admin/testing/metrics/MCP, migrations and dump/restore, CRUD access control. Candidate-bearing: files/uploads, CalDAV/feeds/mail, outbound requests, auth/sessions. **Deferred (not reviewed):** `pkg/richtext` and mention resolution; link-share password/renew flows; user deletion/export/invite/bot/notification lifecycle; S3 storage/repair/dump; avatar providers and LDAP/OIDC group sync; API-token creation/scope expansion; `veans/`; service worker/browser storage; plugin route wiring. Final critic result: 9 gaps accepted, no reassignments, `stop: false`.
