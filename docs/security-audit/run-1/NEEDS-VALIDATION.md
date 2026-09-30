# Needs-validation leads (run-1)
Prioritized leads, no severity. No step here involves probing a live or shared target; local steps assume dependencies are already present and run in an isolated sandbox with dummy data.

## WebSocket connection outlives token expiry, logout, password change and account disable
`websocket-auth-not-revalidated-after-revocation`

The WebSocket upgrade is unauthenticated; the first client message carries a JWT that is parsed once (signature, exp, type==user), and the connection is then registered in the hub under the user ID with no expiry timer, no sid/session lookup and no user-status check. It is not closed when the session is deleted (logout, password change) or the account is disabled. The hub keeps delivering notification.created and timer.* events to that user's subscribed sockets until the client disconnects. A holder of a once-valid access token (for example a stolen one) can keep a read-only stream of the victim's notification and time-entry events after the 10-minute token lifetime and after revocation. A verifier judged the source trace complete (likelihood low, impact low, overall low) but no bounded local observation is possible because the module cache is empty, so the record is held at needs_validation rather than confirmed.

**Claimed root cause:** auth.GetUserIDFromToken (auth.go:305-329) validates only signature, exp and type/id and returns only the user ID; Connection.handleAuth (connection.go:168-199) calls it once and registers the connection. No component keeps the token expiry or sid, no periodic revalidation runs, and nothing closes connections on logout, session deletion, password change or user disable; the only Unregister is the ReadLoop defer on socket close or error.

**Trace**
1. [entrypoint] `pkg/websocket/handler.go:44` UpgradeHandler — Unauthenticated WebSocket upgrade; ReadLoop/WriteLoop start with context.Background(), so the connection has no expiry tied to any credential.
2. [propagation] `pkg/websocket/connection.go:174` Connection.handleAuth — Calls auth.GetUserIDFromToken once, sets userID and authenticated, then hub.Register (line 188); no later revalidation.
3. [propagation] `pkg/modules/auth/auth.go:305` GetUserIDFromToken — jwt.Parse checks signature and exp, then type==user and id; does not read sid, query the session store or user status, and retains no expiry.
4. [sink] `pkg/websocket/hub.go:65` Hub.PublishForUser — Delivers events to every registered connection of the user for as long as the socket stays registered.

**Evidence**
- `pkg/websocket/connection.go:99` The only Unregister path is the ReadLoop defer; no other Unregister call in non-test code.
- `pkg/websocket/connection.go:39` Connection stores only userID, authenticated flag, subscriptions and send channel; no token expiry or sid.
- `pkg/modules/auth/auth.go:200` User JWTs carry a sid claim bound to a session at issue time; GetUserIDFromToken never reads it.
- `pkg/websocket/listener.go:77` Notification payloads are pushed to the user's sockets by user ID.
- `pkg/websocket/listener.go:110` Time-entry events are also published to the hub by user ID.

**Blockers**
- No target build or test execution is possible (empty Go module cache), so the missing bounded local observation cannot be produced in this run.
- Requires a valid, unexpired user access JWT for the victim at the time of the WebSocket auth message.

**Local plan:** With dependencies present locally, in a websocket test using the existing test hub inside the isolated sandbox: mint a JWT with NewUserJWTAuthtoken, connect, send the auth action, then delete the session row and set the user status to disabled, publish a notification.created event for that user through the listener or hub.PublishForUser and assert whether the socket still receives it. Expected: still delivered.

**Owner-observed deployment check:** Not required. Suggested fix: record exp and sid at auth time, close the connection at exp or require re-auth, and close user/session connections on session deletion, password change or user disable (or re-check session and user status on each ping tick).

## OAuth2 authorize endpoint silently issues codes for any vikunja-* or loopback redirect (no consent, no client registry)
`oauth2-authorize-silent-consent-no-client-registry`

