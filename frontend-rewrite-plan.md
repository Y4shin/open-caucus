# Frontend Rewrite Plan

## Goal

Replace the current Go-rendered HTMX/Templ frontend with a SvelteKit single-page application that is built as static assets, embedded into the Go binary, and backed by a typed API plus live-update streams.

This is a deliberate architectural rewrite, not an incremental migration. The goal is to simplify long-term frontend development, make rich client-side state easier to manage, and establish a clearer contract between UI and backend.

## Status Snapshot

Last updated: 2026-03-28

- Current phase: `Phase 6 - Legacy Removal`
- Phase 0 status: `Completed`
- Phase 1 status: `Completed`
- Phase 2 status: `Completed`
- Phase 3 status: `Completed`
- Phase 4 status: `Completed`
- Phase 5 status: `Completed`
- Phase 6 status: `In progress`
- Phase 6a (decoupling): `Completed` — serve.go fully decoupled from internal/handlers
- Phase 6b (Connect streaming): `In progress` — replacing raw SSE + refetch with typed streaming RPC
- Rewrite strategy: `Big-bang rewrite approved`

## Current Implementation Baseline

The repository is still primarily on the original architecture at the time of this update, but the first rewrite scaffolding now exists alongside it.

### Frontend and Delivery

- The UI is currently server-rendered in Go using `Templ`.
- Dynamic interactions are currently implemented with `HTMX`.
- CSS is currently built with Tailwind/DaisyUI via the Node toolchain in `package.json`.
- The current `Dockerfile` already contains a Node-based build step, but only for CSS generation.

### Backend Shape

- HTTP routes are currently defined in `routes.yaml`.
- A custom route generator in `tools/routing/` produces typed router code and URL builders.
- Handlers currently return template input models and server-rendered HTML responses.
- The backend already owns authentication, authorization, domain logic, persistence, and file serving.
- A minimal `Connect + Protobuf` transport slice now exists under `/api` for session bootstrap, login, and logout.

### Realtime

- Realtime behavior already exists via SSE.
- Current SSE streams send server-rendered HTML fragments that HTMX swaps into the page.
- There is already an in-memory event broker that can be reused conceptually in the new architecture.

### Persistence and Auth

- The data layer is currently Go + SQLite + SQLC.
- The app already uses cookie/session-based authentication.
- Authorization logic already exists in middleware and handler flows.

### Testing and Tooling

- Playwright E2E tests already exist and cover a broad set of user-facing workflows.
- The project already uses Task for common development commands.
- The flake dev shell now includes `buf` for protobuf linting and generation.
- Initial API-level tests now exist for the new session transport slice.
- A SvelteKit frontend workspace now exists under `web/`.
- Initial `Connect + Protobuf` contract source now exists under `proto/` for the first Phase 1 slice.
- `buf.yaml` and `buf.gen.yaml` now exist.
- Initial generated Go stubs now exist under `gen/go/`.
- Generated TypeScript clients now exist under `web/src/lib/gen/` and are consumed directly by the SPA.
- The SPA now covers session bootstrap, committee home/overview, meeting join and attendee login, live meeting state, moderation speaker controls, typed agenda-management workflows, vote draft editing, open and secret ballot handling, admin/account management, docs/public verification pages, and attachment/file-serving flows.

## Target Architecture

### Frontend

- Build a new frontend in `SvelteKit`.
- Use the static adapter and run the app fully client-side after load.
- Serve the built assets from the Go binary using `embed`.
- Keep the frontend and API on the same origin to simplify cookies, CSRF, and deployment.
- Use generated TypeScript clients for backend communication instead of handwritten request shapes.

### Backend

- Keep Go as the single backend binary and system of record.
- Move backend responsibilities toward:
  - authentication and session management
  - authorization
  - domain logic
  - persistence
  - file/blob serving
  - realtime event publishing
- Retire server-side HTML rendering and HTMX-specific response patterns once the new frontend is complete.

### Contract Layer

- Introduce a typed backend contract as the new integration boundary.
- Use `Connect + Protobuf` for request/response APIs.
- Keep `SSE` for live updates unless the rewrite reveals a real need for bidirectional streaming.
- Treat the contract as the primary source of truth for frontend/backend integration.
- Keep plain HTTP endpoints where browser-native or streaming semantics are a better fit:
  - file uploads
  - file downloads
  - inline document streaming
  - OAuth redirects/callbacks
  - SSE streams

## Recommended Technology Decisions

### API Style

Use a service-oriented API instead of forcing the domain into strict CRUD REST.

This domain contains many explicit actions and state transitions:

- opening and closing votes
- counting ballots
- activating agenda points
- starting and ending speakers
- toggling attendee flags
- joining meetings and managing live state

These workflows map naturally to typed commands and query methods.

### Auth Model

- Keep cookie-based auth.
- Keep the frontend on the same origin as the API.
- Prefer `HttpOnly` secure session cookies over storing tokens in browser storage.
- Add explicit session endpoints so the SPA can bootstrap current-user and current-session state on startup.

Why this direction:

- same-origin deployment makes cookie auth the simplest fit
- the existing backend session model already aligns with this approach
- `HttpOnly` cookies reduce exposure compared to bearer tokens stored in browser storage
- the app does not need the extra token and refresh-token machinery that a browser-stored bearer model would introduce

### Live Updates

- Preserve the existing event-driven model.
- Replace HTML-over-SSE with JSON event payloads.
- Use SSE events primarily as invalidation or refresh signals for meeting-scoped state.
- Let the frontend refetch the relevant read model after receiving an event.

### Frontend Data Model

Split backend endpoints into:

- screen-oriented read models for complex pages like moderation, meeting management, and live meeting views
- explicit command endpoints for state-changing actions

This avoids excessive client-side orchestration and keeps the SPA simpler.

## Proposed Repository Shape

The rewrite should move toward a repo layout that clearly separates:

- handwritten Go application code
- contract source files
- generated code
- frontend application code
- embedded build artifacts

Concrete target structure:

```text
.
├── cmd/
│   ├── root.go
│   └── serve.go
├── internal/
│   ├── api/
│   │   ├── connect/         # Connect transport handlers, interceptors, error mapping
│   │   ├── http/            # plain HTTP endpoints that remain outside Connect (SSE, uploads, downloads, static)
│   │   └── auth/            # transport-facing auth/session helpers
│   ├── auth/                # core auth/authorization logic
│   ├── services/            # application services implementing contract operations
│   ├── repository/          # persistence interfaces + sqlite implementation
│   ├── realtime/            # broker, event fanout, event payload builders
│   ├── session/             # session storage and cookie/session primitives
│   ├── storage/             # blob/file storage
│   ├── locale/              # locale loading and selection logic retained if reused
│   ├── web/
│   │   ├── embed.go         # embeds frontend build output
│   │   └── static.go        # serves SPA assets / index fallback
│   └── docs/                # retain if docs service stays backend-owned
├── proto/
│   └── conference/
│       ├── common/v1/
│       ├── session/v1/
│       ├── committees/v1/
│       ├── meetings/v1/
│       ├── moderation/v1/
│       ├── agenda/v1/
│       ├── attendees/v1/
│       ├── speakers/v1/
│       ├── votes/v1/
│       ├── admin/v1/
│       ├── realtime/v1/
│       └── docs/v1/
├── gen/
│   ├── go/                  # generated Go protobuf/connect code
│   └── ts/                  # optional generated TS output if not emitted directly into web/src/lib/gen
├── web/
│   ├── src/
│   │   ├── lib/
│   │   │   ├── api/         # client wrappers around generated clients
│   │   │   ├── components/  # shared UI components
│   │   │   ├── features/    # domain-oriented frontend modules
│   │   │   ├── gen/         # generated TS/protobuf/connect code if kept in-repo here
│   │   │   ├── stores/      # app/session/query state
│   │   │   └── utils/
│   │   ├── routes/          # SvelteKit routes mirroring current UI
│   │   ├── app.html
│   │   └── app.css
│   ├── static/
│   ├── tests/
│   ├── package.json
│   ├── svelte.config.js
│   ├── vite.config.ts
│   └── tsconfig.json
├── e2e/                     # end-to-end tests (may continue to use Playwright)
├── doc/
├── Taskfile.yaml
├── Dockerfile
├── buf.yaml
├── buf.gen.yaml
├── package.json             # may be removed later if root-level Node tooling becomes unnecessary
└── package-lock.json        # same as above
```

Ownership notes:

- `internal/services/` should become the main home of app-level business operations that are currently embedded in HTML handlers.
- `internal/api/connect/` should be thin transport glue, not the main home of business logic.
- `internal/api/http/` should hold the endpoints that do not map naturally to Connect, especially:
  - SSE streams
  - file upload endpoints if multipart handling stays plain HTTP
  - file/blob download endpoints
  - auth redirects like OAuth start/callback where plain HTTP semantics are simpler
- `proto/` should contain only source-of-truth contract definitions.
- `gen/` should contain generated Go code and optionally generated TS code if we do not emit that directly into `web/src/lib/gen`.
- `web/` should be a self-contained frontend workspace with its own build and test tooling.
- `internal/web/` should only serve embedded assets and should not contain handwritten frontend source code.

Preferred generation layout:

- Go generated code: `gen/go/...`
- TypeScript generated code:
  - preferred: `web/src/lib/gen/...` for convenient frontend imports
  - acceptable alternative: `gen/ts/...` with a frontend import alias if that proves easier to manage

Legacy areas expected to become transitional and later removable:

- `internal/templates/`
- `routes.yaml`
- `tools/routing/`
- generated route and templ artifacts tied to SSR/HTMX

Status:

- this is the proposed target repository shape for Phase 0
- exact directory names may still shift slightly before implementation starts
- the main goal is to freeze the separation of concerns, not every subfolder name

## Implementation Phases

### Phase 0: Design and Decision Freeze

Define the target before writing production rewrite code.

Current status:

- This phase has started.
- The rewrite direction is agreed in principle.
- Several technical decisions are provisionally aligned but not yet fully frozen.

Deliverables:

- final decision on API contract style
- final decision on live-update mechanism
- top-level frontend route map
- initial protobuf package layout
- initial repository structure for the new frontend and generated code
- success criteria for feature parity

Key decisions to lock:

- route naming and versioning strategy
- how file uploads/downloads will work
- how locale handling will work in the SPA
- whether existing session storage can be reused as-is

Decision tracker:

| Topic | Status | Current position |
|---|---|---|
| Rewrite scope | Confirmed | Big-bang rewrite rather than incremental migration |
| Frontend architecture | Confirmed | `SvelteKit` SPA built with the static adapter and embedded into the Go binary |
| Deployment topology | Confirmed | Same-origin frontend and backend served from the Go binary |
| Auth model | Confirmed | Keep same-origin cookie-based auth with `HttpOnly` session cookies; do not use bearer tokens stored in browser storage |
| Main API contract | Confirmed | Use `Connect + Protobuf` for typed request/response APIs, with plain HTTP reserved for files, SSE, and OAuth/browser-native flows |
| Live updates | Confirmed | Keep meeting-scoped `SSE`, but switch from server-rendered HTML fragments to JSON event payloads and invalidation/version signals |
| Backend role | Confirmed | Go remains the system of record and serves API, static assets, auth, files, and realtime |
| Frontend data style | Provisional | Use screen-oriented read models plus explicit command-style operations |
| SPA route map | Confirmed | The SvelteKit app should mirror the current user-facing UI routes rather than inventing a new navigation structure |
| Repository structure | Confirmed | Separate handwritten Go code, proto contracts, generated code, frontend workspace, and embedded asset serving |
| Locale strategy | Confirmed | Preserve current locale resolution order and optional URL-prefix behavior, but move UI translation ownership to the frontend |
| File handling | Confirmed | Use plain HTTP endpoints for binary upload/download/inline streaming, with the typed contract handling metadata and attachment/workflow commands |
| Session reuse | Confirmed | Reuse the underlying server-side cookie session model and store, while refactoring middleware/transport behavior for API and SPA needs |
| Testing strategy | Confirmed | Treat the current Playwright E2E suite as the primary acceptance contract, keep it passing with minimal changes, and derive API integration coverage from each E2E workflow |
| Route/versioning strategy | Provisional | SPA routes mirror the current UI; API/contract versioning still needs a final scheme |

Initial SPA route map:

The SPA should preserve the current screen-level URLs and user journey wherever possible. This keeps existing mental models, bookmarks, shared links, and E2E workflow expectations stable during the rewrite.

The following routes are proposed as SvelteKit screen routes:

- `/`
  - user login page
- `/admin/login`
  - admin login page
- `/home`
  - authenticated user home / committee overview
- `/admin`
  - admin dashboard
- `/admin/accounts`
  - admin account management
- `/admin/committee/[slug]`
  - admin committee membership and committee-specific account management
- `/committee/[slug]`
  - committee dashboard / meeting list
- `/committee/[slug]/meeting/[meetingId]`
  - canonical live meeting page for attendees and members
- `/committee/[slug]/meeting/[meetingId]/live`
  - compatibility route; may redirect internally to `/committee/[slug]/meeting/[meetingId]`
- `/committee/[slug]/meeting/[meetingId]/join`
  - meeting join page
- `/committee/[slug]/meeting/[meetingId]/attendee-login`
  - guest attendee login / recovery entry page
