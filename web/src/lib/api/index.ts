import { createClient } from '@connectrpc/connect';
import { transport } from './transport.js';
import {
	SessionService,
	DocsService,
	CommitteeService,
	MeetingService,
	AttendeeService,
	AgendaService,
	SpeakerService,
	VoteService,
	AdminService,
	ModerationService
} from './services.js';

export const sessionClient = createClient(SessionService, transport);
export const docsClient = createClient(DocsService, transport);
export const committeeClient = createClient(CommitteeService, transport);
export const meetingClient = createClient(MeetingService, transport);
export const attendeeClient = createClient(AttendeeService, transport);
export const agendaClient = createClient(AgendaService, transport);
export const speakerClient = createClient(SpeakerService, transport);
export const voteClient = createClient(VoteService, transport);
export const adminClient = createClient(AdminService, transport);
export const moderationClient = createClient(ModerationService, transport);

export { transport };
