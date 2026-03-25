# E2E To API Mapping Matrix

## Purpose

This document links the existing browser E2E scenarios to the new typed API contract and the API integration tests that should be created before the legacy SSR/HTMX implementation is removed.

The first version of this matrix covers the initial Phase 1 contract slice:

- `conference.session.v1`
- `conference.committees.v1`
- `conference.meetings.v1`
- `conference.attendees.v1`
- `conference.moderation.v1`
- companion meeting-scoped SSE transport

## First-Slice Acceptance Criteria

The first contract slice is acceptable when all of the following are true:

- the contract scaffold exists in `proto/` and can be linted/generated with `buf`
- `SessionService` can model anonymous bootstrap, password login bootstrap, and logout
- `CommitteeService` can model the `/home` and `/committee/[slug]` read paths
- `MeetingService.GetJoinMeeting` can model the `/committee/[slug]/meeting/[meetingId]/join` and `/attendee-login` read paths
- `MeetingService.GetLiveMeeting` can model the attendee/member live screen read path
- `AttendeeService` can model account signup, guest signup, and attendee secret login
- `ModerationService.GetModerationView` can model the `/committee/[slug]/meeting/[meetingId]/moderate` read path
- `ModerationService.ToggleSignupOpen` can represent the first proving mutation with explicit target state and version metadata
- meeting-scoped SSE invalidation for moderation/live refetch is documented and implemented as a companion transport path
- the API integration equivalents for the mapped first-slice workflows are documented before legacy behavior for those workflows is removed

## Mapping Rules

- `API surface touched` lists the typed contract first and then any transport exceptions that still belong to the workflow
- `Draft API integration tests` are proposed names, not yet implemented tests
- `Parity status` means:
  - `mapped`: covered by this document and expected to become an API integration test in the first slice
  - `later slice`: outside the initial contract slice

## First-Slice Matrix

