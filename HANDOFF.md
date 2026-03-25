# Phase 3 Handoff — SvelteKit Frontend Foundation

## Status: In Progress

Phase 3 (SvelteKit frontend foundation) is **mostly complete** but not yet committed. The work is in uncommitted changes on `main`.

---

## What Is Done

### SvelteKit scaffold (`web/`)
- Created with `npx sv create` — Svelte 5, TypeScript, static adapter, mdsvex, paraglide (en/de), Tailwind v4 + DaisyUI, Vitest (unit + browser), ESLint, Prettier
- `web/svelte.config.js` — static adapter with `fallback: 'index.html'`, output to `../internal/web/build`
- `web/vite.config.ts` — proxy `/api → http://localhost:8080` for dev, `base: '/app/'` for production builds
- `web/src/routes/+layout.ts` — `export const ssr = false` (pure client-side SPA)

### API client layer (`web/src/lib/api/`)
- `transport.ts` — `createConnectTransport({ baseUrl: '/api' })`
- `services.ts` — v2-compatible service descriptors built via `serviceDesc()` from `@bufbuild/protobuf/codegenv2` (workaround for BSR `connectrpc/es` plugin generating v1-style `MethodKind` descriptors incompatible with `@connectrpc/connect` v2)
- `index.ts` — named exports for all 9 service clients (`sessionClient`, `committeeClient`, `meetingClient`, `attendeeClient`, `agendaClient`, `speakerClient`, `voteClient`, `adminClient`, `moderationClient`)

### Session store (`web/src/lib/stores/session.svelte.ts`)
- Svelte 5 runes-based store (`$state`, getters)
- `load()` — calls `sessionClient.getSession({})` on mount, populates state
- `update(bootstrap)` — used after login to hydrate from response
- `clear()` — used on logout
- Fields: `loaded`, `authenticated`, `isAdmin`, `actor`, `availableCommittees`, `locale`

### Routes (SPA pages)
- `web/src/routes/+layout.svelte` — app shell: navbar (committee links, admin link, locale switcher, logout), session bootstrap via `onMount(() => session.load())`
- `web/src/routes/+page.svelte` — redirect hub: unauthenticated → `/login`, admin with no committees → `/admin`, otherwise → `/{firstCommittee}`
- `web/src/routes/login/+page.svelte` — login form, calls `sessionClient.login()`, updates session store on success
- `web/src/routes/admin/+page.svelte` — admin dashboard stub with stats (total committees, total accounts) and nav links to `/admin/committees`, `/admin/accounts`
- `web/src/routes/[committee]/+page.svelte` — committee overview, fetches `committeeClient.getCommittee({ slug })`, shows "Meetings" card linking to `/{slug}/meetings`

### Go embed package (`internal/web/`)
- `handler.go` — `NewSPAHandler()` serves embedded FS, falls back to `index.html` for all non-file paths (SPA routing support)
- `build/` — contains the already-built SPA output (committed in git, from `npm run build`)

### Go server integration (`cmd/serve.go`)
- Mounts SPA at `/app/` in non-development environments: `appMux.Handle("/app/", http.StripPrefix("/app", webassets.NewSPAHandler()))`

### Taskfile additions
- `dev:web` — `cd web && npm run dev`
- `build:web` — `cd web && npm run build`
- `test:web`, `check:web`, `lint:web`, `format:web` — frontend quality tasks
- `generate:clients` — `cd web && npx buf generate --config ../buf.gen.yaml`
- `dev:full` — runs `dev` and `dev:web` in parallel

### buf.gen.yaml
- Updated to use `local: protoc-gen-es` instead of `remote: buf.build/connectrpc/es`
- The `connectrpc/es` remote plugin is removed (generates broken v1-style `_connect.ts`)
- `@bufbuild/buf` installed as devDep in `web/` for npm-based `npx buf generate`
- `@connectrpc/connect`, `@connectrpc/connect-web`, `@bufbuild/protobuf`, `daisyui` added to `web/package.json`

---

## Open Issue: `buf generate` / `generate:clients` task

