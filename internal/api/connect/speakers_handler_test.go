package apiconnect

import (
	"context"
	"fmt"
	"testing"

	connect "connectrpc.com/connect"

	attendeesv1 "github.com/Y4shin/conference-tool/gen/go/conference/attendees/v1"
	sessionv1 "github.com/Y4shin/conference-tool/gen/go/conference/session/v1"
	speakersv1 "github.com/Y4shin/conference-tool/gen/go/conference/speakers/v1"
)

func TestSpeakerService_AddSpeaker_SelfForAttendeeSession(t *testing.T) {
	ts := newCombinedAPITestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	meetingID := ts.seedMeeting(t, "test-committee", "Open Meeting", true)

	agendaPoint, err := ts.repo.CreateAgendaPoint(context.Background(), meetingID, "Main Topic")
	if err != nil {
		t.Fatalf("create agenda point: %v", err)
	}
	if err := ts.repo.SetCurrentAgendaPoint(context.Background(), meetingID, &agendaPoint.ID); err != nil {
		t.Fatalf("set current agenda point: %v", err)
	}

	guest, err := ts.repo.CreateAttendee(context.Background(), meetingID, nil, "Guest Speaker", "secret-guest", false)
	if err != nil {
		t.Fatalf("create attendee: %v", err)
	}

	client := newCombinedTestClient(t, ts)

	if _, err := client.attendees.AttendeeLogin(context.Background(), connect.NewRequest(&attendeesv1.AttendeeLoginRequest{
		CommitteeSlug:  "test-committee",
		MeetingId:      fmt.Sprintf("%d", meetingID),
		AttendeeSecret: "secret-guest",
	})); err != nil {
		t.Fatalf("attendee login: %v", err)
	}

	resp, err := client.speakers.AddSpeaker(context.Background(), connect.NewRequest(&speakersv1.AddSpeakerRequest{
		CommitteeSlug: "test-committee",
		MeetingId:     fmt.Sprintf("%d", meetingID),
		SpeakerType:   "regular",
	}))
	if err != nil {
		t.Fatalf("add speaker: %v", err)
	}

	view := resp.Msg.GetView()
	if !view.GetCanAddSelf() {
		t.Fatal("expected can_add_self=true for attendee session")
	}
	if len(view.GetSpeakers()) != 1 {
		t.Fatalf("expected 1 speaker row, got %d", len(view.GetSpeakers()))
	}
	if got := view.GetSpeakers()[0].GetAttendeeId(); got != fmt.Sprintf("%d", guest.ID) {
		t.Fatalf("unexpected attendee id on speaker row: %q", got)
	}
	if !view.GetSpeakers()[0].GetMine() {
		t.Fatal("expected added speaker row to be marked mine=true")
	}
}

func TestSpeakerService_SetSpeakerSpeaking_ThenDone(t *testing.T) {
	ts := newCombinedAPITestServer(t)
	ts.seedCommittee(t, "Test Committee", "test-committee")
	ts.seedUser(t, "test-committee", "chair1", "pass123", "Chair Person", "chairperson")
	meetingID := ts.seedMeeting(t, "test-committee", "Board Meeting", true)

	agendaPoint, err := ts.repo.CreateAgendaPoint(context.Background(), meetingID, "Main Topic")
	if err != nil {
		t.Fatalf("create agenda point: %v", err)
	}
	if err := ts.repo.SetCurrentAgendaPoint(context.Background(), meetingID, &agendaPoint.ID); err != nil {
		t.Fatalf("set current agenda point: %v", err)
	}

	attendee, err := ts.repo.CreateAttendee(context.Background(), meetingID, nil, "Queued Speaker", "secret-queued", false)
	if err != nil {
		t.Fatalf("create attendee: %v", err)
	}
	entry, err := ts.repo.AddSpeaker(context.Background(), agendaPoint.ID, attendee.ID, "regular", false, false)
	if err != nil {
		t.Fatalf("seed speaker: %v", err)
	}
	if err := ts.repo.RecomputeSpeakerOrder(context.Background(), agendaPoint.ID); err != nil {
		t.Fatalf("recompute speaker order: %v", err)
	}

	client := newCombinedTestClient(t, ts)

	if _, err := client.session.Login(context.Background(), connect.NewRequest(&sessionv1.LoginRequest{
		Username: "chair1",
		Password: "pass123",
	})); err != nil {
		t.Fatalf("login: %v", err)
	}

	startResp, err := client.speakers.SetSpeakerSpeaking(context.Background(), connect.NewRequest(&speakersv1.SetSpeakerSpeakingRequest{
		CommitteeSlug: "test-committee",
		MeetingId:     fmt.Sprintf("%d", meetingID),
		SpeakerId:     fmt.Sprintf("%d", entry.ID),
	}))
	if err != nil {
		t.Fatalf("set speaker speaking: %v", err)
	}
	if len(startResp.Msg.GetView().GetSpeakers()) != 1 {
		t.Fatalf("expected 1 speaker row after start, got %d", len(startResp.Msg.GetView().GetSpeakers()))
	}
	if got := startResp.Msg.GetView().GetSpeakers()[0].GetState(); got != "SPEAKING" {
		t.Fatalf("expected SPEAKING after start, got %q", got)
	}

	doneResp, err := client.speakers.SetSpeakerDone(context.Background(), connect.NewRequest(&speakersv1.SetSpeakerDoneRequest{
		CommitteeSlug: "test-committee",
		MeetingId:     fmt.Sprintf("%d", meetingID),
		SpeakerId:     fmt.Sprintf("%d", entry.ID),
	}))
	if err != nil {
		t.Fatalf("set speaker done: %v", err)
	}
	if len(doneResp.Msg.GetView().GetSpeakers()) != 1 {
		t.Fatalf("expected done speaker to remain visible in queue, got %d rows", len(doneResp.Msg.GetView().GetSpeakers()))
	}
	if got := doneResp.Msg.GetView().GetSpeakers()[0].GetState(); got != "DONE" {
		t.Fatalf("expected DONE state after end, got %q", got)
	}
}
