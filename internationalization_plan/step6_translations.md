# Step 6 – Add translation YAML files and replace hardcoded strings

This step is independent of the path-builder wiring and can be done incrementally, one template at a time.

---

## Setup

`ctxi18n` expects translation files to be loaded via `embed.FS`. The files live in `locales/` at the repo root. The `cmd/serve.go` changes in step 3 already handle loading them.

The YAML key structure is up to you – a nested structure by feature area is recommended.

---

## Translation key pattern in templates

```go
import "github.com/invopop/ctxi18n/i18n"

templ LoginPageTemplate(input LoginPageInput) {
    <h1>{ i18n.T(ctx, "login.title") }</h1>
    <label>{ i18n.T(ctx, "login.username_label") }</label>
    <button>{ i18n.T(ctx, "login.button") }</button>
}
```

`ctx` is the implicit context in every Templ component, already populated by the locale middleware.

---

## Full string catalogue

The following is a complete list of hardcoded UI strings identified in the templates, organised by file.

### `login.templ`

```yaml
login:
  title: "Conference Tool Login"
  committee_label: "Committee:"
  username_label: "Username:"
  password_label: "Password:"
  button: "Login"
```

### `admin_login.templ`

```yaml
admin_login:
  title: "Admin Login"
  password_label: "Password:"
  button: "Login"
```

### `admin_dashboard.templ`

```yaml
admin_dashboard:
  title: "Admin Dashboard"
  committees_heading: "Committees"
  add_committee_heading: "Add New Committee"
  name_label: "Committee Name:"
  slug_label: "Slug (URL-friendly identifier):"
  create_button: "Create Committee"
  existing_heading: "Existing Committees"
  col_name: "Name"
  col_slug: "Slug"
  col_actions: "Actions"
  manage_users_link: "Manage Users"
  delete_button: "Delete"
  delete_confirm: "Are you sure you want to delete this committee?"
  logout_button: "Logout"
  empty_state: "No committees yet."
```

### `admin_committee_users.templ`

```yaml
admin_committee_users:
  username_label: "Username:"
  password_label: "Password:"
  fullname_label: "Full Name:"
  create_button: "Create User"
  back_link: "← Back to Dashboard"
  col_username: "Username"
  col_fullname: "Full Name"
  col_actions: "Actions"
  delete_button: "Delete"
  delete_confirm: "Are you sure you want to delete this user?"
```

### `committee.templ`

```yaml
committee:
  dashboard_heading: "Committee Dashboard"
  welcome: "Welcome to the committee management system."
  create_meeting_heading: "Create New Meeting"
  name_label: "Name:"
  description_label: "Description:"
  signup_label: "Open for signup"
  create_button: "Create Meeting"
  empty_state: "No meetings yet."
  col_name: "Name"
  col_description: "Description"
  col_signup: "Signup Open"
  col_active: "Active"
  col_actions: "Actions"
  view_link: "View"
  manage_link: "Manage"
  activate_button: "Set Active"
  delete_button: "Delete"
  delete_confirm: "Delete this meeting?"
  logged_in_as: "Logged in as: %s (%s)"   # formatted with username and role
  logout_button: "Logout"
```

### `pagination.templ`

```yaml
pagination:
  previous: "← Previous"
  next: "Next →"
  page_info: "Page %d of %d"
```

### `meeting_manage.templ`

```yaml
meeting_manage:
  attendees_heading: "Attendees"
  add_attendee_heading: "Add Attendee"
  fullname_label: "Full Name:"
  add_button: "Add"
  col_name: "Name"
  col_chair: "Chair"
  col_guest: "Guest"
  col_actions: "Actions"
  remove_button: "Remove"
  toggle_chair_button: "Toggle Chair"
  delete_confirm: "Remove this attendee?"
  agenda_heading: "Agenda Points"
  add_agenda_point_heading: "Add Agenda Point"
  title_label: "Title:"
  create_button: "Create"
  activate_button: "Activate"
  delete_ap_confirm: "Delete this agenda point?"
  speakers_heading: "Speakers"
  add_speaker_button: "Add to Speakers List"
  start_button: "Start"
  end_button: "End"
  withdraw_button: "Withdraw"
  priority_button: "Toggle Priority"
  back_link: "← Back to Committee"
  settings_heading: "Meeting Settings"
  signup_open_label: "Signup open"
  protocol_writer_label: "Protocol writer:"
  quotation_label: "Gender quotation:"
  moderator_label: "Moderator:"
```

### `meeting_view.templ` / `meeting_join.templ` / `meeting_live.templ`

Inspect each file and catalogue strings using the same pattern. These pages are not fully enumerated here – use the template source as the ground truth.

### `motion.templ`

```yaml
motion:
  title_label: "Title:"
  file_label: "Document:"
  upload_button: "Upload Motion"
  vote_for_label: "For:"
  vote_against_label: "Against:"
  vote_abstained_label: "Abstained:"
  vote_eligible_label: "Eligible:"
  record_vote_button: "Record Vote"
  delete_button: "Delete"
  delete_confirm: "Delete this motion?"
```

### `attachment.templ`

```yaml
attachment:
  upload_button: "Upload Attachment"
  delete_button: "Delete"
  delete_confirm: "Delete this attachment?"
```

---

## Language switcher UI

Add a small language picker to the layout (e.g., in the navigation bar). Since this is a standard form POST with no HTMX, it does a full page reload which ensures all locale-aware content is re-rendered:

```templ
<form method="POST" action="/locale">
    <input type="hidden" name="lang" value="de"/>
    <button type="submit">DE</button>
</form>
<form method="POST" action="/locale">
    <input type="hidden" name="lang" value="en"/>
    <button type="submit">EN</button>
</form>
```

Or use a single `<select>` with JavaScript form submission.

---

## Notes on `ctxi18n` API

Check the actual `ctxi18n` package for the exact `WithLocale` signature – it may require a `language.Tag` instead of a plain string:

```go
import "golang.org/x/text/language"

ctx = ctxi18n.WithLocale(ctx, language.Make(loc))
```

In that case update `middleware.go` accordingly (the `detect` function already returns a plain string; just convert at the `ctxi18n.WithLocale` call site).
