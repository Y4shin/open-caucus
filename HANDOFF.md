# HANDOFF — 2026-04-02 (evening)

## Session summary

Two topics were covered:

1. **Authentik integration** — attempted, then abandoned and fully reverted
2. **OIDC first-login bug** — root cause identified, fix not yet applied

---

## 1. Authentik integration (reverted)

All changes were reverted. Repo is back to the original OIDC dev-tool state:
- `docker-compose.yml` — original (OIDC dev tool with `setup` + `oidc` + `app` services)
- `docker-compose.desktop.yml` — original
- `Taskfile.yaml` — original
- `dev/authentik/` directory — deleted

---

## 2. OIDC first-login bug

### Symptom

When a user logs in via OIDC for the **first time ever** (no account exists), the callback silently redirects to the login page (`/`). The **second attempt** immediately succeeds.

### Root cause

**File**: `internal/repository/sqlite/repository.go`, `UpsertOAuthIdentity` (line ~285)

The repository's hand-written `UpsertOAuthIdentity` does two things:
1. Runs an `ExecContext` INSERT with a double `ON CONFLICT` clause
2. Immediately follows with a separate `GetOAuthIdentityByIssuerSubject` SELECT to return the inserted row

The double `ON CONFLICT` is actually valid SQLite (multiple upsert targets supported since 3.24), so the INSERT itself runs fine. The problem is the trailing SELECT: **both call sites ignore the returned `*model.OAuthIdentity`** — they only check the error — so the SELECT is pure overhead. If it fails (any error, including transient connection-pool issues), `UpsertOAuthIdentity` returns an error, `resolveOAuthAccount` logs a warning and redirects to `/`.

On the **second login attempt**, `GetOAuthIdentityByIssuerSubject` at the top of `resolveOAuthAccount` finds the identity that was successfully inserted during the first attempt (the INSERT succeeded, only the post-INSERT SELECT failed). Login then takes the "identity found" fast path and succeeds.

The second `ON CONFLICT (account_id)` clause is also semantically wrong: it would silently reassign an identity's issuer/subject if a different OIDC identity tries to claim the same account. An `account_id` uniqueness violation should be a hard error.

### Fix (not yet applied)

Rewrite `UpsertOAuthIdentity` in `internal/repository/sqlite/repository.go`:
- Remove the second `ON CONFLICT (account_id) DO UPDATE` clause
- Remove the trailing `GetOAuthIdentityByIssuerSubject` call — return `nil, nil` on success

```go
func (r *Repository) UpsertOAuthIdentity(
    ctx context.Context,
    issuer, subject string,
    accountID int64,
    username, fullName, email *string,
    groupsJSON *string,
) (*model.OAuthIdentity, error) {
    toNull := func(v *string) sql.NullString {
        if v == nil {
            return sql.NullString{}
        }
        return sql.NullString{String: *v, Valid: true}
    }
    if _, err := r.DB.ExecContext(
        ctx,
        `INSERT INTO oauth_identities (
             issuer, subject, account_id, username, full_name, email, groups_json, created_at, updated_at
         )
         VALUES (?, ?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'))
         ON CONFLICT (issuer, subject) DO UPDATE
         SET account_id = excluded.account_id,
             username = excluded.username,
             full_name = excluded.full_name,
             email = excluded.email,
             groups_json = excluded.groups_json,
             updated_at = datetime('now')`,
        issuer,
        subject,
        accountID,
        toNull(username),
        toNull(fullName),
        toNull(email),
        toNull(groupsJSON),
    ); err != nil {
        return nil, fmt.Errorf("upsert oauth identity: %w", err)
    }
    return nil, nil
}
```

No other files need changing. The `oauth.sql` SQLC query file was already fixed in a prior commit.

### Other small fix applied this session

`.env` was corrected to match what `oidc-dev populate-env` registers:
- `PORT=8080` (was 8081)
- `OAUTH_REDIRECT_URL=http://127.0.0.1:8080/oauth/callback` (was 8081)

---

## Prior work (unchanged)

The previous HANDOFF is backed up at `HANDOFF.bak.md`. It documents the full UI parity expansion and legacy HTML proxy-removal migration (Phases 1–10), all of which are complete and committed.

The unrelated dirty change in `e2e/voting_test.go` mentioned in that file still exists — leave it unless a future task touches that file.