| E2E scenario | User workflow | SPA screen / route | API surface touched | Draft API integration tests | Notes | Parity status |
| --- | --- | --- | --- | --- | --- | --- |
| `TestChairpersonSeesCreateMeetingForm` in [committee_test.go](/home/patric/Projects/conference-tool/e2e/committee_test.go#L23) | Logged-in chair opens committee dashboard and receives chair-capable meeting overview | `/home` then `/committee/[slug]` | `SessionService.GetSession`, `CommitteeService.ListMyCommittees`, `CommitteeService.GetCommitteeOverview` | `TestSessionService_GetSession_UserBootstrap`, `TestCommitteeService_ListMyCommittees_Chairperson`, `TestCommitteeService_GetCommitteeOverview_ChairpersonCapabilities` | Create-meeting mutation is outside the first slice; this row maps the read/bootstrap part only | `mapped` |
| `TestMemberSeesActiveMeetingInfoAndJoinButton` in [committee_test.go](/home/patric/Projects/conference-tool/e2e/committee_test.go#L53) | Logged-in member sees active meeting and can navigate toward join flow | `/committee/[slug]` | `CommitteeService.GetCommitteeOverview` | `TestCommitteeService_GetCommitteeOverview_MemberSeesActiveMeetingJoinCapability` | Join navigation target is plain route navigation; member-specific overview data belongs in the typed contract | `mapped` |
| `TestAttendeeLogin_ValidSecret` in [attendee_login_test.go](/home/patric/Projects/conference-tool/e2e/attendee_login_test.go#L36) | Guest attendee logs in with secret and reaches live screen | `/committee/[slug]/meeting/[meetingId]/attendee-login` then `/committee/[slug]/meeting/[meetingId]` | `MeetingService.GetJoinMeeting`, `AttendeeService.AttendeeLogin`, `MeetingService.GetLiveMeeting` | `TestMeetingService_GetJoinMeeting_GuestEntry`, `TestAttendeeService_AttendeeLogin_CreatesGuestSession`, `TestMeetingService_GetLiveMeeting_AttendeeBootstrap` | The SPA now uses typed attendee login and then hydrates session state from the cookie-backed bootstrap flow | `mapped` |
| `TestAttendeeLogin_InvalidSecret` in [attendee_login_test.go](/home/patric/Projects/conference-tool/e2e/attendee_login_test.go#L63) | Guest attendee receives a validation error for an invalid secret | `/committee/[slug]/meeting/[meetingId]/attendee-login` | `MeetingService.GetJoinMeeting`, `AttendeeService.AttendeeLogin` | `TestMeetingService_GetJoinMeeting_GuestEntry`, `TestAttendeeService_AttendeeLogin_InvalidSecret` | Validation now comes back from the typed attendee login command instead of a plain HTML form submit | `mapped` |
| `TestLivePage_RequiresAttendeeSession` in [attendee_login_test.go](/home/patric/Projects/conference-tool/e2e/attendee_login_test.go#L124) | Unauthenticated visitor is rejected from live meeting screen | `/committee/[slug]/meeting/[meetingId]` | `MeetingService.GetLiveMeeting`, `SessionService.GetSession` | `TestMeetingService_GetLiveMeeting_RequiresMeetingActor`, `TestSessionService_GetSession_Anonymous` | Browser redirect behavior may stay transport-specific; authorization rule belongs in API coverage | `mapped` |
| `TestModeratePage_ChairpersonCanAccess` in [moderate_test.go](/home/patric/Projects/conference-tool/e2e/moderate_test.go#L18) | Chairperson opens moderate screen and receives the expected read model/capabilities | `/committee/[slug]/meeting/[meetingId]/moderate` | `ModerationService.GetModerationView` | `TestModerationService_GetModerationView_Chairperson` | This is the main proving read path for the first moderation slice | `mapped` |
| `TestModeratePage_AttendeeNonChair_Forbidden` in [moderate_test.go](/home/patric/Projects/conference-tool/e2e/moderate_test.go#L45) | Non-chair attendee is forbidden from the moderate screen | `/committee/[slug]/meeting/[meetingId]/moderate` | `ModerationService.GetModerationView` | `TestModerationService_GetModerationView_ForbidsNonChairAttendee` | Keeps permission rules anchored to current behavior before transport/UI changes | `mapped` |
| `TestManagePage_ToggleSignupOpen` in [manage_test.go](/home/patric/Projects/conference-tool/e2e/manage_test.go#L457) | Moderator toggles signup availability without a full reload | `/committee/[slug]/meeting/[meetingId]/moderate` | `ModerationService.GetModerationView`, `ModerationService.ToggleSignupOpen`, `GET /api/realtime/meetings/{meetingId}/events` | `TestModerationService_ToggleSignupOpen_ChangesState`, `TestModerationService_ToggleSignupOpen_RejectsStaleVersion`, `TestRealtime_MeetingEvents_PublishesSignupInvalidation` | This is the first proving mutation and realtime invalidation path for the rewrite | `mapped` |
| `TestJoinPage_GuestSeesFormWhenSignupOpen` in [join_test.go](/home/patric/Projects/conference-tool/e2e/join_test.go#L51) | Anonymous guest sees join form only while signup is open | `/committee/[slug]/meeting/[meetingId]/join` | `MeetingService.GetJoinMeeting` | `TestMeetingService_GetJoinMeeting_AnonymousGuestSeesGuestJoin` | The join screen now reads its open/closed state directly from the typed join read model | `mapped` |
| `TestJoinPage_GuestSeesClosedMessageWhenSignupClosed` in [join_test.go](/home/patric/Projects/conference-tool/e2e/join_test.go#L75) | Anonymous guest sees closed-state messaging when signup is disabled | `/committee/[slug]/meeting/[meetingId]/join` | `MeetingService.GetJoinMeeting`, `ModerationService.ToggleSignupOpen` | `TestMeetingService_GetJoinMeeting_ClosedSignupState`, `TestModerationService_ToggleSignupOpen_ClosedStateMatchesJoinView` | This keeps the guest entry screen aligned with the moderation toggle through the typed contract | `mapped` |
| `TestManagePage_CrossTab_AttendeeChangePropagates` in [manage_test.go](/home/patric/Projects/conference-tool/e2e/manage_test.go#L265) | One manage session updates another via SSE | `/committee/[slug]/meeting/[meetingId]/moderate` | `GET /api/realtime/meetings/{meetingId}/events` | `TestRealtime_MeetingEvents_AreScopedPerMeeting` | The specific attendee-add mutation is later-slice work, but the meeting-scoped invalidation model starts here | `later slice` |
| `TestSync_LiveAndManage_SpeakerLifecycleUpdates` in [session_sync_test.go](/home/patric/Projects/conference-tool/e2e/session_sync_test.go#L24) | Speaker state changes in moderation propagate to live session | `/committee/[slug]/meeting/[meetingId]/moderate` and `/committee/[slug]/meeting/[meetingId]` | `MeetingService.GetLiveMeeting`, `GET /api/realtime/meetings/{meetingId}/events` | `TestMeetingService_GetLiveMeeting_RefetchAfterRealtimeInvalidation` | The speaker mutations themselves are later-slice work, but this remains a target scenario for the invalidation model | `later slice` |

## Immediate Follow-Up

1. Validate the new `proto/` contract with `buf lint` and `buf generate` once `buf` is available in the active environment.
2. Split the first API integration test work into:
   - typed contract tests for `session`, `committees`, `meetings`, `attendees`, and `moderation`
   - meeting-scoped realtime transport tests
3. Extend this matrix row-by-row before removing any legacy workflow implementation.