- `/committee/[slug]/meeting/[meetingId]/moderate`
  - moderator/chair live control surface
- `/committee/[slug]/meeting/[meetingId]/moderate/join-qr`
  - join QR screen
- `/committee/[slug]/meeting/[meetingId]/agenda-point/[agendaPointId]/tools`
  - agenda-point tools / attachment management screen
- `/committee/[slug]/meeting/[meetingId]/attendee/[attendeeId]/recovery`
  - attendee recovery page
- `/receipts`
  - public receipt verification vault
- `/docs/[...docPath]`
  - documentation pages

Routes that should remain backend/API-only and not be treated as SPA pages:

- OAuth start/callback routes
- SSE stream endpoints
- command endpoints for mutations
- blob/file download endpoints
- public verification API endpoints
- docs asset/search transport endpoints unless the frontend later absorbs them fully

Notes:

- Dynamic client-side state should replace HTMX partial updates, but the visible route structure should remain familiar.
- Screen internals may be reorganized in Svelte components without changing the external route contract.
- Query-string behavior for pagination, filters, and deep-linking should be preserved where it is already user-visible.
- Route params should continue to use committee slug plus meeting-scoped IDs unless Phase 0 later finds a strong reason to change canonical URL identity.

Initial protobuf package split:

The initial contract should be split by domain/workflow area rather than by individual page component or low-level database table.

Recommended package naming:

- use domain-scoped packages such as `conference.session.v1` and `conference.votes.v1`
- keep all initial contracts under `v1`
- avoid a single giant `conference.v1` package for all services

Recommended file/package layout:

- `proto/conference/common/v1/types.proto`
  - package: `conference.common.v1`
  - purpose: shared value objects used across domains
  - examples: pagination metadata, actor summaries, meeting references, committee references, capability flags
- `proto/conference/session/v1/session.proto`
  - package: `conference.session.v1`
  - purpose: session bootstrap, login/logout, current actor, auth state
- `proto/conference/committees/v1/committees.proto`
  - package: `conference.committees.v1`
  - purpose: committee list, committee overview, committee-scoped meeting list read models
- `proto/conference/meetings/v1/meetings.proto`
  - package: `conference.meetings.v1`
  - purpose: meeting summary, live meeting read model, join flow, attendee-facing meeting state
- `proto/conference/moderation/v1/moderation.proto`
  - package: `conference.moderation.v1`
  - purpose: moderator screen read model and meeting-level moderation commands
- `proto/conference/agenda/v1/agenda.proto`
  - package: `conference.agenda.v1`
  - purpose: agenda-point list, ordering, activation, agenda import, agenda-point tools state
- `proto/conference/attendees/v1/attendees.proto`
  - package: `conference.attendees.v1`
  - purpose: attendee list, attendee management, guest recovery, chair/quoted toggles
- `proto/conference/speakers/v1/speakers.proto`
  - package: `conference.speakers.v1`
  - purpose: speaker queue read models and speaker actions for both moderator and attendee flows
- `proto/conference/votes/v1/votes.proto`
  - package: `conference.votes.v1`
  - purpose: vote lifecycle, vote panel state, ballot submission, tally/counting flows, receipt verification
- `proto/conference/admin/v1/admin.proto`
  - package: `conference.admin.v1`
  - purpose: admin dashboard, accounts, committee membership management, OAuth committee rules
- `proto/conference/realtime/v1/realtime.proto`
  - package: `conference.realtime.v1`
  - purpose: shared event payload schemas for SSE streams and invalidation messages
- `proto/conference/docs/v1/docs.proto`
  - package: `conference.docs.v1`
  - purpose: docs page state and docs search if those stay under the main SPA/API boundary

Design rules for the split:

- packages should follow stable domain boundaries, not UI component names
- services may return screen-oriented read models even when the package is domain-scoped
- cross-package shared types should go into `conference.common.v1` only when reuse is real and stable
- avoid over-normalizing messages too early; duplicate small read-model fields when that improves clarity
- keep command requests and query read models in the same package when they belong to the same workflow area

Suggested service ownership:

- `conference.session.v1`
  - `SessionService`
- `conference.committees.v1`
  - `CommitteeService`
- `conference.meetings.v1`
  - `MeetingService`
- `conference.moderation.v1`
  - `ModerationService`
- `conference.agenda.v1`
  - `AgendaService`
- `conference.attendees.v1`
  - `AttendeeService`
- `conference.speakers.v1`
  - `SpeakerService`
- `conference.votes.v1`
  - `VoteService`
- `conference.admin.v1`
  - `AdminService`
- `conference.docs.v1`
  - `DocsService`

Rationale:

- `meetings`, `moderation`, `speakers`, and `votes` are kept distinct because they are large workflow surfaces in this app and will likely grow independently.
- `attendees` is separated from `meetings` because attendee management has its own permissions, guest flows, and recovery tools.
- `agenda` is separated because it combines ordering, activation, tools, attachments, and import workflows that would otherwise make `meetings.proto` too broad.
- `realtime` is separated so SSE event payloads can evolve without being tightly coupled to a single page contract.
- `docs` can be removed later if documentation remains largely static and is served outside the main client contract.

Status:

- this is the initial proposed split for Phase 0
- names and boundaries can still be adjusted before Phase 1 starts
- the goal is to freeze this structure before generating production contract code

Session reuse assessment:

Current recommendation:

- reuse the existing server-side session model as the foundation for the rewrite
- do not reuse the current auth/session transport layer unchanged
- introduce a new API-facing session/bootstrap surface for the SPA

What looks reusable:

- server-side sessions stored in the database
- signed `session_id` cookie model
- account and guest session types
- repository-backed session persistence
- OAuth and password login both converging on the same session mechanism

Why reuse looks attractive:

- the current system already matches the target architecture of same-origin cookie auth
- the cookie is already `HttpOnly`, which is desirable for the SPA
- the backend remains the system of record, so token-based redesign is not necessary
- guest and account access already fit one shared session manager

What should be refactored instead of copied forward:

- middleware that redirects directly to HTML pages
- middleware that parses route identity from URL strings
- handler flows that return template input plus redirect metadata
- request-context enrichment that is tightly coupled to the current route/middleware chain

Main findings from the current implementation:

- sessions are persisted in the database and keyed by a signed opaque session ID
- the cookie/session setup is simple and understandable, which is a good baseline
- both account logins and guest attendee logins use the same session manager
- admin access is derived from account data, not from a separate session type
- current auth/access middleware mixes identity loading with redirect/error response behavior

Risks or limitations to address during the rewrite:

- session cookie settings are currently hardcoded and do not yet enforce `Secure`
- session TTL is currently fixed at 24 hours in code
- expired sessions can be deleted, but cleanup is not part of the explored request path
- the current app uses one `session_id` cookie, so only one effective auth identity can exist in a browser profile at a time
- session bootstrap for a SPA does not exist yet and will need a dedicated endpoint/service method

