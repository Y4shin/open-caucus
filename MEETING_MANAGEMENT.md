# Meeting Management — Design Overview

This document describes the data model and planned routes for meeting management,
combining the database schema with product decisions made during planning.

---

## Database schema (meeting-related tables)

### `meetings`
| Column | Type | Notes |
|---|---|---|
| `id` | INTEGER PK | |
| `committee_id` | INTEGER FK → committees | Cascade delete |
| `name` | TEXT | |
| `description` | TEXT | |
| `secret` | TEXT | Embedded in QR code for guest signup |
| `signup_open` | BOOLEAN | Gates **guest** self-registration only |
| `current_agenda_point_id` | INTEGER FK → agenda_points | Which point is live; set null on delete |
| `protocol_writer_id` | INTEGER FK → attendees | Assigned attendee; set null on delete |
| `created_at` / `updated_at` | TEXT | |

### `attendees`
| Column | Type | Notes |
|---|---|---|
| `id` | INTEGER PK | |
| `meeting_id` | INTEGER FK → meetings | Cascade delete |
| `user_id` | INTEGER FK → users (nullable) | NULL for guests |
| `full_name` | TEXT | |
| `secret` | TEXT | Used for attendee session login |
| `is_chair` | BOOLEAN | Chairperson designation for this meeting |
| `quoted` | BOOLEAN | Whether the attendee uses quoted speech in protocols; toggled by the attendee themselves |
| `created_at` | TEXT | |

### `agenda_points`
| Column | Type | Notes |
|---|---|---|
| `id` | INTEGER PK | |
| `meeting_id` | INTEGER FK → meetings | Cascade delete |
| `parent_id` | INTEGER FK → agenda_points (nullable) | Self-referencing for sub-points |
| `position` | INTEGER | Order within sibling group; unique per (meeting, parent) |
| `title` | TEXT | |
| `protocol` | TEXT | Written by the protocol writer |
| `current_speaker_id` | INTEGER FK → speakers_list | Which speaker is currently active |
| `created_at` / `updated_at` | TEXT | |

### `speakers_list`
| Column | Type | Notes |
|---|---|---|
| `id` | INTEGER PK | |
| `agenda_point_id` | INTEGER FK → agenda_points | Cascade delete |
| `attendee_id` | INTEGER FK → attendees | Cascade delete |
| `type` | TEXT | `regular` or `ropm` |
| `status` | TEXT | `WAITING` → `SPEAKING` → `DONE` or `WITHDRAWN` |
| `requested_at` | TEXT | When they joined the queue |
| `start_of_speech` | TEXT (nullable) | Set when status moves to SPEAKING |
| `duration` | TEXT (nullable) | Set when speech ends |

### `motions`
| Column | Type | Notes |
|---|---|---|
| `id` | INTEGER PK | |
| `agenda_point_id` | INTEGER FK → agenda_points | Cascade delete |
| `blob_id` | INTEGER FK → binary_blobs | The motion document |
| `title` | TEXT | |
| `votes_for/against/abstained/eligible` | INTEGER (nullable) | All null until vote is recorded |

### `binary_blobs` / `agenda_attachments`
- `binary_blobs` stores file metadata (filename, content-type, size, storage path)
- `agenda_attachments` links blobs to agenda points with an optional label

---

## Session types

There are two session types in the system:

| Session type | Who | Carries |
|---|---|---|
| `user` | Committee members logged in via `/login` | `user_id`, `committee_slug`, `username`, `role`, `quoted` |
| `attendee` | Meeting participants logged in via QR code or join page | `attendee_id`, `meeting_id`, `full_name`, `is_chair`, `quoted` |

---

## Route plan

### Access rules summary

| Page | Who can access | Session required |
|---|---|---|
| `/manage` | User with role `chairperson` | User session |
| `/join` (GET) | Anyone with the URL / QR code | None (public) |
| `/join` (POST, registered) | Any logged-in committee member | User session |
| `/guest` (POST, self-signup) | Anyone, only when `signup_open = true` | None |
| `/live` | Any signed-up attendee | Attendee session |
| `/protocol` | The assigned `protocol_writer_id` attendee | Attendee session |

---

### QR code / join page

**`GET /committee/{slug}/meeting/{meeting_id}/join`** — public

- Displays a QR code whose URL is this same page
- **Registered users** (user session present): see a one-click "Sign up for this meeting" button — always available regardless of `signup_open`
- **Guests** (no session): see a name-entry form — only rendered when `signup_open = true`; otherwise shows a "signup is closed" message

**`POST /committee/{slug}/meeting/{meeting_id}/join`** — registered user signup

- Requires a user session for this committee
- Creates an attendee row linked to `user_id`; idempotent (no duplicate if already signed up)
- Redirects to `/live`

**`POST /committee/{slug}/meeting/{meeting_id}/guest`** — guest self-signup

