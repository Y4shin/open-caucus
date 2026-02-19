# Speakers List: Quotation, Ordering, Moderator & Priority — Overview

## Background

The speakers list currently stores entries (type + status) but has no ordering logic, no gender quotation awareness, and no moderator support. This improvement set adds:

1. **Gender quotation** — quoted speakers (attendees with `quoted = true`) get priority in the speakers list when gender quotation is enabled.
2. **First-speaker bonus** — speakers addressing an agenda point for the first time get additional priority within their gender group.
3. **Ordering column** — an `order_position` integer is stored on every WAITING speaker row and recomputed after every mutation, so the UI can ORDER BY this column without application-layer sorting.
4. **Moderator** — both meetings and agenda points can have an optional moderator (foreign key into the attendees table).
5. **Priority toggle** — chairpersons can manually promote a speaker to the front of the queue.

## Ordering Algorithm

For WAITING speakers on a given agenda point, `order_position` is assigned by sorting on:

| Key | Direction | Meaning |
|-----|-----------|---------|
| `priority` | DESC | Manually promoted speakers go first |
| `gender_quoted` | DESC | Quoted speakers before non-quoted |
| `first_speaker` | DESC | First-timers before returning speakers (within each gender group) |
| `requested_at` | ASC | Tie-break by earliest request |

After the sort, positions 1, 2, 3, … are assigned to WAITING speakers. The `order_position` column is meaningless for non-WAITING entries; the query uses a CASE expression to place SPEAKING at 0 and DONE/WITHDRAWN after all WAITING entries.

**Key design decision**: `gender_quoted` and `first_speaker` are snapshots stored at insert time (not computed at query time).

- `gender_quoted` = `attendee.quoted AND effective_gender_quotation_enabled`, where the effective setting is the agenda point's value if set (not NULL), otherwise the meeting's value.
- `first_speaker` = no prior SPEAKING or DONE entry exists for (agenda_point_id, attendee_id) at the time of the request.

Recomputation of `order_position` is triggered after:
- A new speaker is added
- A speaker is removed, withdrawn, starts, or finishes speaking
- A speaker's priority is toggled
- A meeting's or agenda point's quotation settings are changed

## Phase Structure

| Phase | File | Scope |
|-------|------|-------|
| 1 | [01_database_schema.md](01_database_schema.md) | Migration, SQL queries, `task generate:db` |
| 2 | [02_models_and_repository.md](02_models_and_repository.md) | Model structs, repository interface, repository implementation, fix `AddSpeaker` call site |
| 3 | [03_manage_ui.md](03_manage_ui.md) | New routes, handlers, template changes for the chairperson management page |
| 4 | [04_attendee_speaker_ui.md](04_attendee_speaker_ui.md) | Attendee-facing UI updates and SSE integration (may be deferred) |

Each phase ends with a verification step. Complete and verify each phase before starting the next.

## Key Files at a Glance

- Migrations: `internal/repository/sqlite/migrations/`
- SQL queries: `internal/repository/sqlite/queries/`
- Generated DB client: `internal/repository/sqlite/client/` (**do not edit manually**)
- Models: `internal/repository/model/`
- Repository interface: `internal/repository/repository.go`
- Repository implementation: `internal/repository/sqlite/repository.go`
- Routes: `routes.yaml` → `task generate:routes` → `internal/routes/routes_gen.go` (**do not edit manually**)
- Templates: `internal/templates/*.templ` → `task generate:templates` → `*_templ.go` (**do not edit manually**)
- Handlers: `internal/handlers/`
- E2E tests: `e2e/` (build tag `e2e`, run with `task test:e2e`)