POST /oauth/authorize accepts any client_id string and any redirect_uri whose scheme starts with vikunja- or is http on a loopback host. The SPA route calls it on mount with the logged-in user's JWT and navigates to the redirect URI with the code, with no consent screen. The PKCE challenge is chosen by the requester, so PKCE does not protect against a hostile requester. A hostile local app that owns a matching custom-scheme handler or loopback listener, and can get a logged-in victim's browser to open /oauth/authorize, can redeem the code for a full-scope session. Source fully shows the missing controls; exploitability depends on the victim environment, and silent approval for native app schemes may be an accepted RFC 8252 trade-off.

**Claimed root cause:** There is no OAuth client registry, no binding of client_id to redirect URIs and no user consent step. ValidateRedirectURI trusts a scheme prefix or loopback host, and token exchange only checks that client_id, redirect_uri and the PKCE verifier match the values the requester supplied at authorize time; sessions minted this way have normal-login scope.

**Trace**
1. [entrypoint] `frontend/src/views/user/OAuthAuthorize.vue:55` authorize() / onMounted — On mount (line 82) posts the route query params to oauth/authorize with the user JWT and, with no approval UI, sets window.location.href to redirect_uri?code=...&state=... (lines 66-75).
2. [propagation] `pkg/modules/auth/oauth2server/authorize.go:95` Authorize — Only response_type==code, ValidateRedirectURI (95) and PKCE S256 presence (100) are checked; client_id is never validated and CreateOAuthCode runs at 113.
3. [propagation] `pkg/modules/auth/oauth2server/client.go:38` ValidateRedirectURI — Accepts any scheme with the vikunja- prefix, or http to localhost or a loopback IP (lines 42-49); no per-client allowlist.
4. [sink] `pkg/modules/auth/oauth2server/token.go:145` exchangeAuthorizationCode — Code consumption checks only the requester-supplied client_id (127), redirect_uri (132) and PKCE verifier (137), then CreateSession (145) and NewUserJWTAuthtoken (158) mint a full session with refresh token.

**Evidence**
- `pkg/modules/auth/oauth2server/authorize.go:95` redirect_uri validation is the only ownership check; client_id passes through unvalidated to CreateOAuthCode at line 113.
- `pkg/modules/auth/oauth2server/client.go:38` Deliberate scheme-prefix or loopback allowlist in the style of RFC 8252, not per-client registration.
- `frontend/src/views/user/OAuthAuthorize.vue:82` Authorization runs automatically on mount; the template has no consent or approve button.
- `pkg/modules/auth/oauth2server/token.go:127` client_id and redirect_uri are compared only against requester-supplied values.

**Blockers**
- Exploitation needs a hostile app or listener on the victim device or host plus a victim with a live Fenster browser session opening a crafted link; not observable from source.
- The Go module cache and node_modules are empty, so the API and SPA cannot be built or run offline.
- Whether silent consent for first-party native-app schemes is an accepted product trade-off is an owner decision that determines severity.

**Local plan:** With a local dev build and dummy user, start a throwaway loopback listener, generate a PKCE verifier and S256 challenge, open /oauth/authorize?response_type=code&client_id=anything&redirect_uri=http://127.0.0.1:PORT/cb&code_challenge=<c>&code_challenge_method=S256&state=x in the logged-in browser, confirm the code arrives with no prompt, then POST /oauth/token with the verifier and confirm access and refresh tokens; repeat with redirect_uri=vikunja-evil://cb.

**Owner-observed deployment check:** Owner-observed and non-destructive: check whether the shipped native clients depend on silent approval and whether a consent screen, a client registry with exact redirect matching, or a first-party-only scheme allowlist would break them.

## CalDAV displayname built from project/task titles may be emitted into multistatus XML unescaped
`caldav/go-caldav-displayname-xml-unescaped-title`

User-controlled Project.Title and Task.Title are copied verbatim into caldav Resource.Name, which the go-vikunja/caldav-go fork renders into PROPFIND/REPORT multistatus XML as D:displayname. This repo does not escape the value; a repo comment at listStorageProvider.go:326 says the fork writes hrefs into XML unescaped, so displayname may be handled the same way. If so, a title with XML markup could break or inject into the XML that CalDAV clients of a user the project is shared with parse. The fork source was unreadable, so exploitability is undetermined.

