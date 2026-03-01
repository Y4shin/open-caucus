# Voting System

## Overview

The voting subsystem is normalized and built around:

- `vote_definitions`: vote configuration and lifecycle
- `vote_options`: selectable choices per vote
- `eligible_voters`: immutable eligibility snapshot per vote
- `vote_casts`: who cast/handed in a vote
- `vote_ballots`: persisted ballots and receipt token linkage
- `vote_ballot_selections`: selected options per ballot

`motions.votes_*` legacy tally columns are removed. Ballot-level data is now the source of truth.

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

## Security and Integrity Invariants

- Eligibility is snapshotted and immutable once vote opens.
- Double casts are blocked by `UNIQUE(vote_definition_id, attendee_id)` on `vote_casts`.
- Open ballots are one-per-attendee (`UNIQUE(vote_definition_id, attendee_id)`).
- Receipt tokens are unique per vote (`UNIQUE(vote_definition_id, receipt_token)`).
- Secret ballots remain identity-unlinked at ballot row level by schema design.
