-- name: CreateOAuthAccount :one
INSERT INTO accounts (username, full_name, email, auth_method, created_at, updated_at)
VALUES (?, ?, ?, 'oauth', datetime('now'), datetime('now'))
RETURNING *;

-- name: UpdateAccountProfile :exec
UPDATE accounts SET full_name = ?, email = ?, updated_at = datetime('now') WHERE id = ?;

-- name: GetOAuthIdentityByIssuerSubject :one
SELECT * FROM oauth_identities
WHERE issuer = ? AND subject = ?;

-- name: UpsertOAuthIdentity :one
INSERT INTO oauth_identities (
    issuer,
    subject,
    account_id,
    username,
    full_name,
    email,
    groups_json,
    created_at,
    updated_at
)
VALUES (?, ?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'))
ON CONFLICT (issuer, subject) DO UPDATE
SET account_id = excluded.account_id,
    username = excluded.username,
    full_name = excluded.full_name,
    email = excluded.email,
    groups_json = excluded.groups_json,
    updated_at = datetime('now')
RETURNING *;

-- name: ListAllOAuthCommitteeGroupRules :many
SELECT * FROM oauth_committee_group_rules
ORDER BY committee_id, group_name;

-- name: ListOAuthCommitteeGroupRulesByCommitteeSlug :many
SELECT
    r.id,
    r.committee_id,
    r.group_name,
    r.role,
    r.created_at,
    r.updated_at,
    c.slug AS committee_slug
FROM oauth_committee_group_rules r
JOIN committees c ON c.id = r.committee_id
WHERE c.slug = ?
ORDER BY r.group_name;

-- name: CreateOAuthCommitteeGroupRuleByCommitteeSlug :one
INSERT INTO oauth_committee_group_rules (
    committee_id,
    group_name,
    role,
    created_at,
    updated_at
)
SELECT c.id, ?, ?, datetime('now'), datetime('now')
FROM committees c
WHERE c.slug = ?
RETURNING *;

-- name: DeleteOAuthCommitteeGroupRuleByIDAndCommitteeSlug :exec
DELETE FROM oauth_committee_group_rules
WHERE oauth_committee_group_rules.id = ?
  AND oauth_committee_group_rules.committee_id = (
      SELECT c.id
      FROM committees c
      WHERE c.slug = ?
  );

-- name: ListMembershipsForAccountWithOAuthManaged :many
SELECT
    u.id,
    u.account_id,
    u.committee_id,
    u.role,
    u.quoted,
    u.created_at,
    u.updated_at,
    CASE WHEN om.user_id IS NULL THEN 0 ELSE 1 END AS oauth_managed
FROM users u
LEFT JOIN oauth_managed_memberships om ON om.user_id = u.id
WHERE u.account_id = ?;

-- name: UpsertOAuthManagedMembership :exec
INSERT INTO oauth_managed_memberships (user_id, last_synced_at)
VALUES (?, datetime('now'))
ON CONFLICT (user_id) DO UPDATE
SET last_synced_at = datetime('now');

-- name: UpdateMembershipRole :exec
UPDATE users
SET role = ?, updated_at = datetime('now')
WHERE id = ?;