**Claimed root cause:** Resource.Name is set from Project.Title/Task.Title in listStorageProvider.go with no XML escaping, relying on the replaced dependency github.com/go-vikunja/caldav-go (go.mod:229) to escape it when rendering multistatus; that code was not readable.

**Trace**
1. [entrypoint] `pkg/routes/caldav/listStorageProvider.go:1202` addTaskResource — Task.Title (settable by any user with write access to the project) is assigned to Resource.Name.
2. [propagation] `pkg/routes/caldav/listStorageProvider.go:105` GetResources / project collection resource — Project.Title is assigned to r.Name (same pattern at lines 164, 302, 314).
3. [sink] `go.mod:229` replace github.com/samedi/caldav-go => github.com/go-vikunja/caldav-go — The external fork renders Resource.Name into D:displayname; whether it escapes is unverified. The sink is in a dependency.

**Evidence**
- `pkg/routes/caldav/listStorageProvider.go:105` r.Name = vcls.project.Title with no escaping.
- `pkg/routes/caldav/listStorageProvider.go:302` r.Name = vcls.project.Tasks[i].Title with no escaping.
- `pkg/routes/caldav/listStorageProvider.go:325` Comment: caldav-go writes hrefs into XML unescaped; only the href is confirmed unescaped.
- `go.mod:229` replace directive pointing at the go-vikunja/caldav-go fork; module cache empty so its rendering code could not be read.

**Blockers**
- The go-vikunja/caldav-go fork source at the pinned version is not present locally (module cache empty), so its displayname rendering cannot be read.
- Target code cannot be built or run here, so a PROPFIND cannot be observed.
- The fork escaping behaviour is decisive: if it uses encoding/xml escaping the candidate is refuted.

**Local plan:** With the pinned caldav-go source already present in a local module cache or vendor directory, read its displayname rendering and check whether Resource.Name passes through xml.EscapeText or html.EscapeString. Then, in the isolated sandbox with a dummy sqlite DB, create a project titled x</D:displayname><D:foo>1</D:foo> and a title a&b<c, send an authenticated PROPFIND with Depth 1 to /dav/projects/<id> and check the raw body is well-formed XML with the title escaped. Reject if escaped.

**Owner-observed deployment check:** Not required for the decision; the result is deterministic from fork source plus a local run. If unescaped, an owner may confirm on a test instance that a CalDAV client misparses a project titled with markup.

## Unsynchronized global map in Unsplash empty-search cache can trigger fatal concurrent map access
`unsplash-empty-search-cache-concurrent-map-fatal`

Provider.Search caches the empty-query result in the package-level pointer emptySearchResult, whose images map is keyed by the client-supplied page number p (any int64). The map is read at unsplash.go:169-176 and lazily created and written at 205-212 with no mutex. Concurrent authenticated requests with an empty s and different or uncached p can hit a concurrent map read/write, which in Go is an unrecoverable fatal error that echo Recover cannot catch, so the process would exit. Reachable only when backgrounds.providers.unsplash.enabled is true (default false). Whether the runtime detector fires in practice is probabilistic and was not observed.

**Claimed root cause:** emptySearchResult (unsplash.go:102) is a shared mutable package global holding a map keyed by attacker-controlled page number, read and written from concurrent HTTP handlers without synchronization.

**Trace**
1. [entrypoint] `pkg/routes/routes.go:904` GET /api/v1/backgrounds/unsplash/search — Authenticated route registered only inside if config.BackgroundsUnsplashEnabled.GetBool() (line 898).
2. [propagation] `pkg/modules/background/handler/background.go:81` SearchBackgrounds — Reads s and p (p parsed as int64, no range check) and calls Search on a fresh per-request Provider; no lock.
3. [propagation] `pkg/modules/background/unsplash/unsplash.go:169` Provider.Search (empty search) — Unlocked read of emptySearchResult.images[page].
4. [sink] `pkg/modules/background/unsplash/unsplash.go:212` Provider.Search — Unlocked lazy init (205-208) and write emptySearchResult.images[page] = result.

