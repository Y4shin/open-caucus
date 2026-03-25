/**
 * V2-compatible service descriptors built from generated file descriptors.
 *
 * The BSR `buf.build/connectrpc/es` plugin currently generates v1-style
 * service types (using `MethodKind`) which are incompatible with
 * `@connectrpc/connect` v2. We build our own descriptors here using
 * `serviceDesc` from `@bufbuild/protobuf/codegenv2`.
 */
import { serviceDesc } from '@bufbuild/protobuf/codegenv2';

import { file_conference_session_v1_session } from '$lib/gen/conference/session/v1/session_pb.js';
import { file_conference_committees_v1_committees } from '$lib/gen/conference/committees/v1/committees_pb.js';
import { file_conference_meetings_v1_meetings } from '$lib/gen/conference/meetings/v1/meetings_pb.js';
import { file_conference_attendees_v1_attendees } from '$lib/gen/conference/attendees/v1/attendees_pb.js';
import { file_conference_agenda_v1_agenda } from '$lib/gen/conference/agenda/v1/agenda_pb.js';
import { file_conference_speakers_v1_speakers } from '$lib/gen/conference/speakers/v1/speakers_pb.js';
import { file_conference_votes_v1_votes } from '$lib/gen/conference/votes/v1/votes_pb.js';
import { file_conference_admin_v1_admin } from '$lib/gen/conference/admin/v1/admin_pb.js';
import { file_conference_moderation_v1_moderation } from '$lib/gen/conference/moderation/v1/moderation_pb.js';

// Each proto file has exactly one service at index 0.
export const SessionService = serviceDesc(file_conference_session_v1_session, 0);
export const CommitteeService = serviceDesc(file_conference_committees_v1_committees, 0);
export const MeetingService = serviceDesc(file_conference_meetings_v1_meetings, 0);
export const AttendeeService = serviceDesc(file_conference_attendees_v1_attendees, 0);
export const AgendaService = serviceDesc(file_conference_agenda_v1_agenda, 0);
export const SpeakerService = serviceDesc(file_conference_speakers_v1_speakers, 0);
export const VoteService = serviceDesc(file_conference_votes_v1_votes, 0);
export const AdminService = serviceDesc(file_conference_admin_v1_admin, 0);
export const ModerationService = serviceDesc(file_conference_moderation_v1_moderation, 0);
