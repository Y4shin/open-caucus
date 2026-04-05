# Real-Time Events & OAuth/OIDC Login

This document is for operators deploying Open Caucus and developers building
clients that consume its real-time events or integrate with its OAuth login
flow.

---

## Real-Time Meeting Events

### What you get

When anything changes in a meeting (speakers queue, votes, agenda, attendees),
Open Caucus pushes a typed event to every connected client. Events are
**invalidation signals** — they tell you *what* changed so you can re-fetch the
relevant data. They do not carry the changed data itself.

Events can be consumed in two ways:

- **Connect RPC stream** — a long-lived HTTP/2 server-stream for connected
  browser clients (described below).
- **Outbound webhooks** — HTTP POST requests to one or more external URLs,
  configured via environment variables. Useful for integrating with automation
  tools such as n8n, Zapier, or custom backends.

### Subscribing

Call the `SubscribeMeetingEvents` RPC on the Meetings service:

```
POST /conference.meetings.v1.MeetingsService/SubscribeMeetingEvents
```

Request body:

```json
{
  "committeeSlug": "budget-committee",
  "meetingId": "42"
}
```

The server validates that the meeting exists and the caller has access, then
keeps the connection open. You will receive a stream of `MeetingEvent` messages,
each with a `kind` field.

### Event kinds

| Kind | Fired when |
|---|---|
| `SPEAKERS_UPDATED` | Speaker added, removed, started, ended, notes edited, queue reordered |
| `VOTES_UPDATED` | Vote opened, closed, eligibility changed, ballot cast |
| `AGENDA_UPDATED` | Agenda point created, updated, deleted, activated, attachment added/removed |
| `ATTENDEES_UPDATED` | Attendee signed up (self or manual), guest added |
| `MEETING_UPDATED` | Signup toggled open/closed, quotation settings changed, moderator assigned |

### Handling events

On each event, re-fetch the data that corresponds to the event kind. Example
in TypeScript (using the generated Connect client):

```typescript
const stream = meetingClient.subscribeMeetingEvents({
  committeeSlug: "budget-committee",
  meetingId: "42",
});

for await (const event of stream) {
  switch (event.kind) {
    case MeetingEventKind.SPEAKERS_UPDATED:
      await loadSpeakers();
      break;
    case MeetingEventKind.VOTES_UPDATED:
      await loadVotes();
      break;
    case MeetingEventKind.AGENDA_UPDATED:
    case MeetingEventKind.ATTENDEES_UPDATED:
    case MeetingEventKind.MEETING_UPDATED:
      await loadMeetingDetails();
      break;
  }
}
```

### Behaviour notes

- An initial `MEETING_UPDATED` event is sent immediately on connection so you
  can load the current state without a separate call.
- Events are scoped to the requested meeting — you will not receive events from
  other meetings.
- Events are **not persisted**. If the server restarts, in-flight events are
  lost. Clients should implement a polling fallback or reconnect logic.
- If your client reads too slowly, events may be dropped. The recommended
  pattern is to treat each event as "something changed" and always re-fetch
  current state rather than building up incremental state from events.

---

## Outbound Webhooks

### Overview

When `WEBHOOK_URLS` is set, Open Caucus POSTs a JSON payload to every
configured URL each time a meeting event is published. Delivery is
fire-and-forget — failed requests are logged and dropped with no retry.

### Configuration

| Variable | Required | Description |
|---|---|---|
| `WEBHOOK_URLS` | Yes | Comma-separated list of URLs to POST events to. Leave unset (or empty) to disable webhooks entirely. |
| `WEBHOOK_SECRET` | No | Shared secret sent as the `X-Webhook-Secret` header on every request. Omit to send no secret header. |
| `WEBHOOK_TIMEOUT_SECONDS` | No | Per-request HTTP timeout in seconds. Default: `10`. |

Example:

```
WEBHOOK_URLS=https://n8n.example.com/webhook/meeting-events,https://hooks.example.com/fallback
WEBHOOK_SECRET=supersecret
WEBHOOK_TIMEOUT_SECONDS=5
```

### Payload

Every request is a `POST` with `Content-Type: application/json`:

```json
{
  "event":      "speakers.updated",
  "meeting_id": 42,
  "timestamp":  "2026-04-05T12:34:56Z"
}
```

| Field | Type | Description |
|---|---|---|
| `event` | string | One of the event kind strings listed in the table below. |
| `meeting_id` | integer \| null | ID of the affected meeting. `null` for broker events not scoped to a meeting. |
| `timestamp` | string (RFC 3339) | UTC time at which the dispatcher fired the request. |

### Event kind strings

| `event` value | Fired when |
|---|---|
| `speakers.updated` | Speaker added, removed, started, ended, notes edited, queue reordered |
| `votes.updated` | Vote opened, closed, eligibility changed, ballot cast |
| `agenda.updated` | Agenda point created, updated, deleted, activated, attachment added/removed |
| `attendees.updated` | Attendee signed up (self or manual), guest added |
| `moderate-updated` | Signup toggled open/closed, quotation settings changed, moderator assigned |

### Headers

| Header | Always sent | Description |
|---|---|---|
| `Content-Type` | Yes | Always `application/json`. |
| `X-Webhook-Secret` | Only when `WEBHOOK_SECRET` is set | Use this to verify the request originates from your Open Caucus instance. |

### Behaviour notes

- A separate goroutine is spawned per URL per event, so a slow or unreachable
  URL does not block other URLs or delay the application.
- There is no retry queue. If a request fails (network error, non-2xx response),
  it is logged at `WARN` level and dropped.
- Webhooks are disabled entirely when `WEBHOOK_URLS` is empty — no HTTP calls
  are made.
- The dispatcher stops cleanly when the server shuts down.