**Evidence**
- `pkg/modules/background/unsplash/unsplash.go:102` var emptySearchResult *initialCollection is a package-level mutable global; no sync.Mutex/RWMutex in the file.
- `pkg/modules/background/unsplash/unsplash.go:106` initialCollection.images is a plain map[int64][]*background.Image.
- `pkg/routes/routes.go:898` Unsplash routes registered only when BackgroundsUnsplashEnabled is true (default false).

**Blockers**
- No execution possible (empty Go module cache): the race detector or a stress run cannot be performed, so the fatal error cannot be observed.
- Deployment fact: backgrounds.providers.unsplash.enabled must be true with an Unsplash access key configured; default is false.
- Process-level impact depends on the best-effort runtime concurrent-map detector firing.

**Local plan:** With dependencies already present locally, add a test in pkg/modules/background/unsplash that stubs the upstream call (doGet) or points it at a local httptest server, then calls Provider.Search(nil, "", n) from many goroutines with distinct n under the race detector, inside the isolated sandbox. Expect a data race report and possibly "fatal error: concurrent map read and map write".

**Owner-observed deployment check:** Owner checks whether backgrounds.providers.unsplash.enabled is true. If false the issue is unreachable. Suggested fix: guard emptySearchResult with a sync.RWMutex (or use the keyvalue cache with a TTL) and bound or validate page.

## 50MP decode cap with no concurrency bound lets small uploads drive large per-request allocations
`imageutils-50mp-decode-unbounded-concurrency-oom`

imageutils.ValidateConfig rejects images over 50M pixels (image.go:25,28-36), a per-image cap only. Avatar upload, background upload and attachment preview each buffer the whole file and fully decode any image under that cap. No semaphore or per-user limit bounds concurrent decodes. A highly compressible flat PNG of about 7000x7000 is tens of KB on the wire but decodes to hundreds of MB. An authenticated user (registration on by default) sending several such requests in parallel could raise aggregate memory to cap x concurrency and OOM the single-process container. Outcome depends on real peak allocation per format and the container memory limit, neither observable here. The cap is a deliberate mitigation for GHSA-4vh2-39rq-rq8j, so this is generic resource-exhaustion hardening unless shown otherwise.

**Claimed root cause:** MaxPixels bounds a single image only. The decode paths (avatar upload, background upload, attachment preview) have no concurrency bound or global memory budget, so aggregate memory is the per-image cap times concurrency, about 1000x the upload size.

**Trace**
1. [entrypoint] `pkg/routes/routes.go:592` PUT /user/settings/avatar/upload (also PUT /projects/:project/backgrounds/upload at routes.go:896 and attachment preview) — Authenticated user supplies image bytes; only request-level bound is the global BodyLimit (routes.go:206) on upload bytes.
2. [propagation] `pkg/modules/imageutils/image.go:28` ValidateConfig / ValidateReader (MaxPixels at line 25) — Header check accepts up to 50,000,000 pixels per image with no global accounting.
3. [sink] `pkg/modules/avatar/upload/upload.go:155` StoreAvatarFile — io.ReadAll (148), ValidateReader (152), image.Decode (155), imaging.Fit (160) with no concurrency limit; same pattern at background.go:175,277 and task_attachment.go:340.

**Evidence**
- `pkg/modules/imageutils/image.go:25` MaxPixels = 50_000_000 is the only decode-size control and applies per image.
- `pkg/modules/avatar/upload/upload.go:148` io.ReadAll then full decode and resize without any semaphore.
- `pkg/modules/background/handler/background.go:277` imaging.Decode after ValidateReader; no concurrency bound.
- `pkg/models/task_attachment.go:340` Preview path: ReadAll at 325, ValidateConfig at 336, then image.Decode; the cache only dedupes repeated requests for the same size.
- `pkg/routes/routes.go:206` Global BodyLimit bounds upload bytes only.

**Blockers**
- Peak allocation per format at 50MP including the imaging.Fit resize cannot be measured because the module cache is empty and target code cannot be built or run.
- Container memory limit, reverse-proxy limits and ratelimit.enabled are deployment facts.

