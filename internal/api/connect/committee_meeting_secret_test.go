package apiconnect

import (
	"context"
	"testing"

	connect "connectrpc.com/connect"

	committeesv1 "github.com/Y4shin/open-caucus/gen/go/conference/committees/v1"
	sessionv1 "github.com/Y4shin/open-caucus/gen/go/conference/session/v1"
)

func TestCreateMeeting_GeneratesSecret(t *testing.T) {
	ts := newCombinedAPITestServer(t)
	ts.seedCommittee(t, "Secret Committee", "secret-comm")
	ts.seedUser(t, "secret-comm", "chair1", "pass123", "Chair One", "chairperson")

	client := newCombinedTestClient(t, ts)
	if _, err := client.session.Login(context.Background(), connect.NewRequest(&sessionv1.LoginRequest{
		Username: "chair1",
		Password: "pass123",
	})); err != nil {
		t.Fatalf("login: %v", err)
	}

	createResp, err := client.committees.CreateMeeting(context.Background(), connect.NewRequest(&committeesv1.CreateMeetingRequest{
		CommitteeSlug: "secret-comm",
		Name:          "Secret Test Meeting",
	}))
	if err != nil {
		t.Fatalf("create meeting: %v", err)
	}

	meetingID := createResp.Msg.GetMeeting().GetMeetingId()
	if meetingID == "" {
		t.Fatal("expected meeting id")
	}

	meetings, err := ts.repo.ListMeetingsForCommittee(context.Background(), "secret-comm", 1, 0)
	if err != nil || len(meetings) == 0 {
		t.Fatalf("list meetings: %v", err)
	}

	secret := meetings[0].Secret
	if secret == "" {
		t.Fatal("expected non-empty meeting secret, got empty string")
	}
	if len(secret) != 32 {
		t.Fatalf("expected 32-char hex secret, got %d chars: %q", len(secret), secret)
	}
}

func TestCreateMeeting_UniqueSecrets(t *testing.T) {
	ts := newCombinedAPITestServer(t)
	ts.seedCommittee(t, "Unique Committee", "unique-comm")
	ts.seedUser(t, "unique-comm", "chair1", "pass123", "Chair One", "chairperson")

	client := newCombinedTestClient(t, ts)
	if _, err := client.session.Login(context.Background(), connect.NewRequest(&sessionv1.LoginRequest{
		Username: "chair1",
		Password: "pass123",
	})); err != nil {
		t.Fatalf("login: %v", err)
	}

	for _, name := range []string{"Meeting A", "Meeting B"} {
		if _, err := client.committees.CreateMeeting(context.Background(), connect.NewRequest(&committeesv1.CreateMeetingRequest{
			CommitteeSlug: "unique-comm",
			Name:          name,
		})); err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
	}

	meetings, err := ts.repo.ListMeetingsForCommittee(context.Background(), "unique-comm", 10, 0)
	if err != nil {
		t.Fatalf("list meetings: %v", err)
	}
	if len(meetings) != 2 {
		t.Fatalf("expected 2 meetings, got %d", len(meetings))
	}
	if meetings[0].Secret == meetings[1].Secret {
		t.Fatalf("expected unique secrets, both are %q", meetings[0].Secret)
	}
}