---

## OAuth/OIDC Login

### Overview

Open Caucus can authenticate users via an external OpenID Connect identity
provider (Keycloak, Authentik, Entra ID, etc.) using the Authorization Code
flow with PKCE.

### Login flow

```
Browser                    Open Caucus                     Identity Provider
───────                   ────────────                    ──────────────────
GET /oauth/start ──────►  generate state + PKCE
                          set state cookie
                     ◄──  302 → IdP authorize URL
                                                    ──►  user authenticates
                     ◄──────────────────────────────────  302 → /oauth/callback?code=…&state=…
GET /oauth/callback ───►  validate state cookie
                          exchange code for tokens
                          extract claims from ID token
                          resolve / create account
                          sync admin & committee groups
                          create session
                     ◄──  302 → /home (or /admin)
```

### Endpoints

#### `GET /oauth/start`

Initiates the login. Redirect your users here.

| Query param | Required | Description |
|---|---|---|
| `target` | No | Set to `"admin"` to land on `/admin` after login. Defaults to `/home`. |

#### `GET /oauth/callback`

Registered with the identity provider as the redirect URI. Do not call this
directly — the IdP redirects the browser here after authentication.

### Configuration

Set these environment variables to enable OAuth:

| Variable | Required | Description |
|---|---|---|
| `OAUTH_ENABLED` | Yes | `true` to enable the OAuth login flow. |
| `OAUTH_ISSUER_URL` | Yes | OIDC issuer URL (e.g. `https://idp.example.com/realms/main`). Used for auto-discovery of endpoints. |
| `OAUTH_CLIENT_ID` | Yes | Client ID registered with the IdP. |
| `OAUTH_CLIENT_SECRET` | Yes | Client secret. |
| `OAUTH_REDIRECT_URL` | Yes | Must point to `/oauth/callback` on your deployment (e.g. `https://caucus.example.com/oauth/callback`). Must match exactly what is registered in the IdP. |
| `OAUTH_SCOPES` | No | Space-separated scopes. Default: `openid profile email`. Add a `groups` scope if your IdP requires it for the groups claim. |
| `OAUTH_REQUIRED_GROUPS` | No | Comma-separated group names. If set, the user must belong to **at least one** of these groups to log in. |
| `OAUTH_ADMIN_GROUP` | No | Group name that grants admin access. Users in this group become admins; users removed from it lose admin on next login. |
| `OAUTH_PROVISIONING_MODE` | No | Set to `auto_create` to automatically create accounts on first login. If unset, accounts must be pre-created by an admin before the user can log in via OAuth. |

### Identity provider setup

1. Register a new OIDC client in your IdP.
2. Set the redirect URI to `https://<your-domain>/oauth/callback`.
3. Ensure the ID token includes these claims:
   - `sub` (subject) — required
   - `preferred_username` or `sub` — used as the account username
   - `name` — used as the display name (falls back to username)
   - `email` — stored but not required
   - `groups` — **required if you use `OAUTH_REQUIRED_GROUPS`, `OAUTH_ADMIN_GROUP`, or committee group mapping**. Many IdPs do not include this by default — you may need to add a groups mapper/scope.

### Account resolution

When a user completes the OAuth flow, Open Caucus resolves their account in
this order:

1. **Required group check** — if `OAUTH_REQUIRED_GROUPS` is set and the user
   has none of them, login is rejected.
2. **Existing OAuth identity** — if the user has logged in before (matched by
   issuer + subject), the linked account is used.
3. **Username match** — if an account with the same username exists and was
   created for OAuth, it is linked. If the account uses a different auth
   method (e.g. password), login is rejected.
4. **Auto-creation** — if no account exists and `OAUTH_PROVISIONING_MODE=auto_create`,
   a new account is created. Otherwise login is rejected.

### Automatic group sync

On every login, Open Caucus syncs the user's roles based on their IdP groups:

- **Admin sync**: If `OAUTH_ADMIN_GROUP` is set, the user's admin flag is set
  or cleared based on group membership.
- **Committee sync**: Administrators can create group rules (via the admin UI)
  that map an IdP group to a committee role (`member` or `chairperson`). On
  login, the user's committee memberships are reconciled — roles are added,
  upgraded, or removed to match the rules. If multiple rules match the same
  committee, the highest role wins (`chairperson` > `member`).

### Troubleshooting login failures

When login fails, Open Caucus logs a warning with full context:

```
WARN oauth account resolution failed
    subject=c1983507…
    issuer=https://idp.example.com
    username=jdoe
    groups=["staff","viewers"]
    err="missing required oauth group: user has groups [staff viewers],
         but needs at least one of [committee-members admin]"
```

Common causes:

| Symptom | Cause | Fix |
|---|---|---|
| "missing required oauth group" | User's groups don't overlap with `OAUTH_REQUIRED_GROUPS` | Add the user to one of the required groups in the IdP, or check that the IdP is sending the `groups` claim (many don't by default). |
| "oauth account is not preprovisioned" | No account exists and auto-creation is off | Either create the account in the admin UI first, or set `OAUTH_PROVISIONING_MODE=auto_create`. |
| "account uses a different auth method" | Account was created with password auth | Migrate the account to OAuth via the admin UI, or create a separate OAuth account. |
| User logs in but has no committee access | IdP groups don't match any committee group rules | Check the group rules in the admin UI and verify the IdP sends the expected group names. |
| User logs in but is not admin | User is not in `OAUTH_ADMIN_GROUP` | Add the user to the admin group in the IdP. |
| Redirect loop or silent redirect to `/` | OAuth callback failed (check server logs) | Look for the `WARN oauth callback failed` log line for details. Common: mismatched redirect URI, expired state cookie, IdP misconfiguration. |