**Local plan:** With dependencies present locally, inside the isolated sandbox with a low memory limit, build a flat 7000x7000 PNG and call StoreAvatarFile and attachment GetPreview against a dummy user; record peak RSS for one call, then 4 and 8 concurrent calls.

**Owner-observed deployment check:** Owner compares the measured per-request peak times plausible concurrency against the container memory limit and checks ratelimit/proxy configuration. Suggested fix: a weighted semaphore around decode, or a lower MaxPixels for avatars and previews.

## Unsplash photo lookup caches empty results for arbitrary IDs without bound (memory growth and cache poisoning)
`unsplash-doget-no-status-check-negative-cache-unbounded`

When the Unsplash provider is enabled, any authenticated user can request the image or thumb proxy with an arbitrary ID. doGet never checks the upstream HTTP status and decodes a JSON error body (404, or 429) into a zero-value Photo with nil error. keyvalue.RememberValue then stores it with Put and no TTL, so each distinct attacker-chosen ID leaves a permanent entry (unbounded in-memory map, or Redis if configured), and a transient 429 for a legitimate ID is cached permanently. The quota-burn framing is weak because an attacker can already spend one upstream call per unique ID. Impact is low-severity authenticated availability at most. The path-injection aspect is not established: the host stays fixed to api.unsplash.com.

**Claimed root cause:** doGet (unsplash.go:104-127) returns the JSON decode result without checking resp.StatusCode; getUnsplashPhotoInfoByID passes that to keyvalue.RememberValue, which caches with no TTL under a key derived from the user-controlled :image param, with no size bound.

**Trace**
1. [entrypoint] `pkg/modules/background/unsplash/proxy.go:137` ProxyUnsplashThumb (ProxyUnsplashImage at :121) — c.Param("image") passed unvalidated into FetchUnsplashThumbByID; routes at routes.go:906-907 behind JWT auth, only when Unsplash is enabled (default false).
2. [propagation] `pkg/modules/background/unsplash/unsplash.go:138` getUnsplashPhotoInfoByID — keyvalue.RememberValue(cachePrefix+photoID, fn); fn calls doGet("photos/"+photoID).
3. [propagation] `pkg/modules/keyvalue/keyvalue.go:150` RememberValue — On nil error, Put(key, val) stores with no TTL; memory backend has no size cap.
4. [sink] `pkg/modules/background/unsplash/unsplash.go:123` doGet — Decodes any response body without checking resp.StatusCode, so 404/429 error JSON becomes an empty Photo with nil error.

**Evidence**
- `pkg/modules/background/unsplash/unsplash.go:116` SSRF-safe client Do followed by Decode with no status check; host fixed to unsplashAPIURL.
- `pkg/modules/keyvalue/keyvalue.go:150` RememberValue caches with Put (no TTL), unlike RememberFor (:172-191).
- `pkg/modules/keyvalue/memory/memory.go:58` Memory Put clears any expiry and stores with no size cap.
- `pkg/routes/routes.go:898` Unsplash routes only registered when BackgroundsUnsplashEnabled is true (default false).

**Blockers**
- Unsplash actual response body and status for unknown IDs and rate limiting are external facts, so whether the body decodes to an empty Photo with nil error is not observable offline.
- The Go module cache is empty, so cache growth cannot be observed.
- Deployment: the provider is disabled by default; cache backend and quota tier are operator choices.

**Local plan:** With dependencies present locally, in a unit test point unsplashAPIURL at a loopback httptest server returning 404 with {"errors":["Couldn't find Photo"]}. Call getUnsplashPhotoInfoByID for N distinct IDs and a repeated ID; confirm nil error, empty Photo, one upstream hit per unique ID and N new keyvalue entries; then return 200 for a cached ID and confirm the stale empty value is still served.

**Owner-observed deployment check:** Owner confirms whether backgrounds.providers.unsplash.enabled is true and which keyvalue backend is used; on a test instance observe memory or key count after a few requests with random IDs. Suggested fix: return an error from doGet when status >= 400, validate the ID (^[A-Za-z0-9_-]{1,32}$), and use a TTL cache.

