# Step 4 – Update all `.templ` helper methods to accept and propagate `ctx`

Every method on a template input struct that calls a path builder must:
1. Add `ctx context.Context` as its **first** parameter.
2. Pass `ctx, ""` to the path builder call.
3. Have its call site in the Templ component body updated to pass `ctx`.

Add `"context"` to the import block of each file that doesn't already have it.

Static file path methods (`HtmxMinJs()`, etc.) are **not** path builder calls – leave them as-is.

---

## Pattern

**Before:**
```go
func (i *CommitteePageInput) LogoutSubmitPost() templ.SafeURL {
    return templ.URL(paths.Route.LogoutSubmitPost())
}
```
```templ
<form hx-post={ string(input.LogoutSubmitPost()) }>
```

**After:**
```go
func (i *CommitteePageInput) LogoutSubmitPost(ctx context.Context) templ.SafeURL {
    return templ.URL(paths.Route.LogoutSubmitPost(ctx, ""))
}
```
```templ
<form hx-post={ string(input.LogoutSubmitPost(ctx)) }>
```

---

## File-by-file change list

The following summarises every method that needs to be updated. All are straightforward `(ctx, "")` additions unless noted.

### `internal/templates/login.templ`

| Method | Signature change |
|--------|-----------------|
| `LoginPageInput.LoginSubmitPost()` | add `ctx context.Context` |

---

### `internal/templates/admin_login.templ`

| Method | Signature change |
|--------|-----------------|
| `AdminLoginInput.AdminLoginSubmitPost()` | add `ctx context.Context` |

---

### `internal/templates/admin_dashboard.templ`

| Method | Signature change |
|--------|-----------------|
| `CommitteeItem.ManageUsersGet()` | add `ctx context.Context` |
| `CommitteeItem.DeletePost()` | add `ctx context.Context` |
| `CommitteeItem.DeletePostStr()` | add `ctx context.Context` |
| `AdminDashboardInput.PageURL(page int)` | add `ctx context.Context` as first param; also update `AdminDashboardGetWithQuery` call to `(ctx, "", q)` |
| `AdminDashboardInput.AdminLogoutPost()` | add `ctx context.Context` |
| `AdminDashboardInput.AdminCreateCommitteePost()` | add `ctx context.Context` |

---

### `internal/templates/admin_committee_users.templ`

| Method | Signature change |
|--------|-----------------|
| `AdminCommitteeUsersInput.PageURL(page int)` | add `ctx context.Context`; update `AdminCommitteeUsersGetWithQuery` call to `(ctx, "", q)` |
| `AdminCommitteeUsersInput.AdminDashboardGet()` | add `ctx context.Context` |
| `AdminCommitteeUsersInput.AdminCreateUserPost()` (if present) | add `ctx context.Context` |
| `AdminCommitteeUsersInput.AdminDeleteUserPost(user)` (if present) | add `ctx context.Context` |
| `UserListPartialInput.AdminCreateUserPost()` | add `ctx context.Context` |
| `UserListPartialInput.AdminDeleteUserPost(user)` | add `ctx context.Context` |

Check the full file – there may be additional helpers.

---

### `internal/templates/committee.templ`

| Method | Signature change |
|--------|-----------------|
| `CommitteePageInput.LogoutSubmitPost()` | add `ctx context.Context` |
| `CommitteePageInput.CommitteeCreateMeetingPost()` | add `ctx context.Context` |
| `CommitteePageInput.MeetingViewGet(m)` | add `ctx context.Context` |
| `CommitteePageInput.MeetingManageGet(m)` | add `ctx context.Context` |
| `CommitteePageInput.MeetingDeletePost(m)` | add `ctx context.Context` |
| `CommitteePageInput.MeetingActivatePost(m)` | add `ctx context.Context` |
| `CommitteePageInput.PageURL(page int)` | add `ctx context.Context`; update `CommitteePageGetWithQuery` call |
| `MeetingListPartialInput.CommitteeCreateMeetingPostStr()` | add `ctx context.Context` |
| `MeetingListPartialInput.MeetingViewGet(m)` | add `ctx context.Context` |
| `MeetingListPartialInput.MeetingManageGet(m)` | add `ctx context.Context` |
| `MeetingListPartialInput.MeetingDeletePostStr(m)` | add `ctx context.Context` |
| `MeetingListPartialInput.MeetingActivatePostStr(m)` | add `ctx context.Context` |
| `MeetingListPartialInput.PageURL(page int)` | add `ctx context.Context`; update `CommitteePageGetWithQuery` call |

---

### `internal/templates/meeting_manage.templ`

