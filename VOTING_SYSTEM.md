# Voting System

## Overview

The voting subsystem is normalized and built around:

- `vote_definitions`: vote configuration and lifecycle
- `vote_options`: selectable choices per vote
- `eligible_voters`: immutable eligibility snapshot per vote
- `vote_casts`: who cast/handed in a vote
- `vote_ballots`: persisted ballots and receipt token linkage
- `vote_ballot_selections`: selected options per ballot

Motions are removed end-to-end. Votes are bound to agenda points only. Ballot-level data is the source of truth.

## Lifecycle

Vote state machine:

- Open visibility votes: `draft -> open -> closed -> archived`
- Secret visibility votes: `draft -> open -> counting -> closed -> archived`

Rules:

- No reopen transitions.
- Vote definition edits and option edits are allowed only in `draft`.
- Eligibility snapshot is populated when opening and is immutable afterward.
- `CloseVote` is adaptive:
  - Open visibility votes always transition `open -> closed`.
  - Secret visibility votes transition `open -> counting` if `casts > secret_ballots`; otherwise `open -> closed`.
  - In `counting`, `CloseVote` is retryable and returns `still_counting` while `casts > secret_ballots`; once equal, it transitions `counting -> closed`.

## Eligibility Snapshot

`OpenVoteWithEligibleVoters(voteID, attendeeIDs)` is transactional:

1. Load and validate vote definition is in `draft`.
2. Insert `eligible_voters` rows for all provided attendees.
3. Set vote state to `open` and `opened_at`.
4. Commit, or rollback everything on any failure.

Eligibility is enforced by FK constraints when creating `vote_casts` and open ballots.

## Open vs Secret Ballots

There is no explicit `ballot_mode` column.

Ballot type is inferred from `vote_ballots.attendee_id`:

- Open ballot: `attendee_id IS NOT NULL`
  - Requires `cast_id IS NOT NULL`
  - Commitment fields must be `NULL`
- Secret ballot: `attendee_id IS NULL`
  - Requires `cast_id IS NULL`
  - Commitment fields must be non-`NULL`

This invariant is enforced by `CHECK` constraints.

## Submission Rules

Common:

- Selected option IDs must belong to the vote.
- Selection count must satisfy `min_selections <= selected <= max_selections`.

Open ballots:

- Vote must be in `open` state.
- Attendee must be eligible.
- Cast row is required and linked.
- Open ballot creation is transactional together with cast creation when needed.

Secret ballots:

- Vote may be in `open` or `counting` state.
- Ballot has no attendee or cast linkage.
- To prevent over-counting, insertion requires:
  - `secret_ballots_count < vote_casts_count`

Cast registration:

- `RegisterVoteCast` is only allowed in `open`.
- No new casts are accepted once vote state is `counting`, `closed`, or `archived`.

## Receipt Verification

Open ballots:

- Lookup key: `vote_id + receipt_token`
- Returned data: vote name, attendee number, selected options
- Intended for client-side participant proof/hash verification workflows

Secret ballots:

- Lookup key: `vote_id + receipt_token`
- Returned data: vote name + `encrypted_commitment` + `commitment_cipher` + `commitment_version`
- Nonce remains client-held and is not persisted server-side

Counting-phase read restriction:

- While vote state is `counting`, all result/verification reads are blocked:
  - `VerifyOpenBallotByReceipt`
  - `VerifySecretBallotByReceipt`
  - `GetVoteTallies`
  - `GetVoteSubmissionStats`

## UI Surface

Moderator flow (`/committee/{slug}/meeting/{meeting_id}/moderate` tools tab):

- create draft votes for the active agenda point
- edit draft vote metadata/options
- open vote (eligibility snapshot = all current attendees)
- adaptive close (`open -> counting/closed`, `counting -> closed`)
- archive closed votes
- register manual casts for secret votes
- count secret ballots in open/counting with receipt tokens
- see final tallies/stats once vote is closed/archived

Participant/live flow (`/committee/{slug}/meeting/{meeting_id}`):

- shows open votes for the active agenda point
- self-submit open or secret ballots
- receives receipt string after successful submission
- stores receipts in browser `IndexedDB`

Receipt vault and public verification:

- public page: `/receipts`
- public APIs:
  - `POST /api/votes/verify/open`
  - `POST /api/votes/verify/secret`
- counting-state restrictions apply to these endpoints as described above

SSE propagation:

- vote mutations publish `meeting-votes-changed`
- moderate and live SSE payloads include vote-panel OOB updates
- agenda-point activation/create/delete also publishes vote updates so the active-point vote panel stays in sync

## Security and Integrity Invariants

- Eligibility is snapshotted and immutable once vote opens.
- Double casts are blocked by `UNIQUE(vote_definition_id, attendee_id)` on `vote_casts`.
- Open ballots are one-per-attendee (`UNIQUE(vote_definition_id, attendee_id)`).
- Receipt tokens are unique per vote (`UNIQUE(vote_definition_id, receipt_token)`).
- Secret ballots remain identity-unlinked at ballot row level by schema design.
