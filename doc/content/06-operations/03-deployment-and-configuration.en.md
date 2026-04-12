---
title-en: Deployment and Configuration Guide
title-de: Deployment- und Konfigurationshandbuch
---

# Deployment and Configuration Guide

This guide covers all configuration options for deploying Open Caucus, including email, OIDC/OAuth, member management, and the development environment.

## Environment Variables

All configuration is done via environment variables or a `.env` file. See `.env.example` for the full list with defaults.

### Application

| Variable | Default | Description |
|----------|---------|-------------|
| `ENVIRONMENT` | `development` | `development`, `staging`, or `production` |
| `HOST` | `0.0.0.0` | HTTP bind address |
| `PORT` | `8080` | HTTP port |
| `SERVICE_NAME` | `conference-tool` | Service identifier |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |
| `LOG_FORMAT` | `json` | `json` or `text` |
| `SESSION_SECRET` | — | **Required.** HMAC key for session cookies (32+ characters) |
| `SESSION_EXPIRATION` | `86400` | Session lifetime in seconds (default: 24h) |
| `DATABASE_PATH` | `conference.db` | Path to the SQLite database file |

### Authentication

| Variable | Default | Description |
|----------|---------|-------------|
| `AUTH_PASSWORD_ENABLED` | `true` | Enable local username/password login |
| `AUTH_OAUTH_ENABLED` | `false` | Enable OIDC/OAuth login |

### OAuth/OIDC

Required when `AUTH_OAUTH_ENABLED=true`:

| Variable | Default | Description |
|----------|---------|-------------|
| `OAUTH_ISSUER_URL` | — | OIDC issuer URL (e.g. `https://auth.example.com`) |
| `OAUTH_CLIENT_ID` | — | OIDC client ID |
| `OAUTH_CLIENT_SECRET` | — | OIDC client secret |
| `OAUTH_REDIRECT_URL` | — | OAuth callback URL (e.g. `https://caucus.example.com/oauth/callback`) |
| `OAUTH_SCOPES` | `openid,profile,email` | Comma-separated OAuth scopes |
| `OAUTH_GROUPS_CLAIM` | `groups` | JWT claim containing user groups |
| `OAUTH_USERNAME_CLAIMS` | `preferred_username,email,sub` | Fallback claims for username (tried in order) |
| `OAUTH_FULL_NAME_CLAIMS` | `name,preferred_username,email` | Fallback claims for display name |
| `OAUTH_PROVISIONING_MODE` | `preprovisioned` | `preprovisioned` (accounts must exist) or `auto_create` (create on first login) |
| `OAUTH_REQUIRED_GROUPS` | — | Comma-separated groups required for login (empty = no restriction) |
| `OAUTH_ADMIN_GROUP` | — | Group that grants sitewide admin privileges |
| `OAUTH_COMMITTEE_GROUP_PREFIX` | — | When set, only OIDC groups with this prefix can be used in committee group rules |
| `OAUTH_STATE_TTL_SECONDS` | `300` | Lifetime of the OAuth state cookie |

#### How OIDC data is used

- **Email**: Always extracted from the `email` claim. Stored on the account and updated on every login. Used for sending meeting invite emails.
- **Display name**: Extracted from the first available claim in `OAUTH_FULL_NAME_CLAIMS`. Updated on every login.
- **Username**: Extracted from the first available claim in `OAUTH_USERNAME_CLAIMS`. Used as the account identifier.
- **Groups**: Extracted from the claim named in `OAUTH_GROUPS_CLAIM`. Used for automatic committee membership sync and admin group assignment.

### Email (SMTP)

Email is used for sending meeting invite emails to committee members.

| Variable | Default | Description |
|----------|---------|-------------|
| `EMAIL_ENABLED` | `false` | Enable email sending |
| `EMAIL_SMTP_HOST` | — | SMTP server hostname |
| `EMAIL_SMTP_PORT` | `587` | SMTP port (587 for STARTTLS, 1025 for Mailpit dev) |
| `EMAIL_USERNAME` | — | SMTP auth username (leave empty for unauthenticated) |
| `EMAIL_PASSWORD` | — | SMTP auth password |
| `EMAIL_FROM_ADDRESS` | — | Sender email address |
| `EMAIL_FROM_NAME` | `Open Caucus` | Sender display name |