The `generate:clients` task (`cd web && npx buf generate --config ../buf.gen.yaml`) **does not work on Windows** because:
1. The `@bufbuild/buf` npm package downloads a platform-specific binary at install-time. Since `npm install` ran on Windows, the Windows binary was downloaded — not the Linux one.
2. When running `npx buf` from WSL Debian, the binary is missing (`downloaded-@bufbuild-buf-linux-x64-buf ENOENT`).
3. Running `buf` from the WinGet install can't find `protoc-gen-es` (not in Windows PATH).

The user wants to use `local: protoc-gen-es` (canonical approach per ConnectRPC docs). The fix is:
- Run `npm install` inside WSL/Linux to get the Linux binary downloaded
- Or: add `buf` (the CLI tool) to the nix flake devshell, so `nix develop` environments provide it alongside node

The current `_pb.ts` generated files (from `buf.build/bufbuild/es` remote plugin) are already correct for `@bufbuild/protobuf` v2 and are committed. The `_connect.ts` files are stale/unused (v1 style) and could be deleted — they are not imported anywhere (we use `services.ts` instead).

The `services.ts` workaround using `serviceDesc()` is the correct v2 approach regardless of which plugin generates `_pb.ts`. This pattern is documented in the `@bufbuild/protobuf` v2 migration guide.

---

## What Still Needs to Be Done

1. **Resolve `generate:clients` on Linux** — run `npm install` in WSL so `@bufbuild/buf` downloads the Linux binary, then test `task generate:clients` from WSL
2. **Delete stale `_connect.ts` files** — they're generated but not imported; optionally clean them up by removing the `connectrpc/es` plugin (already done in `buf.gen.yaml`) and deleting the existing `_connect.ts` files
3. **`npm run check`** — run `svelte-check` to verify no TypeScript errors in the frontend
4. **`npm run build`** — do a clean frontend build to confirm the SPA compiles and the output lands in `internal/web/build/`
5. **`go build ./...`** — confirm Go still compiles cleanly with the embedded build output
6. **`go test ./...`** — run all Go tests
7. **Commit** — stage and commit Phase 3 as a single commit

### Future phases (not part of Phase 3)
- Meeting list/detail pages (`/{committee}/meetings`, `/{committee}/meetings/{id}`)
- Speakers list, agenda, voting UI
- Admin committees/accounts management pages
- E2E tests for the new SPA routes

---

## Key File Locations

| File | Purpose |
|------|---------|
| `web/src/lib/api/transport.ts` | Connect transport (baseUrl `/api`) |
| `web/src/lib/api/services.ts` | v2 service descriptors via `serviceDesc()` |
| `web/src/lib/api/index.ts` | All named service clients |
| `web/src/lib/stores/session.svelte.ts` | Session state store (Svelte 5 runes) |
| `web/src/routes/+layout.svelte` | App shell + navbar |
| `web/src/routes/+layout.ts` | `ssr = false` |
| `web/src/routes/+page.svelte` | Root redirect hub |
| `web/src/routes/login/+page.svelte` | Login page |
| `web/src/routes/admin/+page.svelte` | Admin dashboard |
| `web/src/routes/[committee]/+page.svelte` | Committee overview |
| `internal/web/handler.go` | Go SPA handler with index.html fallback |
| `internal/web/build/` | Built SPA output (embedded into Go binary) |
| `cmd/serve.go` line ~209 | SPA mounted at `/app/` |
| `buf.gen.yaml` | Proto code generation config |
| `Taskfile.yaml` | All task definitions |

---

## Environment Notes

- **Windows** with Git Bash + WSL Debian. Nix devshell available via `wsl -d Debian /nix/var/nix/profiles/default/bin/nix develop .`
- Nix devshell does **not** include `node` or `buf` — only Go tools and task runner
- Use `MSYS_NO_PATHCONV=1` prefix when calling WSL from Git Bash to prevent path mangling
- Go tests: run with `go test ./...` from repo root
- Frontend dev: `task dev:full` runs Go backend + Vite dev server in parallel