Recommended implementation direction:

- keep `session.Manager` semantics or a close equivalent
- keep DB-backed session persistence unless Phase 1 uncovers a scaling reason to change it
- preserve cookie-based auth for both password and OAuth login
- add a contract method such as `SessionService.GetSession` for SPA bootstrap
- make auth/access checks return API-friendly unauthenticated/forbidden responses instead of browser redirects in the new transport layer
- move committee/meeting/role resolution toward service/transport helpers that use parsed route params or explicit request fields rather than string-splitting paths

Conclusion:

- session reuse is recommended at the model/storage/cookie level
- session reuse is not recommended at the current middleware/handler interface level
- this should lower rewrite risk while still allowing a cleaner API-oriented auth surface

Main API contract decision:

Chosen direction:

- use `Connect + Protobuf` as the primary typed application contract
- use plain HTTP endpoints only where the transport semantics are a better fit than RPC

Why this is the best fit for this app:

- the domain is command-heavy and workflow-heavy rather than CRUD-centric
- typed service methods map naturally to actions like opening votes, starting speakers, joining meetings, and fetching screen read models
- the browser-facing TypeScript generation story is strong
- Go and frontend code can share a stricter contract boundary with less request-shape drift
- it fits the repo direction already established in this plan

What belongs in Connect:

- session/bootstrap queries
- authenticated read models for screens
- domain commands
- admin operations
- metadata reads and writes that are not binary transfers

What stays on plain HTTP:

- multipart file uploads
- blob downloads
- inline current-document responses
- OAuth start/callback browser flows
- SSE event streams

What this means architecturally:

- `proto/` becomes the source of truth for typed application APIs
- `internal/api/connect/` handles the main request/response contract
- `internal/api/http/` handles the intentionally non-Connect endpoints
- the frontend should prefer generated Connect clients for application behavior and use `fetch` only for the plain HTTP exceptions above

Why not choose OpenAPI as the main contract:

- it is less natural for the action-oriented parts of this domain
- it would push more design effort into endpoint-shape conventions for commands that already fit an RPC model well
- the typed client ergonomics for this specific app are more attractive with Connect

Conclusion:

- the rewrite should standardize on `Connect + Protobuf` as the main API contract
- plain HTTP remains a deliberate companion for files, SSE, and browser-native auth flows

File handling decision:

Current recommendation:

- keep binary file transfer outside the main Connect contract
- use plain HTTP endpoints for:
  - multipart uploads
  - attachment downloads
  - inline current-document streaming
- use the typed contract for metadata queries and attachment-related commands

Why this direction:

- the current app already uses straightforward multipart upload plus raw streaming successfully
- forcing file bytes through an RPC contract would add ceremony without much value
- browser file uploads and inline/download responses are naturally modeled as normal HTTP
- same-origin cookie auth works cleanly for protected file endpoints
- the current storage abstraction is already transport-agnostic and can be reused

What the split should look like:

- plain HTTP endpoints:
  - upload attachment binary
  - download blob by ID or attachment reference
  - stream the current live document inline
- typed contract methods:
  - list attachments for an agenda point
  - create/update/delete attachment metadata records where appropriate
  - set/clear the current attachment
  - return file references, labels, content types, sizes, and URLs needed by the frontend

Recommended API shape:

- the frontend should obtain attachment/document metadata from the typed API
- the typed API should return stable file URLs for binary fetches
- uploads should use `multipart/form-data` to a protected HTTP endpoint
- uploads should return structured metadata for the created file/attachment, either directly or via a follow-up typed read
- downloads and inline viewing should remain ordinary authenticated `GET` requests

What not to do:

- do not put large binary payloads directly into protobuf messages for normal upload/download flows
- do not redesign the storage layer just to make file transfer fit the RPC contract
- do not introduce presigned-upload complexity unless deployment requirements later make it necessary

Implications for implementation:

- `internal/api/http/` should own binary upload/download endpoints
- `internal/storage/` and blob metadata models can remain conceptually the same
- the frontend should treat files as URLs plus metadata, not as contract-embedded blobs
- CSRF protection will matter for authenticated upload endpoints because cookies are sent automatically

Conclusion:

- file transfer should stay plain HTTP
- attachment/document lifecycle and metadata should live in the typed API
- this gives the SPA a clean model without forcing binary transport through the wrong abstraction

Live-update contract decision:

Current recommendation:

- keep `SSE` as the live-update transport for Phase 1
- scope streams primarily at the meeting level
- send JSON events, not HTML fragments
- let the frontend refetch authoritative read models after relevant events

Why this direction:

- the current product already behaves as a server-to-client update system rather than a bidirectional realtime app
- SSE is simpler to deploy and reason about than WebSockets for this use case
- the current broker/event model already maps naturally to SSE
- moving HTML rendering out of the event stream will simplify both backend and frontend responsibilities

What should change from the current implementation:

- do not stream server-rendered partials
- do not make the SSE handler rebuild whole UI fragments on every event
- do not couple event names directly to HTMX swap behavior
- do not treat the stream as the source of truth for full screen state

Recommended live-update shape:

- one meeting-scoped SSE endpoint, for example:
  - `/api/realtime/meetings/{meetingId}/events`
- optional future split into moderator/live-specific streams only if needed after usage is clear
- event payloads should be small and stable
- events should carry enough information for the client to decide what to refresh

Recommended event envelope:

```json
{
  "type": "votes.updated",
  "meetingId": "123",
  "entityId": "vote_42",
  "scope": ["moderation", "live"],
  "version": 17,
  "originClientId": "optional-client-id",
  "occurredAt": "2026-03-24T12:34:56Z"
}
```

Recommended event categories:

- `meeting.updated`
- `agenda.updated`
- `attendees.updated`
- `speakers.updated`
- `votes.updated`
- `current-document.updated`

Frontend behavior:

- subscribe to the meeting-scoped stream when entering a live screen
- ignore events originating from the same client when appropriate
- invalidate/refetch the relevant screen read model on event receipt
- keep optimistic UI minimal until real needs appear
- treat the latest fetched read model as authoritative

Backend behavior:

- publish small domain events from service-layer operations
- include meeting scope and optional origin client ID
- avoid embedding screen-specific rendering logic in the event transport
- keep event production transport-agnostic where possible so it can outlive SSE if needed

Versioning guidance:

- include a monotonically useful field such as `version` or `updatedAt` when possible
- event payloads should be additive and backward-compatible within `v1`
- the API read models, not the SSE stream, should remain the primary contract for screen state

What not to do:

- do not move to WebSockets by default
- do not stream full screen snapshots as the primary design
- do not create many tiny highly specialized event endpoints too early