| Method | Signature change |
|--------|-----------------|
| `AttendeeListPartialInput.AttendeeCreatePostStr()` | add `ctx context.Context` |
| `AttendeeListPartialInput.AttendeeDeletePostStr(a)` | add `ctx context.Context` |
| `AttendeeListPartialInput.AttendeeToggleChairPostStr(a)` | add `ctx context.Context` |
| `MeetingSettingsPartialInput.ToggleSignupOpenPostStr()` | add `ctx context.Context` |
| `MeetingSettingsPartialInput.SetProtocolWriterPostStr()` | add `ctx context.Context` |
| `MeetingSettingsPartialInput.SetMeetingQuotationPostStr()` | add `ctx context.Context` |
| `MeetingSettingsPartialInput.SetMeetingModeratorPostStr()` | add `ctx context.Context` |
| `AgendaPointListPartialInput.AgendaPointCreatePostStr()` | add `ctx context.Context` |
| `AgendaPointListPartialInput.AgendaPointDeletePostStr(ap)` | add `ctx context.Context` |
| `AgendaPointListPartialInput.AgendaPointActivatePostStr(ap)` | add `ctx context.Context` |
| `SpeakersListPartialInput.SpeakerAddPostStr()` | add `ctx context.Context` |
| `SpeakersListPartialInput.SpeakerRemovePostStr(s)` | add `ctx context.Context` |
| `SpeakersListPartialInput.SpeakerStartPostStr(s)` | add `ctx context.Context` |
| `SpeakersListPartialInput.SpeakerEndPostStr(s)` | add `ctx context.Context` |
| `SpeakersListPartialInput.SpeakerWithdrawPostStr(s)` | add `ctx context.Context` |
| `SpeakersListPartialInput.SpeakerTogglePriorityPostStr(s)` | add `ctx context.Context` |
| `SpeakersListPartialInput.AgendaPointQuotationPostStr()` | add `ctx context.Context` |
| `SpeakersListPartialInput.AgendaPointModeratorPostStr()` | add `ctx context.Context` |
| `MeetingManageInput.CommitteeDashboardGet()` | add `ctx context.Context` |

---

### `internal/templates/motion.templ`

| Method | Signature change |
|--------|-----------------|
| `MotionListPartialInput.MotionCreatePostStr()` | add `ctx context.Context` |
| `MotionItemPartialInput.MotionDeletePostStr()` | add `ctx context.Context` |
| `MotionItemPartialInput.MotionVotePostStr()` | add `ctx context.Context` |
| `MotionItemPartialInput.BlobDownloadGetStr()` | add `ctx context.Context` |

---

### `internal/templates/attachment.templ`

| Method | Signature change |
|--------|-----------------|
| `AttachmentListPartialInput.AttachmentCreatePostStr()` | add `ctx context.Context` |
| `AttachmentListPartialInput.AttachmentDeletePostStr(a)` | add `ctx context.Context` |
| `AttachmentListPartialInput.BlobDownloadGetStr(a)` (if present) | add `ctx context.Context` |

---

### `internal/templates/meeting_join.templ`

| Method | Signature change |
|--------|-----------------|
| `MeetingJoinInput.MeetingJoinSubmitPost()` | add `ctx context.Context` |
| `MeetingJoinInput.MeetingGuestSignupPost()` | add `ctx context.Context` |
| `MeetingJoinInput.MeetingLivePageGet()` | add `ctx context.Context` |
| `MeetingJoinInput.AttendeeLoginPageGet()` | add `ctx context.Context` |

---

### `internal/templates/meeting_view.templ`

| Method | Signature change |
|--------|-----------------|
| `MeetingViewInput.CommitteePageGet()` (or similar back-link method) | add `ctx context.Context` |

---

### `internal/templates/meeting_live.templ`

| Method | Signature change |
|--------|-----------------|
| `MeetingLiveInput.AttendeeLoginPageGet()` | add `ctx context.Context` |

---

### `internal/templates/meeting_protocol.templ`

| Method | Signature change |
|--------|-----------------|
| `MeetingProtocolInput.ProtocolSavePostStr(agendaPointID)` (or similar) | add `ctx context.Context` |
| `MeetingProtocolInput.MeetingManageGet()` | add `ctx context.Context` |

---

### `internal/templates/attendee_login.templ`

| Method | Signature change |
|--------|-----------------|
| `AttendeeLoginInput.AttendeeLoginSubmitPost()` | add `ctx context.Context` |

---

## Templ call-site pattern

Inside every Templ component body where a helper method is called, add `ctx` as the first argument. For example, inside `templ CommitteeListPartial(input CommitteeListPartialInput)`:

```templ
// Before
<form hx-post={ input.CommitteeCreateMeetingPostStr() }>

// After
<form hx-post={ input.CommitteeCreateMeetingPostStr(ctx) }>
```

`ctx` is the implicit request context available in every Templ component body – no extra wiring needed.

---

## Tip: finding all call sites after the method signature changes

Once step 5 (code generation) is done, compile errors from `go build ./...` will point to every call site that still uses the old signature. Use the error list to drive the updates rather than trying to find them manually.