When email is not configured (`EMAIL_ENABLED=false`):
- The invite step in the meeting creation wizard is hidden
- The "Send Invite Emails" button on the moderation page is disabled
- A warning message is shown to chairpersons explaining that email is not configured

#### Invite emails include

- Meeting name, description, and date/time (with timezone)
- Agenda with sub-points
- Custom message from the chairperson
- ICS calendar attachment (when the meeting has start/end datetime)
- Personalized join link (direct login for account members, invite-secret link for email-only members)

### Webhooks

| Variable | Default | Description |
|----------|---------|-------------|
| `WEBHOOK_URLS` | — | Comma-separated POST endpoints for event notifications |
| `WEBHOOK_SECRET` | — | Shared secret sent as `X-Webhook-Secret` header |
| `WEBHOOK_TIMEOUT_SECONDS` | `10` | Per-request HTTP timeout |

## Member Management

### How it works

Committee member management is accessible to **chairpersons** (not just admins). Members can be:

1. **Account-based**: Linked to a sitewide account (login via password or OIDC). The account email is used for notifications.
2. **Email-only**: Identified by email address, no sitewide account needed. They receive invite emails with a personalized login link that creates a guest session for the specific meeting.

### Adding members

Chairpersons can add members two ways:
- **By email**: Provide email + name + role. An invite secret is generated for personalized login links.
- **Assign existing account**: Select from accounts not already in the committee.

### OIDC automatic membership sync

When OIDC is configured, committee memberships can be automatically managed via group rules:
1. Admin creates a rule: committee slug + OIDC group name + role
2. When a user logs in via OIDC, their groups are matched against all rules
3. Memberships are created/updated/removed automatically
4. OAuth-managed memberships have their role locked (changeable only via OIDC groups)

### Invite emails

When creating a meeting, chairpersons can choose to send invite emails to selected members. The invite compose form includes:
- **Language selector**: English or Deutsch
- **Timezone selector**: For datetime display in the email (common IANA timezones, auto-detected from browser)
- **Custom message**: Optional personal note included in the email

Invites can also be sent for existing meetings via the "Send Invite Emails" button on the moderation page.

## Quotation System

The speakers list supports configurable quotation rules that affect speaker ordering:

- **FLINTA\* quotation**: Interleaves FLINTA\* and non-FLINTA\* speakers in round-robin
- **First-speaker quotation**: Gives priority to speakers making their first contribution

Rules can be reordered via drag-and-drop on the moderation page. The order determines which dimension is applied first. Both can be independently enabled or disabled.

## Development Environment

### Quick start

```bash
task dev:oidc
```

This starts four services in parallel:
- **Go backend** (port 8080) with hot reload via `air`
- **Vite SPA dev server** (port 5173) with HMR
- **Local OIDC provider** (port 9096) for OAuth testing
- **Mailpit** (SMTP on port 1025, Web UI on port 8025) for email testing

### Mailpit

Mailpit captures all outgoing emails in development. View them at **http://localhost:8025**.

The `populate-env` command automatically configures the app to send emails via Mailpit when running `task dev:oidc`.

### Docker Compose

```bash
# Linux (host networking)
docker compose up --build app oidc

# macOS/Windows (Docker Desktop)
docker compose -f docker-compose.desktop.yml up --build app oidc
```

Both compose files include a Mailpit service for email testing.

### Database reset

If the database gets into a dirty migration state:

```bash
sqlite3 conference.db "UPDATE schema_migrations SET version = <last_clean_version>, dirty = 0;"
```

Then restart the app to apply pending migrations.

### Test accounts

After running `task dev:oidc`, the following OIDC test accounts are available (see `dev/users.yaml`):

| Username | Password | Groups | Notes |
|----------|----------|--------|-------|
| `alice` | `alice` | `committee-a`, `committee-a-chair`, `ca-admin` | Admin + chairperson |
| `bob` | `bob` | `committee-b` | Regular member |

### Running tests

```bash
task test              # Go unit tests (excludes E2E)
task test:e2e          # Browser-based E2E tests (requires Playwright)
task check             # Format + vet + lint
task ci                # All CI checks
```