Conclusion:

- live updates should remain SSE-based
- the stream should carry JSON invalidation/domain events
- the frontend should refetch typed read models instead of receiving rendered UI fragments

Locale strategy:

Current recommendation:

- keep the current locale resolution semantics:
  - URL prefix
  - then `locale` cookie
  - then `Accept-Language`
  - then default locale
- keep locale prefixes optional rather than mandatory
- preserve the current route shape, including optional prefixed routes such as `/de/home`
- move user-facing application translations to the frontend as the source of truth

Supported locales for the initial rewrite:

- `en`
- `de`

Route behavior:

- all SPA screen routes should work both:
  - without a locale prefix
  - and with an explicit supported locale prefix
- the default locale should not require a prefix
- if a request arrives with a locale prefix, the app should preserve that prefix during in-app navigation
- if a request arrives without a locale prefix, the app should not force a redirect to a prefixed URL

Why this direction:

- it preserves the current user-facing behavior and mental model
- it keeps localized shareable URLs possible without forcing locale prefixes on every route
- it avoids unnecessary route churn while still allowing explicit locale links
- it fits the existing same-origin Go edge handling cleanly

Source of truth for translations:

- frontend UI strings should move into the SvelteKit app
- backend API responses should prefer stable error codes and machine-readable metadata over localized screen copy
- backend locale support may remain for:
  - initial request locale resolution
  - docs or other server-owned content if those remain backend-rendered
  - any server-generated text that still truly needs localization

Recommended implementation shape:

- frontend translation messages live in the `web/` workspace
- locale preference is persisted in a regular `locale` cookie
- the frontend locale store initializes from:
  - the URL prefix when present
  - otherwise the locale cookie
  - otherwise a server-provided bootstrap locale or browser preference
- the language switcher should:
  - update the frontend locale store
  - persist the `locale` cookie
  - preserve whether the current route is prefixed or unprefixed

Server responsibilities:

- continue resolving locale for the initial HTML request
- continue recognizing and stripping supported locale prefixes before handing the request to the app
- serve the SPA shell for both prefixed and unprefixed app URLs
- expose enough bootstrap information for the frontend to know the resolved locale on first load

Frontend responsibilities:

- own application copy and runtime translation lookup
- render the correct locale immediately from bootstrap state to avoid flicker where possible
- keep docs/content handling separate if documentation remains backend-owned

What not to do:

- do not require every route to carry a locale prefix
- do not duplicate all user-facing copy between backend-rendered API messages and frontend translation catalogs
- do not make the frontend and backend compete as separate sources of truth for application UI text

Conclusion:

- preserve the current locale detection and optional-prefix model
- move app-copy translation ownership to the frontend
- keep backend locale logic focused on request resolution and any remaining server-owned content

Feature parity definition:

For this rewrite, feature parity does not mean reproducing the internal implementation. It means that the new system supports the same user-visible workflows, permissions, and realtime behavior that the current application already provides.

Parity should be judged against:

- the current user-facing routes and screens
- the current role-based behavior
- the current major meeting workflows
- the current E2E test surface in `e2e/`

The rewrite should be considered functionally at parity when all of the following are true:

- authenticated users can:
  - log in
  - reach `/home`
  - view their committees
- admins can:
  - log in
  - manage accounts
  - manage committees
  - manage committee memberships and OAuth rules
- committee users can:
  - view committee dashboards
  - create, activate, and delete meetings
  - open and close meeting signup
- meeting participants can:
  - join meetings
  - use guest attendee login/recovery flows
  - access the live meeting page
- moderators/chairpersons can:
  - use the moderation screen
  - manage attendees
  - manage agenda points and agenda ordering/import
  - manage speakers
  - manage attachments and current documents
  - manage votes through the full lifecycle
- public users can:
  - access the receipt vault behavior
  - verify receipts through the public verification flow
  - access documentation pages if docs remain in scope
- realtime behavior works for:
  - speaker updates
  - attendee updates where relevant
  - vote updates
  - current-document changes
- route-level parity exists for the screen routes defined earlier in this document
- role/permission behavior is preserved for admin, chairperson, member, attendee, and guest contexts

Parity does not require:

- preserving HTMX partial endpoints
- preserving Templ input types
- preserving identical DOM structure
- preserving identical generated route code
- preserving the exact same transport endpoint paths behind the SPA, except where compatibility is explicitly desired

Parity checklist source:

- use the current Playwright coverage as the baseline inventory
- treat these existing workflow areas as the minimum parity surface:
  - admin
  - committee
  - manage/moderate
  - join and attendee login
  - speakers and quotation behavior
  - voting and concurrent/session sync behavior
  - attachments and current document behavior
  - docs
  - OAuth
  - access control

Testing strategy decision:

Current recommendation:

- keep Playwright as the primary end-to-end test tool and primary acceptance target
- require the current `e2e/` suite to pass unchanged wherever possible
- allow only minimal E2E adjustments where the server boot/wiring changes make them necessary
- derive lower-level API integration tests from the E2E workflows instead of designing them independently

Primary testing principle:

- E2E defines the user-visible contract
- API integration tests should mirror E2E workflows at the backend boundary
- narrow route/handler tests should support the API integration layer, not replace it

Required test layers:

- end-to-end tests
  - full user workflows in a browser
  - multi-session realtime behavior where relevant
  - primary acceptance signal for parity
- API integration tests
  - backend behavior for each E2E workflow without the browser
  - exercise the real API contract and auth/session behavior
  - verify read models, commands, permissions, and error cases
- backend unit/service tests
  - business rules
  - permission decisions
  - domain transitions
- backend contract/transport tests
  - request/response mapping
  - auth failures
  - validation behavior
  - SSE event formatting/stream behavior
- frontend tests
  - component tests for interaction-heavy UI
  - store/state tests
  - route/bootstrap tests

Testing expectations by phase:

- Phase 1
  - generation/config sanity checks
  - basic contract tests for the first service surface
- Phase 2
  - service-layer tests added as logic moves out of HTML handlers
- Phase 3
  - frontend component/store tests for the new app shell and session bootstrap
- Phase 4 and beyond
  - each vertical slice ships with:
    - unchanged or minimally adapted E2E coverage
    - mapped API integration coverage for the same workflow
    - backend tests
    - frontend tests where state is non-trivial
    - permission/realtime coverage where applicable

E2E parity baseline:

The rewrite should preserve coverage for the workflow areas currently represented in `e2e/`, including:

- `admin`
- `committee`
- `manage`
- `moderate`
- `join`
- `attendee_login`
- `attendee_speakers`
- `agenda_speakers`
- `attachment`
- `current_doc`
- `voting`
- `voting_concurrent`
- `session_sync`
- `oauth`
- `docs`
- `access_control`

E2E compatibility rule:

- the goal is for the existing E2E suite to pass unchanged
- acceptable changes are limited to harness/startup adjustments caused by the new server wiring or app boot process
- avoid rewriting assertions unless the current test is tightly coupled to SSR/HTMX internals rather than user-visible behavior
- preserve existing stable semantic hooks such as route structure and `data-testid` markers wherever practical in the rewrite
- when a current E2E assertion depends on mutable localized copy or tooltip text, document the underlying workflow intent in the E2E-to-API mapping and prefer stable hooks for any newly added coverage

E2E-to-API integration mapping:

For each E2E test file or scenario, add a corresponding API integration test plan and then implement those tests against the real API surface.

Sequencing rule:

- draft the initial contract surface first
- derive and document the API-call equivalents for each E2E workflow while the old codebase is still intact
- derive those API-call flows against the drafted contract, not against an ad hoc endpoint sketch
- do not postpone this mapping until after the SSR/HTMX stack has been removed
- use the intact legacy implementation as the behavioral reference when identifying:
  - commands
  - read models
  - auth/session expectations
  - realtime interactions

Each mapping should answer:

- what user workflow the E2E test proves
- which API queries/commands/SSE events now implement that workflow
- which permission and failure cases belong at the API layer
- which parts still need browser coverage because they are inherently UI-specific

Recommended mapping artifact:

- maintain a living matrix in the repo that links:
  - E2E scenario
  - affected SPA route
  - relevant API methods/endpoints
  - required API integration tests
  - any remaining route/component tests

Suggested matrix columns:

- E2E test name
- user workflow
- SPA screen/route
- API surface touched
- API integration test names
- route/component test notes
- parity status

Testing order per workflow:

1. Draft the relevant contract surface for the workflow.
2. While the old codebase is still intact, preserve or adapt the existing E2E test minimally.
3. While the old codebase is still intact, document the API operations that should back that E2E scenario against the drafted contract.
4. Add API integration tests for the same workflow.
5. Add focused route/component tests only where they add value beyond the API integration coverage.

Why this order:

- it forces API-call mapping to be grounded in an intentional contract
- it keeps the rewrite anchored to user-visible behavior
- it captures the API mapping while the legacy behavior is still easy to inspect and verify
- it prevents overfitting lower-level tests to an implementation that might still change
- it ensures API tests are derived from real workflows rather than abstract endpoint inventories

Test philosophy:

- test workflows and permissions, not implementation details
- treat E2E as the top-level contract and lower-level tests as supporting evidence
- prefer API integration tests derived from E2E workflows over large numbers of isolated endpoint tests
- prefer service tests for domain branching instead of pushing all logic into E2E
- prefer route/screen assertions over fragile HTML-fragment assertions
- prefer stable semantic selectors over localized labels for newly added browser assertions where the UI already exposes a suitable hook
- keep one or two strong realtime E2E scenarios instead of many redundant variants

Definition of acceptable test coverage for a completed feature area:

- the matching E2E workflow still passes
- the matching API integration workflow test exists and passes
- at least one backend test covering the main business path
- at least one negative-path or permission-path test where the feature is sensitive
- at least one browser-level workflow test for user-visible behavior
- realtime verification if the feature is supposed to update another session live

Exit criteria for Phase 0:

- [x] Finalize the contract style and record the rationale.
- [x] Finalize the live-update contract shape and record the rationale.
- [x] Define the top-level SPA route map.
- [x] Define the initial protobuf package split and naming conventions.
- [x] Define the target repository structure for `web/`, `proto/`, and generated code.
- [x] Decide how session reuse will work.
- [x] Define what counts as feature parity for the rewrite.
- [x] Define the rewrite testing strategy and parity baseline.

### Phase 1: Contract Foundation

Create the new backend contract and generation pipeline before porting UI features.

Current status:

- Phase 1 has started.
- An initial contract surface draft exists for the first vertical slice.
- The first real contract scaffold now exists in `proto/` together with `buf.yaml` and `buf.gen.yaml`.
- Generated Go and TypeScript outputs now exist for the first-slice packages.
- The current contract scaffold has been validated with `buf lint` and `buf generate` via `nix develop . --command ...`.
- The first E2E-to-API mapping matrix now exists in `e2e-api-mapping-matrix.md`.
- A shared API error/status mapper now exists for the new transport layer.
- A minimal Connect-backed `SessionService` transport now exists and is mounted under `/api`.
- API-level tests now cover the initial `SessionService` bootstrap, login, and logout flow.
- `CommitteeService` (`ListMyCommittees`, `GetCommitteeOverview`) is implemented, mounted under `/api`, and covered by API integration tests.
- `MeetingService` (`GetLiveMeeting`) is implemented, mounted under `/api`, and covered by API integration tests.
- `ModerationService` (`GetModerationView`, `ToggleSignupOpen`) is implemented, mounted under `/api`, and covered by API integration tests including version-conflict detection.
- Meeting `version` column added (migration 030) and wired into all read and mutation paths.
- A combined test server helper (`newCombinedAPITestServer`) exists in `internal/api/connect/` for multi-service integration tests.
- Meeting-scoped SSE invalidation endpoint (`GET /api/realtime/meetings/{meetingId}/events`) is implemented in `internal/api/http/` and covered by an integration test.
- `ToggleSignupOpen` publishes JSON invalidation events to the broker; the SSE endpoint filters and streams them to subscribed clients.
- All first-slice acceptance criteria are met. Phase 1 is complete.

Deliverables:

- `buf.yaml` and `buf.gen.yaml`
- initial proto packages, likely split by bounded areas such as:
  - `session`
  - `accounts`
  - `committees`
  - `meetings`
  - `attendees`
  - `agenda`
  - `speakers`
  - `votes`
  - `realtime`
- generated Go code
- generated TypeScript clients
- shared error model and status mapping

Implementation status for this phase so far:

- [x] Create `buf.yaml`
- [x] Create `buf.gen.yaml`
- [x] Materialize the first-slice contract draft as actual `.proto` files
- [x] Generate Go contract code
- [x] Generate TypeScript client code
- [x] Add shared transport error/status mapping
- [x] Implement the first generated Connect service on the Go server
- [x] Add initial API-level tests for the session bootstrap slice
- [x] Implement `CommitteeService` with API integration tests
- [x] Implement `MeetingService` with API integration tests
- [x] Implement `ModerationService` (read + `ToggleSignupOpen` mutation) with API integration tests
- [x] Add meeting `version` field (migration 030) for optimistic conflict detection
- [x] Implement meeting-scoped SSE invalidation endpoint (`GET /api/realtime/meetings/{meetingId}/events`)
- [x] Publish `MeetingInvalidationEvent` from `ToggleSignupOpen` via the broker

Design rules:

- favor stable opaque IDs
- keep commands explicit and domain-named
- return denormalized read models for high-interaction screens
- include permissions/capabilities in read models where useful
- define versioning or timestamps for mutable resources

Initial contract surface draft for the first vertical slice:

The first contract slice should be intentionally narrow. Its job is to prove the architecture, not to model the whole application at once.

Proposed first-slice scope:

- session bootstrap
- password login/logout
- current-user/current-session state
- home / committee listing
- committee overview
- one live meeting read model
- one moderation read model
- one simple moderation command
- meeting-scoped realtime subscription

Recommended first moderation command:

- `ToggleSignupOpen`

Why this command:

- it is clearly user-visible
- it exercises auth, permissions, and mutation flow
- it affects other clients and therefore validates realtime updates
- it is much simpler than starting with the full voting lifecycle

Out of scope for this first slice:

- admin surface
- OAuth account provisioning logic beyond keeping the existing browser flow intact
- attachment upload/download flows
- full voting command set
- agenda import
- attendee recovery tooling

Primary E2E workflows this slice should prepare us to map:

- login/session bootstrap
- committee navigation
- join / attendee login fundamentals
- one live meeting read path
- one moderation workflow
- one realtime sync scenario

Draft package/service surface:

`conference.session.v1`

```proto
service SessionService {
  rpc GetSession(GetSessionRequest) returns (GetSessionResponse);
  rpc Login(LoginRequest) returns (LoginResponse);
  rpc Logout(LogoutRequest) returns (LogoutResponse);
}
```

Intent:

- `GetSession`
  - SPA bootstrap call
  - returns auth state, actor summary, locale, capabilities, and minimal app bootstrap data
- `Login`
  - password login only for the typed contract
  - sets the session cookie via transport layer
- `Logout`
  - clears the session

Key response shape:

- `authenticated`
- `actor`
- `isAdmin`
- `availableCommittees`
- `locale`
- `capabilities`
- optional `redirectTo`

Notes:

- OAuth start/callback remains plain HTTP, not Connect
- guest attendee login may land here later or may stay in a meeting/attendee package depending on final flow design

`conference.committees.v1`

```proto
service CommitteeService {
  rpc ListMyCommittees(ListMyCommitteesRequest) returns (ListMyCommitteesResponse);
  rpc GetCommitteeOverview(GetCommitteeOverviewRequest) returns (GetCommitteeOverviewResponse);
}
```

Intent:

- `ListMyCommittees`
  - powers `/home`
- `GetCommitteeOverview`
  - powers `/committee/[slug]`
  - returns the committee summary plus the relevant meeting list/read model for the current user role

Request identity for first slice:

- use `committee_slug` in the request for route alignment
- include stable IDs in responses so later contract evolution can lean more on opaque identifiers

`conference.meetings.v1`

```proto
service MeetingService {
  rpc GetJoinMeeting(GetJoinMeetingRequest) returns (GetJoinMeetingResponse);
  rpc GetLiveMeeting(GetLiveMeetingRequest) returns (GetLiveMeetingResponse);
}
```

Intent:

- `GetJoinMeeting`
  - powers `/committee/[slug]/meeting/[meetingId]/join`
  - powers `/committee/[slug]/meeting/[meetingId]/attendee-login`
  - returns the join/login screen state for account signup, guest signup, and attendee re-entry
- `GetLiveMeeting`
  - powers `/committee/[slug]/meeting/[meetingId]`
  - returns the live meeting read model needed by attendees/members
  - includes:
  - meeting summary
  - active agenda point summary
  - speaker list snapshot
  - current-document metadata
  - live capability flags for the current actor

Request identity for first slice:

- use `committee_slug` + `meeting_id` to align with existing routes and E2E workflows
- return canonical IDs in the response payload

`conference.moderation.v1`

```proto
service ModerationService {
  rpc GetModerationView(GetModerationViewRequest) returns (GetModerationViewResponse);
  rpc ToggleSignupOpen(ToggleSignupOpenRequest) returns (ToggleSignupOpenResponse);
}
```

Intent:

- `GetModerationView`
  - powers `/committee/[slug]/meeting/[meetingId]/moderate`
  - returns a denormalized moderation screen model
- `ToggleSignupOpen`
  - first proving mutation for the new stack
  - should return either the updated meeting state or a compact operation result plus version metadata

Moderation read model should include at least:

- meeting summary
- attendee summary block
- active agenda point summary
- speaker summary block
- current signup-open state
- current actor capabilities
- stream subscription info or stream URL

Companion plain HTTP endpoints for the first slice:

- `GET /oauth/start`
- `GET /oauth/callback`
- `GET /api/realtime/meetings/{meetingId}/events`

Draft realtime behavior for the first slice:

- `ToggleSignupOpen` publishes a meeting-scoped invalidation event
- the moderation and live screens refetch their typed read models on receipt

Contract drafting rule for this slice:

- keep request identity aligned with existing route params where that makes E2E mapping easier
- keep response payloads denormalized and screen-oriented
- avoid premature generalization into tiny reusable messages
- prefer clarity for the first slice over maximal theoretical elegance

### Phase 2: Backend Application Layer

Refactor the Go backend so domain logic is no longer coupled to server-rendered templates.

Deliverables:

- service-layer APIs that return contract DTOs or domain results instead of template inputs
- transport handlers for the new contract
- centralized auth/session bootstrap endpoint
- JSON/event-based SSE endpoints
- reusable backend authorization checks decoupled from page-rendering assumptions

Main goal:

Move logic out of handler/template glue and into service-level operations that can be called by the new transport layer cleanly.

### Phase 3: Frontend Foundation

Create the SvelteKit application and prove the local development/build pipeline.

Deliverables:

- `web/` SvelteKit app
- static adapter setup
- generated client integration
- app shell, layout, routing, and session bootstrap
- common state/query utilities
- error handling strategy
- design system or component primitives for the new UI

Infrastructure requirements:

- local dev workflow that can run frontend and Go backend together
- production build that outputs static assets for embedding
- updated Docker build that installs frontend dependencies, generates clients, builds the app, and embeds the output into the Go binary

### Phase 4: Vertical Slice Implementation

Build one fully working slice end-to-end before porting everything.

Recommended first slice:

- authentication/session bootstrap
- committee list/home
- one meeting read model
- one high-value interactive screen, preferably moderation or voting

Why:

- it validates contract design
- it exercises auth and permissions
- it proves live updates
- it gives the frontend structure for the rest of the port

Success criteria:

- user can sign in
- user can load the new SPA from the Go server
- user can view live meeting state
- user can perform at least one important state-changing workflow
- realtime updates work across two browser sessions
- the matching E2E-to-API mapping has been documented while the legacy implementation is still available

### Phase 5: Full Feature Port