- No session required
- Only accepted when `signup_open = true`; returns 403 otherwise
- Creates an attendee row with `user_id = NULL`, generates a `secret`
- Returns the attendee secret (e.g., as a new QR code or display token) so the guest can log in to `/live`

---

### Meeting manage page (chairperson only)

**`GET /committee/{slug}/meeting/{meeting_id}/manage`**

Sections on this page:

#### Attendee management
- List of current attendees (name, is_chair, user or guest)
- **Add guest manually** — chairperson enters a name on behalf of someone; bypasses `signup_open`
- **Remove attendee** — with confirmation
- **Toggle `is_chair`** — promote/demote an attendee to co-chair for this meeting
- `quoted` is **not** editable here — attendees control their own setting on the live page

#### Meeting settings
- **Toggle `signup_open`** — open or close guest self-registration
- **Assign protocol writer** — pick from current attendees; sets `protocol_writer_id`
- **Advance agenda point** — set `current_agenda_point_id`

#### Speakers list controls
- View the current queue for the active agenda point
- **Add a speaker** — pick an attendee and type (regular / ropm)
- **Remove a speaker** from the queue
- **Start speech** — moves status from `WAITING` → `SPEAKING`, records `start_of_speech`
- **End speech** — moves status to `DONE`, records `duration`
- **Withdraw** — moves status to `WITHDRAWN`

#### Agenda management
- Create / edit / delete agenda points and sub-points
- Reorder (change `position`)
- Add / remove file attachments (`binary_blobs` → `agenda_attachments`)

#### Motions / voting
- Create a motion (attach a document blob, set title)
- Record vote result (for / against / abstained / eligible)

**Mutation routes** (all POST, chairperson session required, return HTMX partials):
```
POST /committee/{slug}/meeting/{meeting_id}/attendee/create       — add guest manually
POST /committee/{slug}/meeting/{meeting_id}/attendee/{id}/delete  — remove attendee
POST /committee/{slug}/meeting/{meeting_id}/attendee/{id}/chair   — toggle is_chair
POST /committee/{slug}/meeting/{meeting_id}/signup-open           — toggle signup_open
POST /committee/{slug}/meeting/{meeting_id}/protocol-writer       — assign protocol writer
POST /committee/{slug}/meeting/{meeting_id}/agenda-point/create
POST /committee/{slug}/meeting/{meeting_id}/agenda-point/{id}/edit
POST /committee/{slug}/meeting/{meeting_id}/agenda-point/{id}/delete
POST /committee/{slug}/meeting/{meeting_id}/agenda-point/{id}/move
POST /committee/{slug}/meeting/{meeting_id}/agenda-point/{id}/activate  — set as current
POST /committee/{slug}/meeting/{meeting_id}/speaker/add
POST /committee/{slug}/meeting/{meeting_id}/speaker/{id}/remove
POST /committee/{slug}/meeting/{meeting_id}/speaker/{id}/start
POST /committee/{slug}/meeting/{meeting_id}/speaker/{id}/end
POST /committee/{slug}/meeting/{meeting_id}/speaker/{id}/withdraw
POST /committee/{slug}/meeting/{meeting_id}/motion/create
POST /committee/{slug}/meeting/{meeting_id}/motion/{id}/vote
```

---

### Attendee live page

**`GET /committee/{slug}/meeting/{meeting_id}/live`** — attendee session required

What an attendee can do here:
- View the current agenda point and speaker
- View the speakers queue
- **Add themselves** to the speakers list (type: regular or ropm)
- **Withdraw themselves** from the queue
- **Toggle own `quoted` flag**

Mutation routes (attendee session required):
```
POST /committee/{slug}/meeting/{meeting_id}/live/speak      — add self to queue
POST /committee/{slug}/meeting/{meeting_id}/live/withdraw   — withdraw self
POST /committee/{slug}/meeting/{meeting_id}/live/quoted     — toggle own quoted flag
```

---

### Protocol page

**`GET /committee/{slug}/meeting/{meeting_id}/protocol`** — attendee session required, must be the assigned `protocol_writer_id`

- Read-only view of agenda structure, attendee list, and motion results
- Editable `protocol` text field per agenda point

Mutation route:
```
POST /committee/{slug}/meeting/{meeting_id}/protocol/{agenda_point_id} — save protocol text
```

---

## Suggested implementation order

1. **Attendee signup** (`/join`, `/guest`) — everything else depends on attendees existing
2. **Attendee session login** — so attendees can reach `/live` and `/protocol`
3. **Manage page skeleton** — attendee list, add guest, remove, toggle is_chair
4. **Toggle signup_open** — simple flag flip, unblocks guest self-registration
5. **Speakers list** — add/remove/start/end/withdraw
6. **Agenda management** — create/edit/delete/reorder points
7. **Protocol page** — write protocol text per agenda point
8. **Motions** — create motion, record vote
9. **File attachments** — binary blob upload and linking
