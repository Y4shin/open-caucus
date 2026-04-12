---
title-en: Patch Notes
title-de: Versionshinweise
---

# Patch Notes

## v1.4.5 — 2026-04-12

### Improvements

- **Documentation**: Add patch notes section to in-app docs. Update chairperson, admin, and quotation docs to reflect v1.4.0 changes (member management, invite emails, quotation rules, OIDC profile sync).
- **UI wording**: Replace "Quoted" label with "FLINTA*" in the member management panel to match the rest of the application.
- **CLAUDE.md**: Add rules for maintaining patch notes on every release and updating docs when implementing new features.

---

## v1.4.4 — 2026-04-12

### Bug Fixes

- **Email headers**: Add RFC 2822 `Date` header to outgoing emails to prevent rejection by mail relays.

---

## v1.4.3 — 2026-04-12

### Bug Fixes

- **Email threading**: Add RFC 5322 `Message-ID` header to outgoing invite emails. Emails without this header were rejected by mail relays (e.g. amavis `BAD-HEADER-0`).
- **Email threading**: Track sent emails in new `sent_emails` table and set `References` header so invite emails for the same committee thread together in the recipient's mail client.

---

## v1.4.2 — 2026-04-12

### Improvements

- **OIDC logging**: Add comprehensive debug logging for OIDC group detection, committee sync, and email invite flows to aid production troubleshooting.

---

## v1.4.1 — 2026-04-12

### Bug Fixes

- **OIDC profile sync**: Fetch the OIDC userinfo endpoint on every login and update the account's display name and email. Previously, only the ID token was read and profiles were only set for new accounts, causing stale or missing display names.

---

## v1.4.0 — 2026-04-12

### New Features

- **Member management for chairpersons**: Chairpersons can now manage committee members directly from the committee page instead of requiring a sitewide admin. Add members by email, assign existing accounts, edit roles, and remove members.
- **Email-only members**: Committee members no longer need a sitewide account. Chairpersons can add members by email address; these members receive a personalized invite link.
- **Meeting invite emails**: Send meeting invite emails to committee members with ICS calendar attachments, agenda overview, custom message, language selection (EN/DE), and timezone support.
- **Meeting creation wizard — Invites step**: The wizard now includes an Invites step where chairpersons select which members receive invite emails. Replaces the old text-based participant entry.
- **Date range picker**: Meetings now support optional start and end datetime with a calendar-based date range picker.
- **Meeting editing**: Edit meeting details (name, description, datetime) after creation.
- **Speakers list quotation rules**: Revamped quotation system with ordered, drag-and-drop rules and an animated step-by-step visualization explaining how quotation sorting works.
- **bits-ui migration**: Migrated interactive UI components (Select, Switch, Collapsible, Dialog, Tooltip, DatePicker, DateRangePicker) from DaisyUI to bits-ui for improved accessibility (ARIA roles, keyboard navigation).
- **Agenda import — title highlighting**: Extracted titles in the agenda import preview are now highlighted in the edit text field.
- **OIDC email extraction**: The OIDC login flow now extracts and stores the user's email from the identity provider.
- **OIDC group prefix**: Configurable `OAUTH_GROUP_PREFIX` to filter which OIDC groups are considered for committee sync.
- **Deployment guide**: Added comprehensive sysadmin deployment documentation covering all environment variables, OIDC, email, and member management configuration.

### Bug Fixes

- **Meeting secrets**: Automatically generate a meeting secret when creating meetings. Existing meetings without secrets are backfilled on startup.
- **Docscapture selectors**: Updated all documentation screenshot selectors for the bits-ui migration (Collapsible, Dialog, etc.).

---

## v1.3.0 — 2026-04-10

### New Features

- **OIDC group prefix configuration**: Add configurable prefix for OIDC group names used in committee sync rules.
- **OIDC debug logging**: Add debug-level logging for OIDC group detection and committee membership synchronization.

---

## v1.2.1 — 2026-04-10

### New Features

- **Webhook header authentication**: Support per-URL custom headers in `WEBHOOK_URLS` for authenticated webhook delivery.

### Bug Fixes

- **Docker build context**: Stop excluding `doc/` and `tools/` directories from the Docker build context, fixing documentation capture in CI.

---

## v1.2.0 — 2026-04-06

### New Features

- **Outbound webhooks**: Add webhook dispatcher for meeting events (meeting created, started, ended) and committee/OIDC group events. Configure via `WEBHOOK_URLS` environment variable.
- **Webhook documentation**: Added `WEBHOOKS.md` with event schemas and configuration guide.

### Bug Fixes

- **OAuth logging**: Include issuer, username, and groups in login failure log messages.
- **Version display**: Remove duplicate "v" prefix in the footer version string.

---

## v1.1.0 — 2026-04-05

### New Features

- **Agenda import redesign**: Combined input and correction into a live two-panel workflow with classification pills, format detection, and diff view with cross-level move detection.
- **Agenda timestamps**: Record timestamps when agenda points are entered and left during a meeting.
- **Vote receipts**: Show voting choices when verifying secret ballot receipts. Added "My Receipts" dialog to the meeting live view.
- **Meeting creation wizard**: Multi-step wizard for meeting creation with agenda and participant import.
- **In-app documentation**: Embedded localized user documentation with search, media variants, and context-sensitive help.
- **SVG logo**: Replaced text title in the header with an SVG logo.
- **Internationalization**: Wired Paraglide translations across all Svelte components.
- **CI pipeline**: Three-stage Dockerfile with screenshot generation, upgraded to Go 1.26.1.

### Improvements

- **Moderate page**: Extracted reusable components (AttendeeRow, VoteCard, AgendaPointCard, SpeakersSection, VotesPanelSection).
- **Admin pages**: Cleaned up admin page layouts with shared AppCard and DataTable components.
- **QR codes**: Converted QR code pages to inline dialog modals.
- **Bug fixes**: Resolved 10 items from IMPROVEMENTS.md including Svelte 5 reactive cycles, speaker flow issues, and UI inconsistencies.

---

## v1.0.0 — 2026-04-05

### Initial Release

Open Caucus — a conference and committee management tool.

- **Committee management**: Create and configure committees with membership roles (chairperson, member).
- **Meeting lifecycle**: Create meetings, manage agenda points with sub-points, and run live sessions.
- **Speakers list**: Real-time speakers list with SSE live updates, gender quotation, priority toggles, and moderator assignment.
- **Voting**: Normalized voting lifecycle on agenda points with secret ballot receipts and public verification.
- **Attendee management**: Join flow with QR codes, guest sessions, and attendee self-service.
- **Authentication**: Account-based login with admin and user roles. OAuth/OIDC support with provider gating and automatic committee membership sync.
- **Agenda import**: Import agenda from text with correction and diff preview.
- **Attachments**: Upload and manage agenda point attachments.
- **Internationalization**: Locale-aware routing with English and German translations.
- **Real-time updates**: SSE-based live updates for speakers list, voting, and meeting state.
- **Resizable panels**: Responsive layout with resizable panels in the moderation view.
- **Documentation capture**: Scripted screenshot and GIF generation for documentation.
- **SPA architecture**: SvelteKit frontend with Connect (gRPC-web) API, fully decoupled from the legacy HTMX layer.