Port the remaining feature areas in domain order.

Recommended order:

1. session/auth/bootstrap
2. home and committee overview
3. meeting join and attendee flows
4. moderation and speaker management
5. agenda management
6. voting
7. admin/account management
8. docs/public verification pages
9. attachments and file-serving flows

Porting rule:

Each feature area should be considered complete only when all of the following exist:

- E2E-to-API workflow mapping documented before the legacy implementation for that area is removed
- contract definitions
- backend implementation
- frontend screens/components
- API integration tests derived from the mapped E2E workflows
- happy-path tests
- role/permission checks
- live update behavior where relevant

### Phase 6: Legacy Removal

After feature parity is reached and validated, remove the old server-rendered stack.

Precondition:

- the E2E-to-API mapping matrix must already exist for the covered workflow surface
- API integration test equivalents for the mapped workflows must already be in place

Expected removals:

- Templ page components and generated `_templ.go` files
- HTMX-specific client assets
- YAML route definitions and route generator
- template-oriented handler return types
- SSR-only middleware and helpers no longer needed

Keep only if still useful:

- repository layer
- storage layer
- session implementation
- parts of middleware and auth logic
- SSE broker (internal pub/sub stays; the raw HTTP SSE endpoint is replaced by the Connect stream)

### Phase 6b: Connect Streaming RPC for Meeting Events

**Goal**: replace the hybrid `GET /api/realtime/meetings/{meetingId}/events` raw SSE endpoint
plus per-event typed-RPC refetch with a single typed Connect server-streaming RPC.

**Why**: Connect server-streaming RPCs are implemented as SSE under HTTP/1.1, so the wire
protocol is unchanged. The benefit is a fully typed contract — no raw `EventSource`, no manual
event-name dispatch, no separate refetch round trip. The client receives the updated view
payload directly in the stream.

**Proto change** (`proto/conference/meetings/v1/meetings.proto`):

```protobuf
rpc SubscribeMeetingEvents(SubscribeMeetingEventsRequest)
    returns (stream MeetingEvent);

message SubscribeMeetingEventsRequest {
  string committee_slug = 1;
  string meeting_id = 2;
}

message MeetingEvent {
  oneof event {
    LiveMeetingView    live_updated       = 1;
    SpeakersUpdated    speakers_updated   = 2;
    AgendaUpdated      agenda_updated     = 3;
    VotesUpdated       votes_updated      = 4;
    AttendeesUpdated   attendees_updated  = 5;
  }
}
```

Each event variant carries the fresh view that the affected page needs — the SPA applies the
payload directly without a follow-up fetch.

**Backend implementation**:
- New `SubscribeMeetingEvents` method on the `MeetingService` Connect handler
- Subscribes to the broker, filters by meeting ID, calls the relevant read-model method on
  each incoming broker event, and streams the typed result to the client
- The raw `GET /realtime/meetings/{meetingId}/events` endpoint (`internal/api/http/realtime.go`)
  is removed after the new RPC is verified

**Frontend change** (`web/src/lib/utils/sse.ts` and the two page components):
- Replace `connectEventStream(url, onInvalidate)` + follow-up RPC calls with a single
  `client.meetings.subscribeMeetingEvents(req)` streaming call
- The stream handler patches the local reactive state directly from the event payload

**Rollout order**:
1. Add proto message + regenerate
2. Implement Go handler, register, verify with `go test ./...`
3. Update SPA pages, rebuild, run E2E suite
4. Delete `internal/api/http/realtime.go` and `web/src/lib/utils/sse.ts`
5. Remove `events_url` field from `LiveMeetingView` proto if no longer needed

## Testing Strategy

The rewrite needs a new test pyramid rather than a direct copy of current tests.

### Backend

- unit tests for service-layer operations
- contract-level handler tests
- repository tests where behavior is complex
- authorization tests for sensitive actions

### Frontend

- component tests for interaction-heavy widgets
- state/store tests where local logic is non-trivial
- route-level tests for app bootstrap and navigation

### End-to-End

- preserve Playwright as the main E2E tool
- rewrite tests around user workflows rather than HTML fragments
- cover at least:
  - login
  - joining a meeting
  - moderation
  - speaker management
  - voting lifecycle
  - admin management flows
  - realtime synchronization across sessions

## Build and Tooling Work

Expected tooling changes:

- add `buf`
- add SvelteKit build tooling
- add TypeScript generation step
- update `Taskfile.yaml` for:
  - contract generation
  - frontend dev
  - frontend build
  - combined dev mode
  - test orchestration
- update `Dockerfile` to build frontend assets before the Go binary

Developer experience goals:

- one command to run backend + frontend in development
- one command to regenerate all contracts and generated clients
- one command to run the full test suite

## Risks and Mitigations

### Risk: Contract churn early in the rewrite

Mitigation:

- design one vertical slice first
- avoid generating a huge proto surface before real usage
- prefer a few stable read models over many tiny low-level endpoints

### Risk: Backend logic remains coupled to old handlers

Mitigation:

- explicitly introduce service-layer boundaries
- move business logic before rewriting large parts of the UI

### Risk: Realtime complexity grows quickly

Mitigation:

- start with SSE invalidation events
- avoid bidirectional protocols until there is a proven need
- keep event payloads small and versioned

### Risk: Rewrite stalls before parity

Mitigation:

- work in vertical slices with clear done criteria
- keep a visible parity checklist
- prioritize core meeting workflows before edge/admin features

### Risk: Frontend state becomes inconsistent

Mitigation:

- define authoritative read models per screen
- refetch after commands or invalidation events
- avoid premature client-side caching complexity

## Suggested Milestones

1. Contract/tooling foundation exists and code generation works.
2. Go binary serves embedded SvelteKit build.
3. Session bootstrap and login work in the SPA.
4. One meeting workflow works end-to-end with realtime updates.
5. Core meeting management and voting flows reach parity.
6. Admin and public-facing flows reach parity.
7. Legacy HTMX/Templ stack is removed.

## Definition of Done

The rewrite is complete when:

- all user-facing flows are available in the SvelteKit frontend
- the Go server exposes only the new contract and required static/file endpoints
- live updates no longer depend on HTML fragments
- feature parity is validated by automated tests
- the old Templ/HTMX/routing-generator stack has been removed
- local development, CI, and Docker builds all use the new architecture

## Immediate Next Steps

1. Expand the E2E-to-API mapping matrix for the completed Phase 5 workflows so the legacy-removal phase has an accurate parity baseline.
2. Continue replacing HTML-over-SSE and HTMX-specific partial refresh paths with typed JSON invalidation plus SPA refetch flows.
3. Identify and remove legacy SSR/HTMX surfaces in low-risk slices, starting with documentation, obsolete helpers, and duplicate transport glue before attempting route-generator removal.
